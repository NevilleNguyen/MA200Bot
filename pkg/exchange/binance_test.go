package exchange

import (
	"context"
	"testing"
	"time"

	"github.com/quangkeu95/binancebot/config"
	"github.com/quangkeu95/binancebot/pkg/model"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type BinanceTestSuite struct {
	suite.Suite
	client *Binance
}

func TestBinanceTestSuite(t *testing.T) {
	suite.Run(t, new(BinanceTestSuite))
}

func (ts *BinanceTestSuite) SetupSuite() {
	assert := ts.Assert()
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)

	config.InitConfig()

	client, err := NewBinance()
	assert.NoError(err)
	assert.NotNil(client)

	ts.client = client
}

func (ts *BinanceTestSuite) TestCandlesSubscription() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var (
		symbol   = "BTCUSDT"
		period   = "1m"
		candleCh = make(chan model.Candle)
		errCh    = make(chan error)
	)

	go ts.client.CandlesSubscription(ctx, symbol, period, candleCh, errCh)

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			ts.client.l.Error(err)
		case candle := <-candleCh:
			ts.client.l.Infof("%+v\n", candle)
		}
	}
}

func (ts *BinanceTestSuite) TestCombinedCandlesSubscription() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var (
		candleCh           = make(chan model.Candle)
		errCh              = make(chan error)
		mapSymbolTimeframe = make(map[string]string)
	)
	mapSymbolTimeframe["BTCUSDT"] = "1m"
	mapSymbolTimeframe["ETHUSDT"] = "1m"
	mapSymbolTimeframe["KNCUSDT"] = "1m"

	go ts.client.CombinedCandlesSubscription(ctx, mapSymbolTimeframe, candleCh, errCh)

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			ts.client.l.Error(err)
		case candle := <-candleCh:
			ts.client.l.Infof("%+v\n", candle)
		}
	}
}

func (ts *BinanceTestSuite) TestMarketStatsSubscription() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var (
		symbol       = "BTCUSDT"
		marketStatCh = make(chan model.MarketStats24h)
		errCh        = make(chan error)
	)

	go ts.client.MarketStatsSubscription(ctx, symbol, marketStatCh, errCh)

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			ts.client.l.Error(err)
		case marketStat := <-marketStatCh:
			ts.client.l.Infof("%+v\n", marketStat)
		}
	}
}
