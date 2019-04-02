///
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
///

package artifactmanager

import (
	"context"

	"github.com/andreyromancev/belt"
	"github.com/andreyromancev/belt/mware"
	"github.com/andreyromancev/belt/workers"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/message"
	"github.com/insolar/insolar/insolar/reply"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/ledger/artifactmanager/conver"
	"github.com/insolar/insolar/ledger/storage/blob"
	"github.com/insolar/insolar/ledger/storage/object"
	"github.com/pkg/errors"
)

func (h *MessageHandler) initBelt(ctx context.Context) {
	h.beltHandlers = map[insolar.MessageType]insolar.MessageHandler{}
	h.events = make(chan belt.Event)
	h.Sorter = conver.NewSorter(mware.Func(h.provide))
	h.Sorter.RegisterInits(h.Handle, h.Handle, h.Handle)
	worker := workers.NewWorker(h.Sorter)
	go func() {
		err := worker.Work(ctx, h.events)
		if err != nil {
			panic(err)
		}
	}()
}

func (h *MessageHandler) WrapMessageBus(ctx context.Context, parcel insolar.Parcel) (insolar.Reply, error) {
	replyTo := make(chan conver.MessageBusReplyWrapper)
	h.events <- conver.MessageBusEventWrapper{Parcel: parcel, ReplyTo: replyTo}
	wrapper := <-replyTo
	return wrapper.Reply, wrapper.Err
}

type Provide interface {
	Provide(mh *MessageHandler, p insolar.PulseNumber, m insolar.Parcel, r chan<- conver.MessageBusReplyWrapper)
}

func (h *MessageHandler) provide(ctx context.Context, i belt.Item) ([]belt.Handler, error) {
	msgHandler, ok := i.Handler().(Provide)
	if !ok {
		return i.Handler().Handle(ctx)
	}
	wrapper, ok := i.Event().(conver.MessageBusEventWrapper)
	if !ok {
		return i.Handler().Handle(ctx)
	}

	msgHandler.Provide(h, wrapper.Parcel.Pulse(), wrapper.Parcel, wrapper.ReplyTo)
	return i.Handler().Handle(ctx)
}

// =====================================================================================================================

func (h *MessageHandler) Handle(ctx context.Context, e belt.Event) belt.Handler {
	wrapper := e.(conver.MessageBusEventWrapper)
	switch wrapper.Parcel.Message().(type) {
	case *message.GetObject:
		return &CheckJet{Next: &WaitForHot{Next: &GetObject{}}}
	default:
	}
	return nil
}

// =====================================================================================================================
type messageHandler struct {
	pulse   insolar.PulseNumber
	parcel  insolar.Parcel
	handler *MessageHandler
	replyTo chan<- conver.MessageBusReplyWrapper
}

func (h *messageHandler) Provide(mh *MessageHandler, p insolar.PulseNumber, m insolar.Parcel, r chan<- conver.MessageBusReplyWrapper) {
	h.pulse = p
	h.parcel = m
	h.handler = mh
	h.replyTo = r
}

type CheckJet struct {
	messageHandler
	returnCtx context.Context
	Next      belt.Handler
}

