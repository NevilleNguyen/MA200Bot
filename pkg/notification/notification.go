package notification

type Notifier interface {
	SendMessage(msg string) error
	OnError(err error)
}
