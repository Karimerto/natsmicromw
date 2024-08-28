// The package introduces a `MicroRequest` type that allows direct data
// manipulation and also includes a `context.Context`.

package natsmicromw

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

type MicroRequest struct {
	Subject string
	Reply   string
	Headers micro.Headers
	Data    []byte

	ctx context.Context
}

// Create a new MicroRequest from an incoming `micro.Request`
func newMicroRequest(req micro.Request, ctx context.Context) *MicroRequest {
	return &MicroRequest{
		Subject: req.Subject(),
		Reply:   req.Reply(),
		Headers: req.Headers(),
		Data:    req.Data(),
		ctx:     ctx,
	}
}

// Context returns the current attached message context.
func (r *MicroRequest) Context() context.Context {
	return r.ctx
}

// WithContext sets a new message context and returns a new MicroRequest.
func (r *MicroRequest) WithContext(ctx context.Context) *MicroRequest {
	return &MicroRequest{
		r.Subject,
		r.Reply,
		r.Headers,
		r.Data,
		ctx,
	}
}

func (r *MicroRequest) HeaderAdd(key, value string) {
	h := nats.Header(r.Headers)
	if h == nil {
		h = nats.Header{}
	}
	h.Add(key, value)
	r.Headers = micro.Headers(h)
}

func (r *MicroRequest) HeaderSet(key, value string) {
	h := nats.Header(r.Headers)
	if h == nil {
		h = nats.Header{}
	}
	h.Set(key, value)
	r.Headers = micro.Headers(h)
}

func (r *MicroRequest) HeaderGet(key string) string {
	return nats.Header(r.Headers).Get(key)
}

func (r *MicroRequest) HeaderValues(key string) []string {
	return r.Headers.Values(key)
}

func (r *MicroRequest) HeaderDel(key string) {
	if r.Headers == nil {
		return
	}
	h := nats.Header(r.Headers)
	delete(h, key)
	r.Headers = micro.Headers(h)
}
