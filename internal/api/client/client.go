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

	"github.com/greyfox12/GoDiplom/internal/api/dbstore"
	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
)

type TRequest struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

// Отправить Запрос
func GetRequest(orderNum string, cfg getparam.ApiParam) (*TRequest, error) {
	var bk TRequest

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest("GET", cfg.AccurualSystemAddress+"/api/orders/"+orderNum, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Head response: %v\n", response.Header)

	if response.StatusCode != 200 {
		logmy.OutLog(fmt.Errorf("client status request: %v", response.StatusCode))
		return nil, fmt.Errorf("client status request: %v", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	defer response.Body.Close()

	if err != nil {
		return nil, err
	}

	fmt.Println("response Body:", string(body))
	logmy.OutLog(fmt.Errorf("client response body: %v", string(body)))

	err = json.Unmarshal(body, &bk)
	if err != nil {
		return nil, err
	}

	return &bk, nil
}

// Повторяю при ошибках вывод
func Resend(orderNum string, cfg getparam.ApiParam) (*TRequest, error) {
	var err error
	var bk *TRequest

	for i := 1; i <= 4; i++ {
		if i > 1 {
			logmy.OutLog(fmt.Errorf("client pause: %v sec", WaitSec(i-1)))
			time.Sleep(time.Duration(WaitSec(i-1)) * time.Second)
		}

		bk, err = GetRequest(orderNum, cfg)
		if err == nil {
			return bk, nil
		}

		logmy.OutLog(fmt.Errorf("post send message: %w", err))
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
func GetOrderNumber(db *sql.DB, cfg getparam.ApiParam) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	str, err := dbstore.GetOrderExec(ctx, db)
	if err != nil {
		logmy.OutLog(fmt.Errorf("client: getOrderExec: %w", err))
		return err
	}
	if str == "" {
		logmy.OutLog(fmt.Errorf("client: getOrderExec: orderNumber is null"))
		return nil
	}

	bk, err := Resend(str, cfg)
	if err != nil {
		logmy.OutLog(fmt.Errorf("client: get data accrpall: orderNum=%v %w", str, err))
		dbstore.ResetOrders(ctx, db, str, cfg)
		return err
	}

	err = dbstore.SetOrders(ctx, db, bk.Order, bk.Status, bk.Accrual)
	if err != nil {
		logmy.OutLog(fmt.Errorf("client: save accrpall: orderNum=%v %w", str, err))
		dbstore.ResetOrders(ctx, db, str, cfg)
		return err
	}
	return nil
}
