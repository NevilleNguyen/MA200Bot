package model

import (
	"fmt"
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

type Dataframe struct {
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
	if len(d.Time) == 0 {
		return false
	}
	if candle.Time == d.Time[len(d.Time)-1] {
		return true
	}
	return false
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
