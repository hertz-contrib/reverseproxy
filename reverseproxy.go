package reverseproxy

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

type ReverseTable map[string]string

var client = &http.Client{}

func Proxy(table ReverseTable) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if dst, ok := table[string(c.Request.URI().Path())]; ok {
			if strings.HasSuffix(dst, "/") {
				dst = strings.TrimSuffix(dst, "/")
			}
			remote, err := url.Parse(dst)
			if err != nil {
				c.Abort()
				return
			}
			c.Request.SetHost(remote.Host)
			c.Request.URI().SetScheme(remote.Scheme)
			c.Request.SetHeader("X-Forwarded-Host", c.Request.Header.Get("Host"))
			u := fmt.Sprintf("%s://%s%s", "http", dst, string(c.Request.RequestURI()))
			proxyReq, err := http.NewRequest(string(c.Request.Method()), u, bytes.NewReader(c.Request.Body()))
			resp, err := client.Do(proxyReq)
			if err != nil {
				c.Abort()
				return
			}
			defer resp.Body.Close() // nolint
			bodyContent, _ := ioutil.ReadAll(resp.Body)
			_, err = c.Response.BodyWriter().Write(bodyContent)
			if err != nil {
				c.Abort()
				return
			}
			for h := range resp.Header {
				c.Response.Header.Set(h, resp.Header.Get(h))
			}
			return
		}
	}
}
