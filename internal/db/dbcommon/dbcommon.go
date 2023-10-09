// Общие функции для рпботы с DB
package dbcommon

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

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

// Переповтор для выполнения скрипта
func ResendDB(ctx context.Context, db *sql.DB, cfg getparam.APIParam, Script string, ids ...any) (int, error) {
	var Errdb error
	var pgErr *pgconn.PgError
	var result sql.Result

	cctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutContexDB)*time.Second)
	defer cancel()

	for i := 1; i <= 4; i++ {
		if i > 1 {
			logmy.OutLogDebug(fmt.Errorf("pause: %v sec", WaitSec(i-1)))
			time.Sleep(time.Duration(WaitSec(i-1)) * time.Second)
		}

		//		fmt.Printf("arguments: %v", ids...)
		result, Errdb = db.ExecContext(cctx, Script, ids...)
		if Errdb == nil {
			rows, err := result.RowsAffected()
			if err == nil {
				return int(rows), nil
			}
			//			return nil
		}

		// Проверяю тип ошибки
		logmy.OutLogDebug(fmt.Errorf("db resendDB: %w", Errdb))

		if errors.As(Errdb, &pgErr) {
			if !pgerrcode.IsConnectionException(pgErr.Code) {
				return 0, Errdb // Ошибка не коннекта
			}
		}
	}
	return 0, Errdb
}

// Прочитать данные из DB
// Повторяю Чтение
func QueryDBRet(ctx context.Context, db *sql.DB, cfg getparam.APIParam, sqlQuery string, ids ...any) (*sql.Rows, error) {
	var err error
	var pgErr *pgconn.PgError

	for i := 1; i <= 4; i++ {
		if i > 1 {
			logmy.OutLogDebug(fmt.Errorf("pause: %v sec", WaitSec(i-1)))
			time.Sleep(time.Duration(WaitSec(i-1)) * time.Second)
		}

		rows, err := db.QueryContext(ctx, sqlQuery, ids...)
		if err == nil {
			return rows, nil
		}

		// Проверяю тип ошибки
		logmy.OutLogDebug(fmt.Errorf("db querydbret querycontext: %w", err))

		if errors.As(err, &pgErr) {

			if !pgerrcode.IsConnectionException(pgErr.Code) {
				return nil, err // Ошибка не коннекта
			}
		}
	}
	return nil, err
}

// Проверка действительности логина. Если есть - вернуть user_id или 0 если нет
func TestLogin(ctx context.Context, db *sql.DB, cfg getparam.APIParam, login string) (int, error) {
	var ret int

	rows, err := QueryDBRet(ctx, db, cfg, "select user_id  from  user_ref "+
		"where  login = $1 ", login)

	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db TestLogin function: execute select query: %w", err))
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&ret)
		if err == sql.ErrNoRows {
			return 0, nil
		}

		if err != nil {
			logmy.OutLogError(fmt.Errorf("get db TestLogin function: scan select query: %w", err))
			return 0, err
		}
	}
	err = rows.Err()
	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db TestLogin function: fetch rows: %w", err))
		return 0, err
	}

	return ret, nil
}

// Сборсить не обработанные задания в ночальное состояние
// конкретное и все с истекшим тайаутом на обработку
func ResetOrders(ctx context.Context, db *sql.DB, orderNum string, cfg getparam.APIParam) error {

	rows, err := ResendDB(ctx, db, cfg,
		"UPDATE orders SET order_status = 'NEW', update_at = now() "+
			" WHERE order_number = $1 OR (order_status = 'PROCESSING' and trunc(EXTRACT( "+
			" EPOCH from now() - update_at)) > $2 )",
		orderNum, cfg.AccurualTimeReset)

	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db resetorders function: execute select query: %w", err))
		return err // внутренняя ошибка сервера
	}

	if rows > 0 {
		logmy.OutLogInfo(fmt.Errorf("get db resetorders function: reset %v orders", rows))
	}
	return nil
}
