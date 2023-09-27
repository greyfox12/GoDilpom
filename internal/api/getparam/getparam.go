// Получаю скроку адреса сервера из переменных среды или ключа командной строки

package getparam

import (
	"flag"
	"fmt"
	"os"
)

type APIParam struct {
	ServiceAddress        string
	AccurualSystemAddress string
	DSN                   string
	AccurualTimeReset     int // Время после которого сбрасывается запрос к системе начисления баллов
	IntervalAccurual      int // Интервал в секудах опроса системы начисления баллов
}

func Param(sp *APIParam) APIParam {
	//	var cfg string
	var ok bool
	var tStr string
	var cfg APIParam

	cfg.AccurualTimeReset = 120 //120 секунд
	cfg.IntervalAccurual = 10   // 10 секунд

	if cfg.ServiceAddress, ok = os.LookupEnv("RUN_ADDRESS"); !ok {
		cfg.ServiceAddress = sp.ServiceAddress
	}
	fmt.Printf("RUN_ADDRESS=%v\n", cfg.ServiceAddress)

	if cfg.AccurualSystemAddress, ok = os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); !ok {
		cfg.AccurualSystemAddress = sp.AccurualSystemAddress
	}

	flag.StringVar(&cfg.ServiceAddress, "a", cfg.ServiceAddress, "Endpoint server IP address host:port")
	flag.StringVar(&cfg.DSN, "d", sp.DSN, "Database URI")
	flag.StringVar(&cfg.AccurualSystemAddress, "r", cfg.AccurualSystemAddress, "Accurual System Address")

	flag.Parse()

	if tStr, ok = os.LookupEnv("DATABASE_URI"); ok {
		fmt.Printf("LookupEnv(DATABASE_URI)=%v\n", tStr)
		cfg.DSN = tStr
	}

	fmt.Printf("After key (ADDRESS)=%v\n", cfg.ServiceAddress)
	fmt.Printf("After key (DATABASE_URI)=%v\n", cfg.DSN)
	fmt.Printf("After key cfg.AccurualSystemAddress=%v\n", cfg.AccurualSystemAddress)

	//	fmt.Printf("os.Args=%v\n", os.Args)
	//	fmt.Printf("os.Environ=%v\n", os.Environ())
	return cfg
}
