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
	"fmt"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/gorilla/websocket"
	hzws "github.com/hertz-contrib/websocket"
)

func TestOptions(t *testing.T) {
	director := func(ctx context.Context, c *app.RequestContext, forwardHeader http.Header) {
		forwardHeader.Add("X-Test-Head", "content")
	}
	dialer := websocket.DefaultDialer
	upgrader := &hzws.HertzUpgrader{
		ReadBufferSize:  64,
		WriteBufferSize: 64,
	}
	options := newOptions(
		WithDirector(director),
		WithDialer(dialer),
		WithUpgrader(upgrader),
	)
	assert.DeepEqual(t, fmt.Sprintf("%p", director), fmt.Sprintf("%p", options.Director))
	assert.DeepEqual(t, fmt.Sprintf("%p", dialer), fmt.Sprintf("%p", options.Dialer))
	assert.DeepEqual(t, fmt.Sprintf("%p", upgrader), fmt.Sprintf("%p", options.Upgrader))
}

func TestDefaultOptions(t *testing.T) {
	options := newOptions()
	assert.Nil(t, options.Director)
	assert.DeepEqual(t, DefaultOptions.Dialer, options.Dialer)
	assert.DeepEqual(t, DefaultOptions.Upgrader, options.Upgrader)
}
