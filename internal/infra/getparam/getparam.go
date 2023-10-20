// Получаю скроку адреса сервера из переменных среды или ключа командной строки

package getparam

import (
	"flag"
	"fmt"
	"os"
)

// Config struct
type APIParam struct {
	ServiceAddress        string
	AccurualSystemAddress string
	DSN                   string
	AccurualTimeReset     int    // Время после которого сбрасывается запрос к системе начисления баллов
	IntervalAccurual      int    // Интервал в секудах опроса системы начисления баллов
	LogLevel              string // Уровень логирования
	TimeoutContexDB       int    // сек. Таймаут для контекста работы c DB
}

func Param(sp *APIParam) APIParam {
	var ok bool
	var tStr string
	var cfg APIParam

	// Копирую параметры в конфигурацию, пока их не настраиваем отдельно
	cfg.AccurualTimeReset = sp.AccurualTimeReset
	cfg.IntervalAccurual = sp.IntervalAccurual
	cfg.TimeoutContexDB = sp.TimeoutContexDB

	if cfg.ServiceAddress, ok = os.LookupEnv("RUN_ADDRESS"); !ok {
		cfg.ServiceAddress = sp.ServiceAddress
	}
	//	fmt.Printf("RUN_ADDRESS=%v\n", cfg.ServiceAddress)

	if cfg.AccurualSystemAddress, ok = os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); !ok {
		cfg.AccurualSystemAddress = sp.AccurualSystemAddress
	}

	if cfg.LogLevel, ok = os.LookupEnv("LOGING_LEVEL"); !ok {
		cfg.LogLevel = sp.LogLevel
	}

	flag.StringVar(&cfg.ServiceAddress, "a", cfg.ServiceAddress, "Endpoint server IP address host:port")
	flag.StringVar(&cfg.DSN, "d", sp.DSN, "Database URI")
	flag.StringVar(&cfg.AccurualSystemAddress, "r", cfg.AccurualSystemAddress, "Accurual System Address")
	flag.StringVar(&cfg.LogLevel, "l", cfg.LogLevel, "Loging level")

	flag.Parse()

	if tStr, ok = os.LookupEnv("DATABASE_URI"); ok {
		fmt.Printf("LookupEnv(DATABASE_URI)=%v\n", tStr)
		cfg.DSN = tStr
	}

	fmt.Printf("Config parametrs:\n")
	fmt.Printf("Http server ADDRESS=%v\n", cfg.ServiceAddress)
	fmt.Printf("DATABASE_URI=%v\n", cfg.DSN)
	fmt.Printf("http AccurualSystemAddress=%v\n", cfg.AccurualSystemAddress)
	fmt.Printf("LogLevel=%v\n", cfg.LogLevel)
	fmt.Printf("AccurualTimeReset=%v\n", cfg.AccurualTimeReset)
	fmt.Printf("IntervalAccurual=%v\n", cfg.IntervalAccurual)
	fmt.Printf("TimeoutContexDB=%v\n", cfg.TimeoutContexDB)

	return cfg
}
