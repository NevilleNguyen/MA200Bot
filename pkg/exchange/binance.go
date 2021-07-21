package exchange

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/quangkeu95/binancebot/lib/app"
	"github.com/quangkeu95/binancebot/pkg/model"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	BinanceApiKeyFlag    = "binance.api_key"
	BinanceApiSecretFlag = "binance.api_secret"
)

const (
	RequestPerSecond = 20
	OrderPerSecond   = 5
	OrderPerDay      = 160000
	RequestTimeout   = 5 * time.Second
)

func init() {
	viper.BindEnv(BinanceApiKeyFlag, "BINANCE_API_KEY")
	viper.BindEnv(BinanceApiSecretFlag, "BINANCE_API_SECRET")
}

type Binance struct {
	l         *zap.SugaredLogger
	apiKey    string
	apiSecret string

	client      *binance.Client
	rateLimiter *app.RateLimiter
}

func NewBinance() (*Binance, error) {
	l := zap.S()
	apiKey := viper.GetString(BinanceApiKeyFlag)
	if err := validation.Validate(apiKey, validation.Required); err != nil {
		l.Errorw("invalid binance api key", "error", err, "api_key", apiKey)
		return nil, err
	}
	apiSecret := viper.GetString(BinanceApiSecretFlag)
	if err := validation.Validate(apiSecret, validation.Required); err != nil {
		l.Errorw("invalid binance api secret", "error", err, "api_secret", apiSecret)
		return nil, err
	}
	client := binance.NewClient(apiKey, apiSecret)

	rateLimiter := app.NewRateLimiter(RequestPerSecond, RequestPerSecond)

	b := &Binance{
		l:           l,
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		client:      client,
		rateLimiter: rateLimiter,
	}

	// test ping
	if err := b.client.NewPingService().Do(context.Background()); err != nil {
		l.Errorw("error ping to binance", "error", err)
		return nil, err
	}
	return b, nil
}

func (b *Binance) GetExchangeInfo(ctx context.Context) (model.ExchangeInfo, error) {
	if err := b.rateLimiter.WaitN(RequestTimeout, 10); err != nil {
		return model.ExchangeInfo{}, err
	}
	resp, err := b.client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		b.l.Errorw("error get binance exchange info", "error", err)
		return model.ExchangeInfo{}, err
	}

	var symbolInfo = make([]model.SymbolInfo, 0)
	for _, item := range resp.Symbols {
		symbolInfo = append(symbolInfo, model.SymbolInfo{
			Symbol:     item.Symbol,
			BaseAsset:  item.BaseAsset,
			QuoteAsset: item.QuoteAsset,
			Status:     item.Status,
		})
	}

	var result = model.ExchangeInfo{
		Symbols: symbolInfo,
	}
	return result, nil
}

func (b *Binance) CandlesByLimit(ctx context.Context, symbol, timeframe string, limit int) ([]model.Candle, error) {
	candles := make([]model.Candle, 0)
	klineService := b.client.NewKlinesService()

	data, err := klineService.Symbol(symbol).
		Interval(timeframe).
		Limit(limit).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	for _, d := range data {
		candles = append(candles, CandleFromKline(symbol, timeframe, *d))
	}

	return candles, nil
}

// CandlesSubscription subscribe kline for specific symbol and timeframe
func (b *Binance) CandlesSubscription(ctx context.Context, symbol, timeframe string, candleCh chan<- model.Candle, errCh chan<- error) {
	wsKlineHandler := func(event *binance.WsKlineEvent) {
		candleCh <- CandleFromWsKline(event.Kline)
	}
	errHandler := func(err error) {
		errCh <- err
	}

	doneCh, stopCh, err := binance.WsKlineServe(symbol, timeframe, wsKlineHandler, errHandler)
	if err != nil {
		b.l.Errorw("candles subscription error", "error", err)
		errCh <- err
		return
	}

	b.l.Debugw("binance candle subscription", "symbol", symbol, "timeframe", timeframe)
	for {
		select {
		case <-ctx.Done():
			stopCh <- struct{}{}
			return
		case <-doneCh:
			errCh <- fmt.Errorf("candles subscription stopped")
			return
		}
	}
}

