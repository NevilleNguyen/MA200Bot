package core

import (
	"fmt"
	"sync"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/looplab/fsm"
	"github.com/quangkeu95/binancebot/pkg/exchange"
	"github.com/quangkeu95/binancebot/pkg/model"
	"github.com/quangkeu95/binancebot/pkg/notification"
	"github.com/quangkeu95/binancebot/pkg/series"
	"github.com/spf13/viper"
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
	VolumePeriodFlag     = "volume_period"
	VolumeMultiplierFlag = "volume_multiplier"
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

type CandleParams struct {
	Symbol             string
	Timeframe          string
	LastUpdate         time.Time
	LastClosePrice     float64
	PreviousPriceMA200 float64
	LastPriceMA200     float64
	PreviousVolume     float64
	LastVolume         float64
}

type AlertOnMAStrategy struct {
	sync.RWMutex
	l                *zap.SugaredLogger
	notifier         notification.Notifier
	state            map[string]*State // store state of previous price vs MA price
	volumePeriod     int
	volumeMultiplier float64
}

func NewAlertOnMAStrategy(notifier notification.Notifier) (*AlertOnMAStrategy, error) {
	l := zap.S()

	volumePeriod := viper.GetInt(VolumePeriodFlag)
	volumeMultiplier := viper.GetFloat64(VolumeMultiplierFlag)
	if err := validation.Validate(volumePeriod, validation.Required); err != nil {
		l.Errorw("parse `volume_period` configuration error", "error", err)
		return nil, err
	}
	if err := validation.Validate(volumeMultiplier, validation.Required); err != nil {
		l.Errorw("parse `volume_multiplier` configuration error", "error", err)
		return nil, err
	}

	return &AlertOnMAStrategy{
		l:                l,
		notifier:         notifier,
		state:            make(map[string]*State),
		volumePeriod:     volumePeriod,
		volumeMultiplier: volumeMultiplier,
	}, nil
}

// Init init is called one time before running strategy
func (s *AlertOnMAStrategy) Init() {
	s.notifier.SendMessage("Start Binance alert bot!!")
}

func (s *AlertOnMAStrategy) WarmupPeriod() int {
	return 201
}

func (s *AlertOnMAStrategy) OnCandle(df *model.Dataframe) {
	prices := df.GetLastValues(model.CandleAttributeClose, 201)
	volumes := df.GetLastValues(model.CandleAttributeVolume, s.volumePeriod+1)

	previousCandleMA200 := series.MA(prices[:200], 200)
	lastCandleMA200 := series.MA(prices[1:], 200)
	lastClosePrice := df.GetLast(model.CandleAttributeClose, 0)

	previousCandleVolume := series.MA(volumes[:s.volumePeriod], s.volumePeriod)
	lastCandleVolume := df.GetLast(model.CandleAttributeVolume, 0)

	s.handleMACross(CandleParams{
		Symbol:             df.Symbol,
		Timeframe:          df.Timeframe,
		LastUpdate:         df.GetLastUpdate(),
		LastClosePrice:     lastClosePrice,
		LastPriceMA200:     lastCandleMA200,
		PreviousPriceMA200: previousCandleMA200,
		PreviousVolume:     previousCandleVolume,
		LastVolume:         lastCandleVolume,
	})
}

func (s *AlertOnMAStrategy) handleMACross(params CandleParams) {
	s.Lock()
	defer s.Unlock()

	key := s.generateKey(params.Symbol, params.Timeframe)

	if _, ok := s.state[key]; !ok {
		var (
			state string
		)
		if params.LastClosePrice > params.LastPriceMA200 {
			state = MAStateAbove.String()
		} else if params.LastClosePrice < params.LastPriceMA200 {
			state = MAStateBelow.String()
		} else {
			state = MAStateEqual.String()
		}

		s.l.Infow("init state", "symbol", params.Symbol,
			"timeframe", params.Timeframe,
			"state", state,
			"last_price", params.LastClosePrice,
			"previous_ma200", params.PreviousPriceMA200,
			"last_ma200", params.LastPriceMA200,
			"last_update", params.LastUpdate,
			"last_update_unix", params.LastUpdate.Unix(),
		)

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
			Symbol:     params.Symbol,
			Timeframe:  params.Timeframe,
			LastUpdate: params.LastUpdate,
			Fsm:        fsmIns,
		}
		return
	}

	currentFsm := s.state[key].Fsm
	currentState := currentFsm.Current()

	if (currentState == MAStateBelow.String() || currentState == MAStateEqual.String()) && params.LastClosePrice >= params.LastPriceMA200 {
		// avoid alert twice in the same timeframe period
		if s.state[key].LastUpdate == params.LastUpdate {
			return
		}

		// if !s.isEnoughVolume(params) {
		// 	return
		// }
		if err := currentFsm.Event(EventMA200CrossUp); err != nil {
			s.l.Errorw("emit event MA cross up error", "error", err)
			return
		}

		maTrend := getMATrend(params.PreviousPriceMA200, params.LastPriceMA200)
		s.l.Infow("event MA 200 cross up", "symbol", params.Symbol,
			"timeframe", params.Timeframe,
			"last_price", params.LastClosePrice,
			"ma_trend", maTrend,
			"next_state", currentFsm.Current(),
			"last_volume", params.LastVolume,
			"previous_volume", params.PreviousVolume,
			"last_update", params.LastUpdate)

		s.sendNotification(true, maTrend, params)
		s.state[key].LastUpdate = params.LastUpdate
	}

	if (currentState == MAStateAbove.String() || currentState == MAStateEqual.String()) && params.LastClosePrice <= params.LastPriceMA200 {
		// avoid alert twice in the same timeframe period
		if s.state[key].LastUpdate == params.LastUpdate {
			return
		}
		// if !s.isEnoughVolume(params) {
		// 	return
		// }
		if err := currentFsm.Event(EventMA200CrossDown); err != nil {
			s.l.Errorw("emit event MA cross down error", "error", err)
			return
		}
		maTrend := getMATrend(params.PreviousPriceMA200, params.LastPriceMA200)
		s.l.Infow("event MA 200 cross down",
			"symbol", params.Symbol,
			"timeframe", params.Timeframe,
			"last_price", params.LastClosePrice,
			"ma_trend", maTrend,
			"next_state", currentFsm.Current(),
			"last_volume", params.LastVolume,
			"previous_volume", params.PreviousVolume,
			"last_update", params.LastUpdate)

		s.sendNotification(false, maTrend, params)
		s.state[key].LastUpdate = params.LastUpdate
	}

}

