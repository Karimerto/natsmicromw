// The package introduces a `MicroReply` type that allows direct data manipulation.

package natsmicromw

import (
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

type MicroReply struct {
	Headers micro.Headers
	Data    []byte
}

// Create a new MicroReply
func NewMicroReply(data []byte) *MicroReply {
	return &MicroReply{
		Headers: micro.Headers{},
		Data:    data,
	}
}

// Create a new MicroReply and copy headers from the original request
func NewMicroReplyFromRequest(data []byte, req *MicroRequest) *MicroReply {
	h := nats.Header{}
	for k, v := range req.Headers {
		h[k] = v
	}
	return &MicroReply{
		Headers: micro.Headers(h),
		Data:    data,
	}
}

func (r *MicroReply) HeaderAdd(key, value string) {
	h := nats.Header(r.Headers)
	if h == nil {
		h = nats.Header{}
	}
	h.Add(key, value)
	r.Headers = micro.Headers(h)
}

func (r *MicroReply) HeaderSet(key, value string) {
	h := nats.Header(r.Headers)
	if h == nil {
		h = nats.Header{}
	}
	h.Set(key, value)
	r.Headers = micro.Headers(h)
}

func (r *MicroReply) HeaderGet(key string) string {
	return nats.Header(r.Headers).Get(key)
}

func (r *MicroReply) HeaderValues(key string) []string {
	return r.Headers.Values(key)
}

func (r *MicroReply) HeaderDel(key string) {
	if r.Headers == nil {
		return
	}
	h := nats.Header(r.Headers)
	delete(h, key)
	r.Headers = micro.Headers(h)
}
