// Example request id middleware for natsmicromw

package middleware

import (
	"context"

	"github.com/Karimerto/natsmicromw"

	// For generating request ID
	"github.com/rs/xid"
)

type requestIdContextKey struct{}

// An example request id middleware
func RequestIdMiddleware(tags ...string) func(next natsmicromw.ContextHandlerFunc) natsmicromw.ContextHandlerFunc {
	return func(next natsmicromw.ContextHandlerFunc) natsmicromw.ContextHandlerFunc {
		// If no tags are defined, then assume "request_id"
		// Try a few variants since NATS headers are case-sensitive
		if len(tags) == 0 {
			tags = []string{"request_id", "Request_id", "Request_Id", "REQUEST_ID"}
		}

		return func(req *natsmicromw.Request) error {
			var requestId string
			// Try all possible tags until something is found
			for _, tag := range tags {
				requestId = req.Headers().Get(tag)
				if requestId != "" {
					break
				}
			}

			// If nothing is found, generate one
			if len(requestId) == 0 {
				requestId = xid.New().String()
			}

			ctx := context.WithValue(req.Context(), requestIdContextKey{}, requestId)
			return next(req.WithContext(ctx))
		}
	}
}

// Get current request Id from the context
func RequestIdFromContext(ctx context.Context) string {
	requestId, ok := ctx.Value(requestIdContextKey{}).(string)
	if !ok {
		return ""
	}
	return requestId
}
