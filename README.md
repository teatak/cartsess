# Cart Session

```go
package main

import (
	"github.com/gimke/cart"
	"github.com/gimke/cartsess"
)

func main() {

	memStore := cartsess.NewMemoryStore()
	cookieStore := cartsess.NewCookieStore([]byte("K5EfWMujNTunxFlOfDT3PP7NPLY"))
	
	r := cart.Default()
	
	r.Use("/",cartsess.NewManager("sess.id",memStore))
	r.Use("/",cartsess.NewManager("cook.id",cookieStore))
	
	r.Route("/").ANY(func(c *cart.Context, next cart.Next) {
		sess := cartsess.Default(c)
		cook := cartsess.GetByName(c,"cook.id")

		token := ""
		if t := cook.Get("token");t != nil {
			token = t.(string)
		} else {
			cook.Set("token", "K5EfWMujNTunxFlOfDT3PP7NPLY")
		}

		count := 0
		if v := sess.Get("count");v !=nil {
			count = v.(int)
			count++
		}
		sess.Set("count",count)

		c.String(200,"tokek:%s count:%d",token,count)
	})

	r.Run()

}
```