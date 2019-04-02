package conver

import (
	"context"

	"github.com/andreyromancev/belt"
	"github.com/andreyromancev/belt/log"
	"github.com/andreyromancev/belt/mware"
	"github.com/pkg/errors"
)

var PresentMiddleware mware.Func = func(ctx context.Context, i belt.Item) ([]belt.Handler, error) {
	return i.Handler().Handle(ctx)
}

type FutureHandler interface {
	Future(ctx context.Context) ([]belt.Handler, error)
}

var FutureMiddleware mware.Func = func(ctx context.Context, i belt.Item) ([]belt.Handler, error) {
	if f, ok := i.Handler().(FutureHandler); ok {
		return f.Future(ctx)
	}

	log.FromContext(ctx).Info("Waiting for present")
	// Future waits for reset by default.
	<-ctx.Done()
	return []belt.Handler{i.Handler()}, nil
}

type PastHandler interface {
	Past(ctx context.Context) ([]belt.Handler, error)
}

var PastMiddleware mware.Func = func(ctx context.Context, i belt.Item) ([]belt.Handler, error) {
	if f, ok := i.Handler().(PastHandler); ok {
		return f.Past(ctx)
	}

	// Past fails by default.
	return nil, errors.New("no past handler")
}

type ContextHandler interface {
	Context() context.Context
}

var ContextMiddleware mware.Func = func(ctx context.Context, i belt.Item) ([]belt.Handler, error) {
	res, err := i.Handler().Handle(ctx)

	if ch, ok := i.Handler().(ContextHandler); ok {
		i.SetContext(ch.Context())
	}

	return res, err
}
