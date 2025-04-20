package chu

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

type Router struct {
	chi chi.Router

	errHandler    ErrorHandler
	routerBuilder func() chi.Router
}

func New(opts ...Option) *Router {
	r := &Router{
		routerBuilder: defaultRouterBuilder,
		errHandler:    defaultErrorHandler,
	}

	for _, opt := range opts {
		opt(r)
	}

	r.chi = r.routerBuilder()

	return r
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.chi.ServeHTTP(w, req)
}

func (r *Router) SetErrorHandler(handler ErrorHandler) {
	r.errHandler = handler
}

func (r *Router) adapt(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if err := h(req.Context(), w, req); err != nil {
			r.errHandler(w, req, err)
		}
	}
}

func (r *Router) Group(fn func(r *Router)) *Router {
	subRouter := &Router{
		chi:        r.routerBuilder(),
		errHandler: r.errHandler,
	}

	fn(subRouter)
	r.chi.Mount("/", subRouter.chi)

	return subRouter
}

func (r *Router) Route(pattern string, fn func(r *Router)) {
	subRouter := &Router{
		chi:        r.routerBuilder(),
		errHandler: r.errHandler,
	}

	fn(subRouter)
	r.chi.Mount(pattern, subRouter.chi)
}

func (r *Router) Mount(pattern string, h http.Handler) {
	r.chi.Mount(pattern, h)
}

func (r *Router) Use(middlewares ...func(Handler) Handler) {
	wrappedMiddlewares := make([]func(http.Handler) http.Handler, len(middlewares))

	for i, middleware := range middlewares {
		wrappedMiddlewares[i] = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				wrappedHandler := middleware(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					next.ServeHTTP(w, r)
					return nil
				})

				if err := wrappedHandler(req.Context(), w, req); err != nil {
					r.errHandler(w, req, err)
				}
			})
		}
	}

	r.chi.Use(wrappedMiddlewares...)
}

func (r *Router) NotFound(h Handler) {
	r.chi.NotFound(r.adapt(h))
}

func (r *Router) MethodNotAllowed(h Handler) {
	r.chi.MethodNotAllowed(r.adapt(h))
}
