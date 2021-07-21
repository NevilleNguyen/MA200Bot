package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/looplab/fsm"
	"github.com/quangkeu95/binancebot/pkg/model"
	"github.com/quangkeu95/binancebot/pkg/notification"
	"github.com/quangkeu95/binancebot/pkg/series"
	"go.uber.org/zap"
)

//go:generate stringer -type=MAState -linecomment
type MAState int

const (
	MAStateAbove MAState = iota // above
	MAStateBelow                // below
	MAStateEqual                // equal
)

const (
	EventMA200CrossUp   = "ma_200_cross_up"
	EventMA200CrossDown = "ma_200_cross_down"

	MATrendUp   = "UP"
	MATrendDown = "DOWN"
)

type State struct {
	Symbol     string
	Timeframe  string
	MATrend    string
	LastUpdate time.Time
	Fsm        *fsm.FSM
}

type AlertOnMAStrategy struct {
	sync.RWMutex
	l        *zap.SugaredLogger
	notifier notification.Notifier
	state    map[string]*State // store state of previous price vs MA price
}

// Init init is called one time before running strategy
func (s *AlertOnMAStrategy) Init() {
	s.l = zap.S()
	teleBot, err := notification.NewTelegram()
	if err != nil {
		s.l.Panicw("error create telegram bot", "error", err)
	}

	s.notifier = teleBot
	s.state = make(map[string]*State)

	s.notifier.SendMessage("Start Binance alert bot!!")
}

func (s *AlertOnMAStrategy) WarmupPeriod() int {
	return 201
}

func (s *AlertOnMAStrategy) OnCandle(df *model.Dataframe) {
	prices := df.Close.LastValues(201)

	previousCandleMA200 := series.MA(prices[:200], 200)
	lastCandleMA200 := series.MA(prices[1:], 200)
	lastClosePrice := df.Close.Last(0)

	go s.handleMACross(df.Symbol, df.Timeframe, df.LastUpdate, lastClosePrice, previousCandleMA200, lastCandleMA200)
}

func (s *AlertOnMAStrategy) handleMACross(symbol, timeframe string, lastUpdate time.Time, lastClosePrice, previousMA200, lastMA200 float64) {
	s.Lock()
	defer s.Unlock()

	key := s.generateKey(symbol, timeframe)

	if _, ok := s.state[key]; !ok {
		var (
			state string
		)
		if lastClosePrice > lastMA200 {
			state = MAStateAbove.String()
		} else if lastClosePrice < lastMA200 {
			state = MAStateBelow.String()
		} else {
			state = MAStateEqual.String()
		}

		s.l.Infow("init state", "symbol", symbol, "timeframe", timeframe, "state", state, "last_price", lastClosePrice, "previous_ma200", previousMA200, "last_ma200", lastMA200)

		// create new state machine for each symbol + timeframe
		fsmIns := fsm.NewFSM(state, fsm.Events{
			{Name: EventMA200CrossUp, Src: []string{MAStateEqual.String(), MAStateBelow.String()}, Dst: MAStateAbove.String()},
			{Name: EventMA200CrossDown, Src: []string{MAStateEqual.String(), MAStateAbove.String()}, Dst: MAStateBelow.String()},
		}, fsm.Callbacks{
			"after_" + EventMA200CrossDown: func(e *fsm.Event) {

			},
			"after_" + EventMA200CrossUp: func(e *fsm.Event) {

			},
		})

		s.state[key] = &State{
			Symbol:     symbol,
			Timeframe:  timeframe,
			LastUpdate: lastUpdate,
			Fsm:        fsmIns,
		}
		return
	}

	currentFsm := s.state[key].Fsm
	currentState := currentFsm.Current()

	if (currentState == MAStateBelow.String() || currentState == MAStateEqual.String()) && lastClosePrice >= lastMA200 {
		// avoid alert twice in the same timeframe period
		if s.state[key].LastUpdate == lastUpdate {
			return
		}
		if err := currentFsm.Event(EventMA200CrossUp); err != nil {
			s.l.Errorw("emit event MA cross up error", "error", err)
			return
		}

		maTrend := getMATrend(previousMA200, lastMA200)
		s.l.Infow("event MA 200 cross up", "symbol", symbol, "timeframe", timeframe, "last_price", lastClosePrice, "ma_trend", maTrend, "next_state", currentFsm.Current(), "last_update", lastUpdate)

		s.sendNotification(true, symbol, timeframe, lastClosePrice, lastMA200, maTrend, lastUpdate)
		s.state[key].LastUpdate = lastUpdate
	}

	if (currentState == MAStateAbove.String() || currentState == MAStateEqual.String()) && lastClosePrice <= lastMA200 {
		// avoid alert twice in the same timeframe period
		if s.state[key].LastUpdate == lastUpdate {
			return
		}
		if err := currentFsm.Event(EventMA200CrossDown); err != nil {
			s.l.Errorw("emit event MA cross down error", "error", err)
			return
		}
		maTrend := getMATrend(previousMA200, lastMA200)
		s.l.Infow("event MA 200 cross down", "symbol", symbol, "timeframe", timeframe, "last_price", lastClosePrice, "ma_trend", maTrend, "next_state", currentFsm.Current(), "last_update", lastUpdate)

		s.sendNotification(false, symbol, timeframe, lastClosePrice, lastMA200, maTrend, lastUpdate)
		s.state[key].LastUpdate = lastUpdate
	}

}

func (s *AlertOnMAStrategy) generateKey(symbol, timeframe string) string {
	key := fmt.Sprintf("%s--%s", symbol, timeframe)
	return key
}

func (s *AlertOnMAStrategy) sendNotification(isUp bool, symbol, timeframe string, lastClosePrice, lastMA200 float64, maTrend string, lastUpdate time.Time) {
	var emoji string
	if isUp {
		emoji = notification.EmojiArrowUp
	} else {
		emoji = notification.EmojiArrowDown
	}

	symbolInfo := fmt.Sprintf("<a href=\"https://www.binance.com/en/trade/%s\">Symbol %s</a>", symbol, symbol)

	msg := fmt.Sprintf("%v MA Cross | %s | Timeframe %v \nLast price: <b>%v</b> \nLast MA200: <b>%v</b> \nMA Trend: <b>%v</b> \nLast Update <b>%v</b>", emoji, symbolInfo, timeframe, lastClosePrice, lastMA200, maTrend, lastUpdate)
	s.notifier.SendMessage(msg)
}

func getMATrend(previousMA200, lastMA200 float64) string {
	var maTrend string
	if previousMA200 < lastMA200 {
		maTrend = MATrendUp
	} else {
		maTrend = MATrendDown
	}
	return maTrend
}
