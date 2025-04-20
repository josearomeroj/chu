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

func TestAdaptMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		middleware     func(http.Handler) http.Handler
		expectedHeader string
		expectedValue  string
		requestMethod  string
		requestPath    string
	}{
		{
			name: "add header middleware",
			middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Test", "middleware-value")
					next.ServeHTTP(w, r)
				})
			},
			expectedHeader: "X-Test",
			expectedValue:  "middleware-value",
			requestMethod:  "GET",
			requestPath:    "/test",
		},
		{
			name: "add multiple headers middleware",
			middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Test-1", "value-1")
					w.Header().Set("X-Test-2", "value-2")
					next.ServeHTTP(w, r)
				})
			},
			expectedHeader: "X-Test-1",
			expectedValue:  "value-1",
			requestMethod:  "POST",
			requestPath:    "/api/resource",
		},
		{
			name: "modify request context middleware",
			middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := context.WithValue(r.Context(), "testKey", "testValue")
					next.ServeHTTP(w, r.WithContext(ctx))
				})
			},
			expectedHeader: "X-Context-Test",
			expectedValue:  "testValue",
			requestMethod:  "PUT",
			requestPath:    "/resource/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adaptedMiddleware := chu.AdaptMiddleware(tt.middleware)

			handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				if tt.name == "modify request context middleware" {
					if val, ok := ctx.Value("testKey").(string); ok {
						w.Header().Set("X-Context-Test", val)
					}
				}

				w.WriteHeader(http.StatusOK)
				return nil
			}

			wrappedHandler := adaptedMiddleware(handler)

			req := httptest.NewRequest(tt.requestMethod, tt.requestPath, nil)
			w := httptest.NewRecorder()

			err := wrappedHandler(req.Context(), w, req)

			assert.NoError(t, err, "Handler should not return error")
			assert.Equal(t, tt.expectedValue, w.Header().Get(tt.expectedHeader),
				"Expected header %s with value %s was not found or had wrong value",
				tt.expectedHeader, tt.expectedValue)
			assert.Equal(t, http.StatusOK, w.Result().StatusCode, "Expected status code OK")
		})
	}
}

func TestAdaptHandler(t *testing.T) {
	tests := []struct {
		name            string
		handler         chu.Handler
		returnError     bool
		expectedStatus  int
		expectedBody    string
		validateHeaders func(t *testing.T, headers http.Header)
	}{
		{
			name: "successful handler",
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("success"))
				return nil
			},
			returnError:    false,
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
			validateHeaders: func(t *testing.T, headers http.Header) {
				assert.Equal(t, "text/plain", headers.Get("Content-Type"))
			},
		},
		{
			name: "json response handler",
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"status":"created"}`))
				return nil
			},
			returnError:    false,
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"status":"created"}`,
			validateHeaders: func(t *testing.T, headers http.Header) {
				assert.Equal(t, "application/json", headers.Get("Content-Type"))
			},
		},
		{
			name: "error handler",
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				return errors.New("handler error")
			},
			returnError:     true,
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    "custom error: handler error\n",
			validateHeaders: nil,
		},
		{
			name: "custom error type",
			handler: func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				type customError struct {
					error
					Code int
				}
				return customError{errors.New("custom typed error"), 400}
			},
			returnError:     true,
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    "custom error: custom typed error\n",
			validateHeaders: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorHandlerCalled := false
			errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
				errorHandlerCalled = true
				http.Error(w, "custom error: "+err.Error(), http.StatusBadRequest)
			}

			adaptedHandler := chu.AdaptHandler(tt.handler, errorHandler)
			require.NotNil(t, adaptedHandler, "AdaptHandler should return a non-nil handler")

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			adaptedHandler.ServeHTTP(w, req)

			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Should be able to read response body")

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code should match expected")
			assert.Equal(t, tt.expectedBody, string(body), "Response body should match expected")
			assert.Equal(t, tt.returnError, errorHandlerCalled,
				"Error handler called: %v, should have been called: %v",
				errorHandlerCalled, tt.returnError)

			if tt.validateHeaders != nil {
				tt.validateHeaders(t, resp.Header)
			}
		})
	}
}

