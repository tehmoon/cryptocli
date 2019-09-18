package main

import (
	"github.com/tehmoon/errors"
	"sync"
	"log"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
)

type ReadFile struct {
	path string
}

func init() {
	MODULELIST.Register("read-file", "Read file from filesystem", NewReadFile)
}

func (m *ReadFile) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.path, "path", "", "File's path")
}

func (m *ReadFile) Init(in, out chan *Message, global *GlobalFlags) (err error) {
	if m.path == "" {
		return errors.Errorf("Flag %q is missing in file module", "--path")
	}

	go func(m *ReadFile, in, out chan *Message) {
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

						wg.Add(2)
						go DrainChannel(inc, wg)
						go func(m *ReadFile, outc MessageChannel, wg *sync.WaitGroup) {
							defer wg.Done()
							defer close(outc)

							file, err := os.Open(filepath.Clean(m.path))
							if err != nil {
								err = errors.Wrap(err, "Error opening file")
								log.Println(err.Error())
								return
							}

							err = ReadBytesSendMessages(file, outc)
							if err != nil {
								err = errors.Wrap(err, "Error reading file")
								log.Println(err.Error())
								return
							}
						}(m, outc, wg)

						if ! global.MultiStreams {
							if ! init {
								close(outc)
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
	}(m, in, out)

	return nil
}

func NewReadFile() (Module) {
	return &ReadFile{}
}