func (h *CheckJet) Handle(ctx context.Context) ([]belt.Handler, error) {
	msg := h.parcel.Message()
	if msg.DefaultTarget() == nil {
		return nil, errors.New("unexpected message")
	}

	// Hack to temporary allow any genesis request.
	if h.parcel.Pulse() == insolar.FirstPulseNumber {
		h.returnCtx = contextWithJet(ctx, insolar.ID(*insolar.NewJetID(0, nil)))
		return []belt.Handler{h.Next}, nil
	}

	// Check token jet.
	token := h.parcel.DelegationToken()
	if token != nil {
		// Calculate jet for target pulse.
		target := *msg.DefaultTarget().Record()
		pulse := target.Pulse()
		switch tm := msg.(type) {
		case *message.GetObject:
			pulse = tm.State.Pulse()
		case *message.GetChildren:
			if tm.FromChild == nil {
				return nil, errors.New("fetching children without child pointer is forbidden")
			}
			pulse = tm.FromChild.Pulse()
		case *message.GetRequest:
			pulse = tm.Request.Pulse()
		}
		jetID, actual := h.handler.JetStorage.ForID(ctx, pulse, target)
		if !actual {
			inslogger.FromContext(ctx).WithFields(map[string]interface{}{
				"msg":   msg.Type().String(),
				"jet":   jetID.DebugString(),
				"pulse": pulse,
			}).Error("jet is not actual")
		}

		h.returnCtx = contextWithJet(ctx, insolar.ID(jetID))
		return []belt.Handler{h.Next}, nil
	}

	// Calculate jet for current pulse.
	var jetID insolar.ID
	if msg.DefaultTarget().Record().Pulse() == insolar.PulseNumberJet {
		jetID = *msg.DefaultTarget().Record()
	} else {
		j, err := h.handler.jetTreeUpdater.fetchJet(ctx, *msg.DefaultTarget().Record(), h.parcel.Pulse())
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch jet tree")
		}

		jetID = *j
	}

	// Check if jet is ours.
	node, err := h.handler.JetCoordinator.LightExecutorForJet(ctx, jetID, h.parcel.Pulse())
	if err != nil {
		return nil, errors.Wrap(err, "failed to calculate executor for jet")
	}

	if *node != h.handler.JetCoordinator.Me() {
		h.replyTo <- conver.MessageBusReplyWrapper{Reply: &reply.JetMiss{JetID: jetID}}
		return nil, nil
	}

	ctx = addJetIDToLogger(ctx, jetID)

	h.returnCtx = contextWithJet(ctx, jetID)
	return []belt.Handler{h.Next}, nil
}

func (h *CheckJet) Context() context.Context {
	return h.returnCtx
}

type WaitForHot struct {
	messageHandler
	Next belt.Handler
}

func (h *WaitForHot) Handle(ctx context.Context) ([]belt.Handler, error) {
	// Hack is needed for genesis:
	// because we don't have hot data on first pulse and without this we would stale.
	if h.parcel.Pulse() == insolar.FirstPulseNumber {
		return []belt.Handler{h.Next}, nil
	}

	// If the call is a call in redirect-chain
	// skip waiting for the hot records
	if h.parcel.DelegationToken() != nil {
		return []belt.Handler{h.Next}, nil
	}

	jetID := jetFromContext(ctx)
	err := h.handler.HotDataWaiter.Wait(ctx, jetID)
	if err != nil {
		h.replyTo <- conver.MessageBusReplyWrapper{Reply: &reply.Error{ErrType: reply.ErrHotDataTimeout}}
		return nil, nil
	}
	return []belt.Handler{h.Next}, nil
}

type GetObject struct {
	messageHandler
}

func (h *GetObject) Handle(ctx context.Context) ([]belt.Handler, error) {
	w := conver.MessageBusReplyWrapper{}
	w.Reply, w.Err = h.handle(ctx, h.parcel)
	h.replyTo <- w
	return nil, nil
}

