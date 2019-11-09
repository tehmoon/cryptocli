package main

import (
	"sync"
	"os"
	"log"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
)

var stdinMutex = &StdinMutex{Init: false,}

type StdinMutex struct {
	Init bool
	sync.Mutex
}

func init() {
	MODULELIST.Register("stdin", "Reads from stdin", NewStdin)
}

type Stdin struct {
	sync *sync.WaitGroup
}

func (m *Stdin) Init(in, out chan *Message, global *GlobalFlags) (error) {
	stdinMutex.Lock()
	defer stdinMutex.Unlock()
	defer func() {
		stdinMutex.Init = false
	}()

	if stdinMutex.Init {
		return errors.New("Module \"stdin\" cannot be added more than once")
	}

	stdinMutex.Init = true

	datac := make(chan []byte)
	// Cancel will tell stdin to stop reading and close the out channel
	cancel := make(chan struct{}, 0)


	go func(in, out chan *Message, datac chan []byte, cancel chan struct{}) {
		wg := &sync.WaitGroup{}
		init := false
		outc := make(MessageChannel)
		out <- &Message{
			Type: MessageTypeChannel,
			Interface: outc,
		}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					close(cancel)
					if ! init {
						close(outc)
					}
					for range datac {}
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
						go func(inc, outc MessageChannel, datac chan []byte, wg *sync.WaitGroup, cancel chan struct{}, mutex *StdinMutex) {
							defer wg.Done()

							mutex.Lock()
							defer mutex.Unlock()

							LOOP: for {
								select {
									case <- cancel:
log.Println("canclled")
										break LOOP
									case _, opened := <- inc:
										if ! opened {
											break LOOP
										}
									case payload, opened := <- datac:
										if ! opened {
											break LOOP
										}

										outc <- payload
								}
							}

							close(outc)
							DrainChannel(inc, nil)
						}(inc, outc, datac, wg, cancel, stdinMutex)

						if ! global.MultiStreams {
							wg.Wait()
							close(cancel)
							for range datac {}
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
	}(in, out, datac, cancel)

	go func(out chan *Message, datac chan []byte, cancel chan struct{}) {
		stdinStartOut(datac, cancel)
		out <- &Message{Type: MessageTypeTerminate,}
		close(datac)
	}(out, datac, cancel)

	return nil
}

func NewStdin() (Module) {
	return &Stdin{
		sync: &sync.WaitGroup{},
	}
}

func stdinStartOutRead(datac chan []byte, closed *StdinCloseSync, syn chan struct{}) {
	err := ReadBytesStep(os.Stdin, func(payload []byte) (bool) {
		closed.RLock()
		if closed.Closed {
			return false
		}

		datac <- payload
		closed.RUnlock()
		return true
	})
	if err != nil {
		err = errors.Wrap(err, "Error copying stdin")
		log.Println(err.Error())
	}

	closed.Lock()
	if ! closed.Closed {
		closed.Closed = true
	}
	closed.Unlock()

	close(syn)
}

type StdinCloseSync struct {
	sync.RWMutex
	Closed bool
}

func stdinStartOut(datac chan []byte, cancel chan struct{}) {
	// Closed will signal the reading stdin callback to stop reading.
	closed := &StdinCloseSync{
		Closed: false,
	}

	syn := make(chan struct{}, 0)

	go stdinStartOutRead(datac, closed, syn)

	select {
		case <- cancel:
		case <- syn:
	}

	closed.Lock()
	if ! closed.Closed {
		closed.Closed = true
	}
	closed.Unlock()
}

func (m *Stdin) SetFlagSet(fs *pflag.FlagSet, args []string) {}
