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
	"github.com/andreyromancev/belt/workers"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/message"
	"github.com/insolar/insolar/ledger/artifactmanager/conver"
)

func (h *MessageHandler) initBelt(ctx context.Context) {
	h.beltHandlers = map[insolar.MessageType]insolar.MessageHandler{}
	h.events = make(chan belt.Event)
	h.Sorter = conver.NewSorter()
	h.Sorter.RegisterInits(h.Handle, h.Handle, h.Handle)
	worker := workers.NewWorker(h.Sorter)
	go func() {
		err := worker.Work(ctx, h.events)
		if err != nil {
			panic(err)
		}
	}()
}

func (h *MessageHandler) Handle(ctx context.Context, e belt.Event) belt.Handler {
	wrapper := e.(conver.MessageBusEventWrapper)
	switch wrapper.Parcel.Message().(type) {
	case *message.GetObject:
		return GetObject{handler: h, parcel: wrapper.Parcel, replyTo: wrapper.ReplyTo}
	default:
	}
	return nil
}

func (h *MessageHandler) WrapMessageBus(ctx context.Context, parcel insolar.Parcel) (insolar.Reply, error) {
	replyTo := make(chan conver.MessageBusReplyWrapper)
	h.events <- conver.MessageBusEventWrapper{Parcel: parcel, ReplyTo: replyTo}
	wrapper := <-replyTo
	return wrapper.Reply, wrapper.Err
}

type GetObject struct {
	parcel  insolar.Parcel
	replyTo chan<- conver.MessageBusReplyWrapper

	handler *MessageHandler
}

func (h GetObject) Handle(ctx context.Context) ([]belt.Handler, error) {
	w := conver.MessageBusReplyWrapper{}
	w.Reply, w.Err = h.handler.beltHandlers[insolar.TypeGetObject](ctx, h.parcel)
	h.replyTo <- w
	return nil, nil
}