func (h *GetObject) handle(
	ctx context.Context, parcel insolar.Parcel,
) (insolar.Reply, error) {
	msg := parcel.Message().(*message.GetObject)
	jetID := jetFromContext(ctx)
	logger := inslogger.FromContext(ctx).WithFields(map[string]interface{}{
		"object": msg.Head.Record().DebugString(),
		"pulse":  parcel.Pulse(),
	})

	h.handler.RecentStorageProvider.GetIndexStorage(ctx, jetID).AddObject(ctx, *msg.Head.Record())

	h.handler.IDLocker.Lock(msg.Head.Record())
	defer h.handler.IDLocker.Unlock(msg.Head.Record())

	// Fetch object index. If not found redirect.
	idx, err := h.handler.ObjectStorage.GetObjectIndex(ctx, jetID, msg.Head.Record())
	if err == insolar.ErrNotFound {
		logger.Debug("failed to fetch index (fetching from heavy)")
		node, err := h.handler.JetCoordinator.Heavy(ctx, parcel.Pulse())
		if err != nil {
			return nil, err
		}
		idx, err = h.handler.saveIndexFromHeavy(ctx, jetID, msg.Head, node)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch index from heavy")
		}
	} else if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch object index %s", msg.Head.Record().String())
	}

	// Determine object state id.
	var stateID *insolar.ID
	if msg.State != nil {
		stateID = msg.State
	} else {
		if msg.Approved {
			stateID = idx.LatestStateApproved
		} else {
			stateID = idx.LatestState
		}
	}
	if stateID == nil {
		return &reply.Error{ErrType: reply.ErrStateNotAvailable}, nil
	}

	var (
		stateJet *insolar.ID
	)
	onHeavy, err := h.handler.JetCoordinator.IsBeyondLimit(ctx, parcel.Pulse(), stateID.Pulse())
	if err != nil {
		return nil, err
	}
	if onHeavy {
		hNode, err := h.handler.JetCoordinator.Heavy(ctx, parcel.Pulse())
		if err != nil {
			return nil, err
		}
		logger.WithFields(map[string]interface{}{
			"state":    stateID.DebugString(),
			"going_to": hNode.String(),
		}).Debug("fetching object (on heavy)")

		obj, err := h.handler.fetchObject(ctx, msg.Head, *hNode, stateID, parcel.Pulse())
		if err != nil {
			if err == insolar.ErrDeactivated {
				return &reply.Error{ErrType: reply.ErrDeactivated}, nil
			}
			return nil, err
		}

		return &reply.Object{
			Head:         msg.Head,
			State:        *stateID,
			Prototype:    obj.Prototype,
			IsPrototype:  obj.IsPrototype,
			ChildPointer: idx.ChildPointer,
			Parent:       idx.Parent,
			Memory:       obj.Memory,
		}, nil
	}

	stateJetID, actual := h.handler.JetStorage.ForID(ctx, stateID.Pulse(), *msg.Head.Record())
	stateJet = (*insolar.ID)(&stateJetID)

	if !actual {
		actualJet, err := h.handler.jetTreeUpdater.fetchJet(ctx, *msg.Head.Record(), stateID.Pulse())
		if err != nil {
			return nil, err
		}
		stateJet = actualJet
	}

	// Fetch state record.
	rec, err := h.handler.ObjectStorage.GetRecord(ctx, *stateJet, stateID)
	if err == insolar.ErrNotFound {
		// The record wasn't found on the current suitNode. Return redirect to the suitNode that contains it.
		// We get Jet tree for pulse when given state was added.
		suitNode, err := h.handler.JetCoordinator.NodeForJet(ctx, *stateJet, parcel.Pulse(), stateID.Pulse())
		if err != nil {
			return nil, err
		}
		logger.WithFields(map[string]interface{}{
			"state":    stateID.DebugString(),
			"going_to": suitNode.String(),
		}).Debug("fetching object (record not found)")

		obj, err := h.handler.fetchObject(ctx, msg.Head, *suitNode, stateID, parcel.Pulse())
		if err != nil {
			if err == insolar.ErrDeactivated {
				return &reply.Error{ErrType: reply.ErrDeactivated}, nil
			}
			return nil, err
		}

		return &reply.Object{
			Head:         msg.Head,
			State:        *stateID,
			Prototype:    obj.Prototype,
			IsPrototype:  obj.IsPrototype,
			ChildPointer: idx.ChildPointer,
			Parent:       idx.Parent,
			Memory:       obj.Memory,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	state, ok := rec.(object.State)
	if !ok {
		return nil, errors.New("invalid object record")
	}
	if state.ID() == object.StateDeactivation {
		return &reply.Error{ErrType: reply.ErrDeactivated}, nil
	}

	var childPointer *insolar.ID
	if idx.ChildPointer != nil {
		childPointer = idx.ChildPointer
	}
	rep := reply.Object{
		Head:         msg.Head,
		State:        *stateID,
		Prototype:    state.GetImage(),
		IsPrototype:  state.GetIsPrototype(),
		ChildPointer: childPointer,
		Parent:       idx.Parent,
	}

	if state.GetMemory() != nil {
		b, err := h.handler.BlobAccessor.ForID(ctx, *state.GetMemory())
		if err == blob.ErrNotFound {
			hNode, err := h.handler.JetCoordinator.Heavy(ctx, parcel.Pulse())
			if err != nil {
				return nil, err
			}
			obj, err := h.handler.fetchObject(ctx, msg.Head, *hNode, stateID, parcel.Pulse())
			if err != nil {
				return nil, err
			}
			err = h.handler.BlobModifier.Set(ctx, *state.GetMemory(), blob.Blob{JetID: insolar.JetID(jetID), Value: obj.Memory})
			if err != nil {
				return nil, err
			}
			b.Value = obj.Memory
		}
		rep.Memory = b.Value
	}

	return &rep, nil
}
