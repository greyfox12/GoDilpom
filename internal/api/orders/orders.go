// Загрузка Номеров Заказов
package orders

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/greyfox12/GoDiplom/internal/api/dbstore"
	"github.com/greyfox12/GoDiplom/internal/api/hash"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
)

func LoadOrderPage(db *sql.DB, authGen hash.AuthGen) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		body := make([]byte, 1000)
		var err error

		if req.Method != http.MethodPost {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Получаю токен авторизации
		login, cod := authGen.CheckAuth(req.Header.Get("Authorization"))
		if cod != 0 {
			logmy.OutLog(fmt.Errorf("orders: error autorization"))
			res.WriteHeader(cod)
			return
		}

		if req.Header.Get("Content-Type") != "text/plain" {
			logmy.OutLog(fmt.Errorf("orders: incorrect content-type head: %v", req.Header.Get("Content-Type")))
			res.WriteHeader(400)
			return
		}

		n, err := req.Body.Read(body)
		if err != nil && n <= 0 {
			fmt.Printf("Error req.Body.Read(body):%v: \n", err)
			fmt.Printf("n =%v, Body: %v \n", n, body)
			logmy.OutLog(fmt.Errorf("orders: read body request: %w", err))
			res.WriteHeader(422)
			return
		}
		defer req.Body.Close()

		// получил номер заказа
		// Проверка корректности
		numeric := regexp.MustCompile(`\d`).MatchString(string(body[0:n]))
		if !numeric {
			logmy.OutLog(fmt.Errorf("orders: number incorrect: %v", string(body[0:n])))
			res.WriteHeader(422)
			return
		}

		// Проверка алгоритмом Луна
		/*		if !hash.ValidLunaStr(string(body[0:n])) {
					logmy.OutLog(fmt.Errorf("orders: number incorrect: %v", string(body[0:n])))
					res.WriteHeader(422)
					return
				}
		*/
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		ret, err := dbstore.LoadOrder(ctx, db, login, string(body[0:n]))
		if err != nil {
			fmt.Printf("Error orders %v \n", err)
			logmy.OutLog(fmt.Errorf("orders: db load_order: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		res.WriteHeader(ret) // тк нет возврата тела - сразу ответ без ZIP
		res.Write(nil)
	}
}
