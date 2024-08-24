package httpserver

import (
	"github.com/textthree/provider/core"
)

type HttpServerService struct {
	container core.Container
	*Engine
}
