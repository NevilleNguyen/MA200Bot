package core

import (
	"context"
	"sync"

	"github.com/quangkeu95/binancebot/pkg/controller"
	"github.com/quangkeu95/binancebot/pkg/exchange"
	"github.com/quangkeu95/binancebot/pkg/strategy"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	SelectedSymbolsFlag = "symbols"
	ExcludedSymbolsFlag = "excluded_symbols"
	ListTimeframesFlag  = "timeframes"
)

type Core struct {
	sync.RWMutex
	l                *zap.SugaredLogger
	exchange         exchange.Exchange
	candleController *controller.CandleController
	symbolController *controller.SymbolsController
	strategy         strategy.Strategy
	// keyValueStorage  storage.KeyValueStorage
}

func New(ex exchange.Exchange, str strategy.Strategy) (*Core, error) {
	symbolController, err := controller.NewSymbolsController(ex)
	if err != nil {
		return nil, err
	}

	// badgerDB, err := storage.NewBadgerDB()
	// if err != nil {
	// 	return nil, err
	// }

	c := &Core{
		l:                zap.S(),
		exchange:         ex,
		candleController: controller.NewCandleController(ex),
		symbolController: symbolController,
		strategy:         str,
		// keyValueStorage:  badgerDB,
	}

	c.strategy.Init()

	return c, nil
}

func (c *Core) Run(ctx context.Context, listTimeframes []string) error {
	c.l.Infow("Running core")

	var listSymbols = make([]string, 0)
	symbolInfos := c.symbolController.GetTradingSymbols()

	selectedSymbols := viper.GetStringSlice(SelectedSymbolsFlag)
	excludedSymbols := viper.GetStringSlice(ExcludedSymbolsFlag)

	if len(selectedSymbols) > 0 {
		for _, symbol := range selectedSymbols {
			if isInList(excludedSymbols, symbol) {
				continue
			}
			listSymbols = append(listSymbols, symbol)
		}
	} else {
		for symbol := range symbolInfos {
			if isInList(excludedSymbols, symbol) {
				continue
			}
			listSymbols = append(listSymbols, symbol)
		}
	}

	c.l.Infof("There are %v pair with USDT", len(listSymbols))

	for _, timeframe := range listTimeframes {
		var mapSymbolTimeframe = make(map[string]string)
		for _, symbol := range listSymbols {
			mapSymbolTimeframe[symbol] = timeframe
		}

		if err := c.SubscribeCandles(ctx, mapSymbolTimeframe); err != nil {
			return err
		}
	}

	c.candleController.Start(ctx)

	return nil
}

func (c *Core) SubscribeCandles(ctx context.Context, mapSymbolTimeframe map[string]string) error {
	strategyController := strategy.NewStategyController(mapSymbolTimeframe, c.strategy)

	var (
		errCh  = make(chan error)
		wg     = &sync.WaitGroup{}
		doneCh = make(chan struct{})
	)

	for symbol, timeframe := range mapSymbolTimeframe {
		wg.Add(1)
		go func(symbol, timeframe string) {
			defer wg.Done()
			c.candleController.Subscribe(symbol, timeframe, strategyController.OnCandle, false)

			// preload candles
			candles, err := c.exchange.CandlesByLimit(ctx, symbol, timeframe, c.strategy.WarmupPeriod())
			if err != nil {
				c.l.Errorw("candles by limit error", "error", err, "symbol", symbol, "timeframe", timeframe)
				errCh <- err
				return
			}

			c.candleController.Preload(symbol, timeframe, candles)
		}(symbol, timeframe)
	}

	go func() {
		wg.Wait()
		close(doneCh)
	}()

	for {
		select {
		case err := <-errCh:
			return err
		case <-doneCh:
			strategyController.Start()
			return nil
		}
	}
}

func isInList(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
