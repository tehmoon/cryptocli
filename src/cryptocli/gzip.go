package main

import (
	"sync"
	"github.com/spf13/pflag"
	"compress/gzip"
	"bytes"
)

func init() {
	MODULELIST.Register("gzip", "Gzip compress", NewGzip)
}

type Gzip struct {}

func (m Gzip) Init(in, out chan *Message, global *GlobalFlags) (error) {
	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		init := false
		mc := NewMessageChannel()

		out <- &Message{
			Type: MessageTypeChannel,
			Interface: mc.Callback,
		}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					if ! init {
						close(mc.Channel)
					}

					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					cb, ok := message.Interface.(MessageChannelFunc)
					if ok {
						if ! init {
							init = true
						} else {
							mc = NewMessageChannel()

							out <- &Message{
								Type: MessageTypeChannel,
								Interface: mc.Callback,
							}
						}

						wg.Add(1)
						go func() {
							buff := bytes.NewBuffer(nil)
							gzipWriter := gzip.NewWriter(buff)

							go func() {
								mc.Start(nil)
								_, inc := cb()

								for payload := range inc {
									_, err := gzipWriter.Write(payload)
									if err != nil {
										break
									}

									gzipWriter.Flush()

									mc.Channel <- CopyResetBuffer(buff)
								}

								close(mc.Channel)
								DrainChannel(inc, nil)
								wg.Done()
							}()
						}()

						if ! global.MultiStreams {
							if ! init {
								close(mc.Channel)
							}
							wg.Wait()
							out <- &Message{Type: MessageTypeTerminate,}
							break LOOP
						}
					}

			}
		}

		wg.Wait()
		// Last message will signal the closing of the channel
		<- in
		close(out)
	}(in, out)

	return nil
}

func NewGzip() (Module) {
	return &Gzip{}
}

func (m *Gzip) SetFlagSet(fs *pflag.FlagSet, args []string) {}
