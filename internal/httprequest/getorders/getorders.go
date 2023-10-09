// получение списка загруженных пользователем номеров заказов, статусов их обработки и информации о начислениях
package getorders

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

func GetOrdersPage(db *sql.DB, cfg getparam.APIParam) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		namefunc := "getorderspage"
		logmy.OutLogDebug(fmt.Errorf("enter in GetOrdersPage"))

		if req.Method != http.MethodGet {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutContexDB)*time.Second)
		defer cancel()

		login := req.Header.Get("LoginUser")
		if login == "" {
			logmy.OutLogWarn(fmt.Errorf("%v: error autorization", namefunc))
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Проверка логина по базе
		userID, err := dbcommon.TestLogin(ctx, db, cfg, login)
		if err != nil {
			logmy.OutLogError(fmt.Errorf("%v: db testLogin: %w", namefunc, err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		str, ret := dbstore.GetOrder(ctx, db, cfg, userID)
		if ret != 0 {
			res.WriteHeader(ret)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		res.Write([]byte(str))
	}
}
