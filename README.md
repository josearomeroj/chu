# chu

chu is a thin wrapper around the [chi router](https://github.com/go-chi/chi) that adds explicit error handling capabilities. Instead of chi's standard HTTP handler signature, chu uses:

```go
func(ctx context.Context, w http.ResponseWriter, r *http.Request) error
```

## Installation

```bash
go get github.com/josearomeroj/chu
```

## Quick Start

```go
package main

import (
    "context"
    "errors"
    "log"
    "net/http"
    
    "github.com/josearomeroj/chu"
)

func main() {
    router := chu.NewRouter(chu.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
        log.Printf("Error: %v", err)
        http.Error(w, "Something went wrong", http.StatusInternalServerError)
    }))
    
    router.Get("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
        if someCondition {
            return errors.New("something went wrong")
        }
        
        w.Write([]byte("Success!"))
        return nil
    })
    
    log.Println("Server starting on :3000")
    http.ListenAndServe(":3000", router)
}
```

## Key Differences from chi

### 1. Error Handling

The primary difference is explicit error handling:

```go
// chi handler
func chiHandler(w http.ResponseWriter, r *http.Request) {
    err := doSomething()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
	
    w.Write([]byte("Success"))
}

// chu handler
func chuHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    err := doSomething()
    if err != nil {
        return err // Let the error handler deal with it
    }
	
    w.Write([]byte("Success"))
    return nil
}
```

### 2. Centralized Error Processing

Set a global or router-specific error handler:

```go
// Custom error handler that provides JSON responses
router := chu.NewRouter(chu.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
    w.Header().Set("Content-Type", "application/json")
    
    // You could check for custom error types here
    code := http.StatusInternalServerError
    if e, ok := err.(CustomError); ok {
        code = e.StatusCode()
    }
    
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}))
```

### 3. Middleware Chain Differences

chu middleware can inspect and handle errors from downstream handlers:

```go
func loggingMiddleware(next chu.Handler) chu.Handler {
    return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
        log.Printf("Request started: %s %s", r.Method, r.URL.Path)
        
		err := next(ctx, w, r)
        if err != nil {
            log.Printf("Request error: %v", err)
        }
		
        return err 
    }
}
```

### 4. Adapting Between chi and chu

chu provides adapters for converting between chi and chu handlers:

```go
// Convert chi middleware to chu middleware
chiMiddleware := middleware.RequestID
chuMiddleware := chu.AdaptMiddleware(chiMiddleware)

// Convert chu handler to standard http.HandlerFunc
chuHandler := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    return nil
}

stdHandler := chu.AdaptHandler(chuHandler, errorHandler)
```