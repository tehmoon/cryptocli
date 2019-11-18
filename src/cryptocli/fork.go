package main

import (
	"io"
	"sync"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"log"
	"os/exec"
	"os"
)

func init() {
	MODULELIST.Register("fork", "Start a program and attach stdin and stdout to the pipeline", NewFork)
}

type Fork struct {
	fs *pflag.FlagSet
}

func (m *Fork) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.SetInterspersed(false)
	m.fs = fs
}

func (m *Fork) Init(in, out chan *Message, global *GlobalFlags) (error) {
	args := SanetizeFlags(m.fs)

	if len(args) == 0 {
		return errors.New("No argument specified in fork module")
	}

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
							defer wg.Done()

							mc.Start(nil)
							_, inc := cb()
							outc := mc.Channel

							cmd := exec.Command(args[0], args[1:]...)
							cmd.Env = make([]string, 0)

							cmdstdin, stdin, err := os.Pipe()
							if err != nil {
								err = errors.Wrap(err, "Error creating pipes for stdin in fork module")
								log.Println(err.Error())
								DrainChannel(inc, nil)
								return
							}

							stdout, cmdstdout, err := os.Pipe()
							if err != nil {
								err = errors.Wrap(err, "Error creating pipes for stdout in fork module")
								log.Println(err.Error())
								DrainChannel(inc, nil)
								return
							}

							cmd.Stdin = cmdstdin
							cmd.Stdout = cmdstdout

							log.Printf("Executing %q with %v in fork module\n", args[0], args[1:])
							cancel := make(chan struct{})

							wg.Add(3)
							go func() {
								err := cmd.Run()
								if err != nil {
									err = errors.Wrap(err, "Error executing command")
									log.Println(err.Error())
								}
								cmdstdout.Close()
								close(cancel)
								wg.Done()
							}()

							go func(stdout io.Reader, outc chan []byte, wg *sync.WaitGroup) {
								defer wg.Done()
								defer close(outc)
								err := ReadBytesSendMessages(stdout, outc)
								if err != nil {
									err = errors.Wrap(err, "Error executing command")
									log.Println(err.Error())
								}
							}(stdout, outc, wg)

							go func() {
								defer wg.Done()
								defer stdin.Close()
								defer DrainChannel(inc, nil)

								LOOP: for {
									select {
										case payload, opened := <- inc:
											if ! opened {
												break LOOP
											}

											_, err := stdin.Write(payload)
											if err != nil {
												err = errors.Wrap(err, "Error writing to forked command")
												log.Println(err.Error())
												break LOOP
											}
										case <- cancel:
											break LOOP
									}
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

func NewFork() (Module) {
	return &Fork{}
}
