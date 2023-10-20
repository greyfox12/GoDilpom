package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type tBallance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

func (c *BaseController) getBalance(res http.ResponseWriter, req *http.Request) {

	namefunc := "getBalance"
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

	bk, err := balanceGetDB(ctx, c, userID)
	if err != nil {
		c.Loger.OutLogWarn(fmt.Errorf("%v: balanceGetDB: %w", namefunc, err))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(bk)
	if err != nil {
		c.Loger.OutLogWarn(fmt.Errorf("%v: convert JSON: %w", namefunc, err))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonData)
}

// BAllanc
func balanceGetDB(ctx context.Context, c *BaseController, userID int) (*tBallance, error) {
	var bk tBallance

	rows, err := c.DB.QueryDBRet(ctx,
		`select ur.ballans, ur.withdrawn 
		   from user_ref ur 
		   where ur.user_id = $1 `, userID)

	if err != nil {
		return nil, fmt.Errorf("execute select query: %w", err) // внутренняя ошибка сервера
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&bk.Current, &bk.Withdrawn)

		if err != nil {
			return nil, fmt.Errorf("scan select query: %w", err) // внутренняя ошибка сервера
		}
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("fetch rows: %w", err) // внутренняя ошибка сервера
	}

	return &bk, nil
}
