package exchange

import (
	"context"
	"time"

	"github.com/quangkeu95/binancebot/pkg/model"
)

// Feeder feeder implementations help fetching market data
type Feeder interface {
	GetExchangeInfo(ctx context.Context) (model.ExchangeInfo, error)
	CandlesByLimit(ctx context.Context, symbol, timeframe string, limit int) ([]model.Candle, error)
	CandlesByPeriod(ctx context.Context, symbol, timeframe string, start, end time.Time) ([]model.Candle, error)

	CandlesSubscription(ctx context.Context, symbol, timeframe string, candleCh chan<- model.Candle, errCh chan<- error)
	// CombinedCandlesSubscription(ctx context.Context, mapSymbolTimeframe map[string]string, candleCh chan<- model.Candle, errCh chan<- error)
	MarketStatsSubscription(ctx context.Context, symbol string, statCh chan<- model.MarketStats24h, errCh chan<- error)
	// CombinedMarketStatsSubscription(ctx context.Context, symbols []string, statCh chan<- model.MarketStats24h, errCh chan<- error)
}

type Exchange interface {
	Feeder
}
