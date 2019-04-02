package conver

import (
	"context"
	"sync"

	"github.com/insolar/insolar/insolar"

	"github.com/andreyromancev/belt/mware"

	"github.com/andreyromancev/belt/items"

	"github.com/andreyromancev/belt"
	"github.com/andreyromancev/belt/slots"
	"github.com/pkg/errors"
)

type MessageBusReplyWrapper struct {
	Reply insolar.Reply
	Err   error
}

type MessageBusEventWrapper struct {
	ReplyTo chan<- MessageBusReplyWrapper
	Parcel  insolar.Parcel
}

type initHandler func(ctx context.Context, e belt.Event) belt.Handler

type Sorter struct {
	sLock                 sync.RWMutex
	past, present, future belt.Slot

	hPast, hPresent, hFuture initHandler
	pulse                    insolar.PulseStorage
}

func NewSorter() *Sorter {
	return &Sorter{
		past:    slots.NewSlot(PastMiddleware),
		present: slots.NewSlot(PresentMiddleware),
		future:  slots.NewSlot(FutureMiddleware),
	}
}

func (s *Sorter) RegisterInits(past, present, future initHandler) {
	s.hPast, s.hPresent, s.hFuture = past, present, future
}

func (s *Sorter) Sort(ctx context.Context, e belt.Event) (belt.Slot, belt.Item, error) {
	if wrapper, ok := e.(MessageBusEventWrapper); ok {
		slot := s.present
		handler := s.hPresent(ctx, wrapper)
		item := items.NewItem(ctx, wrapper, handler)
		return slot, item, nil
	}

	panic("wrong event")
}

func (s *Sorter) OnPulse(pn insolar.PulseNumber) {
	s.sLock.Lock()
	defer s.sLock.Unlock()

	// Deactivate past.
	inactive := mware.Func(func(c context.Context, handler belt.Handler) ([]belt.Handler, error) {
		return nil, errors.New("inactive")
	})
	s.past.Reset(inactive)

	// Move present to past.
	s.present.Reset(PastMiddleware)
	s.past = s.present

	// Move future to present.
	s.future.Reset(PresentMiddleware)
	s.present = s.future

	// Create future.
	future := slots.NewSlot(FutureMiddleware)
	s.future = future
}
