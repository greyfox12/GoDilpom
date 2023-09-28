package dbstore

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/lib/pq"
)

// Создаю объекты БД
func CreateDB(db *sql.DB) error {
	var script string
	//	var errdb error
	var path string

	pwd, _ := os.Getwd()
	fmt.Printf("Create DB\n")

	// заглушка по путям для выполнения на сервере или локально
	if strings.HasPrefix(pwd, "c:\\GoDiplom") {
		path = "../../internal/api/dbstore/Script.sql"
	} else {
		path = "./internal/api/dbstore/Script.sql"
	}

	file, err := os.Open(path)
	if err != nil {
		logmy.OutLog(fmt.Errorf("create db schema: open file: %w", err))
		return error(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		script = script + scanner.Text() + "\n"
	}

	if err := scanner.Err(); err != nil {
		logmy.OutLog(fmt.Errorf("create db schema: scanner file: %w", err))
		return error(err)
	}

	Errdb := ResendDB(db, script)

	if Errdb != nil {
		logmy.OutLog(fmt.Errorf("create db schema: execute script: %w", Errdb))
		return error(Errdb)
	}

	return nil
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

func ResendDB(db *sql.DB, Script string) error {
	var Errdb error
	var pgErr *pgconn.PgError

	for i := 1; i <= 4; i++ {
		if i > 1 {
			fmt.Printf("Pause: %v sec\n", WaitSec(i-1))
			time.Sleep(time.Duration(WaitSec(i-1)) * time.Second)
		}

		//		fmt.Printf("script =%v\n", Script)
		_, Errdb = db.Exec(Script)
		if Errdb == nil {
			return nil
		}

		// Проверяю тип ошибки
		logmy.OutLog(fmt.Errorf("error db create sheme: %w", Errdb))
		//		fmt.Printf("Error DB: %v\n", Errdb)

		if errors.As(Errdb, &pgErr) {
			if !pgerrcode.IsConnectionException(pgErr.Code) {
				return Errdb // Ошибка не коннекта
			}
		}
	}
	return Errdb
}

// Прочитать данные из DB
// Повторяю Чтение
func QueryDBRet(ctx context.Context, db *sql.DB, sql string, ids ...any) (*sql.Rows, error) {
	var err error
	var pgErr *pgconn.PgError

	for i := 1; i <= 4; i++ {
		if i > 1 {
			fmt.Printf("Pause: %v sec\n", WaitSec(i-1))
			time.Sleep(time.Duration(WaitSec(i-1)) * time.Second)
		}

		rows, err := db.QueryContext(ctx, sql, ids...)
		if err == nil {
			return rows, nil
		}

		// Проверяю тип ошибки
		fmt.Printf("Error DB: %v\n", err)

		if errors.As(err, &pgErr) {

			if !pgerrcode.IsConnectionException(pgErr.Code) {
				return nil, err // Ошибка не коннекта
			}
		}
	}
	return nil, err
}

// Регистрация пользователя
func Register(ctx context.Context, db *sql.DB, login string, passwd string) (int, error) {
	var ret int

	rows, err := QueryDBRet(ctx, db, "SELECT register($1, $2) counter", login, passwd)
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db register function: execute select query: %w", err))
		return 500, err // внутренняя ошибка сервера
	}
	defer rows.Close()

	rows.Next()
	err = rows.Scan(&ret)
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db register function: scan select query: %w", err))
		return 500, err // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db register function: fetch rows: %w", err))
		return 500, err // внутренняя ошибка сервера
	}
	return ret, nil
}

// Авторизация пользователя
func Loging(ctx context.Context, db *sql.DB, login string, passwd string) (int, error) {
	var ret int

	rows, err := QueryDBRet(ctx, db, "SELECT loging($1, $2) counter", login, passwd)
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db loging function: execute select query: %w", err))
		return 500, err // внутренняя ошибка сервера
	}
	defer rows.Close()

	rows.Next()
	err = rows.Scan(&ret)
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db loging function: scan select query: %w", err))
		return 500, err // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db loging function: fetch rows: %w", err))
		return 500, err // внутренняя ошибка сервера
	}
	return ret, nil
}

