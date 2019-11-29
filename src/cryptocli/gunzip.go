package main

import (
	"github.com/spf13/pflag"
	"compress/gzip"
	"io"
	"github.com/tehmoon/errors"
	"log"
	"sync"
)

func init() {
	MODULELIST.Register("gunzip", "Gunzip de-compress", NewGunzip)
}

type Gunzip struct {}

func (m *Gunzip) Init(in, out chan *Message, global *GlobalFlags) (error) {
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

						reader, writer := io.Pipe()

						wg.Add(1)
						go func() {
							defer wg.Done()

							mc.Start(nil)
							_, inc := cb()
							outc := mc.Channel

							wg.Add(2)
							go func() {
								for payload := range inc {
									_, err := writer.Write(payload)
									if err != nil {
										err = errors.Wrap(err, "Error wrinting data to pipe")
										log.Println(err.Error())
										break
									}
								}

								writer.Close()
								DrainChannel(inc, nil)
								wg.Done()
							}()

							go func() {
								defer wg.Done()
								defer close(outc)

								gzipReader, err := gzip.NewReader(reader)
								if err != nil {
									err = errors.Wrap(err, "Error initializing gunzip reader")
									log.Println(err.Error())
									return
								}

								err = ReadBytesSendMessages(gzipReader, outc)
								if err != nil {
									err = errors.Wrap(err, "Error reading gzip reader in gunzip")
									log.Println(err.Error())
									return
								}
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

func NewGunzip() (Module) {
	return &Gunzip{}
}

func (m *Gunzip) SetFlagSet(fs *pflag.FlagSet, args []string) {}
