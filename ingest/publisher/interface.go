package publisher

type Publisher interface {
	Publish(message []byte) error
}
