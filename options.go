package chu

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Option func(*Router)

func WithErrorHandler(handler ErrorHandler) Option {
	return func(r *Router) {
		r.errHandler = handler
	}
}

func defaultErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func WithRouterBuilder(builder func() chi.Router) Option {
	return func(r *Router) {
		r.routerBuilder = builder
	}
}

func defaultRouterBuilder() chi.Router {
	return chi.NewRouter()
}
