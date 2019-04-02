package bus

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

type Handle func(*message.Message) error
