package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Хендлер - Логин POST

type TRegister struct {
	Login        string `json:"login"`
	Password     string `json:"password"`
	PasswordHash string
}

func (c *BaseController) postLogin() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		var vRegister TRegister
		namefunc := "postLogin"

		c.Loger.OutLogDebug(fmt.Errorf("enter in %v", namefunc))

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Cfg.TimeoutContexDB)*time.Second)
		defer cancel()

		c.Loger.OutLogDebug(fmt.Errorf("req.Header: %v", req.Header.Get("Content-Encoding")))

		body, err := io.ReadAll(req.Body)
		defer req.Body.Close()

		if err != nil {
			c.Loger.OutLogDebug(fmt.Errorf("%v: read body, Body: %v", namefunc, body))
			c.Loger.OutLogWarn(fmt.Errorf("%v: read body request: %w", namefunc, err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(body) <= 0 {
			c.Loger.OutLogWarn(fmt.Errorf("%v: read empty body request", namefunc))
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(body, &vRegister)
		if err != nil {
			c.Loger.OutLogWarn(fmt.Errorf("%v: decode json: %w", namefunc, err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		if vRegister.Login == "" || vRegister.Password == "" {
			c.Loger.OutLogWarn(fmt.Errorf("%v: empty login/passwd: %v/%v", namefunc, vRegister.Login, vRegister.Password))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		c.Loger.OutLogDebug(fmt.Errorf("%v: login/passwd: %v/%v", namefunc, vRegister.Login, vRegister.Password))

		dbHash, err := loging(ctx, c, vRegister.Login)
		if err != nil {
			c.Loger.OutLogInfo(fmt.Errorf("%v: db loging: %w", namefunc, err))
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "application/json")

		if dbHash == "" {
			c.Loger.OutLogInfo(fmt.Errorf("%v: login %v not found in db", namefunc, vRegister.Login))
			res.WriteHeader(http.StatusUnauthorized) //401
			return
		}

		if err = bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(vRegister.Password)); err != nil {
			c.Loger.OutLogInfo(fmt.Errorf("%v: compare password and hash incorrect", namefunc))
			res.WriteHeader(http.StatusUnauthorized) //401
			return
		}

		token, err := c.Auth.CreateToken(vRegister.Login)
		if err != nil {
			c.Loger.OutLogError(fmt.Errorf("%v: create token: %w", namefunc, err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		c.Loger.OutLogDebug(fmt.Errorf("%v: create token=%v", namefunc, token))

		res.Header().Set("Authorization", "Bearer "+token)
		res.WriteHeader(http.StatusOK) // тк нет возврата тела - сразу ответ без ZIP

		res.Write(nil)
	}
}

// Авторизация пользователя. Возвращаю хранимый хеш пароля по логину
func loging(ctx context.Context, c *BaseController, login string) (string, error) {
	var ret string

	rows, err := c.DB.QueryDBRet(ctx,
		`select user_pass  from  user_ref 	where  login = $1`,
		login)

	if err != nil {
		return "", fmt.Errorf("execute select query: %w", err) // внутренняя ошибка сервера
	}
	defer rows.Close()

	rows.Next()
	err = rows.Scan(&ret)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("scan select query: %w", err) // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		return "", fmt.Errorf("fetch rows: %w", err) // внутренняя ошибка сервера
	}
	//	fmt.Printf("ret=%v", ret)
	return ret, nil
}
