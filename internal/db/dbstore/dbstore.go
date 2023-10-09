package dbstore

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/greyfox12/GoDiplom/internal/api/getparam"
	"github.com/greyfox12/GoDiplom/internal/api/logmy"
	"github.com/greyfox12/GoDiplom/internal/db/dbcommon"
	_ "github.com/lib/pq"
)

// Создаю объекты БД
func CreateDB(db *sql.DB, cfg getparam.APIParam) error {
	var script string
	var path string

	logmy.OutLogDebug(fmt.Errorf("create DB shema"))

	pwd, _ := os.Getwd()
	ctx := context.Background()

	// заглушка по путям для выполнения на сервере или локально
	if strings.HasPrefix(pwd, "c:\\GoDiplom") {
		path = "../../internal/db/dbstore/Script.sql"
	} else {
		path = "./internal/db/dbstore/Script.sql"
	}

	file, err := os.Open(path)
	if err != nil {
		logmy.OutLogFatal(fmt.Errorf("create db schema: open file: %w", err))
		return error(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		script = script + scanner.Text() + "\n"
	}

	if err := scanner.Err(); err != nil {
		logmy.OutLogFatal(fmt.Errorf("create db schema: scanner file: %w", err))
		return error(err)
	}

	_, Errdb := dbcommon.ResendDB(ctx, db, cfg, script)

	if Errdb != nil {
		logmy.OutLogFatal(fmt.Errorf("create db schema: execute script: %w", Errdb))
		return error(Errdb)
	}

	return nil
}

// Регистрация пользователя
func Register(ctx context.Context, db *sql.DB, cfg getparam.APIParam, login string, passwd string) (int, error) {

	rows, err := dbcommon.ResendDB(ctx, db, cfg, "INSERT INTO user_ref (login, user_pass) VALUES ($1,  $2) "+
		"ON CONFLICT (login) DO NOTHING "+
		"returning user_id", login, passwd)

	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db register function: execute insert query: %w", err))
		return http.StatusInternalServerError, err // внутренняя ошибка сервера
	}

	if rows == 0 {
		return http.StatusConflict, nil
	}
	return http.StatusOK, nil
}

// Авторизация пользователя
func Loging(ctx context.Context, db *sql.DB, cfg getparam.APIParam, login string) (string, error) {
	var ret string

	rows, err := dbcommon.QueryDBRet(ctx, db, cfg, "select user_pass  from  user_ref "+
		"where  login = $1", login)
	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db loging function: execute select query: %w", err))
		return "", err // внутренняя ошибка сервера
	}
	defer rows.Close()

	rows.Next()
	err = rows.Scan(&ret)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db loging function: scan select query: %w", err))
		return "", err // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db loging function: fetch rows: %w", err))
		return "", err // внутренняя ошибка сервера
	}
	fmt.Printf("ret=%v", ret)
	return ret, nil
}

// Загрузка наряда
func LoadOrder(ctx context.Context, db *sql.DB, cfg getparam.APIParam, login string, pUserID int, ordNum string) (int, error) {
	var userID int
	var userIDOrd int

	rows, err := dbcommon.QueryDBRet(ctx, db, cfg, "select o.user_id, coalesce(u.user_id, -1)   from orders o "+
		"left join user_ref u on u.user_id =$1 and u.user_id =o.user_id "+
		"where o.order_number = $2", pUserID, ordNum)

	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db loadorder function: execute select query: %w", err))
		return http.StatusInternalServerError, err // внутренняя ошибка сервера
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&userIDOrd, &userID)

		if err != nil && err != sql.ErrNoRows {
			logmy.OutLogInfo(fmt.Errorf("get db loadorder function: scan select query: %w", err))
			return http.StatusInternalServerError, err // внутренняя ошибка сервера
		}

		if err != sql.ErrNoRows {
			if userID == -1 {
				logmy.OutLogInfo(fmt.Errorf("get db loadorder function: order load othes user"))
				return http.StatusConflict, nil // загружено другим пользователем
			}
			logmy.OutLogInfo(fmt.Errorf("get db loadorder function: order load this user"))
			return http.StatusOK, nil // загружено этим пользователем
		}
	}

	if err1 := rows.Err(); err != nil {
		logmy.OutLogError(fmt.Errorf("get db loadorder function: fetch rows: %w", err1))
		return http.StatusInternalServerError, err1 // внутренняя ошибка сервера
	}

	// Загрузка номера
	_, err = dbcommon.ResendDB(ctx, db, cfg, "INSERT INTO orders(user_id, order_number, order_status) "+
		"VALUES ($2, $1,'NEW') "+
		"returning user_id ", ordNum, pUserID)

	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db loadorder function: execute inser query: %w", err))
		return http.StatusInternalServerError, err // внутренняя ошибка сервера
	}

	return http.StatusAccepted, nil
}

