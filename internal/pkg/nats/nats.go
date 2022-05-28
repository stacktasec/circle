package nats

import "github.com/nats-io/nats.go"

func Init() {
	nc, _ := nats.Connect("")

	nc.JetStream(nats.APIPrefix(""))
}
