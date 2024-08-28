// This is a minor example to showcase how Middleware functions can be used

package main

import (
	"flag"
	"log"
	"runtime"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"

	"github.com/Karimerto/natsmicromw"
)

var (
	server string
)

func init() {
	flag.StringVar(&server, "server", nats.DefaultURL, "NATS server address")
	flag.Parse()
}

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

func main() {
	nc, err := nats.Connect(server)
	if err != nil {
		log.Fatal(err)
	}

	// request handler
	echoHandler := func(req micro.Request) {
		req.Respond(req.Data())
	}

	_, err = natsmicromw.AddService(nc, micro.Config{
		Name:        "EchoService",
		Version:     "1.0.0",
		// base handler
		Endpoint: &micro.EndpointConfig{
			Subject: "svc.echo",
			Handler: micro.HandlerFunc(echoHandler),
		},
	}, DurationMiddleware)
	if err != nil {
		log.Fatal(err)
	}

	runtime.Goexit()
}