type tOrders struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float32 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

// Список нарядoв
func GetOrder(ctx context.Context, db *sql.DB, cfg getparam.APIParam, userID int) (string, int) {

	var tm time.Time
	rows, err := dbcommon.QueryDBRet(ctx, db, cfg, "select o.order_number, o.order_status, o.uploaded_at, o.accrual "+
		" from orders o "+
		" where o.user_id = $1 "+
		" order by o.uploaded_at ", userID)

	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db getorder function: execute select query: %w", err))
		return "", http.StatusInternalServerError // внутренняя ошибка сервера
	}
	defer rows.Close()

	stats := make([]*tOrders, 0)
	for rows.Next() {
		bk := new(tOrders)
		err := rows.Scan(&bk.Number, &bk.Status, &tm, &bk.Accrual)
		if err != nil {
			logmy.OutLogError(fmt.Errorf("get db getorder function: scan select query: %w", err))
			return "", http.StatusInternalServerError // внутренняя ошибка сервера
		}

		bk.UploadedAt = tm.Format(time.RFC3339)
		err = rows.Err()
		if err != nil {
			logmy.OutLogError(fmt.Errorf("get db getorder function: fetch rows: %w", err))
			return "", http.StatusInternalServerError // внутренняя ошибка сервера
		}
		stats = append(stats, bk)
	}

	if len(stats) == 0 {
		return "", http.StatusNoContent // 204
	}

	jsonData, err := json.Marshal(stats)
	if err != nil {
		return "", http.StatusInternalServerError
	}

	return string(jsonData), 0
}

// BAllanc
type tBallance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

func GetBalance(ctx context.Context, db *sql.DB, cfg getparam.APIParam, userID int) ([]byte, int) {
	var bk tBallance

	rows, err := dbcommon.QueryDBRet(ctx, db, cfg, "select ur.ballans, ur.withdrawn "+
		" from user_ref ur "+
		" where ur.user_id = $1 ", userID)

	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db getbalance function: execute select query: %w", err))
		return nil, http.StatusInternalServerError // внутренняя ошибка сервера
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&bk.Current, &bk.Withdrawn)

		if err != nil {
			logmy.OutLogError(fmt.Errorf("get db getbalance function: scan select query: %w", err))
			return nil, http.StatusInternalServerError // внутренняя ошибка сервера
		}
	}

	err = rows.Err()
	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db getbalance function: fetch rows: %w", err))
		return nil, http.StatusInternalServerError // внутренняя ошибка сервера
	}

	jsonData, err := json.Marshal(bk)
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	return jsonData, 0
}

// Списание балов
func Debits(ctx context.Context, db *sql.DB, cfg getparam.APIParam, login string, userID int, ordNum string, summ float32) (int, error) {
	var pBallans float32

	// запросить балланс
	rows, err := dbcommon.QueryDBRet(ctx, db, cfg, "select ballans  from  user_ref where  user_id = $1", userID)
	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db debits function: execute select query ballans: %w", err))
		return http.StatusInternalServerError, err // внутренняя ошибка сервера
	}
	defer rows.Close()

	rows.Next()
	err = rows.Scan(&pBallans)
	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db debits function: scan select query ballans: %w", err))
		return http.StatusInternalServerError, err // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db debits function: fetch rows ballans: %w", err))
		return http.StatusInternalServerError, err // внутренняя ошибка сервера
	}

	if pBallans < summ {
		logmy.OutLogDebug(fmt.Errorf("get db ldebitsadorder function: ballans < summ"))
		return http.StatusPaymentRequired, nil // внутренняя ошибка сервера
	}

	// Корректирую балланс и пишу журнал списания
	_, err = dbcommon.ResendDB(ctx, db, cfg,
		"with ins as (insert into withdraw (user_id, order_number, summa) VALUES($1, $2, $3) returning user_id, summa) "+
			"update user_ref u "+
			"		   set withdrawn = withdrawn + $3, "+
			"			   ballans  = ballans - $3 "+
			"		   where u.user_id = (select user_id from ins) ", userID, ordNum, summ)
	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db debits function: execute inser query: %w", err))
		return http.StatusInternalServerError, err // внутренняя ошибка сервера
	}

	return http.StatusOK, nil
}

