package main

import (
	"sync"
)

type MessageType int

const (
	// Channel message signals a new channel.
	// From this channel, data will be sent.
	// The channel will be closed by the sender.
	MessageTypeChannel MessageType = iota

	// Terminate messages is used to instruct the module to terminate.
	// It it expected to be recieved twice:
	// 	once to shutdown the module and
	MessageTypeTerminate
)

// A channel is a way to communicate between modules.
// For now, it only holds an array of byte which represents
// raw data.
// The sender will close the channel to signal the end of the
// transmition.
// The reciever takes care of creating new channels.
type MessageChannel struct {
	started bool
	metadata map[string]interface{}
	Channel chan []byte
	wg *sync.WaitGroup
	Callback MessageChannelFunc
}

func NewMessageChannel() (mc *MessageChannel) {
	mc = &MessageChannel{
		started: false,
		Channel: make(chan []byte),
		wg: &sync.WaitGroup{},
	}

	mc.Callback = func() (metadata map[string]interface{}, inc chan []byte) {
		mc.wg.Wait()

		return mc.metadata, mc.Channel
	}

	mc.wg.Add(1)
	return mc
}

func (mc *MessageChannel) Start(metadata map[string]interface{}) {
	if ! mc.started {
		mc.metadata = metadata
		mc.started = true
		mc.wg.Done()
	}
}

type MessageChannelFunc func() (metadata map[string]interface{}, inc chan []byte)

// MessageType will indicate what is the underlying type
// of the field Interface. Then casting is necessary to use it.
type Message struct {
	Type MessageType
	Interface interface{}
}

func RelayMessages(in, out chan *Message) {
	LOOP: for message := range in {
		switch message.Type {
			case MessageTypeTerminate:
				out <- message
				break LOOP
			default:
				out <- message
		}
	}

	<- in
	close(out)
}
