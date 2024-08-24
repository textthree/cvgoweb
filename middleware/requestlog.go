package middleware

import (
	"fmt"
	"github.com/textthree/cvgoweb"
)

func RequestLog() httpserver.MiddlewareHandler {
	return func(c *httpserver.Context) error {
		fmt.Println("Use goodlog middleware")
		c.Next()
		return nil
	}
}
