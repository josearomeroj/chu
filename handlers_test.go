package chu_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/josearomeroj/chu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter_Method(t *testing.T) {
	const path = "/test"
	const responseText = "test"

	tests := []struct {
		name           string
		method         string
		requestMethod  string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET method",
			method:         "GET",
			requestMethod:  "GET",
			expectedStatus: http.StatusOK,
			expectedBody:   responseText,
		},
		{
			name:           "method not matching",
			method:         "GET",
			requestMethod:  "POST",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chu.New()
			r.Method(tt.method, path, func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(responseText))
				return nil
			})

			req := httptest.NewRequest(tt.requestMethod, path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "should be able to read response body")

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "status code should match expected")

			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, string(body), "response body should match expected")
			}
		})
	}
}

func TestHTTPMethods(t *testing.T) {
	tests := []struct {
		name        string
		setupMethod func(*chu.Router, string, chu.Handler)
		method      string
		expectBody  bool
	}{
		{
			name: "GET",
			setupMethod: func(r *chu.Router, path string, h chu.Handler) {
				r.Get(path, h)
			},
			method:     http.MethodGet,
			expectBody: true,
		},
		{
			name: "POST",
			setupMethod: func(r *chu.Router, path string, h chu.Handler) {
				r.Post(path, h)
			},
			method:     http.MethodPost,
			expectBody: true,
		},
		{
			name: "PUT",
			setupMethod: func(r *chu.Router, path string, h chu.Handler) {
				r.Put(path, h)
			},
			method:     http.MethodPut,
			expectBody: true,
		},
		{
			name: "DELETE",
			setupMethod: func(r *chu.Router, path string, h chu.Handler) {
				r.Delete(path, h)
			},
			method:     http.MethodDelete,
			expectBody: true,
		},
		{
			name: "PATCH",
			setupMethod: func(r *chu.Router, path string, h chu.Handler) {
				r.Patch(path, h)
			},
			method:     http.MethodPatch,
			expectBody: true,
		},
		{
			name: "HEAD",
			setupMethod: func(r *chu.Router, path string, h chu.Handler) {
				r.Head(path, h)
			},
			method:     http.MethodHead,
			expectBody: false,
		},
		{
			name: "OPTIONS",
			setupMethod: func(r *chu.Router, path string, h chu.Handler) {
				r.Options(path, h)
			},
			method:     http.MethodOptions,
			expectBody: true,
		},
		{
			name: "CONNECT",
			setupMethod: func(r *chu.Router, path string, h chu.Handler) {
				r.Connect(path, h)
			},
			method:     http.MethodConnect,
			expectBody: true,
		},
		{
			name: "TRACE",
			setupMethod: func(r *chu.Router, path string, h chu.Handler) {
				r.Trace(path, h)
			},
			method:     http.MethodTrace,
			expectBody: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chu.New()

			const (
				path         = "/test"
				responseText = "test"
			)

			handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				w.WriteHeader(http.StatusOK)
				if tt.method != "HEAD" {
					_, _ = w.Write([]byte(responseText))
				}

				return nil
			}

			tt.setupMethod(r, path, handler)

			req := httptest.NewRequest(tt.method, path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "should be able to read response body")

			assert.Equal(t, http.StatusOK, resp.StatusCode, "status code should be OK")

			if tt.expectBody {
				assert.Equal(t, responseText, string(body), "response body should match expected")
			} else {
				assert.Empty(t, string(body), "response body should be empty")
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	testCases := []struct {
		name           string
		errorMsg       string
		customHandler  bool
		expectedStatus int
		expectedPrefix string
	}{
		{
			name:           "default error handler",
			errorMsg:       "test error",
			customHandler:  false,
			expectedStatus: http.StatusInternalServerError,
			expectedPrefix: "",
		},
		{
			name:           "custom error handler",
			errorMsg:       "custom error",
			customHandler:  true,
			expectedStatus: http.StatusBadRequest,
			expectedPrefix: "handled: ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var r *chu.Router

			if tc.customHandler {
				errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
					http.Error(w, tc.expectedPrefix+err.Error(), tc.expectedStatus)
				}

				r = chu.New(chu.WithErrorHandler(errorHandler))
			} else {
				r = chu.New()
			}

			r.Get("/error", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				return errors.New(tc.errorMsg)
			})

			req := httptest.NewRequest("GET", "/error", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "should be able to read response body")

			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "status code should match expected")
			expectedBody := tc.expectedPrefix + tc.errorMsg + "\n"
			assert.Equal(t, expectedBody, string(body), "response body should match expected")
		})
	}
}
