package main

import (
	"sync"
	"os"
	"log"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
)

var stdinMutex = &StdinMutex{
	Init: false,
	Datac: make(chan []byte),
}

type StdinMutex struct {
	Init bool
	Datac chan []byte
	sync.Mutex
}

func init() {
	MODULELIST.Register("stdin", "Reads from stdin", NewStdin)
}

type Stdin struct {}

func (m *Stdin) Init(in, out chan *Message, global *GlobalFlags) (error) {
	stdinMutex.Lock()
	defer stdinMutex.Unlock()

	if ! stdinMutex.Init {
		stdinMutex.Init = true
		go stdinStartRead(stdinMutex.Datac)
	}

	go func(in, out chan *Message, mutex *StdinMutex) {
		wg := &sync.WaitGroup{}
		init := false
		outc := make(MessageChannel)
		out <- &Message{
			Type: MessageTypeChannel,
			Interface: outc,
		}

		cancel := make(chan struct{})

		LOOP: for {
			select {
				case _, opened := <- cancel:
					if ! opened {
						wg.Wait()
						out <- &Message{Type: MessageTypeTerminate,}
						break LOOP
					}
				case message, opened := <- in:
					if ! opened {
						break LOOP
					}

					switch message.Type {
						case MessageTypeTerminate:
							if ! init {
								close(outc)
							}
							wg.Wait()
							out <- message
							break LOOP

						case MessageTypeChannel:
							inc, ok := message.Interface.(MessageChannel)
							if ok {
								if ! init {
									init = true
								} else {
									outc = make(MessageChannel)

									out <- &Message{
										Type: MessageTypeChannel,
										Interface: outc,
									}
								}

								wg.Add(1)
								go func(inc, outc MessageChannel, mutex *StdinMutex, cancel chan struct{}, wg *sync.WaitGroup) {
									defer wg.Done()

									mutex.Lock()
									defer mutex.Unlock()

									LOOP: for {
										select {
											case _, opened := <- inc:
												if ! opened {
													break LOOP
												}
											case payload, opened := <- mutex.Datac:
												if ! opened {
													close(cancel)
													break LOOP
												}

												outc <- payload
										}
									}

									close(outc)
									DrainChannel(inc, nil)
								}(inc, outc, stdinMutex, cancel, wg)

								if ! global.MultiStreams {
									wg.Wait()
									out <- &Message{Type: MessageTypeTerminate,}
									break LOOP
								}
							}

					}
			}
		}

		wg.Wait()
		// Last message will signal the closing of the channel
		<- in
		close(out)
	}(in, out, stdinMutex)

	return nil
}

func NewStdin() (Module) {
	return &Stdin{}
}

// Make this a global that is started once when the module starts.
// Everything will read from it. There is not clean way to close
// stdin so leave it open, this way it can be re-used.
func stdinStartRead(datac chan []byte) {
	defer close(datac)

	err := ReadBytesStep(os.Stdin, func(payload []byte) (bool) {
		datac <- payload
		return true
	})
	if err != nil {
		err = errors.Wrap(err, "Error copying stdin")
		log.Println(err.Error())
		return
	}
}

func (m *Stdin) SetFlagSet(fs *pflag.FlagSet, args []string) {}