// Список Списаний балов
type tWithdrawals struct {
	Order       string  `json:"order"`
	Sum         float32 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

func GetWithdrawals(ctx context.Context, db *sql.DB, cfg getparam.APIParam, userID int) (string, int) {

	var tm time.Time
	rows, err := dbcommon.QueryDBRet(ctx, db, cfg, "select w.order_number, w.summa, w.uploaded_at "+
		" from withdraw w "+
		" where w.user_id = $1 "+
		" order by w.uploaded_at ", userID)

	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db getwithdrawals query: execute select query: %w", err))
		return "", http.StatusInternalServerError // внутренняя ошибка сервера
	}
	defer rows.Close()

	stats := make([]*tWithdrawals, 0)
	for rows.Next() {
		bk := new(tWithdrawals)
		err := rows.Scan(&bk.Order, &bk.Sum, &tm)
		if err != nil {
			logmy.OutLogError(fmt.Errorf("get db getwithdrawals query: scan select query: %w", err))
			return "", http.StatusInternalServerError // внутренняя ошибка сервера
		}

		bk.ProcessedAt = tm.Format(time.RFC3339)
		err = rows.Err()
		if err != nil {
			logmy.OutLogError(fmt.Errorf("get db getwithdrawals query: fetch rows: %w", err))
			return "", http.StatusInternalServerError // внутренняя ошибка сервера
		}
		stats = append(stats, bk)
	}

	if len(stats) == 0 {
		return "", http.StatusNoContent //204
	}

	jsonData, err := json.Marshal(stats)
	if err != nil {
		return "", http.StatusInternalServerError //500
	}

	return string(jsonData), 0
}

// Получить строку для расчета балов
func GetOrderExec(ctx context.Context, db *sql.DB, cfg getparam.APIParam) (string, error) {

	var ordNumber string

	rows, err := dbcommon.QueryDBRet(ctx, db, cfg, "with q as (select o.order_number, id from orders o "+
		"where order_status = 'NEW' or "+
		"	 (order_status = 'PROCESSING' and trunc(EXTRACT( "+
		"		EPOCH from now() -o.update_at)) > $1 ) "+
		"order by update_at "+
		"limit 1 "+
		"for update nowait) "+
		"update orders o "+
		" set order_status = 'PROCESSING', "+
		"	 update_at  = now() "+
		"	 from Q  where o.id = q.id "+
		"returning o.order_number ", cfg.AccurualTimeReset)

	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db getorderexec function: execute select query: %w", err))
		return "", err // внутренняя ошибка сервера
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&ordNumber)

		if err == nil {
			return ordNumber, nil
		}
	}

	if err == sql.ErrNoRows { // Нет строк
		return "", nil
	}

	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db getorderexec function: scan select query: %w", err))
		return "", err // внутренняя ошибка сервера
	}

	err = rows.Err()
	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db getorderexec function: fetch rows: %w", err))
		return "", err // внутренняя ошибка сервера
	}
	return ordNumber, nil
}

// Добавить новое начисление баллов
func SetOrders(ctx context.Context, db *sql.DB, order string, status string, accrual float32, cfg getparam.APIParam) error {

	logmy.OutLogDebug(fmt.Errorf("get db SetOrdes function: order:%v status:%v accrual:%v", order, status, accrual))

	rows, err := dbcommon.ResendDB(ctx, db, cfg,
		"with sel as (select o.user_id, order_number  from orders o  where o.order_number = $1), "+
			" up1  as (update user_ref "+
			"	set ballans  = ballans  + $3 "+
			"	where user_id = (select user_id from sel) "+
			"	returning user_id) "+
			"	update orders "+
			"	  set order_status = $2, "+
			"		  accrual  = $3, "+
			"		  update_at = now() "+
			"	where order_number = $1 "+
			"	  and user_id = (select user_id from up1)",
		order, status, accrual)

	if err != nil {
		logmy.OutLogError(fmt.Errorf("get db SetOrdes function: execute update query: %w", err))
		return err // внутренняя ошибка сервера
	}

	if rows == 0 {
		logmy.OutLogError(fmt.Errorf("get db SetOrdes function: no update rows: %w", err))
		return err // внутренняя ошибка сервера
	}

	return nil
}