// Загрузка наряда
func LoadOrder(ctx context.Context, db *sql.DB, login string, ordNum string) (int, error) {
	var ret int

	rows, err := QueryDBRet(ctx, db, "SELECT load_order($1, $2) counter", login, ordNum)
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db loadorder function: execute select query: %w", err))
		return 500, err // внутренняя ошибка сервера
	}
	defer rows.Close()

	rows.Next()
	err = rows.Scan(&ret)
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db loadorder function: scan select query: %w", err))
		return 500, err // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db logloadorderng function: fetch rows: %w", err))
		return 500, err // внутренняя ошибка сервера
	}
	return ret, nil
}

type tOrders struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float32 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

// Список нарядoв
func GetOrder(ctx context.Context, db *sql.DB, login string) ([]*tOrders, int) {

	var tm time.Time
	rows, err := QueryDBRet(ctx, db, "select o.order_number, o.order_status, o.uploaded_at, o.accrual "+
		" from orders o "+
		"	join user_ref ur on ur.user_id = o.user_id "+
		" where ur.login = $1 "+
		" order by o.uploaded_at ", login)

	if err != nil {
		logmy.OutLog(fmt.Errorf("get db getorder function: execute select query: %w", err))
		return nil, 500 // внутренняя ошибка сервера
	}
	defer rows.Close()

	stats := make([]*tOrders, 0)
	for rows.Next() {
		bk := new(tOrders)
		err := rows.Scan(&bk.Number, &bk.Status, &tm, &bk.Accrual)
		if err != nil {
			logmy.OutLog(fmt.Errorf("get db getorder function: scan select query: %w", err))
			return nil, 500 // внутренняя ошибка сервера
		}

		bk.UploadedAt = tm.Format(time.RFC3339)
		err = rows.Err()
		if err != nil {
			logmy.OutLog(fmt.Errorf("get db getorder function: fetch rows: %w", err))
			return nil, 500 // внутренняя ошибка сервера
		}
		stats = append(stats, bk)
	}

	if len(stats) == 0 {
		return nil, 204
	}
	return stats, 0
}

type tBallance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

// BAllanc
func GetBalance(ctx context.Context, db *sql.DB, login string) ([]byte, int) {

	rows, err := QueryDBRet(ctx, db, "select ur.ballans, ur.withdrawn "+
		" from user_ref ur "+
		" where ur.login = $1 ", login)

	if err != nil {
		logmy.OutLog(fmt.Errorf("get db getbalance function: execute select query: %w", err))
		return nil, 500 // внутренняя ошибка сервера
	}
	defer rows.Close()

	rows.Next()
	bk := new(tBallance)
	err = rows.Scan(&bk.Current, &bk.Withdrawn)

	if err != nil {
		logmy.OutLog(fmt.Errorf("get db getbalance function: scan select query: %w", err))
		return nil, 500 // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db getbalance function: fetch rows: %w", err))
		return nil, 500 // внутренняя ошибка сервера
	}

	jsonData, err := json.Marshal(bk)
	if err != nil {
		return nil, 500
	}
	return jsonData, 0
}

// Списание балов
func Debits(ctx context.Context, db *sql.DB, login string, ordNum string, summ float32) (int, error) {
	var ret int

	rows, err := QueryDBRet(ctx, db, "SELECT debeting($1, $2, $3) counter", login, ordNum, summ)
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db loadorder function: execute select query: %w", err))
		return 500, err // внутренняя ошибка сервера
	}
	defer rows.Close()

	rows.Next()
	err = rows.Scan(&ret)
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db loadorder function: scan select query: %w", err))
		return 500, err // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db logloadorderng function: fetch rows: %w", err))
		return 500, err // внутренняя ошибка сервера
	}
	return ret, nil
}

