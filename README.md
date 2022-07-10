# reverseproxy
reverseproxy for Hertz [WIP]

**A Hertz based reverse proxy middleware**

## Getting started

1. Download swag by using:
```
$ go get github.com/hertz-contrib/reverseproxy
```

## usage

```
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
```


