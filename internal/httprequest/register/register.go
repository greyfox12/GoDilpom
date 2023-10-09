package register

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/hash"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
	"github.com/greyfox12/GoDiplom/internal/db/dbstore"
)

type TRegister struct {
	Login        string `json:"login"`
	Password     string `json:"password"`
	PasswordHash string
}

func RegisterPage(db *sql.DB, cfg getparam.APIParam, authGen hash.AuthGen) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		var vRegister TRegister
		var err error
		namefunc := "registerpage"

		ctx := context.Background()
		body := make([]byte, 1000)
		logmy.OutLogDebug(fmt.Errorf("enter in RegisterPage"))

		if req.Method != http.MethodPost {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		n, err := req.Body.Read(body)
		if err != nil && n <= 0 {
			logmy.OutLogDebug(fmt.Errorf("%v: read body n: %v, Body: %v", namefunc, n, body))
			logmy.OutLogWarn(fmt.Errorf("%v: read body request: %w", namefunc, err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		bodyS := body[0:n]

		err = json.Unmarshal(bodyS, &vRegister)
		if err != nil {
			logmy.OutLogWarn(fmt.Errorf("%v: decode json: %w", namefunc, err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		if vRegister.Login == "" || vRegister.Password == "" {
			logmy.OutLogWarn(fmt.Errorf("%v login or password empty: login/passwd: %v/%v", namefunc, vRegister.Login, vRegister.Password))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		logmy.OutLogDebug(fmt.Errorf("%v vRegister =%v", namefunc, vRegister))

		if vRegister.PasswordHash, err = hash.GetBcryptHash(vRegister.Password); err != nil {
			logmy.OutLogError(fmt.Errorf("%v hash password: %w", namefunc, err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		ret, err := dbstore.Register(ctx, db, cfg, vRegister.Login, vRegister.PasswordHash)
		if err != nil {
			logmy.OutLogError(fmt.Errorf("%v: db register: %w", namefunc, err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if ret == http.StatusOK {
			token, err := authGen.CreateToken(vRegister.Login)
			if err != nil {
				logmy.OutLogError(fmt.Errorf("%v: create token: %w", namefunc, err))
				res.WriteHeader(http.StatusBadRequest)
				return
			}

			logmy.OutLogDebug(fmt.Errorf("%v: create token=%v", namefunc, token))

			res.Header().Set("Authorization", "Bearer "+token)
		}
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(ret) // тк нет возврата тела - сразу ответ без ZIP
		res.Write(nil)
	}
}
