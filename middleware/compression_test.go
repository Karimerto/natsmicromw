package middleware

import (
	"bytes"
	"testing"
	"time"

	"github.com/Karimerto/natsmicromw"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

func getServerServiceAndConn(t *testing.T) (*server.Server, *natsmicromw.Service, *nats.Conn) {
	s := getServer(t)

	// Create router and connect to test server
	nc, err := nats.Connect(s.Addr().String())
	if err != nil {
		t.Fatalf("Could not connect to NATS server: %v", err)
	}
	nm, err := natsmicromw.AddMicroService(nc, micro.Config{
		Name:    "TestService",
		Version: "1.0.0",
	})
	if err != nil {
		t.Fatalf("Could not create micro service: %v", err)
	}
	return s, nm, nc
}

func microEcho(req *natsmicromw.MicroRequest) (*natsmicromw.MicroReply, error) {
	// Send response with same content
	return natsmicromw.NewMicroReply(req.Data), nil
}

func TestCompressionMiddleware(t *testing.T) {
	// Create test server
	// s := getServer(t)
	// defer s.Shutdown()
	s, nm, nc := getServerServiceAndConn(t)
	nm = nm.UseMicro(CompressionMiddleware)
	defer nc.Close()
	defer s.Shutdown()

	t.Run("no compression", func(t *testing.T) {
		if err := nm.AddMicroEndpoint("foo", microEcho); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message and send a request
		msg := nats.NewMsg("foo")
		msg.Data = []byte("data")

		// There are no headers, so no compression will happen
		reply, err := nc.RequestMsg(msg, 1*time.Second)
		// Verify contents
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !bytes.Equal(msg.Data, reply.Data) {
			t.Errorf("responses do not match, expected %s, received %s", string(msg.Data), string(reply.Data))
		}
	})

	t.Run("compression with not enough data", func(t *testing.T) {
		if err := nm.AddMicroEndpoint("foo2", microEcho); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message and send a request
		msg := nats.NewMsg("foo2")
		msg.Data = []byte("data")

		// Add header to support reply compression
		msg.Header.Add(HeaderAcceptEncoding, string(CompressionGzip))

		reply, err := nc.RequestMsg(msg, 1*time.Second)
		// Verify contents
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !bytes.Equal(msg.Data, reply.Data) {
			t.Errorf("responses do not match, expected %s, received %s", string(msg.Data), string(reply.Data))
		}
		if reply.Header.Get(HeaderEncoding) != "" {
			t.Errorf("encoding header should be empty")
		}
	})

	t.Run("request compression", func(t *testing.T) {
		if err := nm.AddMicroEndpoint("foo3", microEcho); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message and send a request
		msg := nats.NewMsg("foo3")

		// Compress request data
		longdata := bytes.Repeat([]byte("data"), 500)
		compressed, err := compressGzip(longdata)
		if err != nil {
			t.Errorf("test compression failed: %v", err)
		}
		msg.Data = compressed

		// Add header to indicate request compression
		msg.Header.Add(HeaderEncoding, string(CompressionGzip))

		reply, err := nc.RequestMsg(msg, 1*time.Second)
		// Verify contents
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !bytes.Equal(longdata, reply.Data) {
			t.Errorf("responses do not match, expected %s, received %s", string(longdata), string(reply.Data))
		}
		if reply.Header.Get(HeaderEncoding) != "" {
			t.Errorf("encoding header should be empty")
		}
	})

	t.Run("reply compression", func(t *testing.T) {
		if err := nm.AddMicroEndpoint("foo4", microEcho); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message and send a request
		msg := nats.NewMsg("foo4")
		msg.Data = bytes.Repeat([]byte("data"), 500)

		// Add header to support reply compression
		msg.Header.Add(HeaderAcceptEncoding, string(CompressionGzip))

		reply, err := nc.RequestMsg(msg, 1*time.Second)
		// Verify contents
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Compress data, should be equal
		d, err := compressGzip(msg.Data)
		if err != nil {
			t.Errorf("test compression failed: %v", err)
		}
		if !bytes.Equal(d, reply.Data) {
			t.Errorf("responses do not match, expected %s, received %s", string(msg.Data), string(reply.Data))
		}
		if reply.Header.Get(HeaderEncoding) != string(CompressionGzip) {
			t.Errorf("incorrect encoding header found, expected %s, received %s", string(CompressionGzip), string(msg.Header.Get(HeaderEncoding)))
		}
	})

	t.Run("both compression", func(t *testing.T) {
		if err := nm.AddMicroEndpoint("foo5", microEcho); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message and send a request
		msg := nats.NewMsg("foo5")

		// Compress request data
		longdata := bytes.Repeat([]byte("data"), 500)
		compressed, err := compressGzip(longdata)
		if err != nil {
			t.Errorf("test compression failed: %v", err)
		}
		msg.Data = compressed

		// Add header to indicate request and reply compressions
		msg.Header.Add(HeaderEncoding, string(CompressionGzip))
		msg.Header.Add(HeaderAcceptEncoding, string(CompressionGzip))

		reply, err := nc.RequestMsg(msg, 1*time.Second)
		// Verify contents
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		decompressed, err := decompressGzip(reply.Data)
		if err != nil {
			t.Errorf("unexpected decompression error: %v", err)
		}
		if !bytes.Equal(longdata, decompressed) {
			t.Errorf("responses do not match, expected %s, received %s", string(longdata), string(decompressed))
		}
		if reply.Header.Get(HeaderEncoding) != string(CompressionGzip) {
			t.Errorf("incorrect encoding header found, expected %s, received %s", string(CompressionGzip), string(msg.Header.Get(HeaderEncoding)))
		}
	})


	t.Run("mismatch compression", func(t *testing.T) {
		if err := nm.AddMicroEndpoint("foo6", microEcho); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Create message and send a request
		msg := nats.NewMsg("foo6")

		// Compress request data
		longdata := bytes.Repeat([]byte("data"), 500)
		compressed, err := compressGzip(longdata)
		if err != nil {
			t.Errorf("test compression failed: %v", err)
		}
		msg.Data = compressed

		// Add header to indicate request and reply compressions
		msg.Header.Add(HeaderEncoding, string(CompressionGzip))
		msg.Header.Add(HeaderAcceptEncoding, string(CompressionDeflate))

		reply, err := nc.RequestMsg(msg, 1*time.Second)
		// Verify contents
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		decompressed, err := decompressDeflate(reply.Data)
		if err != nil {
			t.Errorf("unexpected decompression error: %v", err)
		}
		if !bytes.Equal(longdata, decompressed) {
			t.Errorf("responses do not match, expected %s, received %s", string(longdata), string(decompressed))
		}
		if reply.Header.Get(HeaderEncoding) != string(CompressionDeflate) {
			t.Errorf("incorrect encoding header found, expected %s, received %s", string(CompressionDeflate), string(msg.Header.Get(HeaderEncoding)))
		}
	})
}
