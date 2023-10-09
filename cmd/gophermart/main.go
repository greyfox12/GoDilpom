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

	"github.com/greyfox12/GoDiplom/internal/api/compress"
	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/hash"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
	"github.com/greyfox12/GoDiplom/internal/db/dbstore"
	"github.com/greyfox12/GoDiplom/internal/getaccrual/client"
	"github.com/greyfox12/GoDiplom/internal/httprequest/erroreq"
	"github.com/greyfox12/GoDiplom/internal/httprequest/getbalance"
	"github.com/greyfox12/GoDiplom/internal/httprequest/getorders"
	"github.com/greyfox12/GoDiplom/internal/httprequest/getwithdrawals"
	"github.com/greyfox12/GoDiplom/internal/httprequest/loging"
	"github.com/greyfox12/GoDiplom/internal/httprequest/orders"
	"github.com/greyfox12/GoDiplom/internal/httprequest/postwithdraw"
	"github.com/greyfox12/GoDiplom/internal/httprequest/register"
)

// Конфигурация по умолчанию
const (
	defServiceAddress        = "localhost:8080"
	defDSN                   = "host=localhost user=videos password=videos dbname=postgres sslmode=disable"
	defAccurualSystemAddress = "http://localhost:8090"
	defLogLevel              = "Debug"
	defAccurualTimeReset     = 120 //120 секунд - Время до сброса в БД состояния отправленных на обработку ордеров
	defIntervalAccurual      = 1   // 1 секунд - Задержка перед циклом выбора для отправки на обработку ордеров
	defTimeoutContexDB       = 10  // сек. Таймаут для контекста работы c DB
)

func main() {

	serverStart()
}

// Запускаю сервер
func serverStart() {
	var db *sql.DB
	var authGen hash.AuthGen
	var err error

	// Собрать конфигурацию приложения из Умолчаний, ключей и Переменнных среды
	apiParam := getparam.APIParam{
		ServiceAddress:        defServiceAddress,
		AccurualSystemAddress: defAccurualSystemAddress,
		DSN:                   defDSN,
		LogLevel:              defLogLevel,
		AccurualTimeReset:     defAccurualTimeReset,
		IntervalAccurual:      defIntervalAccurual,
		TimeoutContexDB:       defTimeoutContexDB,
	}
	// запрашиваю параметры ключей-переменных окружения
	apiParam = getparam.Param(&apiParam)

	// Инициализирую логирование
	if ok := logmy.Initialize(apiParam.LogLevel); ok != nil {
		panic(ok)
	}

	// Подключение к БД
	db, err = sql.Open("pgx", apiParam.DSN)
	if err != nil {
		logmy.OutLogFatal(err)
		panic(err)
	}
	defer db.Close()

	if err = dbstore.CreateDB(db, apiParam); err != nil {
		logmy.OutLogFatal(err)
		panic(err)
	}

	// Инициация шифрования
	if err = authGen.Init(); err != nil {
		logmy.OutLogFatal(err)
		panic(err)
	}

	// запускаю Опрос системы начисления баллов
	if apiParam.IntervalAccurual > 0 {
		go func(*sql.DB, getparam.APIParam) {

			if apiParam.IntervalAccurual > 0 {
				ticker := time.NewTicker(time.Second * time.Duration(apiParam.IntervalAccurual))
				defer ticker.Stop()
				for {
					client.GetOrderNumber(db, apiParam)
					<-ticker.C
				}
			}
		}(db, apiParam)
	}

	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)
	//	r.Use(hash.Autoriz)

	// определяем хендлер
	r.Route("/", func(r chi.Router) {
		//получение списка загруженных пользователем номеров заказов, статусов их обработки и информации о начислениях
		r.Get("/api/user/orders", logmy.RequestLogger(getorders.GetOrdersPage(db, apiParam)))
		//получение текущего баланса счёта баллов лояльности пользователя
		r.Get("/api/user/balance", logmy.RequestLogger(getbalance.GetBalancePage(db, apiParam)))
		//запрос на списание баллов с накопительного счёта в счёт оплаты нового заказа
		r.Get("/api/user/withdrawals", logmy.RequestLogger(getwithdrawals.GetWithdrawalsPage(db, apiParam)))

		r.Get("/*", logmy.RequestLogger(erroreq.ErrorReq))

		// регистрация пользователя
		r.Post("/api/user/register", logmy.RequestLogger(register.RegisterPage(db, apiParam, authGen)))
		//аутентификация пользователя
		r.Post("/api/user/login", logmy.RequestLogger(loging.LoginPage(db, apiParam, authGen)))
		//загрузка пользователем номера заказа для расчёта
		r.Post("/api/user/orders", logmy.RequestLogger(orders.LoadOrderPage(db, apiParam)))
		//запрос на списание баллов с накопительного счёта в счёт оплаты нового заказа
		r.Post("/api/user/balance/withdraw", logmy.RequestLogger(postwithdraw.DebitingPage(db, apiParam)))

		r.Post("/*", logmy.RequestLogger(erroreq.ErrorReq))

	})

	fmt.Printf("Start Server %v\n", apiParam.ServiceAddress)

	hd := compress.GzipHandle(compress.GzipRead(r))
	log.Fatal(http.ListenAndServe(apiParam.ServiceAddress, hash.Autoriz(hd, authGen)))
}
