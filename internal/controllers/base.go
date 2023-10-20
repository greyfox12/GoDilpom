package controllers

import (
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	adapters "github.com/greyfox12/GoDiplom/internal/adapters/db"
	"github.com/greyfox12/GoDiplom/internal/infra/compress"
	"github.com/greyfox12/GoDiplom/internal/infra/getparam"
	"github.com/greyfox12/GoDiplom/internal/infra/hash"
	"github.com/greyfox12/GoDiplom/internal/infra/logmy"
)

type BaseController struct {
	DB    *adapters.DBAdapter
	Cfg   *getparam.APIParam
	Loger *logmy.Log
	Auth  *hash.AuthGen
}

func NewBaseController(db *adapters.DBAdapter, cfg *getparam.APIParam, loger *logmy.Log, auth *hash.AuthGen) *BaseController {
	return &BaseController{
		DB:    db,
		Cfg:   cfg,
		Loger: loger,
		Auth:  auth}
}

func (c *BaseController) Route() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.StripSlashes)

	// определяем хендлер
	//	router.Route("/", func(r chi.Router) {
	router.Group(func(r chi.Router) {
		r.Use(c.Autoriz)
		//получение списка загруженных пользователем номеров заказов, статусов их обработки и информации о начислениях
		r.Get("/api/user/orders", c.Loger.RequestLogger(c.getOrders))
		//получение текущего баланса счёта баллов лояльности пользователя
		r.Get("/api/user/balance", c.Loger.RequestLogger(c.getBalance))
		//запрос на списание баллов с накопительного счёта в счёт оплаты нового заказа
		r.Get("/api/user/withdrawals", c.Loger.RequestLogger(c.getWithdrawals))

		//загрузка пользователем номера заказа для расчёта
		r.Post("/api/user/orders", c.Loger.RequestLogger(c.postOrder))
		//запрос на списание баллов с накопительного счёта в счёт оплаты нового заказа
		r.Post("/api/user/balance/withdraw", c.Loger.RequestLogger(c.postWithdraw))

	})

	router.Group(func(r chi.Router) {
		r.Post("/api/user/register", c.Loger.RequestLogger(c.postRegister()))
		//аутентификация пользователя
		r.Post("/api/user/login", c.Loger.RequestLogger(c.postLogin()))
		// Ошибочный путь
		r.Post("/*", c.Loger.RequestLogger(c.errorPath))
		r.Get("/*", c.Loger.RequestLogger(c.errorPath))
	})

	hd := compress.GzipHandle(compress.GzipRead(router))

	return hd

}
