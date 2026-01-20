# Cart Session

`cartsess` is a session management middleware for Go, originally for the `cart` framework but now compatible with any standard `net/http` handler (including Gin, Chi, Echo, etc.).

It supports multiple storage backends:
- **Memory**: Simple in-memory storage (default).
- **Cookie**: Secure, encrypted cookie-based storage.
- **Redis**: Distributed session storage using Redis.
- **JWT**: Stateless session storage using JSON Web Tokens.

## Installation

```bash
go get github.com/teatak/cartsess/v2
```

## Quick Start (Standard net/http)

`cartsess` implements the standard `net/http` middleware pattern.

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/teatak/cartsess/v2"
)

func main() {
	// 1. Create a session store
	// store := cartsess.NewMemoryStore()
	store := cartsess.NewCookieStore([]byte("your-secret-key-32-bytes"))

	// 2. Create the middleware
	// NewManager returns: func(http.Handler) http.Handler
	sessionMiddleware := cartsess.NewManager("sessionid", store)

	// 3. Create your handler
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 4. Get session from context
		// Use GetByName if you have multiple managers, or Default() if you setup strict context keys.
		manager := cartsess.GetByName(r.Context(), "sessionid")

		// Get value
		count := 0
		if v, _ := manager.Get("count"); v != nil {
			count = int(v.(float64)) // Note: JSON/Gob serialization types may vary
		}
		
		// Set value
		count++
		manager.Set("count", count)

		w.Write([]byte(fmt.Sprintf("Count: %d", count)))
	})

	// 5. Wrap and serve
	http.ListenAndServe(":8080", sessionMiddleware(mux))
}
```

---

## Integrations

### Gin Framework

Use a simple wrapper to adapt `cartsess` (standard middleware) to `gin.HandlerFunc`.

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/teatak/cartsess/v2"
)

// GinMiddleware wraps cartsess middleware for Gin
func GinMiddleware(name string, store cartsess.Store) gin.HandlerFunc {
	mw := cartsess.NewManager(name, store)
	
	return func(c *gin.Context) {
		// Wrap the remaining handlers in the chain as a single http.Handler
		// This requires a bit of trickery because middlewares usually wrap "next".
		// Since Gin controls the chain, we can't easily wrap "c.Next()".
		
		// IMPROVED APPROACH:
		// Modify the Request in Context manually using the inner logic of NewManager
		// OR just use an adapter that is compatible.
		
		// Simplified Adapter:
		// Since cartsess middleware essentially just needs to wrap standard w/r and call next.
		// We can use a helper or manual implementation.
		
		// Actually, standard middleware wrapping gin logic:
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r // Update request with new context
			c.Next()
		}))
		
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

func main() {
	r := gin.Default()
	store := cartsess.NewMemoryStore()

	// Apply middleware
	r.Use(GinMiddleware("gin-sess", store))

	r.GET("/", func(c *gin.Context) {
		manager := cartsess.GetByName(c.Request.Context(), "gin-sess")
		manager.Set("user", "gin-user")
		
		val, _ := manager.Get("user")
		c.String(200, "User: %s", val)
	})

	r.Run(":8080")
}
```

### Teatak Cart (v2)

`cart` v2 is fully compatible with standard `net/http`. You can wrap the middleware easily.

```go
package main

import (
	"net/http"
	"github.com/teatak/cart/v2"
	"github.com/teatak/cartsess/v2"
)

func main() {
	c := cart.Default()
	store := cartsess.NewMemoryStore()

	// 1. Define middleware
	sessMW := cartsess.NewManager("cart-sess", store)

	// 2. Use middleware (Wrap standard middleware to Cart middleware)
	// Assuming cart has a Wrap function, or manually:
	c.Use("/", func(ctx *cart.Context, next cart.Next) {
		// Create a temporary handler that calls next()
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx.Request = r // crucial: update context with new request containing session
			next()
		})
		
		// Execute session middleware
		sessMW(nextHandler).ServeHTTP(ctx.Response, ctx.Request)
	})

	c.Route("/").GET(func(ctx *cart.Context, next cart.Next) {
		manager := cartsess.GetByName(ctx.Request.Context(), "cart-sess")
		manager.Set("foo", "bar")
		ctx.String(200, "Session Set")
	})

	c.Run(":8080")
}
```

---

## Storage Backends

### Redis Store

Requires `github.com/redis/go-redis/v9`.

```go
import (
    "github.com/redis/go-redis/v9"
    "github.com/teatak/cartsess/v2"
)

// Default (localhost:6379)
store := cartsess.NewRedisStore()

// Custom Client
rdb := redis.NewClient(&redis.Options{
    Addr: "redis-server:6379",
    Password: "secure",
})
store := cartsess.NewRedisStoreWithClient(rdb)
```

### JWT Store

Stateless session using JWT. Token is stored in Cookie and also returned in `X-JWT-Token` header.

```go
store := cartsess.NewJWTStoreWithKeyValidation([]byte("must-be-at-least-32-bytes-long-secret"))

// Will check headers: Authorization: Bearer <token>
// Will check cookie: session_name=<token>
```