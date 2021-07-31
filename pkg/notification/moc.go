package notification

type MocNotifier struct{}

func NewMocNotifier() *MocNotifier {
	return &MocNotifier{}
}

func (m *MocNotifier) SendMessage(msg string) error {
	return nil
}

func (m *MocNotifier) OnError(err error) {

}
