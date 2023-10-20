package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/greyfox12/GoDiplom/internal/closer"
)

func RunServer(ctx context.Context, c *BaseController, mux http.Handler) error {

	var srv = &http.Server{
		Addr:    c.Cfg.ServiceAddress,
		Handler: mux,
	}
	var closer = &closer.Closer{}

	closer.Add(srv.Shutdown)

	closer.Add(func(ctx context.Context) error {
		time.Sleep(3 * time.Second)

		return nil
	})
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.Loger.OutLogFatal(fmt.Errorf("listen and serve: %v", err))
		}
	}()

	c.Loger.OutLogInfo(fmt.Errorf("listening on %s", c.Cfg.ServiceAddress))
	<-ctx.Done()

	c.Loger.OutLogInfo(fmt.Errorf("shutting down server gracefully"))

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := closer.Close(shutdownCtx); err != nil {
		return fmt.Errorf("closer: %v", err)
	}

	return nil

}
