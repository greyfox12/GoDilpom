package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/greyfox12/GoDiplom/internal/api/client"
	"github.com/greyfox12/GoDiplom/internal/api/compress"
	"github.com/greyfox12/GoDiplom/internal/api/dbstore"
	"github.com/greyfox12/GoDiplom/internal/api/erroreq"
	"github.com/greyfox12/GoDiplom/internal/api/getbalance"
	"github.com/greyfox12/GoDiplom/internal/api/getorders"
	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/getwithdrawals"
	"github.com/greyfox12/GoDiplom/internal/api/hash"
	"github.com/greyfox12/GoDiplom/internal/api/loging"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
	"github.com/greyfox12/GoDiplom/internal/api/orders"
	"github.com/greyfox12/GoDiplom/internal/api/postwithdraw"
	"github.com/greyfox12/GoDiplom/internal/api/register"
)

const (
	defServiceAddress        = "localhost:8080"
	defDSN                   = "host=localhost user=videos password=videos dbname=postgres sslmode=disable"
	defAccurualSystemAddress = "http://localhost:8090"
)

func main() {

	serverStart()
}

// Запускаю сервер
func serverStart() {
	var db *sql.DB
	var authGen hash.AuthGen

	apiParam := getparam.ApiParam{
		ServiceAddress:        defServiceAddress,
		AccurualSystemAddress: defAccurualSystemAddress,
		DSN:                   defDSN,
	}
	// запрашиваю параметры ключей-переменных окружения
	apiParam = getparam.Param(&apiParam)

	// Инициализирую логирование
	if ok := logmy.Initialize("info"); ok != nil {
		panic(ok)
	}

	// Подключение к БД
	var err error
	fmt.Printf("DSN: %v\n", apiParam.DSN)
	db, err = sql.Open("pgx", apiParam.DSN)
	if err != nil {
		logmy.OutLog(err)
		fmt.Printf("Error connect DB: %v\n", err)
	}
	defer db.Close()

	if err = dbstore.CreateDB(db); err != nil {
		logmy.OutLog(err)
		panic(err)
	}

	// Инициация шифрования
	if err = authGen.Init(); err != nil {
		logmy.OutLog(err)
		panic(err)
	}

	// запускаю Опрос системы начисления баллов

	go func(*sql.DB, getparam.ApiParam) {

		ticker := time.NewTicker(time.Second * time.Duration(apiParam.IntervalAccurual))
		defer ticker.Stop()
		for {
			<-ticker.C
			client.GetOrderNumber(db, apiParam)
		}
	}(db, apiParam)

	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)

	// определяем хендлер
	r.Route("/", func(r chi.Router) {
		//получение списка загруженных пользователем номеров заказов, статусов их обработки и информации о начислениях
		r.Get("/api/user/orders", logmy.RequestLogger(getorders.GetOrdersPage(db, authGen)))
		//получение текущего баланса счёта баллов лояльности пользователя
		r.Get("/api/user/balance", logmy.RequestLogger(getbalance.GetBalancePage(db, authGen)))
		//запрос на списание баллов с накопительного счёта в счёт оплаты нового заказа
		r.Get("/api/user/withdrawals", logmy.RequestLogger(getwithdrawals.GetWithdrawalsPage(db, authGen)))

		r.Get("/*", logmy.RequestLogger(erroreq.ErrorReq))

		// регистрация пользователя
		r.Post("/api/user/register", logmy.RequestLogger(register.RegisterPage(db, authGen)))
		//аутентификация пользователя
		r.Post("/api/user/login", logmy.RequestLogger(loging.LoginPage(db, authGen)))
		//загрузка пользователем номера заказа для расчёта
		r.Post("/api/user/orders", logmy.RequestLogger(orders.LoadOrderPage(db, authGen)))
		//запрос на списание баллов с накопительного счёта в счёт оплаты нового заказа
		r.Post("/api/user/balance/withdraw", logmy.RequestLogger(postwithdraw.DebitingPage(db, authGen)))

		r.Post("/*", logmy.RequestLogger(erroreq.ErrorReq))

	})

	fmt.Printf("Start Server %v\n", apiParam.ServiceAddress)

	hd := compress.GzipHandle(compress.GzipRead(r))
	log.Fatal(http.ListenAndServe(apiParam.ServiceAddress, hd))
}