type tWithdrawals struct {
	Order       string  `json:"number"`
	Sum         float32 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

// Список Списаний балов
func GetWithdrawals(ctx context.Context, db *sql.DB, login string) ([]*tWithdrawals, int) {

	var tm time.Time
	rows, err := QueryDBRet(ctx, db, "select w.order_number, w.summa, w.uploaded_at "+
		" from withdraw w "+
		"	join user_ref ur on ur.user_id = w.user_id "+
		" where ur.login = $1 "+
		" order by w.uploaded_at ", login)

	if err != nil {
		logmy.OutLog(fmt.Errorf("get db getwithdrawals query: execute select query: %w", err))
		return nil, 500 // внутренняя ошибка сервера
	}
	defer rows.Close()

	stats := make([]*tWithdrawals, 0)
	for rows.Next() {
		bk := new(tWithdrawals)
		err := rows.Scan(&bk.Order, &bk.Sum, &tm)
		if err != nil {
			logmy.OutLog(fmt.Errorf("get db getwithdrawals query: scan select query: %w", err))
			return nil, 500 // внутренняя ошибка сервера
		}

		bk.ProcessedAt = tm.Format(time.RFC3339)
		err = rows.Err()
		if err != nil {
			logmy.OutLog(fmt.Errorf("get db getwithdrawals query: fetch rows: %w", err))
			return nil, 500 // внутренняя ошибка сервера
		}
		stats = append(stats, bk)
	}

	if len(stats) == 0 {
		return nil, 204
	}
	return stats, 0
}

// Получить строку для расчета балов
func GetOrderExec(ctx context.Context, db *sql.DB) (string, error) {

	var ordNumber string
	rows, err := QueryDBRet(ctx, db, "select get_order()")

	if err != nil {
		logmy.OutLog(fmt.Errorf("get db getorderexec function: execute select query: %w", err))
		return "", err // внутренняя ошибка сервера
	}
	defer rows.Close()

	rows.Next()
	err = rows.Scan(&ordNumber)

	if err != nil {
		logmy.OutLog(fmt.Errorf("get db getbalance function: scan select query: %w", err))
		return "", err // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db getbalance function: fetch rows: %w", err))
		return "", err // внутренняя ошибка сервера
	}

	return ordNumber, nil
}

// Сборсить не обработанные задания в ночальное состояние
func ResetOrders(ctx context.Context, db *sql.DB, orderNum string, cfg getparam.APIParam) error {

	rows, err := QueryDBRet(ctx, db, "UPDATE orders SET order_status = 'NEW', update_at = now() "+
		" WHERE order_number = $1 OR (order_status = 'PROCESSING' and trunc(EXTRACT( "+
		" EPOCH from now() - update_at)) > $2 )", orderNum, cfg.AccurualTimeReset)

	if err != nil {
		logmy.OutLog(fmt.Errorf("get db resetorders function: execute select query: %w", err))
		return err // внутренняя ошибка сервера
	}
	defer rows.Close()

	err = rows.Err()
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db resetorders function: fetch rows: %w", err))
		return err // внутренняя ошибка сервера
	}

	return nil
}

// Добавить новое начисление баллов
func SetOrders(ctx context.Context, db *sql.DB, order string, status string, accrual float32) error {
	var ret int

	logmy.OutLog(fmt.Errorf("get db SetOrdes function: order:%v status:%v accrual:%v", order, status, accrual))
	rows, err := QueryDBRet(ctx, db, "select add_accrual($1, $2, $3)", order, status, accrual)

	if err != nil {
		logmy.OutLog(fmt.Errorf("get db SetOrdes function: execute select query: %w", err))
		return err // внутренняя ошибка сервера
	}
	defer rows.Close()

	rows.Next()
	err = rows.Scan(&ret)
	if err != nil || ret != 0 {
		logmy.OutLog(fmt.Errorf("get db SetOrdes function: scan select query: %w", err))
		return err // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		logmy.OutLog(fmt.Errorf("get db SetOrdes function: fetch rows: %w", err))
		return err // внутренняя ошибка сервера
	}

	return nil
}
