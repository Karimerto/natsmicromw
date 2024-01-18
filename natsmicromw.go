/*
Package `natsmicromw` provides middleware functionality for NATS-based
Microservices in Go. It implements `micro.Service` and provides a thin wrapper
for the underlying Service. This also adds a context-enabled custom `Request`
for carrying the same context through the whole middleware stack.
*/

package natsmicromw

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

// Service represents a Microservice with middleware support.
type Service struct {
	svc        micro.Service
	mw         []MiddlewareFunc
	cmw        []ContextMiddlewareFunc
	defaultCtx context.Context
}

// Group represents a Microservice group with middleware support.
type Group struct {
	svc *Service
	grp micro.Group
}

// MiddlewareFunc defines the type for middleware functions.
type MiddlewareFunc func(micro.Handler) micro.Handler

type ContextHandlerFunc func(*Request) error

// Middleware function that takes a `ContextHandlerFunc` and returns a new `ContextHandlerFunc`
type ContextMiddlewareFunc func(ContextHandlerFunc) ContextHandlerFunc

func wrapHandler(handler micro.Handler, mws ...MiddlewareFunc) micro.Handler {
	// Create a chain of middleware handlers
	var wrappedHandler micro.Handler = handler
	for i := len(mws) - 1; i >= 0; i-- {
		wrappedHandler = mws[i](wrappedHandler)
	}

	return wrappedHandler
}

// AddService creates a new Microservice with middleware support.
func AddService(nc *nats.Conn, config micro.Config, fns ...MiddlewareFunc) (*Service, error) {
	// Check if `Endpoint` is defined and there are middleware functions,
	// and if so, wrap the handler
	if len(fns) > 0 && config.Endpoint != nil && config.Endpoint.Handler != nil {
		config.Endpoint.Handler = wrapHandler(config.Endpoint.Handler, fns...)
	}

	svc, err := micro.AddService(nc, config)
	if err != nil {
		return nil, err
	}

	s := &Service{svc: svc, mw: fns}
	return s, nil
}

// WithMiddleware adds middleware functions to the Microservice.
func (s *Service) WithMiddleware(fns ...MiddlewareFunc) *Service {
	return &Service{
		svc:        s.svc,
		mw:         append(s.mw, fns...),
		cmw:        s.cmw,
		defaultCtx: s.defaultCtx,
	}
}

// Use is an alias for WithMiddleware, adding middleware functions to the Microservice.
func (s *Service) Use(fns ...MiddlewareFunc) *Service {
	return s.WithMiddleware(fns...)
}

func wrapContextHandler(s *Service, handler ContextHandlerFunc) micro.HandlerFunc {
	return micro.HandlerFunc(func(req micro.Request) {
		// Use the default context if available, otherwise use background context
		var ctx context.Context
		if s.defaultCtx != nil {
			ctx = s.defaultCtx
		} else {
			ctx = context.Background()
		}

		ctxReq := &Request{req, ctx}

		// Wrap handler in middleware calls
		var wrappedCtxHandler ContextHandlerFunc = handler
		for i := len(s.cmw) - 1; i >= 0; i-- {
			wrappedCtxHandler = s.cmw[i](wrappedCtxHandler)
		}

		// Call the top-level handler
		err := wrappedCtxHandler(ctxReq)

		// If an error is encountered, respond with it automatically
		if err != nil {
			handlerErr, ok := err.(*HandlerError)
			if !ok {
				handlerErr = &HandlerError{
					Description: err.Error(),
					Code:        "500",
				}
			}

			// Send the entire error in the body as well
			errData, _ := json.Marshal(handlerErr)

			req.Error(handlerErr.Code, handlerErr.Description, errData)
		}
	})
}

// AddContextService creates a new Microservice with middleware support.
// Note that this version does not support defining an endpoint in the initial config.
// If any is defined, it will not use any of the context-based middlewares.
func AddContextService(nc *nats.Conn, config micro.Config, fns ...ContextMiddlewareFunc) (*Service, error) {
	svc, err := micro.AddService(nc, config)
	if err != nil {
		return nil, err
	}

	s := &Service{svc: svc, cmw: fns}
	return s, nil
}

