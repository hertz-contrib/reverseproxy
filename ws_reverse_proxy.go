package reverseproxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

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

func (w *WSReverseProxy) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	forwardHeader := prepareForwardHeader(ctx, c)
	// customer Director will overwrite existed header if they have the same header key
	if w.options.director != nil {
		appendHeader := w.options.director(ctx, c)
		appendHeader.VisitAll(func(key, value []byte) {
			forwardHeader.SetBytesKV(key, value)
		})
	}
	connBackend, respBackend, err := w.options.dialer.Dial(w.target, ConvertHZHeaderToStdHeader(forwardHeader))
	if err != nil {
		hlog.Errorf("can not dial to remote backend(%v): %v", w.target, err)
		if respBackend != nil {
			if err = wsCopyResponse(&c.Response, respBackend); err != nil {
				hlog.Errorf("can not copy response: %v", err)
			}
		} else {
			c.AbortWithMsg(err.Error(), consts.StatusServiceUnavailable)
		}
		return
	}
	if err := w.options.upgrader.Upgrade(c, func(connClient *hzws.Conn) {
		defer connClient.Close()

		var (
			errClientC  = make(chan error, 1)
			errBackendC = make(chan error, 1)
			errMsg      string
		)

		hlog.Debugf("upgrade handler working...")

		go replicateWSConn(connClient, connBackend, errClientC, false)
		go replicateWSConn(connClient, connBackend, errBackendC, true)

		for {
			select {
			case err = <-errClientC:
				errMsg = "copy websocket response err: %v"
			case err = <-errBackendC:
				errMsg = "copy websocket request err: %v"
			}

			var ce *websocket.CloseError
			var hzce *hzws.CloseError
			if !errors.As(err, &ce) || !errors.As(err, &hzce) {
				hlog.Errorf(errMsg, err)
			}
		}
	}); err != nil {
		hlog.Errorf("can not upgrade to websocket: %v", err)
	}
}

func ConvertHZHeaderToStdHeader(hzHeader *protocol.RequestHeader) http.Header {
	header := make(http.Header)
	hzHeader.VisitAll(func(key, value []byte) {
		k, v := string(key), string(value)
		// refer to http.Request.Header
		if k == "Host" {
			return
		}
		header.Add(k, v)
	})
	return header
}

func prepareForwardHeader(_ context.Context, c *app.RequestContext) *protocol.RequestHeader {
	forwardHeader := &protocol.RequestHeader{}
	if origin := c.Request.Header.Peek("Origin"); origin != nil {
		forwardHeader.SetBytesKV([]byte("Origin"), origin)
	}
	if proto := c.Request.Header.Peek("Sec-Websocket-Protocol"); proto != nil {
		forwardHeader.SetBytesKV([]byte("Sec-Websocket-Protocol"), proto)
	}
	if cookies := c.Request.Header.Cookies(); cookies != nil {
		for _, cookie := range cookies {
			forwardHeader.SetCookie(string(cookie.Key()), string(cookie.Value()))
		}
	}
	if host := c.Request.Host(); host != nil {
		forwardHeader.SetHost(string(host))
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

func replicateWSConn(connClient *hzws.Conn, connBackend *websocket.Conn, errC chan error, c2b bool) {
	if c2b {
		src := connClient
		dst := connBackend
		for {
			msgType, msg, err := src.ReadMessage()
			if err != nil {
				hlog.Errorf("read message failed when replicating websocket conn: msgType=%v msg=%v err=%v", msgType, msg, err)
				var ce *hzws.CloseError
				if errors.As(err, &ce) {
					msg = hzws.FormatCloseMessage(ce.Code, ce.Text)
				} else {
					hlog.Errorf("read message failed when replicate websocket conn: err=%v", err)
					msg = hzws.FormatCloseMessage(hzws.CloseAbnormalClosure, err.Error())
				}
				errC <- err

				if err = dst.WriteMessage(websocket.CloseMessage, msg); err != nil {
					hlog.Errorf("write message failed when replicate websocket conn: err=%v", err)
				}
				break
			}

			err = dst.WriteMessage(msgType, msg)
			if err != nil {
				hlog.Errorf("write message failed when replicating websocket conn: msgType=%v msg=%v err=%v", msgType, msg, err)
				errC <- err
				break
			}
		}
	} else {
		src := connBackend
		dst := connClient
		for {
			msgType, msg, err := src.ReadMessage()
			if err != nil {
				hlog.Errorf("read message failed when replicating websocket conn: msgType=%v msg=%v err=%v", msgType, msg, err)
				var ce *websocket.CloseError
				if errors.As(err, &ce) {
					msg = websocket.FormatCloseMessage(ce.Code, ce.Text)
				} else {
					hlog.Errorf("read message failed when replicate websocket conn: err=%v", err)
					msg = websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, err.Error())
				}
				errC <- err

				if err = dst.WriteMessage(hzws.CloseMessage, msg); err != nil {
					hlog.Errorf("write message failed when replicate websocket conn: err=%v", err)
				}
				break
			}

			err = dst.WriteMessage(msgType, msg)
			if err != nil {
				hlog.Errorf("write message failed when replicating websocket conn: msgType=%v msg=%v err=%v", msgType, msg, err)
				errC <- err
				break
			}
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
