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

	"github.com/greyfox12/GoDiplom/internal/api/dbstore"
	"github.com/greyfox12/GoDiplom/internal/api/hash"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
)

type TRequest struct {
	Order string `json:"order"`
	Sum   int    `json:"sum"`
}

func DebitingPage(db *sql.DB, authGen hash.AuthGen) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		body := make([]byte, 1000)
		var err error
		var vRequest TRequest

		if req.Method != http.MethodPost {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Получаю токен авторизации
		login, cod := authGen.CheckAuth(req.Header.Get("Authorization"))
		if cod != 0 {
			logmy.OutLog(fmt.Errorf("debitingpage: error autorization"))
			res.WriteHeader(cod)
			return
		}

		if req.Header.Get("Content-Type") != "application/json" {
			logmy.OutLog(fmt.Errorf("debitingpage: incorrect content-type head: %v", req.Header.Get("Content-Type")))
			res.WriteHeader(400)
			return
		}

		n, err := req.Body.Read(body)
		if err != nil && n <= 0 {
			fmt.Printf("Error req.Body.Read(body):%v: \n", err)
			fmt.Printf("n =%v, Body: %v \n", n, body)
			logmy.OutLog(fmt.Errorf("debitingpage: read body request: %w", err))
			res.WriteHeader(422)
			return
		}
		defer req.Body.Close()

		err = json.Unmarshal(body[0:n], &vRequest)
		if err != nil {
			fmt.Printf("Error decode %v \n", err)
			logmy.OutLog(fmt.Errorf("debitingpage: decode json: %w", err))
			res.WriteHeader(422)
			return
		}
		if vRequest.Order == "" || vRequest.Sum == 0 {
			fmt.Printf("Error order/sum %v/%v \n", vRequest.Order, vRequest.Sum)
			logmy.OutLog(fmt.Errorf("debitingpage: order/sum: %v/%v", vRequest.Order, vRequest.Sum))
			res.WriteHeader(422)
			return
		}

		// получил номер заказа
		// Проверка корректности
		numeric := regexp.MustCompile(`\d`).MatchString(vRequest.Order)
		if !numeric {
			logmy.OutLog(fmt.Errorf("debitingpage: number incorrect: %v", vRequest.Order))
			res.WriteHeader(422)
			return
		}

		// Проверка алгоритмом Луна
		if !hash.ValidLunaStr(vRequest.Order) {
			logmy.OutLog(fmt.Errorf("debitingpage: number incorrect: %v", vRequest.Order))
			res.WriteHeader(422)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		ret, err := dbstore.Debits(ctx, db, login, vRequest.Order, vRequest.Sum)
		if err != nil {
			fmt.Printf("Error orders %v \n", err)
			logmy.OutLog(fmt.Errorf("debitingpage: db debits: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		res.WriteHeader(ret) // тк нет возврата тела - сразу ответ без ZIP
		res.Write(nil)
	}
}
