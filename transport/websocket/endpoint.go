package websocket

import "context"

type WebSocketEndpoint func(ctx context.Context) (ingress chan interface{}, egress chan interface{}, err error)
