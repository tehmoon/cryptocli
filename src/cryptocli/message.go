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
// The sender needs to call NewMessageChannel() in order to allocate
// a new one. Then it passes the Callback method to the next module.
// The Callback method is a function that returns two things:
//   - A map[string]interface{} that is metadata associated with the channel
//   - A chan []byte that is the raw bytes to be transfered
// The sender must call Start() in order to unlock that callback function.
// It is possible to pass a nil metadata, under the hood it will never be nil.
// The sender will close the channel to signal the end of the
// transmition.
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

// Callable only once
func (mc *MessageChannel) Start(metadata map[string]interface{}) {
	if ! mc.started {
		if metadata == nil {
			metadata = make(map[string]interface{})
		}

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
