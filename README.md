Atreugo
=======

Micro-framework to make simple the use of routing and middlewares in fasthttp.

Is based on [erikdubbelboer's fasthttp fork](https://github.com/erikdubbelboer/fasthttp) that it more active than [valyala's fasthttp](https://github.com/valyala/fasthttp)

***The project use [dep](https://golang.github.io/dep/) manager dependencies.***

## Go dependencies:

- [fasthttp](https://github.com/erikdubbelboer/fasthttp)
- [fasthttprouter](https://github.com/thehowl/fasthttprouter)
- [go-logger](https://github.com/savsgio/go-logger)


## Atreugo configuration:

- Host *(string)*
- Port *(int)*
- LogLevel *(string)*: [See levels](https://github.com/savsgio/go-logger/blob/master/README.md)
- Compress *(bool)*:  Compress response body
- TLSEnable *(bool)*:  Enable HTTPS
- CertKey *(string)*: Path of cert.key file
- CertFile *(string)*: Path of cert.pem file
- GracefulEnable *(bool)*: Start server with graceful shutdown


## Example:

```go
package main

import (
	"errors"

	"github.com/erikdubbelboer/fasthttp"
	"github.com/savsgio/atreugo"
)

func main() {
	// Configuration for Atreugo server
	config := &atreugo.Config{
		Host: "0.0.0.0",
		Port: 8000,
	}

	// New instance of atreugo server with your config
	server := atreugo.New(config)

	// Middlewares
	fnMiddlewareOne := func(ctx *fasthttp.RequestCtx) (int, error) {
		return fasthttp.StatusOK, nil
	}

	fnMiddlewareTwo := func(ctx *fasthttp.RequestCtx) (int, error) {
		return fasthttp.StatusBadRequest, errors.New("Error message")
	}

	// Register middlewares
	server.UseMiddleware(fnMiddlewareOne, fnMiddlewareTwo)


	// Views
	server.Path("GET", "/", func(ctx *fasthttp.RequestCtx) error {
		return atreugo.HttpResponse(ctx, []byte("<h1>Atreugo Micro-Framework</h1>"))
	})

	server.Path("GET", "/jsonPage", func(ctx *fasthttp.RequestCtx) error {
		return atreugo.JsonResponse(ctx, atreugo.Json{"Atreugo": true})
	})

	// Start server
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

```

Contributing
============

**Feel free to contribute it or fork me...** :wink:
