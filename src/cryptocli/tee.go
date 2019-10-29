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

	teeIn, teeOut, _, err := InitPipeline(m.pipe, &GlobalFlags{})
	if err != nil {
		return errors.Wrap(err, "Error creating pipeline in tee module")
	}

	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		LOOP: for {
			select {
				case message, opened := <- teeIn:
					if ! opened {
						break LOOP
					}

					switch message.Type {
						case MessageTypeTerminate:
							wg.Wait()
							teeOut <- message
							break LOOP
						case MessageTypeChannel:
							inc, ok := message.Interface.(MessageChannel)
							if ok {
								wg.Add(1)
								go DrainChannel(inc, wg)
							}

					}
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
							inc, ok := message.Interface.(MessageChannel)
							if ok {
								outc := make(MessageChannel)
								teeOutc := make(MessageChannel)

								out <- &Message{
									Type: MessageTypeChannel,
									Interface: outc,
								}
								teeOut <- &Message{
									Type: MessageTypeChannel,
									Interface: teeOutc,
								}
								wg.Add(1)

								go func () {
									for payload := range inc {
										buff := make([]byte, len(payload))
										copy(buff, payload)

										teeOutc <- buff
										outc <- payload
									}

									close(teeOutc)
									close(outc)
									wg.Done()
								}()
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
