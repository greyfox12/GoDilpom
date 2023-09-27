package getbalance

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/greyfox12/Gophermart/internal/api/dbstore"
	"github.com/greyfox12/Gophermart/internal/api/hash"
	"github.com/greyfox12/Gophermart/internal/api/logmy"
)

func GetBalancePage(db *sql.DB, authGen hash.AuthGen) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		//		fmt.Printf("OneMetricPage \n")

		if req.Method != http.MethodGet {
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

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		jsonData, ret := dbstore.GetBalance(ctx, db, login)
		if ret != 0 {
			res.WriteHeader(ret)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		res.Write(jsonData)
	}
}
