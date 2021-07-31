package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/quangkeu95/binancebot/pkg/series"
)

//go:generate stringer -type=SymbolStatus -linecomment
type SymbolStatus int

const (
	SymbolStatusTrading  SymbolStatus = iota // TRADING
	SymbolStatusBreaking                     // BREAKING
)

type Candle struct {
	Symbol    string
	Timeframe string
	Time      time.Time
	Open      float64
	Close     float64
	Low       float64
	High      float64
	Volume    float64
	Trades    int64
	Complete  bool
}

func CandleFieldLength() int {
	return 9
}

func (c Candle) ToSlice() []string {
	return []string{
		c.Symbol,
		c.Timeframe,
		fmt.Sprintf("%d", c.Time.Unix()),
		fmt.Sprintf("%f", c.Open),
		fmt.Sprintf("%f", c.Close),
		fmt.Sprintf("%f", c.Low),
		fmt.Sprintf("%f", c.High),
		fmt.Sprintf("%.1f", c.Volume),
		fmt.Sprintf("%d", c.Trades),
	}
}

type CandleAttribute int

const (
	CandleAttributeClose CandleAttribute = iota
	CandleAttributeOpen
	CandleAttributeHigh
	CandleAttributeLow
	CandleAttributeVolume
)

type Dataframe struct {
	sync.RWMutex
	Symbol    string
	Timeframe string

	Close  series.Series
	Open   series.Series
	High   series.Series
	Low    series.Series
	Volume series.Series

	Time       []time.Time
	LastUpdate time.Time

	// Custom user metadata
	Metadata map[string]series.Series
}

func (d *Dataframe) IsLastCandle(candle Candle) bool {
	d.RLock()
	defer d.RUnlock()
	if len(d.Time) == 0 {
		return false
	}
	if candle.Time == d.Time[len(d.Time)-1] {
		return true
	}
	return false
}

func (d *Dataframe) Length() int {
	d.RLock()
	defer d.RUnlock()
	return len(d.Close)
}

func (d *Dataframe) UpdateWithIndex(index int, candle Candle) {
	d.Lock()
	defer d.Unlock()
	if index < 0 || index >= len(d.Close) {
		return
	}
	d.Close[index] = candle.Close
	d.Open[index] = candle.Open
	d.High[index] = candle.High
	d.Low[index] = candle.Low
	d.Volume[index] = candle.Volume
	d.Time[index] = candle.Time
	d.LastUpdate = candle.Time
}

func (d *Dataframe) AddNewCandle(candle Candle) {
	d.Lock()
	defer d.Unlock()
	d.Close = append(d.Close, candle.Close)
	d.Open = append(d.Open, candle.Open)
	d.High = append(d.High, candle.High)
	d.Low = append(d.Low, candle.Low)
	d.Volume = append(d.Volume, candle.Volume)
	d.Time = append(d.Time, candle.Time)
	d.LastUpdate = candle.Time
}

func (d *Dataframe) GetLastValues(candleAttr CandleAttribute, periods int) []float64 {
	d.RLock()
	defer d.RUnlock()
	switch candleAttr {
	case CandleAttributeClose:
		return d.Close.LastValues(periods)
	case CandleAttributeHigh:
		return d.High.LastValues(periods)
	case CandleAttributeLow:
		return d.Low.LastValues(periods)
	case CandleAttributeOpen:
		return d.Open.LastValues(periods)
	case CandleAttributeVolume:
		return d.Volume.LastValues(periods)
	}
	return []float64{}
}

func (d *Dataframe) GetLast(candleAttr CandleAttribute, index int) float64 {
	d.RLock()
	defer d.RUnlock()
	switch candleAttr {
	case CandleAttributeClose:
		return d.Close.Last(index)
	case CandleAttributeHigh:
		return d.High.Last(index)
	case CandleAttributeLow:
		return d.Low.Last(index)
	case CandleAttributeOpen:
		return d.Open.Last(index)
	case CandleAttributeVolume:
		return d.Volume.Last(index)
	}
	return 0
}

func (d *Dataframe) GetLastUpdate() time.Time {
	d.RLock()
	defer d.RUnlock()
	return d.LastUpdate
}

type MarketStats24h struct {
	Symbol             string
	PriceChange        string
	PriceChangePercent string
	LastPrice          float64
	LastQty            float64
	BaseVolume         float64
	QuoteVolume        float64
	OpenTime           time.Time
	CloseTime          time.Time
	FirstTradeId       int64
	LastTradeId        int64
	TotalTrades        int64
}

type ExchangeInfo struct {
	Symbols []SymbolInfo
}

type SymbolInfo struct {
	Symbol     string
	Status     string
	BaseAsset  string
	QuoteAsset string
}
