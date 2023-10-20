// Получение списка списания баллов
package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Список Списаний балов
type tWithdrawals struct {
	Order       string  `json:"order"`
	Sum         float32 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

func (c *BaseController) getWithdrawals(res http.ResponseWriter, req *http.Request) {

	namefunc := "getWithdrawals"
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

	str, err := withdrawalsGetDB(ctx, c, userID)
	if err != nil {
		c.Loger.OutLogWarn(fmt.Errorf("%v: withdrawalsGetDB: %w", namefunc, err))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(str) == 0 {
		c.Loger.OutLogInfo(fmt.Errorf("%v: withdrawalsGetDB: empty", namefunc))
		res.WriteHeader(http.StatusNoContent)
		return //204
	}

	jsonData, err := json.Marshal(str)
	if err != nil {
		c.Loger.OutLogWarn(fmt.Errorf("%v: convert JSON: %w", namefunc, err))
		res.WriteHeader(http.StatusInternalServerError)
		return //500
	}

	c.Loger.OutLogDebug(fmt.Errorf("%v login: %v return: %v", namefunc, login, str))
	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonData)
}

// Выборка
func withdrawalsGetDB(ctx context.Context, c *BaseController, userID int) ([]*tWithdrawals, error) {

	var tm time.Time
	rows, err := c.DB.QueryDBRet(ctx,
		`select w.order_number, w.summa, w.uploaded_at 
		   from withdraw w 
	       where w.user_id = $1 
		 order by w.uploaded_at`,
		userID)

	if err != nil {
		return nil, fmt.Errorf("execute select query: %w", err) // внутренняя ошибка сервера
	}
	defer rows.Close()

	stats := make([]*tWithdrawals, 0)
	for rows.Next() {
		bk := new(tWithdrawals)
		err := rows.Scan(&bk.Order, &bk.Sum, &tm)
		if err != nil {
			return nil, fmt.Errorf("scan select query: %w", err) // внутренняя ошибка сервера
		}

		bk.ProcessedAt = tm.Format(time.RFC3339)
		err = rows.Err()
		if err != nil {
			return nil, fmt.Errorf("fetch rows: %w", err) // внутренняя ошибка сервера
		}
		stats = append(stats, bk)
	}

	return stats, nil
}
