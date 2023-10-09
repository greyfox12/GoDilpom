// Загрузка Номеров Заказов
package orders

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/hash"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
	"github.com/greyfox12/GoDiplom/internal/db/dbcommon"
	"github.com/greyfox12/GoDiplom/internal/db/dbstore"
)

func LoadOrderPage(db *sql.DB, cfg getparam.APIParam, authGen hash.AuthGen) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		body := make([]byte, 1000)

		logmy.OutLogDebug(fmt.Errorf("enter in LoadOrderPage"))

		if req.Method != http.MethodPost {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutContexDB)*time.Second)
		defer cancel()

		// логин из  токена авторизации
		login, cod := authGen.CheckAuth(req.Header.Get("Authorization"))
		if cod != 0 {
			logmy.OutLogWarn(fmt.Errorf("debitingpage: error autorization"))
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Проверка логина по базе
		userID, err := dbcommon.TestLogin(ctx, db, cfg, login)
		if err != nil {
			logmy.OutLogError(fmt.Errorf("orders: db testLogin: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if userID == 0 { // Логин не найден в базе
			logmy.OutLogWarn(fmt.Errorf("orders: error autorization"))
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		if req.Header.Get("Content-Type") != "text/plain" {
			logmy.OutLogInfo(fmt.Errorf("orders: incorrect content-type head: %v", req.Header.Get("Content-Type")))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		n, err := req.Body.Read(body)
		if err != nil && n <= 0 {
			logmy.OutLogDebug(fmt.Errorf("orders: read body n: %v, Body: %v", n, body))
			logmy.OutLogWarn(fmt.Errorf("orders: read body request: %w", err))
			res.WriteHeader(http.StatusUnprocessableEntity) //422
			return
		}
		defer req.Body.Close()

		// получил номер заказа
		// Проверка корректности
		numeric := regexp.MustCompile(`\d`).MatchString(string(body[0:n]))
		if !numeric {
			logmy.OutLogInfo(fmt.Errorf("orders: number incorrect symbol: %v", string(body[0:n])))
			res.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		// Проверка алгоритмом Луна
		if !hash.ValidLunaStr(string(body[0:n])) {
			logmy.OutLogInfo(fmt.Errorf("orders: number incorrect luna: %v", string(body[0:n])))
			res.WriteHeader(http.StatusUnprocessableEntity) //422
			return
		}

		ret, err := dbstore.LoadOrder(ctx, db, cfg, login, userID, string(body[0:n]))
		if err != nil {
			logmy.OutLogError(fmt.Errorf("orders: db loadorder: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		logmy.OutLogDebug(fmt.Errorf("orders load ok: number: %v ret:%v", string(body[0:n]), ret))
		res.WriteHeader(ret) // тк нет возврата тела - сразу ответ без ZIP
		res.Write(nil)
	}
}
