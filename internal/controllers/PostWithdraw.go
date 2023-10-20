// Списание баллов с баланса
package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

type TRequest struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

func (c *BaseController) postWithdraw(res http.ResponseWriter, req *http.Request) {

	namefunc := "postWithdraw"
	var err error
	var vRequest TRequest

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

	if userID == 0 { // Логин не найден в базе
		c.Loger.OutLogWarn(fmt.Errorf("%v: error autorization", namefunc))
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	if req.Header.Get("Content-Type") != "application/json" {
		c.Loger.OutLogInfo(fmt.Errorf("%v: incorrect content-type head: %v", namefunc, req.Header.Get("Content-Type")))
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	defer req.Body.Close()

	if err != nil {
		c.Loger.OutLogWarn(fmt.Errorf("%v: read body request: %w", namefunc, err))
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if len(body) <= 0 {
		c.Loger.OutLogWarn(fmt.Errorf("%v: read empty body request", namefunc))
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	err = json.Unmarshal(body, &vRequest)
	if err != nil {
		c.Loger.OutLogWarn(fmt.Errorf("%v: decode json: %w", namefunc, err))
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if vRequest.Order == "" || vRequest.Sum == 0 {
		c.Loger.OutLogWarn(fmt.Errorf("%v: empty order/sum: %v/%v", namefunc, vRequest.Order, vRequest.Sum))
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	// получил номер заказа
	// Проверка корректности
	numeric := regexp.MustCompile(`\d`).MatchString(vRequest.Order)
	if !numeric {
		c.Loger.OutLogWarn(fmt.Errorf("%v: number incorrect: %v", namefunc, vRequest.Order))
		res.WriteHeader(http.StatusUnprocessableEntity) //422
		return
	}

	// Проверка алгоритмом Луна
	/*		if !hash.ValidLunaStr(vRequest.Order) {
				logmy.OutLog(fmt.Errorf("debitingpage: number incorrect: %v", vRequest.Order))
				res.WriteHeader(422)
				return
			}
	*/

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Cfg.TimeoutContexDB)*time.Second)
	defer cancel()

	retCod, err := debits(ctx, c, userID, vRequest.Order, vRequest.Sum)
	if err != nil {
		c.Loger.OutLogError(fmt.Errorf("%v: db debits: %w", namefunc, err))
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	res.WriteHeader(retCod) // тк нет возврата тела - сразу ответ без ZIP
	res.Write(nil)
}

// Списание балов
func debits(ctx context.Context, c *BaseController, userID int, ordNum string, summ float32) (int, error) {
	var pBallans float32

	// запросить балланс
	rows, err := c.DB.QueryDBRet(ctx, `select ballans  from  user_ref where  user_id = $1`, userID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("execute select query ballans: %w", err) // внутренняя ошибка сервера
	}
	defer rows.Close()

	rows.Next()
	err = rows.Scan(&pBallans)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("scan select query ballans: %w", err) // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("fetch rows ballans: %w", err) // внутренняя ошибка сервера
	}

	if pBallans < summ {
		c.Loger.OutLogDebug(fmt.Errorf("ballans < summ"))
		return http.StatusPaymentRequired, nil // внутренняя ошибка сервера
	}

	// Корректирую балланс и пишу журнал списания
	_, err = c.DB.ResendDB(ctx,
		`with ins as (insert into withdraw (user_id, order_number, summa) VALUES($1, $2, $3) returning user_id, summa) 
			update user_ref u 
					   set withdrawn = withdrawn + $3, 
						   ballans  = ballans - $3 
					   where u.user_id = (select user_id from ins) `,
		userID, ordNum, summ)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("execute insert query: %w", err) // внутренняя ошибка сервера
	}

	return http.StatusOK, nil
}
