package middleware

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/Karimerto/natsmicromw"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

func runServer(opts *server.Options) (*server.Server, error) {
	s, err := server.NewServer(opts)
	if err != nil || s == nil {
		return nil, err
	}

	// Run server in Go routine.
	go s.Start()

	// Wait for accept loop(s) to be started
	if !s.ReadyForConnections(10 * time.Second) {
		return nil, errors.New("Unable to start NATS Server in Go Routine")
	}

	return s, nil
}

func getServer(t *testing.T) *server.Server {
	// Create test server
	opts := &server.Options{Host: "localhost", Port: server.RANDOM_PORT, NoSigs: true}
	s, err := runServer(opts)
	if err != nil {
		t.Fatalf("Could not start NATS server: %v", err)
	}

	return s
}

func TestRequestIdMiddleware(t *testing.T) {
	// Create test server
	s := getServer(t)
	defer s.Shutdown()

	t.Run("default request_id header", func(t *testing.T) {
		nc, err := nats.Connect(s.Addr().String())
		if err != nil {
			t.Fatalf("Could not connect to NATS server: %v", err)
		}
		// Create router and connect to test server
		nm, err := natsmicromw.AddContextService(nc, micro.Config{
			Name:    "TestService",
			Version: "1.0.0",
		})
		if err != nil {
			t.Fatalf("Could not create micro service: %v", err)
		}
		nm = nm.UseContext(RequestIdMiddleware())

		defer nc.Close()

		reqId := "req-1"

		err = nm.AddContextEndpoint("foo", func(req *natsmicromw.Request) error {
			if RequestIdFromContext(req.Context()) != reqId {
				t.Errorf("request id does not match/not found")
			}

			// Send response with same content
			if err := req.Respond(req.Data(), micro.WithHeaders(req.Headers())); err != nil {
				t.Errorf("Failed to publish reply: %v", err)
			}

			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message and send a request
		msg := nats.NewMsg("foo")
		msg.Data = []byte("data")
		msg.Header.Add("request_id", reqId)

		reply, err := nc.RequestMsg(msg, 1*time.Second)
		// Verify contents
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !bytes.Equal(msg.Data, reply.Data) {
			t.Errorf("responses do not match, expected %s, received %s", string(msg.Data), string(reply.Data))
		}
		if msg.Header.Get("request_id") != reply.Header.Get("request_id") {
			t.Errorf("request_id does not match")
		}
	})

	t.Run("custom request_id header", func(t *testing.T) {
		headerTag := "reqid"

		nc, err := nats.Connect(s.Addr().String())
		if err != nil {
			t.Fatalf("Could not connect to NATS server: %v", err)
		}
		// Create router and connect to test server
		nm, err := natsmicromw.AddContextService(nc, micro.Config{
			Name:    "TestService",
			Version: "1.0.0",
		})
		if err != nil {
			t.Fatalf("Could not create micro service: %v", err)
		}
		nm = nm.UseContext(RequestIdMiddleware(headerTag))

		defer nc.Close()

		reqId := "req-1"

		err = nm.AddContextEndpoint("foo", func(req *natsmicromw.Request) error {
			if RequestIdFromContext(req.Context()) != reqId {
				t.Errorf("request id does not match/not found")
			}

			// Send response with same content
			if err := req.Respond(req.Data(), micro.WithHeaders(req.Headers())); err != nil {
				t.Errorf("Failed to publish reply: %v", err)
			}

			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message and send a request
		msg := nats.NewMsg("foo")
		msg.Data = []byte("data")
		msg.Header.Add(headerTag, reqId)

		reply, err := nc.RequestMsg(msg, 1*time.Second)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !bytes.Equal(msg.Data, reply.Data) {
			t.Errorf("responses do not match, expected %s, received %s", string(msg.Data), string(reply.Data))
		}
		if msg.Header.Get(headerTag) != reply.Header.Get(headerTag) {
			t.Errorf("request_id does not match")
		}
	})

	t.Run("missing request_id header", func(t *testing.T) {
		nc, err := nats.Connect(s.Addr().String())
		if err != nil {
			t.Fatalf("Could not connect to NATS server: %v", err)
		}
		// Create router and connect to test server
		nm, err := natsmicromw.AddContextService(nc, micro.Config{
			Name:    "TestService",
			Version: "1.0.0",
		})
		if err != nil {
			t.Fatalf("Could not create micro service: %v", err)
		}
		nm = nm.UseContext(RequestIdMiddleware())

		defer nc.Close()

		err = nm.AddContextEndpoint("foo", func(req *natsmicromw.Request) error {
			reqId := RequestIdFromContext(req.Context())
			if len(reqId) == 0 {
				t.Errorf("no request id found")
			}

			// Send response with same content
			headers := make(map[string][]string)
			headers["request_id"] = []string{reqId}
			if err := req.Respond(req.Data(), micro.WithHeaders(headers)); err != nil {
				t.Errorf("Failed to publish reply: %v", err)
			}

			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message without request_id and send a request
		msg := nats.NewMsg("foo")
		msg.Data = []byte("data")

		reply, err := nc.RequestMsg(msg, 1*time.Second)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !bytes.Equal(msg.Data, reply.Data) {
			t.Errorf("responses do not match, expected %s, received %s", string(msg.Data), string(reply.Data))
		}
		if len(reply.Header.Get("request_id")) == 0 {
			t.Errorf("request_id not found")
		}
	})
}
