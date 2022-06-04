package hub

import (
	"io"
	"net/http"
)

type Session interface {
	io.ReadWriteCloser

	ID() string
	Request() *http.Request
	Set(string, any)
	Get(string) (any, bool)
}

type Hub interface {
	Register(session Session) error
	Unregister(session Session) error
	Send(id string, msg []byte) error
	Broadcast(fn func(Session) bool, msg []byte) error

	OnUpgrade(fn http.HandlerFunc)
	OnMessage(fn func(Session, []byte))
	OnError(fn func(Session, error))
	OnClose(fn func(Session, int))
}
