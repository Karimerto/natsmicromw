// The package introduces a custom `Request` type, extending the `micro.Request` type to include a context,
// allowing users to manage and customize the context associated with each message.

package natsmicromw

import (
	"context"

	"github.com/nats-io/nats.go/micro"
)

// Request extends the micro.Request type to include a custom context.
type Request struct {
	micro.Request
	ctx context.Context
}

// Context returns the current attached message context.
func (r *Request) Context() context.Context {
	return r.ctx
}

// WithContext sets a new message context and returns a new Request.
func (r *Request) WithContext(ctx context.Context) *Request {
	return &Request{
		r.Request,
		ctx,
	}
}
