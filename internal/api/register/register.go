package register

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/greyfox12/Gophermart/internal/api/dbstore"
	"github.com/greyfox12/Gophermart/internal/api/hash"
	"github.com/greyfox12/Gophermart/internal/api/logmy"
)

type TRegister struct {
	Login       string `json:"login"`
	Password    string `json:"password"`
	PasswordMD5 string
}

func RegisterPage(db *sql.DB, authGen hash.AuthGen) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		var vRegister TRegister
		body := make([]byte, 1000)
		var err error

		if req.Method != http.MethodPost {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		fmt.Printf("req.Header: %v \n", req.Header.Get("Content-Encoding"))

		n, err := req.Body.Read(body)
		if err != nil && n <= 0 {
			fmt.Printf("Error req.Body.Read(body):%v: \n", err)
			fmt.Printf("n =%v, Body: %v \n", n, body)
			logmy.OutLog(fmt.Errorf("registerpage: read body request: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		bodyS := body[0:n]

		err = json.Unmarshal(bodyS, &vRegister)
		if err != nil {
			fmt.Printf("Error decode %v \n", err)
			logmy.OutLog(fmt.Errorf("registerpage: decode json: %w", err))
			res.WriteHeader(400)
			return
		}
		if vRegister.Login == "" || vRegister.Password == "" {
			fmt.Printf("Error login/passwd %v/%v \n", vRegister.Login, vRegister.Password)
			logmy.OutLog(fmt.Errorf("registerpage: login/passwd: %w/%w", vRegister.Login, vRegister.Password))
			res.WriteHeader(400)
			return
		}

		fmt.Printf("vRegister =%v\n", vRegister)

		vRegister.PasswordMD5 = hash.GetMD5Hash(vRegister.Password)
		ret, err := dbstore.Register(ctx, db, vRegister.Login, vRegister.PasswordMD5)
		if err != nil {
			fmt.Printf("Error register %v \n", err)
			logmy.OutLog(fmt.Errorf("registerpage: db register: %w", err))
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if ret == 200 {
			token, err := authGen.CreateToken(vRegister.Login)
			if err != nil {
				logmy.OutLog(fmt.Errorf("registerpage: create token: %w", err))
				res.WriteHeader(http.StatusBadRequest)
				return
			}

			fmt.Printf("token=%v\n", token)

			res.Header().Set("Authorization", "Bearer "+token)
		}
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(ret) // тк нет возврата тела - сразу ответ без ZIP
		res.Write(nil)
	}
}
