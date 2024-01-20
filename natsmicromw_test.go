package natsmicromw

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

func emptyHandler(req micro.Request) {
}

func emptyRequestHandler(req *Request) error {
	return nil
}

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

func TestRunServer(t *testing.T) {
	opts := &server.Options{Host: "localhost", Port: server.RANDOM_PORT, NoSigs: true}
	s, err := runServer(opts)
	if err != nil {
		t.Fatalf("Could not start NATS server: %v", err)
	}
	defer s.Shutdown()

	nc, err := nats.Connect(s.Addr().String())
	if err != nil {
		t.Fatalf("Could not connect to NATS server: %v", err)
	}
	defer nc.Close()
}

func TestAddService(t *testing.T) {
	// Create test server
	opts := &server.Options{Host: "localhost", Port: server.RANDOM_PORT, NoSigs: true}
	s, err := runServer(opts)
	if err != nil {
		t.Fatalf("Could not start NATS server: %v", err)
	}
	defer s.Shutdown()

	// Create router and connect to test server
	nc, err := nats.Connect(s.Addr().String())
	if err != nil {
		t.Fatalf("Could not connect to NATS server: %v", err)
	}
	_, err = AddService(nc, micro.Config{
		Name:    "TestService",
		Version: "1.0.0",
	})
	if err != nil {
		t.Fatalf("Could not create micro service: %v", err)
	}
	defer nc.Close()
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

func getServerAndService(t *testing.T) (*server.Server, *Service) {
	s := getServer(t)

	// Create router and connect to test server
	nc, err := nats.Connect(s.Addr().String())
	if err != nil {
		t.Fatalf("Could not connect to NATS server: %v", err)
	}
	nm, err := AddService(nc, micro.Config{
		Name:    "TestService",
		Version: "1.0.0",
	})
	if err != nil {
		t.Fatalf("Could not create micro service: %v", err)
	}
	return s, nm
}

func getServerServiceAndConn(t *testing.T) (*server.Server, *Service, *nats.Conn) {
	s := getServer(t)

	// Create router and connect to test server
	nc, err := nats.Connect(s.Addr().String())
	if err != nil {
		t.Fatalf("Could not connect to NATS server: %v", err)
	}
	nm, err := AddService(nc, micro.Config{
		Name:    "TestService",
		Version: "1.0.0",
	})
	if err != nil {
		t.Fatalf("Could not create micro service: %v", err)
	}
	return s, nm, nc
}

func TestBasicEndpoint(t *testing.T) {
	// Create test server and client
	s, nm := getServerAndService(t)
	defer s.Shutdown()

	t.Run("simple endpoint", func(t *testing.T) {
		err := nm.AddEndpoint("foo", micro.HandlerFunc(emptyHandler))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("context-based simple endpoint", func(t *testing.T) {
		err := nm.AddContextEndpoint("foz", emptyRequestHandler)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestGroupEndpoint(t *testing.T) {
	// Create test server and client
	s, nm := getServerAndService(t)
	grp := nm.AddGroup("foo")
	defer s.Shutdown()

	t.Run("single group endpoint", func(t *testing.T) {
		err := grp.AddEndpoint("bar", micro.HandlerFunc(emptyHandler))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("single context-based group endpoint", func(t *testing.T) {
		err := grp.AddContextEndpoint("baz", emptyRequestHandler)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("multi-part group endpoint", func(t *testing.T) {
		grp2 := grp.AddGroup("foo2")
		grp3 := grp2.AddGroup("foo3")
		err := grp3.AddEndpoint("bar", micro.HandlerFunc(emptyHandler))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("multi-part context-based group endpoint", func(t *testing.T) {
		grp2 := grp.AddGroup("foo_2")
		grp3 := grp2.AddGroup("foo_3")
		err := grp3.AddContextEndpoint("baz", emptyRequestHandler)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestMiddlewareChain(t *testing.T) {
	// define some middleware functions
	middleware1 := func(next ContextHandlerFunc) ContextHandlerFunc {
		return func(req *Request) error {
			ctx := context.WithValue(req.Context(), "key1", "value1")
			return next(req.WithContext(ctx))
		}
	}

	middleware2 := func(next ContextHandlerFunc) ContextHandlerFunc {
		return func(req *Request) error {
			ctx := context.WithValue(req.Context(), "key2", "value2")
			return next(req.WithContext(ctx))
		}
	}

	// define a final handler function
	handler := func(req *Request) error {
		if req.Context().Value("key1") != "value1" {
			t.Errorf("Expected key1 to be value1")
		}

		if req.Context().Value("key2") != "value2" {
			t.Errorf("Expected key2 to be value2")
		}

		// Send response with same content
		if err := req.Respond(req.Data()); err != nil {
			t.Errorf("Failed to publish reply: %v", err)
		}

		return nil
	}

	// Create test server and client
	s, nm, nc := getServerServiceAndConn(t)

	nm = nm.UseContext(middleware1, middleware2)
	defer nc.Close()
	defer s.Shutdown()

	t.Run("endpoint with middleware", func(t *testing.T) {
		err := nm.AddContextEndpoint("foo1", handler)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message and send a request
		msg := nats.NewMsg("foo1")
		msg.Data = []byte("data")

		reply, err := nc.RequestMsg(msg, 1*time.Second)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !bytes.Equal(msg.Data, reply.Data) {
			t.Errorf("responses do not match, expected %s, received %s", string(msg.Data), string(reply.Data))
		}
	})

	t.Run("endpoint with middleware and group", func(t *testing.T) {
		grp := nm.AddGroup("grp")
		err := grp.AddContextEndpoint("foo2", handler)
		// sub := nr.Queue("group").Subject("foo2")
		// _, err := sub.Subscribe(handler)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message and send a request
		msg := nats.NewMsg("grp.foo2")
		msg.Data = []byte("data")

		reply, err := nc.RequestMsg(msg, 1*time.Second)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !bytes.Equal(msg.Data, reply.Data) {
			t.Errorf("responses do not match, expected %s, received %s", string(msg.Data), string(reply.Data))
		}
	})
}

func TestError(t *testing.T) {
	// define a handler that always fails
	errHandler := func(req *Request) error {
		return errors.New("request failed")
	}

	t.Run("return default error", func(t *testing.T) {
		// Create test server and client
		s, nm, nc := getServerServiceAndConn(t)
		defer nc.Close()
		defer s.Shutdown()

		err := nm.AddContextEndpoint("foo", errHandler)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message and send a request
		msg := nats.NewMsg("foo")
		msg.Data = []byte("data")

		reply, err := nc.RequestMsg(msg, 1*time.Second)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		errHdr := reply.Header.Get("Nats-Service-Error")
		errCode := reply.Header.Get("Nats-Service-Error-Code")
		if errHdr != "request failed" {
			t.Errorf("error header does not match, expected %s, got %s", "request failed", errHdr)
		}
		if errCode != "500" {
			t.Errorf("error header does not match, expected %s, got %s", "500", errHdr)
		}
		errJson := []byte("{\"description\":\"request failed\",\"code\":\"500\"}")
		if !bytes.Equal(errJson, reply.Data) {
			t.Errorf("responses do not match, expected %s, received %s", string(msg.Data), string(reply.Data))
		}
	})
}
