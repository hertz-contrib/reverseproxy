# reverseproxy (This is a community driven project)
English | [中文](README_CN.md)

Hertz middleware to enable `reserve proxy` support.

`reverseproxy` features provided:
- Support customize error handler
- Support customize request, response messages
- Support gateway services in combination with subcomponents in [registry](https://github.com/hertz-contrib/registry)

## Usage

Download and install:

```shell
go get github.com/hertz-contrib/reverseproxy
```

Simple usage:
```go
package main

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/reverseproxy"
)

func main() {
	h := server.Default()
	// simple
	proxy, _ := reverseproxy.NewSingleHostReverseProxy("http://127.0.0.1:8000/proxy")
	h.GET("/proxy/backend", func(cc context.Context, c *app.RequestContext) {
		c.JSON(200, utils.H{"msg": "proxy success!!"})
	})
	
	h.GET("/backend", proxy1.ServeHTTP)
	h.Spin()
}
```
See [example](https://github.com/cloudwego/hertz-examples/tree/main/reverseproxy) for full details of the actual 
code used.
