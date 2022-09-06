# reverseproxy (This is a community driven project)
English | [中文](README_CN.md)

Hertz middleware to enable `reserve proxy` support.

## Usage

Download and install it:

```sh
go get github.com/hertz-contrib/reverseproxy
```

Import it in your code:

```go
import "github.com/hertz-contrib/reverseproxy"
```

Canonical example:
```go
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/cloudwego/hertz/pkg/protocol"
	
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/reverseproxy"
)

func main() {
	h := server.Default(server.WithHostPorts("127.0.0.1:8000"))
	// simple
	proxy1, _ := reverseproxy.NewSingleHostReverseProxy("http://127.0.0.1:8000/proxy")
	// with tls
	proxy2, _ := reverseproxy.NewSingleHostReverseProxy("https://www.baidu.com",
		client.WithTLSConfig(&tls.Config{
			MinVersion: tls.VersionTLS12,
			MaxVersion: tls.VersionTLS12,
		}),
	)
	// customize error handler
	proxy3, _ := reverseproxy.NewSingleHostReverseProxy("http://127.0.0.1:8000/proxy")
	proxy3.SetErrorHandler(func(c *app.RequestContext, err error) {
		err = fmt.Errorf("fake 404 not found")
		c.Response.SetStatusCode(404)
		c.String(404, "fake 404 not found")
	})
	
	// modify response
	proxy4, _ := reverseproxy.NewSingleHostReverseProxy("http://127.0.0.1:8000/proxy")
	proxy4.SetModifyResponse(func(resp *protocol.Response) error {
		resp.SetStatusCode(200)
		resp.SetBodyRaw([]byte("change response success"))
		return nil
	})
	h.GET("/proxy/fake", func(cc context.Context, c *app.RequestContext) {
		c.GetConn().Close()
	})
	h.GET("/proxy/backend", func(cc context.Context, c *app.RequestContext) {
		if param := c.Query("who"); param != "" {
			c.JSON(200, utils.H{
				"who": param,
				"msg": "proxy success!!",
			})
		} else {
			c.JSON(200, utils.H{
				"msg": "proxy success!!",
			})
		}
	})
	h.GET("/proxy/changeResponse", func(cc context.Context, c *app.RequestContext) {
		c.String(200, "normal response")
	})
	
	h.GET("/", proxy2.ServeHTTP)
	h.GET("/fake", proxy3.ServeHTTP)
	h.GET("/backend", proxy1.ServeHTTP)
	h.GET("/changeResponse", proxy4.ServeHTTP)
	h.Spin()
}

```

