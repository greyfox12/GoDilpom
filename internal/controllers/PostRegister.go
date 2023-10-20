package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func (c *BaseController) postRegister() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		var vRegister TRegister
		namefunc := "postRegister"

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Cfg.TimeoutContexDB)*time.Second)
		defer cancel()

		c.Loger.OutLogDebug(fmt.Errorf("enter in %v", namefunc))

		body, err := io.ReadAll(req.Body)
		defer req.Body.Close()

		if err != nil {
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
			c.Loger.OutLogWarn(fmt.Errorf("%v login or password empty: login/passwd: %v/%v", namefunc, vRegister.Login, vRegister.Password))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		c.Loger.OutLogDebug(fmt.Errorf("%v vRegister =%v", namefunc, vRegister))

		if vRegister.PasswordHash, err = c.Auth.GetBcryptHash(vRegister.Password); err != nil {
			c.Loger.OutLogError(fmt.Errorf("%v hash password: %w", namefunc, err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		ret, err := registerDB(ctx, c, vRegister.Login, vRegister.PasswordHash)
		if err != nil {
			c.Loger.OutLogError(fmt.Errorf("%v: db register: %w", namefunc, err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if ret == http.StatusOK {
			token, err := c.Auth.CreateToken(vRegister.Login)
			if err != nil {
				c.Loger.OutLogError(fmt.Errorf("%v: create token: %w", namefunc, err))
				res.WriteHeader(http.StatusBadRequest)
				return
			}

			c.Loger.OutLogDebug(fmt.Errorf("%v: create token=%v", namefunc, token))

			res.Header().Set("Authorization", "Bearer "+token)
		}
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(ret) // тк нет возврата тела - сразу ответ без ZIP
		res.Write(nil)
	}
}

// Регистрация пользователя
func registerDB(ctx context.Context, c *BaseController, login string, passwd string) (int, error) {

	rows, err := c.DB.ResendDB(ctx,
		`INSERT INTO user_ref (login, user_pass) VALUES ($1,  $2) 
		  ON CONFLICT (login) DO NOTHING 
		  returning user_id`,
		login, passwd)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("execute insert query: %w", err) // внутренняя ошибка сервера
	}

	if rows == 0 {
		return http.StatusConflict, nil
	}
	return http.StatusOK, nil
}
