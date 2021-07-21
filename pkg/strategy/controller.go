package strategy

import (
	"fmt"
	"sync"

	"github.com/quangkeu95/binancebot/pkg/model"
	"github.com/quangkeu95/binancebot/pkg/series"
	"go.uber.org/zap"
)

type Controller struct {
	sync.RWMutex
	l          *zap.SugaredLogger
	dataframes map[string]*model.Dataframe
	strategy   Strategy
	started    bool
}

func NewStategyController(mapSymbolTimeframe map[string]string, strategy Strategy) *Controller {
	c := &Controller{
		l:          zap.S(),
		dataframes: make(map[string]*model.Dataframe),
		strategy:   strategy,
	}

	for symbol, timeframe := range mapSymbolTimeframe {
		key := c.generateKey(symbol, timeframe)
		c.dataframes[key] = &model.Dataframe{
			Symbol:    symbol,
			Timeframe: timeframe,
			Metadata:  make(map[string]series.Series),
		}
	}

	return c
}

func (s *Controller) Start() {
	s.started = true
}

func (c *Controller) OnCandle(candle model.Candle) {
	c.Lock()
	defer c.Unlock()
	key := c.generateKey(candle.Symbol, candle.Timeframe)
	dataframe, ok := c.dataframes[key]
	if !ok {
		c.l.Warnw("cannot found dataframe for entry", "symbol", candle.Symbol, "timeframe", candle.Timeframe)
		return
	}

	if dataframe.IsLastCandle(candle) && c.started {
		lastIndex := len(dataframe.Close) - 1
		dataframe.Close[lastIndex] = candle.Close
		dataframe.Open[lastIndex] = candle.Open
		dataframe.High[lastIndex] = candle.High
		dataframe.Low[lastIndex] = candle.Low
		dataframe.Volume[lastIndex] = candle.Volume
	} else {
		dataframe.Symbol = candle.Symbol
		dataframe.Timeframe = candle.Timeframe
		dataframe.Close = append(dataframe.Close, candle.Close)
		dataframe.Open = append(dataframe.Open, candle.Open)
		dataframe.High = append(dataframe.High, candle.High)
		dataframe.Low = append(dataframe.Low, candle.Low)
		dataframe.Volume = append(dataframe.Volume, candle.Volume)
		dataframe.Time = append(dataframe.Time, candle.Time)
		dataframe.LastUpdate = candle.Time
	}

	if len(dataframe.Close) >= c.strategy.WarmupPeriod() {
		if c.started {
			c.strategy.OnCandle(dataframe)
		}
	}
}

func (c *Controller) generateKey(symbol, timeframe string) string {
	return fmt.Sprintf("%s--%s", symbol, timeframe)
}
