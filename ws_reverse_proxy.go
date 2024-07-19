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
//
// MIT License
//
// Copyright (c) 2018 YeQiang
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// This file may have been modified by CloudWeGo authors.
// All CloudWeGo Modifications are Copyright 2023 CloudWeGo Authors.

package reverseproxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/gorilla/websocket"
	hzws "github.com/hertz-contrib/websocket"
)

type WSReverseProxy struct {
	target  string
	options *Options
}

// NewWSReverseProxy new a proxy which will provide handler for websocket reverse proxy
func NewWSReverseProxy(target string, opts ...Option) *WSReverseProxy {
	if target == "" {
		panic("target string must not be empty")
	}
	options := newOptions(opts...)
	wsrp := &WSReverseProxy{
		target:  target,
		options: options,
	}
	return wsrp
}

// ServeHTTP provides websocket reverse proxy service
func (w *WSReverseProxy) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	forwardHeader := prepareForwardHeader(ctx, c)
	// NOTE: customer Director will overwrite existed header if they have the same header key
	if w.options.Director != nil {
		w.options.Director(ctx, c, forwardHeader)
	}
	target := w.target
	if w.options.DynamicRoute {
		target = w.target + b2s(c.Path())
	}
	connBackend, respBackend, err := w.options.Dialer.Dial(target, forwardHeader)
	if err != nil {
		hlog.CtxErrorf(ctx, "can not dial to remote backend(%v): %v", target, err)
		if respBackend != nil {
			if err = wsCopyResponse(&c.Response, respBackend); err != nil {
				hlog.CtxErrorf(ctx, "can not copy response: %v", err)
			}
		} else {
			c.AbortWithMsg(err.Error(), consts.StatusServiceUnavailable)
		}
		return
	}
	if err := w.options.Upgrader.Upgrade(c, func(connClient *hzws.Conn) {
		defer connClient.Close()

		var (
			errClientC  = make(chan error, 1)
			errBackendC = make(chan error, 1)
			errMsg      string
		)

		hlog.CtxDebugf(ctx, "upgrade handler working...")

		//                       replicateWSRespConn
		//               ┌─────────────────────────────────┐
		//               │          errClientC             │
		//         ┌─────▼──────┐                   ┌──────┴──────┐
		//         │ connClient │                   │ connBackend │
		//         └─────┬──────┘                   └──────▲──────┘
		//               │          errBackendC            │
		//               └─────────────────────────────────┘
		//                       replicateWSReqConn
		//
		// ┌──────────┐           ┌────────────────┐             ┌──────────┐
		// │          ├───────────► wsreverseproxy ├─────────────►  backend │
		// │  client  │           │                │             │          │
		// │          ◄───────────┤    (server)    ◄─────────────┤ (server) │
		// └──────────┘           └────────────────┘             └──────────┘

		gopool.CtxGo(ctx, func() {
			replicateWSRespConn(ctx, connClient, connBackend, errClientC)
		})
		gopool.CtxGo(ctx, func() {
			replicateWSReqConn(ctx, connBackend, connClient, errBackendC)
		})

		for {
			select {
			case err = <-errClientC:
				errMsg = "copy websocket response err: %v"
			case err = <-errBackendC:
				errMsg = "copy websocket request err: %v"
			}

			var ce *websocket.CloseError
			var hzce *hzws.CloseError
			if !errors.As(err, &ce) && !errors.As(err, &hzce) {
				hlog.CtxErrorf(ctx, errMsg, err)
				continue
			}

			break
		}
	}); err != nil {
		hlog.CtxErrorf(ctx, "can not upgrade to websocket: %v", err)
	}
}

func prepareForwardHeader(_ context.Context, c *app.RequestContext) http.Header {
	forwardHeader := make(http.Header, 4)
	if origin := string(c.Request.Header.Peek("Origin")); origin != "" {
		forwardHeader.Add("Origin", origin)
	}
	if proto := string(c.Request.Header.Peek("Sec-Websocket-Protocol")); proto != "" {
		forwardHeader.Add("Sec-WebSocket-Protocol", proto)
	}
	if cookie := string(c.Request.Header.Peek("Cookie")); cookie != "" {
		forwardHeader.Add("Cookie", cookie)
	}
	if host := string(c.Request.Host()); host != "" {
		forwardHeader.Set("Host", host)
	}
	clientIP := c.ClientIP()
	if prior := c.Request.Header.Peek("X-Forwarded-For"); prior != nil {
		clientIP = string(prior) + ", " + clientIP
	}
	forwardHeader.Set("X-Forwarded-For", clientIP)
	forwardHeader.Set("X-Forwarded-Proto", "http")
	if string(c.Request.URI().Scheme()) == "https" {
		forwardHeader.Set("X-Forwarded-Proto", "https")
	}
	return forwardHeader
}

func replicateWSReqConn(ctx context.Context, dst *websocket.Conn, src *hzws.Conn, errC chan error) {
	for {
		msgType, msg, err := src.ReadMessage()
		if err != nil {
			hlog.CtxErrorf(ctx, "read message failed when replicating websocket conn: msgType=%v msg=%v err=%v", msgType, msg, err)
			var ce *hzws.CloseError
			if errors.As(err, &ce) {
				msg = hzws.FormatCloseMessage(ce.Code, ce.Text)
			} else {
				hlog.CtxErrorf(ctx, "read message failed when replicate websocket conn: err=%v", err)
				msg = hzws.FormatCloseMessage(hzws.CloseAbnormalClosure, err.Error())
			}
			errC <- err

			if err = dst.WriteMessage(websocket.CloseMessage, msg); err != nil {
				hlog.CtxErrorf(ctx, "write message failed when replicate websocket conn: err=%v", err)
			}
			break
		}

		err = dst.WriteMessage(msgType, msg)
		if err != nil {
			hlog.CtxErrorf(ctx, "write message failed when replicating websocket conn: msgType=%v msg=%v err=%v", msgType, msg, err)
			errC <- err
			break
		}
	}
}

func replicateWSRespConn(ctx context.Context, dst *hzws.Conn, src *websocket.Conn, errC chan error) {
	for {
		msgType, msg, err := src.ReadMessage()
		if err != nil {
			hlog.CtxErrorf(ctx, "read message failed when replicating websocket conn: msgType=%v msg=%v err=%v", msgType, msg, err)
			var ce *websocket.CloseError
			if errors.As(err, &ce) {
				msg = websocket.FormatCloseMessage(ce.Code, ce.Text)
			} else {
				hlog.CtxErrorf(ctx, "read message failed when replicate websocket conn: err=%v", err)
				msg = websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, err.Error())
			}
			errC <- err

			if err = dst.WriteMessage(hzws.CloseMessage, msg); err != nil {
				hlog.CtxErrorf(ctx, "write message failed when replicate websocket conn: err=%v", err)
			}
			break
		}

		err = dst.WriteMessage(msgType, msg)
		if err != nil {
			hlog.CtxErrorf(ctx, "write message failed when replicating websocket conn: msgType=%v msg=%v err=%v", msgType, msg, err)
			errC <- err
			break
		}
	}
}

func wsCopyResponse(dst *protocol.Response, src *http.Response) error {
	for k, vs := range src.Header {
		for _, v := range vs {
			dst.Header.Add(k, v)
		}
	}
	dst.SetStatusCode(src.StatusCode)
	defer src.Body.Close()
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, src.Body); err != nil {
		return err
	}
	dst.SetBody(buf.Bytes())
	return nil
}
