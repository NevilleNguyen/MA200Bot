package strategy

import "github.com/quangkeu95/binancebot/pkg/model"

type Strategy interface {
	Init()
	WarmupPeriod() int
	OnCandle(dataframe *model.Dataframe)
}
