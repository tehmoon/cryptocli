package main

import (
	"os"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"path/filepath"
	"bytes"
	"text/template"
)

func init() {
	MODULELIST.Register("write-file", "Writes to a file.", NewWriteFile)
}

type WriteFile struct {
	file *os.File
	mode uint32
	append bool
	pathTemplate string
	tpl *template.Template
}

func (m *WriteFile) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.pathTemplate, "path-template", "", "Metadata template for file path")
	fs.Uint32Var(&m.mode, "mode", 0640, "Set file's mode if created when writting")
	fs.BoolVar(&m.append, "append", false, "Append data instead of truncating when writting")
}

func (m *WriteFile) Init(in, out chan *Message, global *GlobalFlags) (error) {
	if m.pathTemplate == "" {
		return errors.Errorf("Flag %q must be set", "--path-template")
	}

	var err error
	m.tpl, err = template.New("root").Parse(m.pathTemplate)
	if err != nil {
		return errors.Wrap(err, "Error parsing template for \"--path-template\" flag")
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
						go fileWriteStart(m, cb, mc, wg)

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

func NewWriteFile() (Module) {
	return &WriteFile{}
}

func WriteFileOpenWrite(p string, append bool, mode os.FileMode) (*os.File, error) {
	perm := os.O_WRONLY | os.O_CREATE | os.O_TRUNC

	if append {
		perm |= os.O_APPEND

		if perm & os.O_TRUNC == os.O_TRUNC {
			perm ^= os.O_TRUNC
		}
	}

	file, err := os.OpenFile(filepath.Clean(p), perm, os.FileMode(mode))
	if err != nil {
		return nil, err
	}

	log.Printf("File %q is opened\n", file.Name())

	return file, nil
}

func fileWriteStart(m *WriteFile, cb MessageChannelFunc, mc *MessageChannel, wg *sync.WaitGroup) {
	defer wg.Done()

	mc.Start(map[string]interface{}{
		"path-template": m.pathTemplate,
	})

	metadata, inc := cb()

	buff := bytes.NewBuffer(make([]byte, 0))
	err := m.tpl.Execute(buff, metadata)
	if err != nil {
		err = errors.Wrap(err, "Error executing template file")
		log.Println(err.Error())
		close(mc.Channel)
		DrainChannel(inc, nil)
		return
	}

	p := filepath.Clean(string(buff.Bytes()[:]))
	buff.Reset()

	outc := mc.Channel

	file, err := WriteFileOpenWrite(p, m.append, os.FileMode(m.mode))
	if err != nil {
		err = errors.Wrap(err, "Error opening file in write mode")
		log.Println(err.Error())
		close(mc.Channel)
		DrainChannel(inc, nil)
		return
	}

	for message := range inc {
		_, err := file.Write(message)
		if err != nil {
			log.Println(errors.Wrap(err, "Error writing to file"))
			break
		}
	}

	file.Close()
	close(outc)

	DrainChannel(inc, nil)
}
