package main

import (
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
)

func init() {
	MODULELIST.Register("tee", "Create a new one way pipeline to copy the data over", NewTee)
}

type Tee struct {
	pipe string
	pipeline *Pipeline
}

func (m *Tee) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.pipe, "pipe", "", "Pipeline definition")
}

func (m *Tee) Init(in, out chan *Message, global *GlobalFlags) (err error) {
	if m.pipe == "" {
		return errors.Wrapf(err, "Flag %q must be specified in tee module", "pipe")
	}

	teeIn, teeOut, _, err := InitPipeline(m.pipe, global)
	if err != nil {
		return errors.Wrap(err, "Error creating pipeline in tee module")
	}

	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		wg.Add(1)

		go func() {
			defer wg.Done()
			syn := &sync.WaitGroup{}

			LOOP: for {
				select {
					case message, opened := <- teeIn:
						if ! opened {
							break LOOP
						}

						switch message.Type {
							case MessageTypeTerminate:
								syn.Wait()
								teeOut <- message
								break LOOP
							case MessageTypeChannel:
								cb, ok := message.Interface.(MessageChannelFunc)
								if ok {
									syn.Add(1)
									go func() {
										defer syn.Done()

										_, inc := cb()
										DrainChannel(inc, nil)
									}()
								}
						}
				}
			}
		}()

		LOOP: for {
			select {
				case message, opened := <- in:
					if ! opened {
						break LOOP
					}

					switch message.Type {
						case MessageTypeTerminate:
							teeOut <- message
							wg.Wait()
							out <- message
							break LOOP
						case MessageTypeChannel:
							cb, ok := message.Interface.(MessageChannelFunc)
							if ok {
								mc := NewMessageChannel()
								teemc := NewMessageChannel()

								out <- &Message{
									Type: MessageTypeChannel,
									Interface: mc.Callback,
								}
								teeOut <- &Message{
									Type: MessageTypeChannel,
									Interface: teemc.Callback,
								}
								wg.Add(1)

								go func () {
									mc.Start(nil)
									teemc.Start(nil)
									_, inc := cb()

									for payload := range inc {
										buff := make([]byte, len(payload))
										copy(buff, payload)

										teemc.Channel <- buff
										mc.Channel <- payload
									}

									close(teemc.Channel)
									close(mc.Channel)
									wg.Done()
								}()
							}

							if ! global.MultiStreams {
								wg.Wait()
								out <- &Message{Type: MessageTypeTerminate,}
								teeOut <- &Message{Type: MessageTypeTerminate,}
								break LOOP
							}
					}
			}
		}

		wg.Wait()
		// Last message will signal the closing of the channel
		<- teeIn
		<- in
		close(teeOut)
		close(out)
	}(in, out)

	return nil
}

func NewTee() (Module) {
	return &Tee{}
}
