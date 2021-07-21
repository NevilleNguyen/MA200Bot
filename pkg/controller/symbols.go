package controller

import (
	"context"
	"sync"
	"time"

	"github.com/quangkeu95/binancebot/pkg/exchange"
	"github.com/quangkeu95/binancebot/pkg/model"
	"go.uber.org/zap"
)

const FetchSymbolInterval = 30 * time.Minute

type SymbolsController struct {
	sync.RWMutex
	l                 *zap.SugaredLogger
	exchange          exchange.Exchange
	tradingSymbols    map[string]model.SymbolInfo
	deprecatedSymbols map[string]model.SymbolInfo
}

func NewSymbolsController(ex exchange.Exchange) (*SymbolsController, error) {
	c := &SymbolsController{
		l:                 zap.S(),
		exchange:          ex,
		tradingSymbols:    make(map[string]model.SymbolInfo),
		deprecatedSymbols: make(map[string]model.SymbolInfo),
	}

	if err := c.FetchSymbolsPairUSDT(context.Background()); err != nil {
		c.l.Errorw("error initial fetch symbols", "error", err)
		return nil, err
	}
	go c.intervalFetchSymbols()
	return c, nil
}

func (c *SymbolsController) FetchSymbolsPairUSDT(ctx context.Context) error {
	exchangeInfo, err := c.exchange.GetExchangeInfo(ctx)
	if err != nil {
		c.l.Errorw("error get exchange info", "error", err)
		return err
	}
	var newSymbolInfo = make(map[string]model.SymbolInfo)
	for _, item := range exchangeInfo.Symbols {
		if item.QuoteAsset != "USDT" || item.Status != model.SymbolStatusTrading.String() {
			continue
		}
		newSymbolInfo[item.Symbol] = item
	}

	c.filterDeprecatedSymbols(newSymbolInfo)

	c.saveTradingSymbols(newSymbolInfo)

	return nil
}

func (c *SymbolsController) GetTradingSymbols() map[string]model.SymbolInfo {
	c.RLock()
	defer c.RUnlock()
	return c.tradingSymbols
}

func (c *SymbolsController) GetDeprecatedSymbols() map[string]model.SymbolInfo {
	c.RLock()
	defer c.RUnlock()
	return c.deprecatedSymbols
}

func (c *SymbolsController) intervalFetchSymbols() {
	ticker := time.NewTicker(FetchSymbolInterval)
	defer ticker.Stop()

	for {
		if err := c.FetchSymbolsPairUSDT(context.Background()); err != nil {
			c.l.Warnw("interval fetch symbol infos error", "error", err)
			continue
		}
		<-ticker.C
	}
}

func (c *SymbolsController) filterDeprecatedSymbols(newSymbolInfo map[string]model.SymbolInfo) {
	c.Lock()
	defer c.Unlock()
	var deprecatedSymbols = make(map[string]model.SymbolInfo)
	for symbol := range c.tradingSymbols {
		if val, ok := newSymbolInfo[symbol]; !ok {
			deprecatedSymbols[symbol] = val
		}
	}

	for symbol, val := range deprecatedSymbols {
		c.deprecatedSymbols[symbol] = val
	}
}

func (c *SymbolsController) saveTradingSymbols(info map[string]model.SymbolInfo) {
	c.Lock()
	defer c.Unlock()
	c.tradingSymbols = info
}
