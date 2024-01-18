# NATS micro Middleware

NATS micro Middleware is a Go package that provides middleware functionality for building middleware-enabled microservices using NATS.go. It is built on top of the NATS [`micro` package](https://github.com/nats-io/nats.go/tree/main/micro)

- [Overview](#overview)
- [Basic usage](#basic-usage)

## Overview

The `natsmicromw` provides a thin wrapper around `micro.Service` and `micro.Group` as well as implements the same interfaces.

## Basic usage

To start using the `natsmicromw` package, import it in your application:

```go
import "github.com/Karimerto/natsmicromw"
```

Usage is almost identical to the `micro` package itself. The only difference is the middleware(s) as well as support for context-included `Request`.

```go
func DurationMiddleware(next micro.Handler) micro.Handler {
    return micro.HandlerFunc(func(req micro.Request) {
        // Record start time
        start := time.Now()

        // Call the next middleware or handler function
        // Note that it must call the `Handle` function specifically
        next.Handle(req)

        // Record elapsed time and payload size
        elapsed := time.Since(start)
        log.Printf("Duration: %s", elapsed)
    })
}

nc, _ := nats.Connect(nats.DefaultURL)

// request handler
echoHandler := func(req micro.Request) {
    req.Respond(req.Data())
}

srv, err := micro.AddService(nc, micro.Config{
    Name:        "EchoService",
    Version:     "1.0.0",
    // base handler
    Endpoint: &micro.EndpointConfig{
        Subject: "svc.echo",
        Handler: micro.HandlerFunc(echoHandler),
    },
}, DurationMiddleware)
```

After creating the service, it can be accessed by publishing a request on
endpoint subject. For given configuration, run:

```sh
nats req svc.echo "hello!"
```

To get:

```sh
17:37:32 Sending request on "svc.echo"
17:37:32 Received with rtt 365.875µs
hello!
```

As well as:

```sh
17:37:32 Duration: 28.634µs
```

## Context-based usage

Context-based middleware adds a custom `Request` that includes a `context.Context` that can be carried through the entire middleware chain.

```go
type startContextKey struct{}

func DurationContextMiddleware(next natsmicromw.ContextHandlerFunc) natsmicromw.ContextHandlerFunc {
    // Note that this is a regular function that is returned
    return func(req *natsmicromw.Request) error {
        // Record start time
        start := time.Now()

        // Call the next middleware or handler function
        ctx := context.WithValue(req.Context(), requestIdContextKey{}, start)
        err := next(req.WithContext(ctx))

        // Record elapsed time and payload size
        elapsed := time.Since(start)
        log.Printf("Duration: %s", elapsed)

        return err
    }
}

func StartFromContext(ctx context.Context) time.Time {
    start, ok := ctx.Value(startContextKey{}).(time.Time)
    if !ok {
        return time.Now()
    }
    return start
}

nc, _ := nats.Connect(nats.DefaultURL)

// request handler
echoHandler := func(req micro.Request) {
    req.Respond(req.Data())
}

srv, err := micro.AddService(nc, micro.Config{
    Name:        "EchoService",
    Version:     "1.0.0",
    // Note that base handler does not work with context-based middleware
}, DurationContextMiddleware)

srv.AddContextEndpoint("echo", func(req *natsmicromw.Request) error {
    started := StartFromContext(req.Context())
    log.Println("Request was started at:", started)
    req.Respond(req.Data())
    return nil
})
```

## Contributing

Contributions are welcome! If you find a bug or have a feature request, please [open an issue](https://github.com/Karimerto/natsmicromw/issues/new). If you would like to contribute code, please fork the repository and create a pull request.
