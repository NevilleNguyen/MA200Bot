package notification

import (
	validation "github.com/go-ozzo/ozzo-validation"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	EmojiArrowDown = "\U00002B07"
	EmojiArrowUp   = "\U00002B06"
)

const (
	TelegramTokenFlag  = "telegram.token"
	TelegramChatIdFlag = "telegram.chat_id"
)

type Telegram struct {
	l      *zap.SugaredLogger
	api    *tgbotapi.BotAPI
	chatId int64
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
		l:      l,
		api:    api,
		chatId: chatId,
	}, nil
}

func (t Telegram) SendMessage(msg string) {
	sendMsg := tgbotapi.NewMessage(t.chatId, msg)
	sendMsg.ParseMode = tgbotapi.ModeHTML
	sendMsg.DisableWebPagePreview = true

	_, err := t.api.Send(sendMsg)
	if err != nil {
		t.l.Debugw("telegram bot send message error", "error", err)
	}
}

func (t Telegram) OnError(err error) {

}
