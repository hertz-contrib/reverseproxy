# reverseproxy (This is a community driven project)

`reserve proxy` extension for Hertz

## Quick Start

```go
package main

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/reverseproxy"
)

func main() {
	h := server.New()
	rp, _ := reverseproxy.NewSingleHostReverseProxy("http://localhost:8082/test")
	h.GET("/ping", rp.ServeHTTP)
	h.Spin()
}
```

### Use tls

Currently [netpoll](https://github.com/cloudwego/netpoll) does not support tls，we need to use the `net` (standard library)

```go
package main

import (
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/network/standard"
	"github.com/hertz-contrib/reverseproxy"
)

func main() {
	h := server.New()
	rp, _ := reverseproxy.NewSingleHostReverseProxy("https://localhost:8082/test",client.WithDialer(standard.NewDialer()))
	h.GET("/ping", rp.ServeHTTP)
	h.Spin()
}
```

### Use service discovery

Use `nacos` as example and more information refer to [registry](https://github.com/hertz-contrib/registry)

```go
package main

import (
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/middlewares/client/sd"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/hertz-contrib/registry/nacos"
	"github.com/hertz-contrib/reverseproxy"
)

func main() {
	cli, err := client.NewClient()
	if err != nil{
		panic(err)
	}
	r, err := nacos.NewDefaultNacosResolver()
	if err != nil{
		panic(err)
	}
	cli.Use(sd.Discovery(r))
	h := server.New()
	rp, _ := reverseproxy.NewSingleHostReverseProxy("http://test.demo.api/test")
	rp.SetClient(cli)
	rp.SetDirector(func(req *protocol.Request){
		req.SetRequestURI(string(reverseproxy.JoinURLPath(req, rp.Target)))
		req.Header.SetHostBytes(req.URI().Host())
		req.Options().Apply([]config.RequestOption{config.WithSD(true)})
	})
	h.GET("/ping", rp.ServeHTTP)
	h.Spin()
}
```

### Request/Response

`ReverseProxy` provides `SetDirector`、`SetModifyResponse`、`SetErrorHandler` to modify `Request` and `Response`.

### Websocket Reverse Proxy

```go
package main

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/reverseproxy"
)

func main() {
	h := server.Default()
	h.GET("/backend", reverseproxy.NewWSReverseProxy("ws://example.com").ServeHTTP)
	h.Spin()
}
```

| Configuration  | Default                   | Description                  |
|----------------|---------------------------|------------------------------|
| `WithDirector` | `nil`                     | customize the forward header |
| `WithDialer`   | `gorillaws.DefaultDialer` | for dialer customization     |
| `WithUpgrader` | `hzws.HertzUpgrader`      | for upgrader customization   |

### More info
See [example](https://github.com/cloudwego/hertz-examples)
