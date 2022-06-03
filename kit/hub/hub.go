package hub

import (
	"io"
	"net/http"
)

type Session interface {
	io.WriteCloser

	ID() string
	Request() *http.Request
	Set(string, any)
	Get(string) (any, bool)
}

type Hub interface {
	OnUpgrade(fn http.HandlerFunc)
	OnConnect(fn func(Session))
	OnDisconnect(fn func(Session))
	OnPong(fn func(Session))
	OnMessage(fn func(Session, []byte))
	OnError(fn func(Session, error))
	OnClose(fn func(Session, int))
	OnBroadcast(fn func(Session) bool, msg []byte)
}
