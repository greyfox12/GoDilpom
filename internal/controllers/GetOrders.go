// получение списка загруженных пользователем номеров заказов, статусов их обработки и информации о начислениях
package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func (c *BaseController) getOrders(res http.ResponseWriter, req *http.Request) {

	namefunc := "getOrders"
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Cfg.TimeoutContexDB)*time.Second)
	defer cancel()

	str, err := getOrderDB(ctx, c, userID)
	if err != nil {
		c.Loger.OutLogError(fmt.Errorf("%v: db getOrderDB: %w", namefunc, err))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(str) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return // 204
	}

	jsonData, err := json.Marshal(str)
	if err != nil {
		c.Loger.OutLogError(fmt.Errorf("%v: Marshal: %w", namefunc, err))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonData)
}

// Список нарядoв
type tOrders struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float32 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

func getOrderDB(ctx context.Context, c *BaseController, userID int) ([]*tOrders, error) {

	var tm time.Time
	rows, err := c.DB.QueryDBRet(ctx,
		`select o.order_number, o.order_status, o.uploaded_at, o.accrual 
		    from orders o 
		    where o.user_id = $1 
		 order by o.uploaded_at `,
		userID)

	if err != nil {
		return nil, fmt.Errorf("execute select query: %w", err) // внутренняя ошибка сервера
	}
	defer rows.Close()

	stats := make([]*tOrders, 0)
	for rows.Next() {
		bk := new(tOrders)
		err := rows.Scan(&bk.Number, &bk.Status, &tm, &bk.Accrual)
		if err != nil {
			return nil, fmt.Errorf("scan select query: %w", err) // внутренняя ошибка сервера
		}

		bk.UploadedAt = tm.Format(time.RFC3339)
		err = rows.Err()
		if err != nil {
			return nil, fmt.Errorf("fetch rows: %w", err) // внутренняя ошибка сервера
		}
		stats = append(stats, bk)
	}

	return stats, nil
}
