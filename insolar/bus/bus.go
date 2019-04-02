package bus

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/insolar/insolar/insolar/bus/tcp"
)

type Bus struct {
	router     *message.Router
	subscriber message.Subscriber
	publisher  message.Publisher
}

func NewBus() *Bus {
	sub := tcp.NewSubscriber()
	pub := tcp.NewPublisher()
	r, err := message.NewRouter(
		message.RouterConfig{},
		watermill.NopLogger{},
	)
	if err != nil {
		panic(err)
	}
	return &Bus{
		router:     r,
		subscriber: sub,
		publisher:  pub,
	}
}

func (b *Bus) Run() {
	if err := b.router.Run(); err != nil {
		panic(err)
	}
}

func (b *Bus) Subscribe(topic, name string, handle Handle) {
	b.router.AddNoPublisherHandler(
		name,
		topic,
		tcp.NewSubscriber(),
		wrapHandle(handle),
	)
}

func wrapHandle(handle Handle) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		return nil, handle(msg)
	}
}
