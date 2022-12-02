package websocket

import (
	"context"
)

// Decode request on receiving.
type DecodeIngressFunc func(context.Context, []byte) (request interface{}, err error)

// Encode response before sending.
type EncodeEgressFunc func(context.Context, interface{}) (response []byte, err error)
