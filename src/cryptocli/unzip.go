package main

import (
	"io"
	"io/ioutil"
	"sync"
	"github.com/spf13/pflag"
	"github.com/tehmoon/errors"
	"log"
	"os"
	"archive/zip"
	"regexp"
)

func init() {
	MODULELIST.Register("unzip", "Buffer the zip file to disk and read selected file patterns.", NewUnzip)
}

type Unzip struct {
	in chan *Message
	out chan *Message
	wg *sync.WaitGroup
	patterns []string
	rePatterns []*regexp.Regexp
}

func (m *Unzip) Init(global *GlobalFlags) (error) {
	for _, pattern := range m.patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return errors.Wrapf(err, "Err compiling pattern %q", pattern)
		}

		m.rePatterns = append(m.rePatterns, re)
	}

	m.patterns = nil

	return nil
}

type UnzipTempFile struct {
	Err error
	Path string
}

func (m *Unzip) Start() {
	m.wg.Add(1)

	// Used to signal when the zip file as been buffered to
	// disk so the unzip library can safely open it.
	donec := make(chan *UnzipTempFile, 0)

	go UnzipStartIn(m.in, donec)
	go UnzipStartOut(m.rePatterns, m.out, donec, m.wg)
}

func UnzipStartIn(in chan *Message, donec chan *UnzipTempFile) {
	utf := &UnzipTempFile{}
	defer func(utf *UnzipTempFile) {
		donec <- utf
	}(utf)

	tempfile, err := ioutil.TempFile("", "cryptocli-zip")
	if err != nil {
		utf.Err = errors.Wrap(err, "Err writing to temporary file")
		return
	}
	defer tempfile.Close()

	utf.Path = tempfile.Name()

	for message := range in {
		_, err = tempfile.Write(message.Payload)
		if err != nil {
			utf.Err = errors.Wrap(err, "Err writing to temporary file")
			return
		}
	}
}

func UnzipStartOut(patterns []*regexp.Regexp, out chan *Message, donec chan *UnzipTempFile, wg *sync.WaitGroup) {
	defer close(out)
	defer wg.Done()

	utf := <- donec
	if utf.Err != nil {
		if utf.Path != "" {
			os.Remove(utf.Path)
		}

		log.Println(utf.Err.Error())
		return
	}

	defer os.Remove(utf.Path)

	reader, err := zip.OpenReader(utf.Path)
	if err != nil {
		err = errors.Wrap(err, "Err opening zip file")
		log.Println(err.Error())
		return
	}

	err = UnzipReadZip(reader, patterns, out)
	if err != nil {
		err = errors.Wrap(err, "Err reading zipped files")
		log.Println(err.Error())
		return
	}
}

func UnzipReadZippedFile(zfile *zip.File, out chan *Message) (error) {
	file, err := zfile.Open()
	if err != nil {
		return errors.Wrapf(err, "Err opening zipped file %q", zfile.Name)
	}

	defer file.Close()

	err = ReadBytesSendMessages(file, out)
	if err != nil {
		if err != io.EOF {
			return errors.Wrapf(err, "Err reading zipped file %q", zfile.Name)
		}
	}

	return nil
}

func UnzipReadZip(reader *zip.ReadCloser, patterns []*regexp.Regexp, out chan *Message) (error) {
	for _, zfile := range reader.File {
		for _, pattern := range patterns {
			ok := pattern.MatchString(zfile.Name)
			if ok {
				err := UnzipReadZippedFile(zfile, out)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (m *Unzip) Wait() {
	m.wg.Wait()

	for range m.in {}
}

func (m *Unzip) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Unzip) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewUnzip() (Module) {
	return &Unzip{
		wg: &sync.WaitGroup{},
		rePatterns: make([]*regexp.Regexp, 0),
	}
}

func (m *Unzip) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringArrayVar(&m.patterns, "pattern", []string{".*",}, "Read the file each time it matches a pattern.")
}
