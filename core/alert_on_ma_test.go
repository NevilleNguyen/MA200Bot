package core

import (
	"log"
	"testing"
	"time"

	"github.com/quangkeu95/binancebot/config"
	"github.com/stretchr/testify/suite"
)

type AlertOnMAStrategyTestSuite struct {
	suite.Suite
	strategy *AlertOnMAStrategy
}

func TestAlertOnMAStrategyTestSuite(t *testing.T) {
	suite.Run(t, new(AlertOnMAStrategyTestSuite))
}

func (ts *AlertOnMAStrategyTestSuite) SetupSuite() {
	assert := ts.Assert()
	config.InitConfig()
	strategy, err := NewAlertOnMAStrategy()
	assert.NoError(err)
	assert.NotNil(strategy)

	ts.strategy = strategy
	ts.strategy.Init()
}

func (ts *AlertOnMAStrategyTestSuite) TestIsEnoughVolume() {
	// assert := ts.Assert()

	params := CandleParams{
		Symbol:         "BTCUSDT",
		Timeframe:      "4h",
		LastUpdate:     time.Unix(1627315200, 0),
		PreviousVolume: 10000.0,
		LastVolume:     300.0,
	}
	isEnoughVolume := ts.strategy.isEnoughVolume(params)
	log.Println(isEnoughVolume)
}