func (s *AlertOnMAStrategy) generateKey(symbol, timeframe string) string {
	key := fmt.Sprintf("%s--%s", symbol, timeframe)
	return key
}

func (s *AlertOnMAStrategy) sendNotification(isUp bool, maTrend string, params CandleParams) {
	var emoji string
	if isUp {
		emoji = notification.EmojiArrowUp
	} else {
		emoji = notification.EmojiArrowDown
	}

	symbolInfo := fmt.Sprintf("<a href=\"https://www.binance.com/en/trade/%s\">Symbol %s</a>", params.Symbol, params.Symbol)
	lastPriceInfo := fmt.Sprintf("Last price: <b>%v</b>", params.LastClosePrice)
	lastMA200Info := fmt.Sprintf("Last MA200: <b>%v</b>", params.LastPriceMA200)
	maTrendInfo := fmt.Sprintf("MA Trend: <b>%v</b>", maTrend)
	lastUpdateInfo := fmt.Sprintf("Last Update: <b>%v</b>", params.LastUpdate)
	lastVolumeInfo := fmt.Sprintf("Last Volume: <b>%f</b>", params.LastVolume)
	previousVolumeInfo := fmt.Sprintf("Previous Volume: <b>%f</b>", params.PreviousVolume)

	compareVolumeInfo := fmt.Sprintf("Compare volume: %v", s.getVolumeInfo(params))
	// compareVolumeInfo := ""

	msg := fmt.Sprintf("%v MA Cross | %s | Timeframe %v \n%v \n%v \n%v \n%v \n%v \n%v \n%v",
		emoji, symbolInfo, params.Timeframe,
		lastPriceInfo, lastMA200Info, lastVolumeInfo, previousVolumeInfo, maTrendInfo, compareVolumeInfo, lastUpdateInfo)
	s.notifier.SendMessage(msg)
}

func (s *AlertOnMAStrategy) isEnoughVolume(params CandleParams) bool {
	timeframeInSeconds := exchange.ParseTimeframeToSeconds(params.Timeframe)
	lastUpdate := params.LastUpdate.UnixNano() / 1_000_000
	now := time.Now().UnixNano() / 1_000_000
	ratio := float64(now-lastUpdate) / float64(timeframeInSeconds*1000)

	return params.LastVolume >= s.volumeMultiplier*params.PreviousVolume*ratio
}

func (s *AlertOnMAStrategy) getVolumeInfo(params CandleParams) string {
	timeframeInSeconds := exchange.ParseTimeframeToSeconds(params.Timeframe)
	lastUpdate := params.LastUpdate.UnixNano() / 1_000_000
	now := time.Now().UnixNano() / 1_000_000
	ratio := float64(now-lastUpdate) / float64(timeframeInSeconds*1000)

	return fmt.Sprintf("Current volume <b>%f</b> - Previous Volume with ratio <b>%f</b>", params.LastVolume, s.volumeMultiplier*params.PreviousVolume*ratio)
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
