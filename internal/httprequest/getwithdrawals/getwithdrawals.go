// Получение списка списания баллов
package getwithdrawals

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/hash"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
	"github.com/greyfox12/GoDiplom/internal/db/dbcommon"
	"github.com/greyfox12/GoDiplom/internal/db/dbstore"
)

func GetWithdrawalsPage(db *sql.DB, cfg getparam.APIParam, authGen hash.AuthGen) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		logmy.OutLogDebug(fmt.Errorf("enter in withdrawalspage"))

		if req.Method != http.MethodGet {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Получаю токен авторизации
		login, cod := authGen.CheckAuth(req.Header.Get("Authorization"))
		if cod != 0 {
			logmy.OutLogWarn(fmt.Errorf("withdrawals: error autorization"))
			res.WriteHeader(cod)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutContexDB)*time.Second)
		defer cancel()

		// Проверка логина по базе
		userID, err := dbcommon.TestLogin(ctx, db, cfg, login)
		if err != nil {
			logmy.OutLogError(fmt.Errorf("orders: db testLogin: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		str, ret := dbstore.GetWithdrawals(ctx, db, cfg, userID)
		if ret != 0 {
			res.WriteHeader(ret)
			return
		}

		logmy.OutLogDebug(fmt.Errorf("withdrawals login: %v return: %v", login, str))
		res.Header().Set("Content-Type", "application/json")
		res.Write([]byte(str))
	}
}
