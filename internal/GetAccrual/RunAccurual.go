// Старт сервиса работы с системой расчерта бонусов
package accrual

import (
	"context"
	"fmt"
	"time"

	"github.com/greyfox12/GoDiplom/internal/closer"
	"github.com/greyfox12/GoDiplom/internal/controllers"
)

func RunAccurual(ctx context.Context, c *controllers.BaseController) error {

	var closer = &closer.Closer{}

	ticker := time.NewTicker(time.Second * time.Duration(c.Cfg.IntervalAccurual))
	stop := make(chan bool)

	closer.Add(func(ctx context.Context) error {
		time.Sleep(1 * time.Second)
		stop <- true
		return nil
	})

	go func() {
		defer func() { stop <- true }()
		for {
			select {
			case <-ticker.C:
				OrderNumberGet(c)
			case <-stop:
				return
			}
		}
	}()

	c.Loger.OutLogInfo(fmt.Errorf("start RunAccurual on %s", c.Cfg.AccurualSystemAddress))
	<-ctx.Done()

	c.Loger.OutLogInfo(fmt.Errorf("shutting down RunAccurual gracefully"))

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := closer.Close(shutdownCtx); err != nil {
		return fmt.Errorf("closer: %v", err)
	}

	return nil

}
