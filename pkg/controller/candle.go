package controller

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/quangkeu95/binancebot/pkg/exchange"
	"github.com/quangkeu95/binancebot/pkg/model"
	"go.uber.org/zap"
)

type CandleConsumer func(model.Candle)

type Subscription struct {
	onCandleClose bool
	consumer      CandleConsumer
}

type CandleController struct {
	sync.RWMutex
	l             *zap.SugaredLogger
	exchange      exchange.Exchange
	Feeds         []string
	Subscriptions map[string][]Subscription // each symbol_timeframe is a key, value is list of subscriber
}

// NewCandleController manage list of candle subscriptions for each symbol + timeframe
func NewCandleController(ex exchange.Exchange) *CandleController {
	return &CandleController{
		l:             zap.S(),
		exchange:      ex,
		Feeds:         make([]string, 0),
		Subscriptions: make(map[string][]Subscription),
	}
}

func (c *CandleController) generateKey(symbol, timeframe string) string {
	return fmt.Sprintf("%s--%s", symbol, timeframe)
}

func (c *CandleController) extractKey(key string) (symbol, timeframe string) {
	parts := strings.Split(key, "--")
	return parts[0], parts[1]
}

// Subscribe subscribe single symbol to consume data
func (c *CandleController) Subscribe(symbol, timeframe string, consumer CandleConsumer, onCandleClose bool) {
	c.Lock()
	defer c.Unlock()
	key := c.generateKey(symbol, timeframe)
	if !isInList(c.Feeds, key) {
		c.Feeds = append(c.Feeds, key)
		// c.l.Infow("subscribe candle", "symbol", symbol, "timeframe", timeframe)
	}

	if _, ok := c.Subscriptions[key]; !ok {
		c.Subscriptions[key] = make([]Subscription, 0)
	}

	c.Subscriptions[key] = append(c.Subscriptions[key], Subscription{
		onCandleClose: onCandleClose,
		consumer:      consumer,
	})
}

func (c *CandleController) Preload(symbol, timeframe string, candles []model.Candle) {
	c.Lock()
	defer c.Unlock()
	key := c.generateKey(symbol, timeframe)
	for _, candle := range candles {
		for _, sub := range c.Subscriptions[key] {
			sub.consumer(candle)
		}
	}
	// c.l.Infow("preloading candles", "symbol", symbol, "timeframe", timeframe)
}

func (c *CandleController) CandlesSubscription(ctx context.Context, feed string, wg *sync.WaitGroup) {
	var (
		candleCh = make(chan model.Candle)
		errCh    = make(chan error)
	)
	symbol, timeframe := c.extractKey(feed)

	go c.exchange.CandlesSubscription(ctx, symbol, timeframe, candleCh, errCh)

	for {
		select {
		case <-ctx.Done():
			close(candleCh)
			return
		case err := <-errCh:
			// try to reset subscription
			c.l.Debugw("candle subscription error", "error", err, "symbol", symbol, "timeframe", timeframe)
			go c.exchange.CandlesSubscription(ctx, symbol, timeframe, candleCh, errCh)
		case candle, ok := <-candleCh:
			if !ok {
				c.l.Debugw("no more candles", "symbol", symbol, "timeframe", timeframe)
				wg.Done()
				return
			}
			c.onCandle(feed, candle)
		}
	}
}

// func (c *CandleController) CombinedCandlesSubscription(ctx context.Context, feeds []string) {
// 	var (
// 		candleCh           = make(chan model.Candle)
// 		errCh              = make(chan error)
// 		mapSymbolTimeframe = make(map[string]string)
// 	)
// 	for _, feed := range feeds {
// 		symbol, timeframe := c.extractKey(feed)
// 		mapSymbolTimeframe[symbol] = timeframe
// 	}
// }

func (c *CandleController) onCandle(feed string, candle model.Candle) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.Subscriptions[feed]; !ok {
		return
	}
	for _, sub := range c.Subscriptions[feed] {
		if sub.onCandleClose && !candle.Complete {
			continue
		}
		sub.consumer(candle)
	}
}

func (c *CandleController) Start(ctx context.Context) {
	wg := new(sync.WaitGroup)
	for _, feed := range c.Feeds {
		wg.Add(1)
		go c.CandlesSubscription(ctx, feed, wg)
	}
	c.l.Infow("start candle controller")

	wg.Wait()
	c.l.Infow("candle controller finishes")
}

func isInList(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
