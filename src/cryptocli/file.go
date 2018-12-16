package main

import (
	"os"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
)

func init() {
	MODULELIST.Register("file", "Reads from a file or write to a file.", NewFile)
}

type File struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
	path string
	file *os.File
	mode uint32
	append bool
	read bool
	write bool
}

func (m *File) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.path, "path", "", "File's path")
	fs.BoolVar(&m.read, "read", false, "Read from a file")
	fs.BoolVar(&m.write, "write", false, "Write to a file")
	fs.Uint32Var(&m.mode, "mode", 0640, "Set file's mode if created when writting")
	fs.BoolVar(&m.append, "append", false, "Append data instead of truncating when writting")
}

func (m *File) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *File) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func (m *File) Init(global *GlobalFlags) (error) {
	if m.path == "" {
		return errors.Errorf("Flag %q is missing in file module", "--path")
	}

	if (! m.read && ! m.write) || (m.read && m.write) {
		return errors.Errorf("Specify one of %q or %q", "--read", "--write")
	}

	var (
		file *os.File
		err error
	)

	if m.read {
		file, err = os.OpenFile(m.path, os.O_RDONLY, 0220)
		if err != nil {
			return errors.Wrap(err, "Error opening file in read mode")
		}
	}

	if m.write {
		file, err = FileOpenWrite(m.path, m.append, os.FileMode(m.mode))
		if err != nil {
			return errors.Wrap(err, "Error opening file in write mode")
		}
	}

	if file == nil {
		log.Fatal("Unhandled error")
	}

	log.Printf("File %q is opened", m.path)

	m.file = file

	return nil
}

func (m File) Start() {
	m.sync.Add(2)

	go func() {
		if m.read {
			go fileReadStartIn(m.in, m.sync)
			go fileReadStartOut(m.file, m.out, m.sync)

			return
		}

		if m.write {
			go fileWriteStartIn(m.file, m.in, m.sync)
			go fileWriteStartOut(m.out, m.sync)

			return
		}
	}()
}

func (m File) Wait() {
	m.sync.Wait()

	for range m.in {}
}

func fileReadStartIn(in chan *Message, wg *sync.WaitGroup) {
	for range in {}
	wg.Done()
}

func fileReadStartOut(file *os.File, out chan *Message, wg *sync.WaitGroup) {
	err := ReadBytesStep(file, func(payload []byte) {
		SendMessage(payload, out)
	})
	if err != nil {
		log.Println(errors.Wrap(err, "Error reading from file"))
	}

	file.Close()
	close(out)
	wg.Done()
}

func NewFile() (Module) {
	return &File{
		sync: &sync.WaitGroup{},
	}
}

func FileOpenWrite(p string, append bool, mode os.FileMode) (*os.File, error) {
	perm := os.O_WRONLY | os.O_CREATE | os.O_TRUNC

	if append {
		perm |= os.O_APPEND

		if perm & os.O_TRUNC == os.O_TRUNC {
			perm ^= os.O_TRUNC
		}
	}

	file, err := os.OpenFile(p, perm, os.FileMode(mode))
	if err != nil {
		return nil, err
	}

	return file, nil
}

func fileWriteStartIn(file *os.File, in chan *Message, wg *sync.WaitGroup) {
	for message := range in {
		_, err := file.Write(message.Payload)
		if err != nil {
			log.Println(errors.Wrap(err, "Error writing to file"))
			break
		}
	}

	file.Close()
	wg.Done()
}

func fileWriteStartOut(out chan *Message, wg *sync.WaitGroup) {
	close(out)
	wg.Done()
}