// WithContextMiddleware adds middleware functions to the Microservice.
func (s *Service) WithContextMiddleware(fns ...ContextMiddlewareFunc) *Service {
	return &Service{
		svc:        s.svc,
		mw:         s.mw,
		cmw:        append(s.cmw, fns...),
		defaultCtx: s.defaultCtx,
	}
}

// UseContext is an alias for WithContextMiddleware, adding middleware functions to the Microservice.
func (s *Service) UseContext(fns ...ContextMiddlewareFunc) *Service {
	return s.WithContextMiddleware(fns...)
}

// SetDefaultContext sets the default context to be used by the service.
// This context will be used if no custom context is provided during endpoint registration.
func (s *Service) SetDefaultContext(ctx context.Context) {
	s.defaultCtx = ctx
}

// AddEndpoint registers an endpoint with the given name on a specific subject.
func (s *Service) AddEndpoint(name string, handler micro.Handler, opts ...micro.EndpointOpt) error {
	return s.svc.AddEndpoint(name, wrapHandler(handler, s.mw...), opts...)
}

// AddContextEndpoint registers an endpoint with the given name on a specific subject.
func (s *Service) AddContextEndpoint(name string, handler ContextHandlerFunc, opts ...micro.EndpointOpt) error {
	return s.svc.AddEndpoint(name, wrapContextHandler(s, handler), opts...)
}

// AddGroup returns a Group interface, allowing for more complex endpoint topologies.
// A group can be used to register endpoints with a given prefix.
func (s *Service) AddGroup(name string, opts ...micro.GroupOpt) *Group {
	grp := s.svc.AddGroup(name, opts...)
	return &Group{s, grp}
}

// Info returns the service info.
func (s *Service) Info() micro.Info {
	return s.svc.Info()
}

// Stats returns statistics for the service endpoint and all monitoring endpoints.
func (s *Service) Stats() micro.Stats {
	return s.svc.Stats()
}

// Reset resets all statistics (for all endpoints) on a service instance.
func (s *Service) Reset() {
	s.svc.Reset()
}

// Stop drains the endpoint subscriptions and marks the service as stopped.
func (s *Service) Stop() error {
	return s.svc.Stop()
}

// Stopped informs whether [Stop] was executed on the service.
func (s *Service) Stopped() bool {
	return s.svc.Stopped()
}

// AddGroup creates a new group, prefixed by this group's prefix.
func (g *Group) AddGroup(name string, opts ...micro.GroupOpt) *Group {
	grp := g.grp.AddGroup(name, opts...)
	return &Group{g.svc, grp}
}

// AddEndpoint registers new endpoints on a service.
// The endpoint's subject will be prefixed with the group prefix.
func (g *Group) AddEndpoint(name string, handler micro.Handler, opts ...micro.EndpointOpt) error {
	return g.grp.AddEndpoint(name, wrapHandler(handler, g.svc.mw...), opts...)
}

// AddContextEndpoint registers an endpoint with the given name on a specific subject within a group.
func (g *Group) AddContextEndpoint(name string, handler ContextHandlerFunc, opts ...micro.EndpointOpt) error {
	return g.grp.AddEndpoint(name, wrapContextHandler(g.svc, handler), opts...)
}

// WithMiddleware adds middleware functions to the Microservice group.
func (g *Group) WithMiddleware(fns ...MiddlewareFunc) *Group {
	return &Group{
		svc: &Service{
			svc:        g.svc.svc,
			mw:         append(g.svc.mw, fns...),
			cmw:        g.svc.cmw,
			defaultCtx: g.svc.defaultCtx,
		},
		grp: g.grp,
	}
}

// Use is an alias for WithMiddleware, adding middleware functions to the Microservice group.
func (g *Group) Use(fns ...MiddlewareFunc) *Group {
	return g.WithMiddleware(fns...)
}

// WithContextMiddleware adds context middleware functions to the Microservice group.
func (g *Group) WithContextMiddleware(fns ...ContextMiddlewareFunc) *Group {
	return &Group{
		svc: &Service{
			svc:        g.svc.svc,
			mw:         g.svc.mw,
			cmw:        append(g.svc.cmw, fns...),
			defaultCtx: g.svc.defaultCtx,
		},
		grp: g.grp,
	}
}

// UseContext is an alias for WithContextMiddleware, adding context middleware functions to the Microservice group.
func (g *Group) UseContext(fns ...ContextMiddlewareFunc) *Group {
	return g.WithContextMiddleware(fns...)
}
