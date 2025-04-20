package chu

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func AdaptMiddleware(stdMiddleware func(http.Handler) http.Handler) func(Handler) Handler {
	return func(next Handler) Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			var err error

			stdMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				err = next(r.Context(), w, r)
			})).ServeHTTP(w, r)

			return err
		}
	}
}

func AdaptHandler(h Handler, errHandler ErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(r.Context(), w, r); err != nil {
			errHandler(w, r, err)
		}
	}
}

func StandardHandler(h http.HandlerFunc) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		h(w, r)
		return nil
	}
}

func URLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

func URLParamFromCtx(ctx context.Context, key string) string {
	return chi.URLParamFromCtx(ctx, key)
}
