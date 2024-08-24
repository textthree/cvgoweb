package middleware

import (
	"cvgo/provider/httpserver"
	"fmt"
)

func RequestLog() httpserver.MiddlewareHandler {
	return func(c *httpserver.Context) error {
		fmt.Println("Use goodlog middleware")
		c.Next()
		return nil
	}
}
