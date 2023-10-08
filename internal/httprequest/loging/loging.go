// Страница авторизации
package loging

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/hash"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
	"github.com/greyfox12/GoDiplom/internal/db/dbstore"
	"golang.org/x/crypto/bcrypt"
)

type TRegister struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func LoginPage(db *sql.DB, cfg getparam.APIParam, authGen hash.AuthGen) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		var vRegister TRegister
		body := make([]byte, 1000)
		var err error

		logmy.OutLogDebug(fmt.Errorf("enter in LoginPage"))

		if req.Method != http.MethodPost {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutContexDB)*time.Second)
		defer cancel()

		logmy.OutLogDebug(fmt.Errorf("req.Header: %v", req.Header.Get("Content-Encoding")))

		n, err := req.Body.Read(body)
		if err != nil && n <= 0 {
			logmy.OutLogDebug(fmt.Errorf("logingpage: read body n: %v, Body: %v", n, body))
			logmy.OutLogWarn(fmt.Errorf("logingpage: read body request: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		bodyS := body[0:n]

		err = json.Unmarshal(bodyS, &vRegister)
		if err != nil {
			logmy.OutLogWarn(fmt.Errorf("logingpage: decode json: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		if vRegister.Login == "" || vRegister.Password == "" {
			logmy.OutLogWarn(fmt.Errorf("logingpage: empty login/passwd: %v/%v", vRegister.Login, vRegister.Password))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		logmy.OutLogDebug(fmt.Errorf("logingpage: login/passwd: %v/%v", vRegister.Login, vRegister.Password))

		dbHash, err := dbstore.Loging(ctx, db, cfg, vRegister.Login)
		if err != nil {
			logmy.OutLogInfo(fmt.Errorf("logingpage: db loging: %w", err))
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "application/json")

		if dbHash == "" {
			logmy.OutLogInfo(fmt.Errorf("logingpage: login %v not found in db", vRegister.Login))
			res.WriteHeader(http.StatusUnauthorized) //401
			return
		}

		if err = bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(vRegister.Password)); err != nil {
			logmy.OutLogInfo(fmt.Errorf("logingpage: compare password and hash incorrect"))
			res.WriteHeader(http.StatusUnauthorized) //401
			return
		}

		token, err := authGen.CreateToken(vRegister.Login)
		if err != nil {
			logmy.OutLogError(fmt.Errorf("logingpage: create token: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		logmy.OutLogDebug(fmt.Errorf("logingpage: create token=%v", token))

		res.Header().Set("Authorization", "Bearer "+token)
		res.WriteHeader(http.StatusOK) // тк нет возврата тела - сразу ответ без ZIP

		res.Write(nil)
	}
}
