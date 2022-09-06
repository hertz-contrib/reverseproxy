# reverseproxy (这是一个社区驱动的项目)
[English](README.md) | 中文

通过该扩展可以使 hertz 支持`反向代理`

reverseproxy 实现了:
- 支持自定义错误处理
- 支持自定义请求，响应信息
- 支持结合 [registry](https://github.com/hertz-contrib/registry) 中的子组件实现网关服务

## 使用

下载并且安装:

```shell
go get github.com/hertz-contrib/reverseproxy
```

简易使用:
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

完整使用实际代码详见 [example](https://github.com/cloudwego/hertz-examples/tree/main/reverseproxy)

