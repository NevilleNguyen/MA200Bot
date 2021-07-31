package exchange

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/quangkeu95/binancebot/pkg/model"
	"go.uber.org/zap"
)

var ErrInsufficientData = errors.New("insufficient data")

type SymbolFeed struct {
	SymbolInfo model.SymbolInfo
	File       string
	Timeframe  string
}

type CSVFeed struct {
	sync.RWMutex
	l       *zap.SugaredLogger
	Feeds   map[string]SymbolFeed
	Candles map[string][]model.Candle

	exchangeInfo model.ExchangeInfo
}

func NewCSVFeed(feeds ...SymbolFeed) (*CSVFeed, error) {
	l := zap.S()
	csvFeed := &CSVFeed{
		l:       l,
		Feeds:   make(map[string]SymbolFeed),
		Candles: make(map[string][]model.Candle),
	}

	for _, feed := range feeds {
		key := csvFeed.feedTimeframeKey(feed.SymbolInfo.Symbol, feed.Timeframe)
		csvFeed.Feeds[key] = feed

		candles, err := csvFeed.parseCandlesFromCsv(feed.File)
		if err != nil {
			l.Errorw("parse candles error", "error", err)
			return nil, err
		}

		csvFeed.Candles[key] = candles
	}
	return csvFeed, nil
}

func (c *CSVFeed) GetExchangeInfo(ctx context.Context) (model.ExchangeInfo, error) {
	c.RLock()
	defer c.RUnlock()
	var symbols = make([]model.SymbolInfo, 0)

	for _, item := range c.Feeds {
		symbols = append(symbols, item.SymbolInfo)
	}
	return model.ExchangeInfo{
		Symbols: symbols,
	}, nil
}

func (c *CSVFeed) CandlesByLimit(ctx context.Context, symbol, timeframe string, limit int) ([]model.Candle, error) {
	c.RLock()
	defer c.RUnlock()
	var result = make([]model.Candle, 0)
	key := c.feedTimeframeKey(symbol, timeframe)
	if len(c.Candles[key]) < limit {
		return nil, fmt.Errorf("%w: %s -- %s", ErrInsufficientData, symbol, timeframe)
	}

	// each time we get candles, we get from the beginning, and continue from the next limit index
	result, c.Candles[key] = c.Candles[key][:limit], c.Candles[key][limit:]
	return result, nil
}

func (c *CSVFeed) CandlesByPeriod(ctx context.Context, symbol, timeframe string, start, end time.Time) ([]model.Candle, error) {
	c.RLock()
	defer c.RUnlock()
	var result = make([]model.Candle, 0)
	key := c.feedTimeframeKey(symbol, timeframe)
	for _, candle := range c.Candles[key] {
		if candle.Time.Before(start) || candle.Time.After(end) {
			continue
		}
		result = append(result, candle)
	}
	return result, nil
}

func (c *CSVFeed) CandlesSubscription(ctx context.Context, symbol, timeframe string, candleCh chan<- model.Candle, errCh chan<- error) {
	key := c.feedTimeframeKey(symbol, timeframe)
	for _, candle := range c.Candles[key] {
		candleCh <- candle
	}
	// after we emit all the candles, we close the candle channel to indicate that no more candle is emitted
	close(candleCh)
}

func (c *CSVFeed) MarketStatsSubscription(ctx context.Context, symbol string, statCh chan<- model.MarketStats24h, errCh chan<- error) {
	c.l.Errorw("MarketStatsSubscription not implemented")
}

func (c *CSVFeed) feedTimeframeKey(symbol, timeframe string) string {
	return fmt.Sprintf("%s--%s", symbol, timeframe)
}

func (c *CSVFeed) parseCandlesFromCsv(csvFilepath string) ([]model.Candle, error) {
	csvFile, err := os.Open(csvFilepath)
	if err != nil {
		return nil, err
	}

	csvLines, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		return nil, err
	}

	var candles = make([]model.Candle, 0)
	for _, line := range csvLines {
		if len(line) < model.CandleFieldLength() {
			return nil, fmt.Errorf("invalid csv candle data")
		}

		candle := model.Candle{
			Symbol:    line[0],
			Timeframe: line[1],
			Complete:  true,
		}

		timestamp, err := strconv.ParseInt(line[2], 10, 64)
		if err != nil {
			return nil, err
		}
		candle.Time = time.Unix(timestamp, 0)
		candle.Open, err = strconv.ParseFloat(line[3], 64)
		if err != nil {
			return nil, err
		}

		candle.Close, err = strconv.ParseFloat(line[4], 64)
		if err != nil {
			return nil, err
		}

		candle.Low, err = strconv.ParseFloat(line[5], 64)
		if err != nil {
			return nil, err
		}

		candle.High, err = strconv.ParseFloat(line[6], 64)
		if err != nil {
			return nil, err
		}

		candle.Volume, err = strconv.ParseFloat(line[7], 64)
		if err != nil {
			return nil, err
		}

		candle.Trades, err = strconv.ParseInt(line[8], 10, 64)
		if err != nil {
			return nil, err
		}

		candles = append(candles, candle)
	}

	return candles, nil
}
