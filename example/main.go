package main

import (
	"context"
	
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/reverseproxy"
)

func main() {
	h := server.Default(server.WithHostPorts("127.0.0.1:8080"))
	h.Use(reverseproxy.Proxy(map[string]string{
		"/s": "localhost:8080/host/",
	}))

	h.GET("/host/s", func(ctx context.Context, c *app.RequestContext) {
		age := c.DefaultQuery("age", "100")
		c.String(consts.StatusOK, "age = %s", age)
	})
	h.Spin()
}
