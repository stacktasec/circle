package nats

func InitNats() {
	nc, _ := nats.Connect("")

	nc.JetStream(nats.APIPrefix(""))
}
