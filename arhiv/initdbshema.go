package infradb

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/greyfox12/GoDiplom/internal/controllers"
)

// Создаю объекты БД
func InitDBShema(c controllers.BaseController) error {
	var script string
	var path string

	c.Loger.OutLogDebug(fmt.Errorf("create DB shema"))

	pwd, _ := os.Getwd()
	fmt.Printf("pwd=%v\n", pwd)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Cfg.TimeoutContexDB)*time.Second)
	defer cancel()

	// заглушка по путям для выполнения на сервере или локально
	if strings.HasPrefix(pwd, "c:\\GoDiplom") {
		path = "../../internal/db/dbstore/Script.sql"
	} else {
		path = "./internal/db/dbstore/Script.sql"
	}

	file, err := os.Open(path)
	if err != nil {
		c.Loger.OutLogFatal(fmt.Errorf("create db schema: open file: %w", err))
		return error(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		script = script + scanner.Text() + "\n"
	}

	if err := scanner.Err(); err != nil {
		c.Loger.OutLogFatal(fmt.Errorf("create db schema: scanner file: %w", err))
		return error(err)
	}

	_, Errdb := c.DB.ResendDB(ctx, script)

	if Errdb != nil {
		c.Loger.OutLogFatal(fmt.Errorf("create db schema: execute script: %w", Errdb))
		return error(Errdb)
	}

	return nil
}
