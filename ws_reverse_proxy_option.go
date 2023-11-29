package reverseproxy

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/gorilla/websocket"
	hzws "github.com/hertz-contrib/websocket"
)

type Director func(ctx context.Context, c *app.RequestContext) *protocol.RequestHeader

type Option func(o *Options)

type Options struct {
	director Director
	dialer   *websocket.Dialer
	upgrader *hzws.HertzUpgrader
}

var defaultOptions = &Options{
	director: nil,
	dialer:   websocket.DefaultDialer,
	upgrader: &hzws.HertzUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	},
}

func newOptions(opts ...Option) *Options {
	options := &Options{
		director: defaultOptions.director,
		dialer:   defaultOptions.dialer,
		upgrader: defaultOptions.upgrader,
	}
	options.apply(opts...)
	return options
}

func (o *Options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

func WithDialer(dialer *websocket.Dialer) Option {
	return func(o *Options) {
		o.dialer = dialer
	}
}

func WithDirector(director Director) Option {
	return func(o *Options) {
		o.director = director
	}
}

func WithUpgrader(upgrader *hzws.HertzUpgrader) Option {
	return func(o *Options) {
		o.upgrader = upgrader
	}
}
