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
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/gorilla/websocket"
	hzws "github.com/hertz-contrib/websocket"
)

func BenchmarkNewWSReverseProxy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewWSReverseProxy("ws://localhost:8888/echo")
	}
}

var (
	proxyURL   = "ws://127.0.0.1:7777"
	backendURL = "ws://127.0.0.1:8888"
)

func TestProxy(t *testing.T) {
	// websocket proxy
	supportedSubProtocols := []string{"test-protocol"}
	upgrader := &hzws.HertzUpgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(c *app.RequestContext) bool {
			return true
		},
		Subprotocols: supportedSubProtocols,
	}

	proxy := NewWSReverseProxy(backendURL, WithUpgrader(upgrader))

	// proxy server
	ps := server.Default(server.WithHostPorts(":7777"))
	ps.NoHijackConnPool = true
	ps.GET("/proxy", proxy.ServeHTTP)
	go ps.Spin()

	time.Sleep(time.Millisecond * 100)

	go func() {
		// backend server
		bs := server.Default()
		bs.NoHijackConnPool = true
		bs.GET("/", func(ctx context.Context, c *app.RequestContext) {
			// Don't upgrade if original host header isn't preserved
			host := string(c.Host())
			if host != "127.0.0.1:7777" {
				hlog.Errorf("Host header set incorrectly.  Expecting 127.0.0.1:7777 got %s", host)
				return
			}

			if err := upgrader.Upgrade(c, func(conn *hzws.Conn) {
				msgType, msg, err := conn.ReadMessage()
				assert.Nil(t, err)

				if err = conn.WriteMessage(msgType, msg); err != nil {
					return
				}
			}); err != nil {
				hlog.Errorf("upgrade error: %v", err)
				return
			}
		})
		bs.Spin()
	}()

	time.Sleep(time.Millisecond * 100)

	// only one is supported by the server
	clientSubProtocols := []string{"test-protocol", "test-notsupported"}
	h := http.Header{}
	for _, subproto := range clientSubProtocols {
		h.Add("Sec-WebSocket-Protocol", subproto)
	}

	// client
	conn, resp, err := websocket.DefaultDialer.Dial(proxyURL+"/proxy", h)
	assert.Nil(t, err)

	// check if the server really accepted the correct protocol
	in := func(desired string) bool {
		for _, proto := range resp.Header[http.CanonicalHeaderKey("Sec-WebSocket-Protocol")] {
			if desired == proto {
				return true
			}
		}
		return false
	}

	assert.True(t, in("test-protocol"))
	assert.False(t, in("test-notsupported"))

	// now write a message and send it to the proxy
	msg := "hello world"
	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	assert.Nil(t, err)

	msgType, data, err := conn.ReadMessage()
	assert.Nil(t, err)
	assert.DeepEqual(t, websocket.TextMessage, msgType)
	assert.DeepEqual(t, msg, string(data))
}

var dynamicBackendURL = "ws://127.0.0.1:8888/api"

func TestProxyWithDynamicRoute(t *testing.T) {
	// websocket proxy
	supportedSubProtocols := []string{"test-protocol"}
	upgrader := &hzws.HertzUpgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(c *app.RequestContext) bool {
			return true
		},
		Subprotocols: supportedSubProtocols,
	}

	// enable dynamic route option
	proxy := NewWSReverseProxy(dynamicBackendURL, WithUpgrader(upgrader), WithDynamicRoute())

	// proxy server
	ps := server.Default(server.WithHostPorts(":7777"))
	ps.NoHijackConnPool = true
	ps.GET("/test", proxy.ServeHTTP)
	ps.GET("/test2", proxy.ServeHTTP)
	go ps.Spin()

	time.Sleep(time.Millisecond * 100)

	go func() {
		// backend server
		bs := server.Default()
		bs.NoHijackConnPool = true
		bs.GET("/api/test", func(ctx context.Context, c *app.RequestContext) {
			// Don't upgrade if original host header isn't preserved
			host := string(c.Host())
			if host != "127.0.0.1:7777" {
				hlog.Errorf("Host header set incorrectly.  Expecting 127.0.0.1:7777 got %s", host)
				return
			}

			if err := upgrader.Upgrade(c, func(conn *hzws.Conn) {
				msgType, msg, err := conn.ReadMessage()
				assert.Nil(t, err)

				if err = conn.WriteMessage(msgType, msg); err != nil {
					return
				}
			}); err != nil {
				hlog.Errorf("upgrade error: %v", err)
				return
			}
		})
		bs.GET("/api/test2", func(ctx context.Context, c *app.RequestContext) {
			// Don't upgrade if original host header isn't preserved
			host := string(c.Host())
			if host != "127.0.0.1:7777" {
				hlog.Errorf("Host header set incorrectly.  Expecting 127.0.0.1:7777 got %s", host)
				return
			}

			if err := upgrader.Upgrade(c, func(conn *hzws.Conn) {
				msgType, msg, err := conn.ReadMessage()
				assert.Nil(t, err)

				if err = conn.WriteMessage(msgType, msg); err != nil {
					return
				}
			}); err != nil {
				hlog.Errorf("upgrade error: %v", err)
				return
			}
		})
		bs.Spin()
	}()

	time.Sleep(time.Millisecond * 100)

	// only one is supported by the server
	clientSubProtocols := []string{"test-protocol", "test-notsupported"}
	h := http.Header{}
	for _, subproto := range clientSubProtocols {
		h.Add("Sec-WebSocket-Protocol", subproto)
	}

	// client
	conn, resp, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:7777/test", h)
	assert.Nil(t, err)
	conn2, resp2, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:7777/test2", h)
	assert.Nil(t, err)

	// check if the server really accepted the correct protocol
	in := func(desired string) bool {
		for _, proto := range resp.Header[http.CanonicalHeaderKey("Sec-WebSocket-Protocol")] {
			if desired == proto {
				return true
			}
		}
		return false
	}
	in2 := func(desired string) bool {
		for _, proto := range resp2.Header[http.CanonicalHeaderKey("Sec-WebSocket-Protocol")] {
			if desired == proto {
				return true
			}
		}
		return false
	}

	assert.True(t, in("test-protocol"))
	assert.True(t, in2("test-protocol"))
	assert.False(t, in("test-notsupported"))
	assert.False(t, in2("test-notsupported"))

	// now write a message and send it to the proxy
	msg := "hello world"
	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	assert.Nil(t, err)

	msg2 := "hello world2"
	err = conn2.WriteMessage(websocket.TextMessage, []byte(msg2))
	assert.Nil(t, err)

	msgType, data, err := conn.ReadMessage()
	assert.Nil(t, err)
	assert.DeepEqual(t, websocket.TextMessage, msgType)
	assert.DeepEqual(t, msg, string(data))

	msgType2, data2, err := conn2.ReadMessage()
	assert.Nil(t, err)
	assert.DeepEqual(t, websocket.TextMessage, msgType2)
	assert.DeepEqual(t, msg2, string(data2))
}
