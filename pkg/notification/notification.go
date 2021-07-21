package notification

type Notifier interface {
	SendMessage(msg string)
	OnError(err error)
}
