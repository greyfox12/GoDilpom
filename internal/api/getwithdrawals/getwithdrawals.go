// Получение списка списания баллов
package getwithdrawals

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/greyfox12/GoDiplom/internal/api/dbstore"
	"github.com/greyfox12/GoDiplom/internal/api/hash"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
)

func GetWithdrawalsPage(db *sql.DB, authGen hash.AuthGen) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		//		fmt.Printf("OneMetricPage \n")

		if req.Method != http.MethodGet {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Получаю токен авторизации
		login, cod := authGen.CheckAuth(req.Header.Get("Authorization"))
		if cod != 0 {
			logmy.OutLog(fmt.Errorf("withdrawals: error autorization"))
			res.WriteHeader(cod)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		str, ret := dbstore.GetWithdrawals(ctx, db, login)
		if ret != 0 {
			res.WriteHeader(ret)
			return
		}

		jsonData, err := json.Marshal(str)
		if err != nil {
			res.WriteHeader(500)
			return
		}
		logmy.OutLog(fmt.Errorf("withdrawals login: %v return: %v", login, string(jsonData)))
		res.Header().Set("Content-Type", "application/json")
		res.Write([]byte(jsonData))
	}
}
