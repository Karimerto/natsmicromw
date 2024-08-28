// This is a minor example to showcase how MicroMiddleware functions can be used

package main

import (
	"flag"
	"log"
	"runtime"

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

func ContentChangeMiddleware(next natsmicromw.MicroHandlerFunc) natsmicromw.MicroHandlerFunc {
	return func(req *natsmicromw.MicroRequest) (*natsmicromw.MicroReply, error) {
		// Call the next function in the middleware chain (or the actual handler)
		res, err := next(req)
		if err != nil {
			return res, err
		}

		// Prepend some data to the response
		res.Data = append([]byte("Hello from Middleware: "), res.Data...)
		return res, err
	}
}

func main() {
	nc, err := nats.Connect(server)
	if err != nil {
		log.Fatal(err)
	}

	// request handler
	echoHandler := func(req *natsmicromw.MicroRequest) (*natsmicromw.MicroReply, error) {
		return natsmicromw.NewMicroReply(req.Data), nil
	}

	svc, err := natsmicromw.AddMicroService(nc, micro.Config{
		Name:        "EchoService",
		Version:     "1.0.0",
	}, ContentChangeMiddleware)
	if err != nil {
		log.Fatal(err)
	}
	g := svc.AddGroup("svc")

	if err := g.AddMicroEndpoint("echo", echoHandler); err != nil {
		log.Fatal(err)
	}

	runtime.Goexit()
}