func (b *Binance) CombinedCandlesSubscription(ctx context.Context, mapSymbolTimeframe map[string]string, candleCh chan<- model.Candle, errCh chan<- error) {
	wsKlineHandler := func(event *binance.WsKlineEvent) {
		candleCh <- CandleFromWsKline(event.Kline)
	}
	errHandler := func(err error) {
		errCh <- err
	}

	doneCh, stopCh, err := binance.WsCombinedKlineServe(mapSymbolTimeframe, wsKlineHandler, errHandler)
	if err != nil {
		b.l.Errorw("combined candles subscription error", "error", err)
		errCh <- err
		return
	}

	for {
		select {
		case <-ctx.Done():
			stopCh <- struct{}{}
			return
		case <-doneCh:
			errCh <- fmt.Errorf("combined candles subscription stopped")
			return
		}
	}
}

func (b *Binance) MarketStatsSubscription(ctx context.Context, symbol string, statCh chan<- model.MarketStats24h, errCh chan<- error) {
	wsMarketStatHandler := func(event *binance.WsMarketStatEvent) {
		statCh <- MarketStatsFromEvent(event)
	}
	errHandler := func(err error) {
		errCh <- err
	}
	doneCh, stopCh, err := binance.WsMarketStatServe(symbol, wsMarketStatHandler, errHandler)
	if err != nil {
		b.l.Errorw("market stats subscription error", "error", err)
		errCh <- err
		return
	}
	for {
		select {
		case <-ctx.Done():
			stopCh <- struct{}{}
			return
		case <-doneCh:
			errCh <- fmt.Errorf("market stats subscription stopped")
			return
		}
	}
}

func (b *Binance) CombinedMarketStatsSubscription(ctx context.Context, symbols []string, statCh chan<- model.MarketStats24h, errCh chan<- error) {
	wsMarketStatHandler := func(event *binance.WsMarketStatEvent) {
		statCh <- MarketStatsFromEvent(event)
	}
	errHandler := func(err error) {
		errCh <- err
	}
	doneCh, stopCh, err := binance.WsCombinedMarketStatServe(symbols, wsMarketStatHandler, errHandler)
	if err != nil {
		b.l.Errorw("combined market stats subscription error", "error", err)
		errCh <- err
		return
	}
	for {
		select {
		case <-ctx.Done():
			stopCh <- struct{}{}
			return
		case <-doneCh:
			errCh <- fmt.Errorf("combined market stats subscription stopped")
			return
		}
	}
}

func CandleFromKline(symbol, timeframe string, k binance.Kline) model.Candle {
	candle := model.Candle{
		Symbol:    symbol,
		Timeframe: timeframe,
		Time:      time.Unix(0, k.OpenTime*int64(time.Millisecond)),
	}
	candle.Open, _ = strconv.ParseFloat(k.Open, 64)
	candle.Close, _ = strconv.ParseFloat(k.Close, 64)
	candle.High, _ = strconv.ParseFloat(k.High, 64)
	candle.Low, _ = strconv.ParseFloat(k.Low, 64)
	candle.Volume, _ = strconv.ParseFloat(k.Volume, 64)
	candle.Trades = k.TradeNum
	candle.Complete = true
	return candle
}

func CandleFromWsKline(k binance.WsKline) model.Candle {
	candle := model.Candle{
		Symbol:    k.Symbol,
		Timeframe: k.Interval,
		Time:      time.Unix(0, k.StartTime*int64(time.Millisecond)),
	}
	candle.Open, _ = strconv.ParseFloat(k.Open, 64)
	candle.Close, _ = strconv.ParseFloat(k.Close, 64)
	candle.High, _ = strconv.ParseFloat(k.High, 64)
	candle.Low, _ = strconv.ParseFloat(k.Low, 64)
	candle.Volume, _ = strconv.ParseFloat(k.Volume, 64)
	candle.Trades = k.TradeNum
	candle.Complete = k.IsFinal
	return candle
}

func MarketStatsFromEvent(event *binance.WsMarketStatEvent) model.MarketStats24h {
	stat := model.MarketStats24h{
		Symbol:             event.Symbol,
		PriceChange:        event.PriceChange,
		PriceChangePercent: event.PriceChangePercent,
		OpenTime:           time.Unix(0, event.OpenTime*int64(time.Millisecond)),
		CloseTime:          time.Unix(0, event.CloseTime*int64(time.Millisecond)),
		FirstTradeId:       event.FirstID,
		LastTradeId:        event.LastID,
		TotalTrades:        event.Count,
	}
	stat.LastPrice, _ = strconv.ParseFloat(event.LastPrice, 64)
	stat.LastQty, _ = strconv.ParseFloat(event.CloseQty, 64)
	stat.BaseVolume, _ = strconv.ParseFloat(event.BaseVolume, 64)
	stat.QuoteVolume, _ = strconv.ParseFloat(event.QuoteVolume, 64)
	return stat
}
