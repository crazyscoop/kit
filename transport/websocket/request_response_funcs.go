package websocket

import (
	"context"
	"net/http"
)

type RequestFunc func(context.Context, *http.Request) context.Context
