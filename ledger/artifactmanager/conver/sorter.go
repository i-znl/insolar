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

	mPast, mPresent, mFuture belt.Middleware

	hPast, hPresent, hFuture initHandler
	pulse                    insolar.PulseStorage
}

func NewSorter(mw belt.Middleware) *Sorter {
	past := mware.NewChain()
	past.AddMiddleware(mw)
	past.AddMiddleware(ContextMiddleware)
	past.AddMiddleware(PastMiddleware)

	present := mware.NewChain()
	present.AddMiddleware(mw)
	present.AddMiddleware(ContextMiddleware)
	present.AddMiddleware(PresentMiddleware)

	future := mware.NewChain()
	future.AddMiddleware(mw)
	future.AddMiddleware(ContextMiddleware)
	future.AddMiddleware(FutureMiddleware)

	return &Sorter{
		mPast:    past,
		mPresent: present,
		mFuture:  future,
		past:     slots.NewSlot(past),
		present:  slots.NewSlot(present),
		future:   slots.NewSlot(future),
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
	inactive := mware.Func(func(c context.Context, handler belt.Item) ([]belt.Handler, error) {
		return nil, errors.New("inactive")
	})
	s.past.Reset(inactive)

	// Move present to past.
	s.present.Reset(s.mPast)
	s.past = s.present

	// Move future to present.
	s.future.Reset(s.mPresent)
	s.present = s.future

	// Create future.
	future := slots.NewSlot(s.mFuture)
	s.future = future
}
