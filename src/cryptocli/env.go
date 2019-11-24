package main

import (
	"sync"
	"github.com/spf13/pflag"
	"github.com/tehmoon/errors"
	"os"
	"bytes"
	"text/template"
	"log"
)

func init() {
	MODULELIST.Register("env", "Read an environment variable", NewEnv)
}

type Env struct {
	v string
}

func (m Env) Init(in, out chan *Message, global *GlobalFlags) (err error) {
	if m.v == "" {
		return errors.Errorf("Flag %q must be specified in module init", "var")
	}

	tpl, err := template.New("root").Parse(m.v)
	if err != nil {
		return errors.Wrap(err, "Error parsing template for \"--var\" flag")
	}

	go func() {
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
							defer wg.Done()

							mc.Start(nil)
							metadata, inc := cb()

							buff := bytes.NewBuffer(make([]byte, 0))
							err := tpl.Execute(buff, metadata)
							if err != nil {
								err = errors.Wrap(err, "Error executing template env")
								log.Println(err.Error())
								close(mc.Channel)
								DrainChannel(inc, nil)
								return
							}

							env := string(buff.Bytes()[:])
							buff.Reset()

							wg.Add(1)
							go DrainChannel(inc, wg)

							mc.Channel <- []byte(os.Getenv(env))

							close(mc.Channel)
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
	}()

	return nil
}

func NewEnv() (Module) {
	return &Env{}
}

func (m *Env) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.v, "var", "", "Variable to read from")
}
