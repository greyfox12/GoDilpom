package main

import (
	"context"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"

	accrual "github.com/greyfox12/GoDiplom/internal/GetAccrual"
	adapters "github.com/greyfox12/GoDiplom/internal/adapters/db"
	"github.com/greyfox12/GoDiplom/internal/controllers"
	infradb "github.com/greyfox12/GoDiplom/internal/infra/db"
	"github.com/greyfox12/GoDiplom/internal/infra/getparam"
	"github.com/greyfox12/GoDiplom/internal/infra/hash"
	"github.com/greyfox12/GoDiplom/internal/infra/logmy"
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
	//	var db *sql.DB
	var authGen hash.AuthGen
	var DBadapter adapters.DBAdapter
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
	log := new(logmy.Log)
	log, ok := log.Initialize(apiParam.LogLevel)
	if ok != nil {
		panic(ok)
	}

	// Подключение к БД
	adap, err := DBadapter.Init(apiParam.DSN, apiParam.TimeoutContexDB)
	if err != nil {
		log.OutLogFatal(err)
		panic(err)
	}
	defer adap.DB.Close()

	// Инициация шифрования
	if err = authGen.Init(); err != nil {
		log.OutLogFatal(err)
		panic(err)
	}

	controller := controllers.NewBaseController(adap, &apiParam, log, &authGen)
	// Миграция схемы
	if err = infradb.MigrateSchema(controller.DB.DB); err != nil {
		log.OutLogFatal(err)
		panic(err)
	}

	r := controller.Route()
	//	log.OutLogInfo(fmt.Errorf("start Server %v", apiParam.ServiceAddress))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// запускаю Опрос системы начисления баллов
	if apiParam.IntervalAccurual > 0 {
		go func() {
			accrual.RunAccurual(ctx, controller)
		}()
	}

	// Запускаю сервер HTTP
	if err := controllers.RunServer(ctx, controller, r); err != nil {
		log.OutLogFatal(err)
	}
}
