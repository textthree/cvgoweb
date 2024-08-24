package httpserver

import (
	"cvgo/provider/core"
)

type HttpServerService struct {
	container core.Container
	*Engine
}
