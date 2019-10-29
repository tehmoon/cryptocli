package main

import (
	"os"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"path/filepath"
)

func init() {
	MODULELIST.Register("write-file", "Writes to a file.", NewWriteFile)
}

type WriteFile struct {
	path string
	file *os.File
	mode uint32
	append bool
}

func (m *WriteFile) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.path, "path", "", "File's path")
	fs.Uint32Var(&m.mode, "mode", 0640, "Set file's mode if created when writting")
	fs.BoolVar(&m.append, "append", false, "Append data instead of truncating when writting")
}

func (m *WriteFile) Init(in, out chan *Message, global *GlobalFlags) (error) {
	if m.path == "" {
		return errors.Errorf("Flag %q is missing in file module", "--path")
	}

	var (
		file *os.File
		err error
	)

	file, err = WriteFileOpenWrite(m.path, m.append, os.FileMode(m.mode))
	if err != nil {
		return errors.Wrap(err, "Error opening file in write mode")
	}

	if file == nil {
		log.Fatal("Unhandled error")
	}

	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					inc, ok := message.Interface.(MessageChannel)
					if ok {
						outc := make(MessageChannel)

						out <- &Message{
							Type: MessageTypeChannel,
							Interface: outc,
						}
						wg.Add(1)
						go fileWriteStart(file, inc, outc, wg)

						wg.Wait()
						out <- &Message{
							Type: MessageTypeTerminate,
						}
						break LOOP
					}

			}
		}

		wg.Wait()
		// Last message will signal the closing of the channel
		<- in
		close(out)
	}(in, out)

	log.Printf("File %q is opened", m.path)

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

	return file, nil
}

func fileWriteStart(file *os.File, inc, outc MessageChannel, wg *sync.WaitGroup) {
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
	wg.Done()
}
