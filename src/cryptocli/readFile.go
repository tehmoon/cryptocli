package main

import (
	"github.com/tehmoon/errors"
	"sync"
	"log"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
	"text/template"
	"bytes"
)

type ReadFile struct {
	path string
	pathTmpl string
}

func init() {
	MODULELIST.Register("read-file", "Read file from filesystem", NewReadFile)
}

func (m *ReadFile) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.path, "path", "", "File's path")
	fs.StringVar(&m.pathTmpl, "path-template", "", "Metadata template for file path")
}

func (m *ReadFile) Init(in, out chan *Message, global *GlobalFlags) (err error) {
	if (m.path == "" && m.pathTmpl == "") || (m.path != "" && m.pathTmpl != "") {
		return errors.Errorf("One of %q or %q must be set", "--path", "--path-template")
	}

	var tplPath *template.Template

	tplPath, err = template.New("root").Parse(m.pathTmpl)
	if err != nil {
		return errors.Wrap(err, "Error parsing template for \"--path-template\" flag")
	}

	go func(m *ReadFile, in, out chan *Message) {
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

							p := filepath.Clean(m.path)
							metadata := make(map[string]interface{})
							if m.path != "" {
								metadata["path"] = p
							}

							mc.Start(metadata)
							metadata, inc := cb()
							if m.pathTmpl != "" {
								buff := bytes.NewBuffer(make([]byte, 0))
								err := tplPath.Execute(buff, metadata)
								if err != nil {
									err = errors.Wrap(err, "Error executing template url")
									log.Println(err.Error())
									close(mc.Channel)
									DrainChannel(inc, nil)
									return
								}

								p = string(buff.Bytes()[:])
								buff.Reset()
							}

							outc := mc.Channel

							wg.Add(2)
							go DrainChannel(inc, wg)
							go func(m *ReadFile, outc chan []byte, wg *sync.WaitGroup) {
								defer wg.Done()
								defer close(outc)

								file, err := os.Open(p)
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
	}(m, in, out)

	return nil
}

func NewReadFile() (Module) {
	return &ReadFile{}
}
