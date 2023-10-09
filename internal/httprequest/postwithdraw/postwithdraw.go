// Списание баллов с баланса
package postwithdraw

import (
	"context"
	"database/sql"
	"encoding/json"
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

type TRequest struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

func DebitingPage(db *sql.DB, cfg getparam.APIParam, authGen hash.AuthGen) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		body := make([]byte, 1000)
		var err error
		var vRequest TRequest

		logmy.OutLogDebug(fmt.Errorf("enter in DebitingPage"))

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
			logmy.OutLogError(fmt.Errorf("debitingpage: db testLogin: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if userID == 0 { // Логин не найден в базе
			logmy.OutLogWarn(fmt.Errorf("debitingpage: error autorization"))
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		if req.Header.Get("Content-Type") != "application/json" {
			logmy.OutLogInfo(fmt.Errorf("debitingpage: incorrect content-type head: %v", req.Header.Get("Content-Type")))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		n, err := req.Body.Read(body)
		if err != nil && n <= 0 {
			logmy.OutLogDebug(fmt.Errorf("debitingpage: read body n: %v, Body: %v", n, body))
			logmy.OutLogWarn(fmt.Errorf("debitingpage: read body request: %w", err))
			res.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		defer req.Body.Close()

		err = json.Unmarshal(body[0:n], &vRequest)
		if err != nil {
			logmy.OutLogWarn(fmt.Errorf("debitingpage: decode json: %w", err))
			res.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		if vRequest.Order == "" || vRequest.Sum == 0 {
			logmy.OutLogWarn(fmt.Errorf("debitingpage: empty order/sum: %v/%v", vRequest.Order, vRequest.Sum))
			res.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		// получил номер заказа
		// Проверка корректности
		numeric := regexp.MustCompile(`\d`).MatchString(vRequest.Order)
		if !numeric {
			logmy.OutLogWarn(fmt.Errorf("debitingpage: number incorrect: %v", vRequest.Order))
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

		ret, err := dbstore.Debits(ctx, db, cfg, login, userID, vRequest.Order, vRequest.Sum)
		if err != nil {
			logmy.OutLogError(fmt.Errorf("debitingpage: db debits: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		res.WriteHeader(ret) // тк нет возврата тела - сразу ответ без ZIP
		res.Write(nil)
	}
}
