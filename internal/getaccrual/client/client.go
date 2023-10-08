// Взаимодейстаие с системой расчета вознаграждкемя
package client

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
	"github.com/greyfox12/GoDiplom/internal/db/dbcommon"
	"github.com/greyfox12/GoDiplom/internal/db/dbstore"
)

type TRequest struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual"`
}

// Отправить Запрос
func GetRequest(orderNum string, cfg getparam.APIParam) (*TRequest, error) {
	var bk TRequest

	client := &http.Client{
		Timeout: time.Second * 20,
	}
	req, err := http.NewRequest("GET", cfg.AccurualSystemAddress+"/api/orders/"+orderNum, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	//	fmt.Printf("Head response: %v\n", response.Header)
	logmy.OutLogDebug(fmt.Errorf("getrequest: head response: %v", response.Header))

	if response.StatusCode != http.StatusOK {
		logmy.OutLogDebug(fmt.Errorf("client status request: %v", response.StatusCode))
		return nil, fmt.Errorf("client status request: %v", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	defer response.Body.Close()

	if err != nil {
		return nil, err
	}

	logmy.OutLogDebug(fmt.Errorf("client response body: %v", string(body)))

	err = json.Unmarshal(body, &bk)
	if err != nil {
		logmy.OutLogError(fmt.Errorf("client unmarshal body: %v", string(body)))
		return nil, err
	}

	return &bk, nil
}

// Повторяю при ошибках вывод
func Resend(orderNum string, cfg getparam.APIParam) (*TRequest, error) {
	var err error
	var bk *TRequest

	for i := 1; i <= 4; i++ {
		if i > 1 {
			logmy.OutLogDebug(fmt.Errorf("client pause: %v sec", WaitSec(i-1)))
			time.Sleep(time.Duration(WaitSec(i-1)) * time.Second)
		}

		bk, err = GetRequest(orderNum, cfg)
		if err == nil {
			return bk, nil
		}

		logmy.OutLogWarn(fmt.Errorf("post send message: %w", err))
		if _, yes := err.(net.Error); !yes {
			return nil, err
		}
	}
	return nil, err
}

// Считаю задержку - по номеру повторения возвращаю длительность в сек
func WaitSec(period int) int {
	switch period {
	case 1:
		return 1
	case 2:
		return 3
	case 3:
		return 5
	default:
		return 0
	}
}

// Запрашиваю базу номер заказа
// Основной модуль
func GetOrderNumber(db *sql.DB, cfg getparam.APIParam) error {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutContexDB)*time.Second)
	defer cancel()

	str, err := dbstore.GetOrderExec(ctx, db, cfg)
	if err != nil {
		logmy.OutLogError(fmt.Errorf("client: getOrderExec: %w", err))
		return err
	}
	if str == "" {
		logmy.OutLogDebug(fmt.Errorf("client: getOrderExec: orderNumber is null"))
		return nil
	}

	bk, err := Resend(str, cfg)
	if err != nil {
		logmy.OutLogInfo(fmt.Errorf("client: get data accrpall: orderNum=%v %w", str, err))
		dbcommon.ResetOrders(ctx, db, str, cfg)
		return err
	}

	err = dbstore.SetOrders(ctx, db, bk.Order, bk.Status, bk.Accrual, cfg)
	if err != nil {
		logmy.OutLogError(fmt.Errorf("client: error save accrpall: orderNum=%v %w", str, err))
		dbcommon.ResetOrders(ctx, db, str, cfg)
		return err
	}
	return nil
}
