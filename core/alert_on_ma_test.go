package core

import (
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
	config.InitConfig()
	ts.strategy = &AlertOnMAStrategy{}
	ts.strategy.Init()
}

func (ts *AlertOnMAStrategyTestSuite) TestHandleMACross() {
	assert := ts.Assert()

	var (
		symbol    = "BTCUSDT"
		timeframe = "15m"
		key       = ts.strategy.generateKey(symbol, timeframe)
	)

	ts.strategy.handleMACross(symbol, timeframe, time.Unix(1625205600, 0), 33000.0, 33100.0)

	stateIns := ts.strategy.state[key]
	assert.Equal(MAStateBelow.String(), stateIns.Fsm.Current())

	// new price, same timeframe period, expect nothing changes
	ts.strategy.handleMACross(symbol, timeframe, time.Unix(1625205600, 0), 33050.0, 33100.0)
	assert.Equal(MAStateBelow.String(), stateIns.Fsm.Current())

	// new price cross up ma200, same timeframe period, expect change state
	ts.strategy.handleMACross(symbol, timeframe, time.Unix(1625205600, 0), 33100.0, 33100.0)
	assert.Equal(MAStateAbove.String(), stateIns.Fsm.Current())
	assert.Equal(time.Unix(1625205600, 0), stateIns.LastUpdate)

	// new price still above ma200, next timeframe period, expect same state
	ts.strategy.handleMACross(symbol, timeframe, time.Unix(1625206500, 0), 33200.0, 33100.0)
	assert.Equal(MAStateAbove.String(), stateIns.Fsm.Current())

	// new price cross
	ts.strategy.handleMACross(symbol, timeframe, time.Unix(1625206500, 0), 33200.0, 33100.0)
	assert.Equal(MAStateAbove.String(), stateIns.Fsm.Current())

	// new price cross down ma200, next timeframe period, expect change state
	ts.strategy.handleMACross(symbol, timeframe, time.Unix(1625207400, 0), 33000.0, 33100.0)
	assert.Equal(MAStateBelow.String(), stateIns.Fsm.Current())
	assert.Equal(time.Unix(1625207400, 0), stateIns.LastUpdate)
}
