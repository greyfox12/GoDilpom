// Взаимодейстаие с системой расчета вознаграждкемя
package accrual

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/greyfox12/GoDiplom/internal/controllers"
	"github.com/greyfox12/GoDiplom/internal/infra/getparam"
)

type TRequest struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual"`
}

// Запрашиваю базу номер заказа
// Основной модуль
func OrderNumberGet(c *controllers.BaseController) error {
	namefunc := "orderNumberGet"

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Cfg.TimeoutContexDB)*time.Second)
	defer cancel()

	str, err := getOrderDB(ctx, c)
	if err != nil {
		c.Loger.OutLogError(fmt.Errorf("%v: getOrderExec: %w", namefunc, err))
		return err
	}
	if str == "" {
		c.Loger.OutLogDebug(fmt.Errorf("%v: getOrderExec: orderNumber is null", namefunc))
		return nil
	}

	bk, err := getRequestHTTP(ctx, str, c.Cfg)
	if err != nil {
		c.Loger.OutLogInfo(fmt.Errorf("%v: getRequest: orderNum=%v %w", namefunc, str, err))

		cn, err1 := resetOrdersDB(ctx, c, str)
		c.Loger.OutLogInfo(fmt.Errorf("%v: resetOrdersDB: reset %v orders", namefunc, cn))
		if err1 != nil {
			c.Loger.OutLogError(fmt.Errorf("%v: resetOrdersDB: orderNum=%v %w", namefunc, str, err))
		}
		return err
	}

	err = saveOrdersDB(ctx, c, bk.Order, bk.Status, bk.Accrual)
	if err != nil {
		c.Loger.OutLogError(fmt.Errorf("%v: saveOrdersDB: orderNum=%v %w", namefunc, str, err))

		cn, err1 := resetOrdersDB(ctx, c, str)
		c.Loger.OutLogInfo(fmt.Errorf("%v: resetOrdersDB: reset %v orders", namefunc, cn))
		if err1 != nil {
			c.Loger.OutLogError(fmt.Errorf("%v: resetOrdersDB: orderNum=%v %w", namefunc, str, err))
		}
		return err
	}
	return nil
}

// Отправить Запрос http
func getRequestHTTP(ctx context.Context, orderNum string, cfg *getparam.APIParam) (*TRequest, error) {
	var bk TRequest

	client := &http.Client{
		Timeout: time.Second * 20,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", cfg.AccurualSystemAddress+"/api/orders/"+orderNum, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	//	fmt.Printf("Head response: %v\n", response.Header)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request status: %v", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	defer response.Body.Close()

	if err != nil {
		return nil, err
	}

	//	logmy.OutLogDebug(fmt.Errorf("client response body: %v", string(body)))

	err = json.Unmarshal(body, &bk)
	if err != nil {
		return nil, fmt.Errorf("unmarshal body: %v", string(body))
	}

	return &bk, nil
}

// Получить строку для расчета балов из БД
func getOrderDB(ctx context.Context, c *controllers.BaseController) (string, error) {

	var ordNumber string

	rows, err := c.DB.QueryDBRet(ctx,
		`with q as (select o.order_number, id from orders o 
		where order_status = 'NEW' or 
			 (order_status = 'PROCESSING' and trunc(EXTRACT( 
				EPOCH from now() -o.update_at)) > $1 ) 
		order by update_at 
		limit 1 
		for update nowait) 
		update orders o 
		 set order_status = 'PROCESSING', 
			 update_at  = now() 
			 from Q  where o.id = q.id 
		returning o.order_number `,
		c.Cfg.AccurualTimeReset)

	if err != nil {
		return "", fmt.Errorf("execute select query: %w", err) // внутренняя ошибка сервера
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&ordNumber)

		if err == nil {
			return ordNumber, nil
		}
	}

	if errors.Is(err, sql.ErrNoRows) { // Нет строк
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("scan select query: %w", err) // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		return "", fmt.Errorf("fetch rows: %w", err) // внутренняя ошибка сервера
	}
	return ordNumber, nil
}

// Сборсить не обработанные задания в ночальное состояние
// конкретное и все с истекшим тайаутом на обработку
func resetOrdersDB(ctx context.Context, c *controllers.BaseController, orderNum string) (int, error) {

	rows, err := c.DB.ResendDB(ctx,
		`UPDATE orders SET order_status = 'NEW', update_at = now() 
			 WHERE order_number = $1 OR (order_status = 'PROCESSING' and trunc(EXTRACT( 
			 EPOCH from now() - update_at)) > $2 )`,
		orderNum, c.Cfg.AccurualTimeReset)

	if err != nil {
		return 0, fmt.Errorf("execute select query: %w", err) // внутренняя ошибка сервера
	}

	return rows, nil
}

//if rows > 0 {
//	logmy.OutLogInfo(fmt.Errorf("get db resetorders function: reset %v orders", rows))
//}

// Добавить новое начисление баллов
func saveOrdersDB(ctx context.Context, c *controllers.BaseController, order string, status string, accrual float32) error {

	//	logmy.OutLogDebug(fmt.Errorf("get db SetOrdes function: order:%v status:%v accrual:%v", order, status, accrual))
	rows, err := c.DB.ResendDB(ctx,
		`with sel as (select o.user_id, order_number  from orders o  where o.order_number = $1), 
			 up1  as (update user_ref 
				set ballans  = ballans  + $3 
				where user_id = (select user_id from sel) 
				returning user_id) 
				update orders 
				  set order_status = $2, 
					  accrual  = $3, 
					  update_at = now() 
				where order_number = $1 
				  and user_id = (select user_id from up1)`,
		order, status, accrual)

	if err != nil {
		return fmt.Errorf("execute update query: %w", err) // внутренняя ошибка сервера
	}

	if rows == 0 {
		return fmt.Errorf("no update rows") // внутренняя ошибка сервера
	}

	return nil
}
