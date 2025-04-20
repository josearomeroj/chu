package chu_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/josearomeroj/chu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter_New(t *testing.T) {
	tests := []struct {
		name    string
		options []chu.Option
		want    bool
	}{
		{
			name:    "default constructor",
			options: []chu.Option{},
			want:    true,
		},
		{
			name:    "with custom error handler",
			options: []chu.Option{chu.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {})},
			want:    true,
		},
		{
			name:    "with custom router builder",
			options: []chu.Option{chu.WithRouterBuilder(func() chi.Router { return chi.NewRouter() })},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chu.New(tt.options...)
			if tt.want {
				assert.NotNil(t, r, "Router should not be nil")
			} else {
				assert.Nil(t, r, "Router should be nil")
			}
		})
	}
}

func TestRouter_ServeHTTP(t *testing.T) {
	r := chu.New()
	r.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
		return nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Reading response body should not fail")

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be OK")
	assert.Equal(t, "test", string(body), "Response body should match expected content")
}

func TestRouter_SetErrorHandler(t *testing.T) {
	r := chu.New()

	called := false
	customHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		called = true
		w.WriteHeader(http.StatusBadRequest)
	}

	r.SetErrorHandler(customHandler)

	r.Get("/error", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return errors.New("test error")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.True(t, called, "Custom error handler should be called")
	assert.Equal(t, http.StatusBadRequest, w.Code, "Status code should be Bad Request")
}

func TestRouter_Group(t *testing.T) {
	r := chu.New()

	r.Group(func(gr *chu.Router) {
		gr.Get("/group", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("group"))
			return nil
		})
	})

	req := httptest.NewRequest("GET", "/group", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Reading response body should not fail")

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be OK")
	assert.Equal(t, "group", string(body), "Response body should match expected content")
}

func TestRouter_Route(t *testing.T) {
	r := chu.New()

	r.Route("/api", func(api *chu.Router) {
		api.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("api test"))
			return nil
		})
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Reading response body should not fail")

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be OK")
	assert.Equal(t, "api test", string(body), "Response body should match expected content")
}

func TestRouter_Mount(t *testing.T) {
	r := chu.New()

	subHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mounted"))
	})

	r.Mount("/sub", subHandler)

	req := httptest.NewRequest("GET", "/sub", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Reading response body should not fail")

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Status code should be OK")
	assert.Equal(t, "mounted", string(body), "Response body should match expected content")
}

func TestRouter_Use(t *testing.T) {
	r := chu.New()

	middleware := func(next chu.Handler) chu.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("X-Test", "test-value")
			return next(ctx, w, r)
		}
	}

	r.Use(middleware)

	r.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, "test-value", w.Header().Get("X-Test"), "Middleware should set X-Test header")
}

func TestRouter_NotFound(t *testing.T) {
	r := chu.New()

	r.NotFound(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("custom not found"))
		return nil
	})

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Reading response body should not fail")

	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Status code should be Not Found")
	assert.Equal(t, "custom not found", string(body), "Response body should match expected content")
}

func TestRouter_MethodNotAllowed(t *testing.T) {
	r := chu.New()

	r.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	r.MethodNotAllowed(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("custom method not allowed"))
		return nil
	})

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Reading response body should not fail")

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "Status code should be Method Not Allowed")
	assert.Equal(t, "custom method not allowed", string(body), "Response body should match expected content")
}

func TestRouter_UseWithErrorHandling(t *testing.T) {
	r := chu.New()

	errorHandlerCalled := false
	r.SetErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
		errorHandlerCalled = true
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("middleware error"))
	})

	errorMiddleware := func(next chu.Handler) chu.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			return errors.New("middleware error")
		}
	}

	r.Use(errorMiddleware)

	r.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
		return nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Reading response body should not fail")

	assert.True(t, errorHandlerCalled, "Error handler should be called when middleware returns an error")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "Status code should be Internal Server Error")
	assert.Equal(t, "middleware error", string(body), "Response body should match expected content")
}
