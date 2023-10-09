package getbalance

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
	"github.com/greyfox12/GoDiplom/internal/db/dbcommon"
	"github.com/greyfox12/GoDiplom/internal/db/dbstore"
)

func GetBalancePage(db *sql.DB, cfg getparam.APIParam) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		namefunc := "getbalancepage"
		logmy.OutLogDebug(fmt.Errorf("enter in GetBalancePage"))

		if req.Method != http.MethodGet {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		login := req.Header.Get("LoginUser")
		if login == "" {
			logmy.OutLogWarn(fmt.Errorf("%v: error autorization", namefunc))
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutContexDB)*time.Second)
		defer cancel()

		// Проверка логина по базе
		userID, err := dbcommon.TestLogin(ctx, db, cfg, login)
		if err != nil {
			logmy.OutLogError(fmt.Errorf("%v: db testLogin: %w", namefunc, err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if userID == 0 { // Логин не найден в базе
			logmy.OutLogWarn(fmt.Errorf("%v: error autorization", namefunc))
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		jsonData, ret := dbstore.GetBalance(ctx, db, cfg, userID)
		if ret != 0 {
			res.WriteHeader(ret)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		res.Write(jsonData)
	}
}
