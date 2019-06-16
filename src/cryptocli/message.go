package main

type Message struct {
	Payload []byte
}

func SendMessage(payload []byte, out chan *Message) {
	out <- &Message{Payload: payload,}
}

func SendMessageLine(payload []byte, out chan *Message) {
	SendMessage(append(payload, '\n'), out)
}

func RelayMessages(read, write chan *Message) {
	for message := range read {
		write <- message
	}

	close(write)
}

func NewPipeMessages() (chan *Message, chan *Message) {
	buff := 5

	in, out := make(chan *Message, buff), make(chan *Message, buff)

	return in, out
}
