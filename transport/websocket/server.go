package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/crazyscoop/kit/log"
	"github.com/crazyscoop/kit/transport"
	"github.com/gorilla/websocket"
)

type SocketConfigFunc func(context.Context, *websocket.Conn)

type ConnectionCloseHandler func(code int, text string) error

type ServerOption func(*Server)

type Server struct {
	conn             websocket.Conn
	upgrader         websocket.Upgrader
	e                WebSocketEndpoint
	ingress          chan interface{}
	egress           chan interface{}
	connCloseHandler ConnectionCloseHandler
	dec              DecodeIngressFunc
	enc              EncodeEgressFunc
	socketConfig     []SocketConfigFunc
	before           []RequestFunc
	errorHandler     transport.ErrorHandler
}

func NewServer(
	e WebSocketEndpoint,
	dec DecodeIngressFunc,
	enc EncodeEgressFunc,
	options ...ServerOption,
) *Server {
	s := &Server{
		e:            e,
		dec:          dec,
		enc:          enc,
		errorHandler: transport.NewLogErrorHandler(log.NewNopLogger()),
	}
	for _, option := range options {
		option(s)
	}
	return s
}

func SetConnCloseHandler(handler ConnectionCloseHandler) ServerOption {
	return func(s *Server) { s.connCloseHandler = handler }
}

func SetSocketConfig(socketConfig ...SocketConfigFunc) ServerOption {
	return func(s *Server) { s.socketConfig = append(s.socketConfig, socketConfig...) }
}

func ServerBefore(before ...RequestFunc) ServerOption {
	return func(s *Server) { s.before = append(s.before, before...) }
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.errorHandler.Handle(ctx, err)
		return
	}
	s.conn = *conn

	for _, f := range s.before {
		ctx = f(ctx, r)
	}

	for _, f := range s.socketConfig {
		f(ctx, &s.conn)
	}

	if s.connCloseHandler == nil {
		s.connCloseHandler = func(code int, text string) error {
			close(s.ingress)
			return nil
		}
	}

	s.conn.SetCloseHandler(s.connCloseHandler)

	s.ingress, s.egress, err = s.e(ctx)
	if err != nil {
		return
	}

	go s.egressMessage(ctx)
	go s.ingressMessage(ctx)
}

func (s Server) egressMessage(ctx context.Context) {
	for {

		message, ok := <-s.egress
		if !ok {
			s.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		resp, err := s.enc(ctx, message)
		if err != nil {
			return
		}
		s.conn.WriteMessage(websocket.BinaryMessage, resp)

	}
}

func (s Server) ingressMessage(ctx context.Context) {
	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			return
		}

		req, err := s.dec(ctx, message)
		if err != nil {
			return
		}
		s.conn.SetReadDeadline(time.Now().Add(time.Second * 10))
		s.ingress <- req

	}
}