func TestStandardHandler(t *testing.T) {
	tests := []struct {
		name           string
		handlerFunc    http.HandlerFunc
		expectedStatus int
		expectedBody   string
		requestMethod  string
		requestPath    string
		requestHeaders map[string]string
	}{
		{
			name: "standard handler",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("standard"))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "standard",
			requestMethod:  "GET",
			requestPath:    "/test",
			requestHeaders: nil,
		},
		{
			name: "handler with request headers",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				userAgent := r.Header.Get("User-Agent")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("UA: " + userAgent))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "UA: test-agent",
			requestMethod:  "GET",
			requestPath:    "/headers-test",
			requestHeaders: map[string]string{
				"User-Agent": "test-agent",
			},
		},
		{
			name: "not found handler",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "404 page not found\n",
			requestMethod:  "GET",
			requestPath:    "/not-found",
			requestHeaders: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adaptedHandler := chu.StandardHandler(tt.handlerFunc)
			require.NotNil(t, adaptedHandler, "StandardHandler should return a non-nil handler")

			req := httptest.NewRequest(tt.requestMethod, tt.requestPath, nil)

			if tt.requestHeaders != nil {
				for key, value := range tt.requestHeaders {
					req.Header.Set(key, value)
				}
			}

			w := httptest.NewRecorder()

			err := adaptedHandler(req.Context(), w, req)
			assert.NoError(t, err, "Handler should not return error")

			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Should be able to read response body")

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code should match expected")
			assert.Equal(t, tt.expectedBody, string(body), "Response body should match expected")
		})
	}
}

func TestURLParam(t *testing.T) {
	tests := []struct {
		name          string
		setupRouter   func() *chu.Router
		requestPath   string
		expectedParam string
		paramName     string
	}{
		{
			name: "simple id parameter",
			setupRouter: func() *chu.Router {
				r := chu.New()
				r.Get("/users/{id}", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					id := chu.URLParam(r, "id")
					_, _ = w.Write([]byte(id))

					return nil
				})

				return r
			},
			requestPath:   "/users/123",
			expectedParam: "123",
			paramName:     "id",
		},
		{
			name: "complex path parameter",
			setupRouter: func() *chu.Router {
				r := chu.New()
				r.Get("/api/{version}/resources/{resourceId}", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					version := chu.URLParam(r, "version")
					resourceId := chu.URLParam(r, "resourceId")
					_, _ = w.Write([]byte(version + ":" + resourceId))

					return nil
				})

				return r
			},
			requestPath:   "/api/v2/resources/abc-xyz",
			expectedParam: "v2:abc-xyz",
			paramName:     "combined",
		},
		{
			name: "missing parameter",
			setupRouter: func() *chu.Router {
				r := chu.New()
				r.Get("/plain/path", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					missing := chu.URLParam(r, "missing")
					_, _ = w.Write([]byte("missing:" + missing))

					return nil
				})

				return r
			},
			requestPath:   "/plain/path",
			expectedParam: "missing:",
			paramName:     "missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := tt.setupRouter()
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Should be able to read response body")

			switch tt.paramName {
			case "combined":
				assert.Equal(t, tt.expectedParam, string(body), "Expected combined parameters value did not match")
			default:
				if tt.expectedParam == "" {
					assert.Empty(t, string(body), "Parameter should be empty")
				} else {
					assert.Equal(t, tt.expectedParam, string(body), "URL parameter value did not match expected")
				}
			}

			assert.Equal(t, http.StatusOK, resp.StatusCode, "Status should be 200 OK")
		})
	}
}

func TestURLParamFromCtx(t *testing.T) {
	tests := []struct {
		name          string
		setupRouter   func() *chu.Router
		requestPath   string
		expectedParam string
		paramName     string
	}{
		{
			name: "simple id from context",
			setupRouter: func() *chu.Router {
				r := chu.New()
				r.Get("/users/{id}", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					id := chu.URLParamFromCtx(r.Context(), "id")
					_, _ = w.Write([]byte(id))

					return nil
				})

				return r
			},
			requestPath:   "/users/456",
			expectedParam: "456",
			paramName:     "id",
		},
		{
			name: "multiple params from context",
			setupRouter: func() *chu.Router {
				r := chu.New()
				r.Get("/organizations/{orgId}/users/{userId}", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					orgId := chu.URLParamFromCtx(r.Context(), "orgId")
					userId := chu.URLParamFromCtx(r.Context(), "userId")
					_, _ = w.Write([]byte(orgId + "-" + userId))

					return nil
				})

				return r
			},
			requestPath:   "/organizations/org123/users/user456",
			expectedParam: "org123-user456",
			paramName:     "combined",
		},
		{
			name: "empty context param",
			setupRouter: func() *chu.Router {
				r := chu.New()
				r.Get("/test", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					notFound := chu.URLParamFromCtx(r.Context(), "notFound")
					_, _ = w.Write([]byte("not-found:" + notFound))

					return nil
				})

				return r
			},
			requestPath:   "/test",
			expectedParam: "not-found:",
			paramName:     "notFound",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := tt.setupRouter()
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Should be able to read response body")

			assert.Equal(t, http.StatusOK, resp.StatusCode, "Status should be 200 OK")
			assert.Equal(t, tt.expectedParam, string(body), "Context parameter value did not match expected")
		})
	}
}
