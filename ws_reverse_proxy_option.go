// Copyright 2023 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reverseproxy

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/gorilla/websocket"
	hzws "github.com/hertz-contrib/websocket"
)

type Director func(ctx context.Context, c *app.RequestContext, forwardHeader http.Header)

type Option func(o *Options)

type Options struct {
	Director     Director
	Dialer       *websocket.Dialer
	Upgrader     *hzws.HertzUpgrader
	DynamicRoute bool
}

var DefaultOptions = &Options{
	Director: nil,
	Dialer:   websocket.DefaultDialer,
	Upgrader: &hzws.HertzUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	},
	DynamicRoute: false,
}

func newOptions(opts ...Option) *Options {
	options := &Options{
		Director: DefaultOptions.Director,
		Dialer:   DefaultOptions.Dialer,
		Upgrader: DefaultOptions.Upgrader,
	}
	options.apply(opts...)
	return options
}

func (o *Options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithDialer for dialer customization
func WithDialer(dialer *websocket.Dialer) Option {
	return func(o *Options) {
		o.Dialer = dialer
	}
}

// WithDirector user can edit the forward header by using custom Director
// NOTE: custom Director will overwrite default forward header field if they have the same key
func WithDirector(director Director) Option {
	return func(o *Options) {
		o.Director = director
	}
}

// WithUpgrader for upgrader customization
func WithUpgrader(upgrader *hzws.HertzUpgrader) Option {
	return func(o *Options) {
		o.Upgrader = upgrader
	}
}

// WithDynamicRoute enable dynamic route
// backend url = handler url + proxy url
func WithDynamicRoute() Option {
	return func(o *Options) {
		o.DynamicRoute = true
	}
}
