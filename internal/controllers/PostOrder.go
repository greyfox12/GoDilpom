// Загрузка Номеров Заказов
package controllers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/greyfox12/GoDiplom/internal/infra/hash"
)

func (c *BaseController) postOrder(res http.ResponseWriter, req *http.Request) {

	namefunc := "postOrder"

	c.Loger.OutLogDebug(fmt.Errorf("enter in %v", namefunc))

	login := req.Header.Get("LoginUser")
	if login == "" {
		c.Loger.OutLogWarn(fmt.Errorf("%v: error autorization", namefunc))
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(req.Header.Get("UserID"))
	if err != nil {
		c.Loger.OutLogWarn(fmt.Errorf("%v: error autorization: convert userID: %v", namefunc, err))
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	if req.Header.Get("Content-Type") != "text/plain" {
		c.Loger.OutLogInfo(fmt.Errorf("%v: incorrect content-type head: %v", namefunc, req.Header.Get("Content-Type")))
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	defer req.Body.Close()

	if err != nil {
		c.Loger.OutLogWarn(fmt.Errorf("%v: read body request: %w", namefunc, err))
		res.WriteHeader(http.StatusUnprocessableEntity) //422
		return
	}
	if len(body) <= 0 {
		c.Loger.OutLogWarn(fmt.Errorf("%v: read empty body request", namefunc))
		res.WriteHeader(http.StatusUnprocessableEntity) //422
		return
	}

	// получил номер заказа
	// Проверка корректности
	numeric := regexp.MustCompile(`\d`).MatchString(string(body))
	if !numeric {
		c.Loger.OutLogInfo(fmt.Errorf("%v: number incorrect symbol: %v", namefunc, string(body)))
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	// Проверка алгоритмом Луна
	if !hash.ValidLunaStr(string(body)) {
		c.Loger.OutLogInfo(fmt.Errorf("%v: number incorrect luna: %v", namefunc, string(body)))
		res.WriteHeader(http.StatusUnprocessableEntity) //422
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Cfg.TimeoutContexDB)*time.Second)
	defer cancel()

	ret, err := loadOrder(ctx, c, userID, string(body))
	if err != nil {
		c.Loger.OutLogError(fmt.Errorf("%v: db loader: %w", namefunc, err))
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	c.Loger.OutLogDebug(fmt.Errorf("%v load ok: number: %v ret:%v", namefunc, string(body), ret))
	res.WriteHeader(ret) // тк нет возврата тела - сразу ответ без ZIP
	res.Write(nil)
}

// Загрузка наряда
func loadOrder(ctx context.Context, c *BaseController, pUserID int, ordNum string) (int, error) {
	var userID int
	var userIDOrd int

	// Загрузка номера
	userID, err := c.DB.ResendDB(ctx,
		`INSERT INTO orders(user_id, order_number, order_status) 
		       VALUES ($2, $1,'NEW') 
			   ON CONFLICT (order_number) DO NOTHING
		       returning user_id `,
		ordNum, pUserID)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("execute insert query: %w", err) // внутренняя ошибка сервера
	}

	if userID > 0 {
		return http.StatusAccepted, nil
	} // Записб добавлена

	// Запись конфликтует. Ищу причину
	rows, err := c.DB.QueryDBRet(ctx,
		`select o.user_id, coalesce(u.user_id, -1)
		     from orders o 
		     left join user_ref u on u.user_id =$2 and u.user_id =o.user_id 
		   where o.order_number = $1`,
		ordNum, pUserID)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("execute select query: %w", err) // внутренняя ошибка сервера
	}
	defer rows.Close()

	if rows.Next() {

		err = rows.Scan(&userIDOrd, &userID)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("scan select query: %w", err) // внутренняя ошибка сервера
		}

		if userID == -1 {
			c.Loger.OutLogDebug(fmt.Errorf("order load othes user"))
			return http.StatusConflict, nil // загружено другим пользователем
		}
		c.Loger.OutLogDebug(fmt.Errorf("order load this user"))
		return http.StatusOK, nil // загружено этим пользователем
	}

	if err = rows.Err(); err != nil {
		c.Loger.OutLogError(fmt.Errorf("fetch rows: %w", err))
		return http.StatusInternalServerError, err // внутренняя ошибка сервера
	}

	return http.StatusInternalServerError, nil // Непонятная ошибка
}
