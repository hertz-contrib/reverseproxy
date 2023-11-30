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
	"github.com/gorilla/websocket"
	hzws "github.com/hertz-contrib/websocket"
)

func BenchmarkNewWSReverseProxy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewWSReverseProxy("ws://localhost:8888/echo")
	}
}

var (
	serverURL  = "ws://127.0.0.1:7777"
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
	ps.GET("/proxy", proxy.ServeHTTP)
	go ps.Spin()

	time.Sleep(time.Millisecond * 100)

	go func() {
		// backend server
		bs := server.Default()
		bs.GET("/", func(ctx context.Context, c *app.RequestContext) {
			// Don't upgrade if original host header isn't preserved
			host := string(c.Host())
			if host != "127.0.0.1:7777" {
				hlog.Errorf("Host header set incorrectly.  Expecting 127.0.0.1:7777 got %s", host)
				return
			}

			if err := upgrader.Upgrade(c, func(conn *hzws.Conn) {
				msgType, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}

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

	// let us define two subprotocols, only one is supported by the server
	clientSubProtocols := []string{"test-protocol", "test-notsupported"}
	h := http.Header{}
	for _, subproto := range clientSubProtocols {
		h.Add("Sec-WebSocket-Protocol", subproto)
	}

	// frontend server, dial now our proxy, which will reverse proxy our
	// message to the backend websocket server.
	conn, resp, err := websocket.DefaultDialer.Dial(serverURL+"/proxy", h)
	if err != nil {
		t.Fatal(err)
	}

	// check if the server really accepted only the first one
	in := func(desired string) bool {
		for _, prot := range resp.Header[http.CanonicalHeaderKey("Sec-WebSocket-Protocol")] {
			if desired == prot {
				return true
			}
		}
		return false
	}

	if !in("test-protocol") {
		t.Error("test-protocol should be available")
	}

	if in("test-notsupported") {
		t.Error("test-notsupported should be not recevied from the server.")
	}

	// now write a message and send it to the backend server (which goes through proxy)
	msg := "hello kite"
	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		t.Error(err)
	}

	messageType, p, err := conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}

	if messageType != websocket.TextMessage {
		t.Error("incoming message type is not Text")
	}

	if msg != string(p) {
		t.Errorf("expecting: %s, got: %s", msg, string(p))
	}
}
