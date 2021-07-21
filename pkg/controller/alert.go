package controller

import (
	"fmt"
	"sync"
	"time"

	"github.com/quangkeu95/binancebot/pkg/notification"
	"go.uber.org/zap"
)

type Alert struct {
	Name       string
	Symbol     string
	Timeframe  string
	LastUpdate time.Time
	Message    string
}

type AlertController struct {
	sync.RWMutex
	l        *zap.SugaredLogger
	alerts   map[string]Alert
	notifier notification.Notifier
}

func NewAlertController(notifier notification.Notifier) *AlertController {
	notification.NewTelegram()
	return &AlertController{
		l:        zap.S(),
		notifier: notifier,
	}
}

func (c *AlertController) AddAlert(alert Alert) {
	c.Lock()
	defer c.Unlock()
	key := c.generateKey(alert)
	if val, ok := c.alerts[key]; ok {
		if val.LastUpdate == alert.LastUpdate {
			return
		}
	}

	c.l.Debugw("add alert", "name", alert.Name, "symbol", alert.Symbol, "timeframe", alert.Timeframe, "time", alert.LastUpdate)
	c.alerts[key] = alert

	// c.notifier.SendMessage()
}

func (c *AlertController) generateKey(alert Alert) string {
	key := fmt.Sprintf("%s--%s--%s", alert.Symbol, alert.Timeframe, alert.Name)
	return key
}
