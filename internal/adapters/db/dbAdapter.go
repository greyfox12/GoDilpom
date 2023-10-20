// Общие функции для рпботы с DB
package adapters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBAdapter struct {
	DB        *sql.DB
	timeoutDB int
}

// инициализирую
func (pDB *DBAdapter) Init(dbDNS string, timeout int) (*DBAdapter, error) {
	// Подключение к БД
	db, err := sql.Open("pgx", dbDNS)

	return &DBAdapter{DB: db, timeoutDB: timeout}, err
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

// Переповтор для выполнения скрипта
func (pDB *DBAdapter) ResendDB(ctx context.Context, Script string, ids ...any) (int, error) {
	//	var Errdb error
	var err error
	var pgErr *pgconn.PgError
	var result sql.Result

	//	cctx, cancel := context.WithTimeout(ctx, time.Duration(pDB.timeoutDB)*time.Second)
	//	defer cancel()

	for i := 1; i <= 4; i++ {
		if i > 1 {
			//			logmy.OutLogDebug(fmt.Errorf("pause: %v sec", WaitSec(i-1)))
			time.Sleep(time.Duration(WaitSec(i-1)) * time.Second)
		}

		//		fmt.Printf("arguments: %v", ids...)
		result, err = pDB.DB.ExecContext(ctx, Script, ids...)
		if err != nil {
			if errors.As(err, &pgErr) {
				if !pgerrcode.IsConnectionException(pgErr.Code) {
					return 0, err // Ошибка не коннекта
				}
			}
			continue
		}

		rows, err := result.RowsAffected()
		if err == nil {
			return int(rows), nil
		}

		// Проверяю тип ошибки
		//		logmy.OutLogDebug(fmt.Errorf("db resendDB: %w", Errdb))

	}
	return 0, err
}

// Прочитать данные из DB
// Повторяю Чтение
func (pDB *DBAdapter) QueryDBRet(ctx context.Context, sqlQuery string, ids ...any) (*sql.Rows, error) {
	var err error
	var pgErr *pgconn.PgError

	for i := 1; i <= 4; i++ {
		if i > 1 {
			//			logmy.OutLogDebug(fmt.Errorf("pause: %v sec", WaitSec(i-1)))
			time.Sleep(time.Duration(WaitSec(i-1)) * time.Second)
		}

		rows, err := pDB.DB.QueryContext(ctx, sqlQuery, ids...)
		if err == nil {
			return rows, nil
		}

		// Проверяю тип ошибки
		//		logmy.OutLogDebug(fmt.Errorf("db querydbret querycontext: %w", err))

		if errors.As(err, &pgErr) {

			if !pgerrcode.IsConnectionException(pgErr.Code) {
				return nil, err // Ошибка не коннекта
			}
		}
	}
	return nil, err
}

// Проверка действительности логина. Если есть - вернуть user_id или 0 если нет
func (pDB *DBAdapter) TestLogin(ctx context.Context, login string) (int, error) {
	var ret int

	rows, err := pDB.QueryDBRet(ctx, `select user_id  from  user_ref where  login = $1 `, login)

	if err != nil {
		return 0, fmt.Errorf("testLogin function: execute select query: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&ret)
		if err == sql.ErrNoRows {
			return 0, nil
		}

		if err != nil {
			return 0, fmt.Errorf("testLogin function: scan select query: %w", err)
		}
	}
	err = rows.Err()
	if err != nil {
		return 0, fmt.Errorf("testLogin function: fetch rows: %w", err)
	}

	return ret, nil
}
