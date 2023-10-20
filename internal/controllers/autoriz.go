package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Autoriz — middleware-авторизация для входящих HTTP-запросов.

func (c *BaseController) Autoriz(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		c.Loger.OutLogDebug(fmt.Errorf("enter in Autoriz"))
		// Пропускаю авторизацию для регистрации и логина
		//		if r.URL.String() == "/api/user/register" || r.URL.String() == "/api/user/login" {
		//			next.ServeHTTP(w, r)
		//			return
		//		}

		// Получаю токен авторизации
		login, err := c.Auth.CheckAuth(r.Header.Get("Authorization"))
		if err != nil {
			c.Loger.OutLogWarn(fmt.Errorf("autorization: error autorization: %w", err))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		c.Loger.OutLogDebug(fmt.Errorf("autorization: login %v ", login))
		// Добавляю логин для дальнейшего использования в хендлере
		r.Header.Add("LoginUser", login)

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Cfg.TimeoutContexDB)*time.Second)
		defer cancel()

		userID, err := c.DB.TestLogin(ctx, login)
		if err != nil {
			c.Loger.OutLogWarn(fmt.Errorf("autorization: error get userID: %w", err))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Добавляю логин для дальнейшего использования в хендлере
		r.Header.Add("UserID", fmt.Sprint(userID))

		next.ServeHTTP(w, r)
	})
}
