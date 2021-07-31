package notification

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/quangkeu95/binancebot/lib/app"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	EmojiArrowDown = "\U00002B07"
	EmojiArrowUp   = "\U00002B06"
)

const (
	TelegramTokenFlag        = "telegram.token"
	TelegramChatIdFlag       = "telegram.chat_id"
	DefaultTelegramRateLimit = 30
	DefaultTelegramRateBurst = 1
	DefaultTelegramTimeout   = 5 * time.Second
)

type Telegram struct {
	l           *zap.SugaredLogger
	api         *tgbotapi.BotAPI
	chatId      int64
	rateLimiter *app.RateLimiter
}

func init() {
	viper.BindEnv(TelegramTokenFlag, "TELEGRAM_TOKEN")
	viper.BindEnv(TelegramChatIdFlag, "TELEGRAM_CHAT_ID")
}

func NewTelegram() (*Telegram, error) {
	l := zap.S()

	token := viper.GetString(TelegramTokenFlag)
	if err := validation.Validate(token, validation.Required); err != nil {
		l.Errorw("initialize telegram client error", "error", err)
		return nil, err
	}
	chatId := viper.GetInt64(TelegramChatIdFlag)
	if err := validation.Validate(chatId, validation.Required); err != nil {
		l.Errorw("initialize telegram client error", "error", err)
		return nil, err
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		l.Errorw("initialize telegram bot error", "error", err)
		return nil, err
	}

	return &Telegram{
		l:           l,
		api:         api,
		chatId:      chatId,
		rateLimiter: app.NewRateLimiter(DefaultTelegramRateLimit, DefaultTelegramRateBurst),
	}, nil
}

func (t Telegram) SendMessage(msg string) error {
	sendMsg := tgbotapi.NewMessage(t.chatId, msg)
	sendMsg.ParseMode = tgbotapi.ModeHTML
	sendMsg.DisableWebPagePreview = true

	if err := t.rateLimiter.WaitN(DefaultTelegramTimeout, 1); err != nil {
		t.l.Errorw("telegram bot send message error rate limit", "error", err)
		return err
	}

	_, err := t.api.Send(sendMsg)
	if err != nil {
		t.l.Errorw("telegram bot send message error", "error", err)
		return err
	}
	return nil
}

func (t Telegram) OnError(err error) {

}
