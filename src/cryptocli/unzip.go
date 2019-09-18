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
	patterns []string
	rePatterns []*regexp.Regexp
}

func (m *Unzip) Init(in, out chan *Message, global *GlobalFlags) (error) {
	rePatterns := make([]*regexp.Regexp, 0)

	for _, pattern := range m.patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return errors.Wrapf(err, "Err compiling pattern %q", pattern)
		}

		rePatterns = append(rePatterns, re)
	}

	m.patterns = nil

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

						go func(inc, outc MessageChannel, patterns []*regexp.Regexp, wg *sync.WaitGroup) {
							defer wg.Done()
							defer close(outc)

							tempfile, err := ioutil.TempFile("", "cryptocli-zip")
							if err != nil {
								err = errors.Wrap(err, "Err writing to temporary file")
								log.Println(err.Error())
								DrainChannel(inc, nil)
								return
							}
							defer os.Remove(tempfile.Name())
							for payload := range inc {
								_, err = tempfile.Write(payload)
								if err != nil {
									err = errors.Wrap(err, "Err writing to temporary file")
									log.Println(err.Error())
									tempfile.Close()
									DrainChannel(inc, nil)
									return
								}
							}

							// Let's close the file so we can open it with the zip reader
							tempfile.Close()

							reader, err := zip.OpenReader(tempfile.Name())
							if err != nil {
								err = errors.Wrap(err, "Err opening zip file")
								log.Println(err.Error())
								return
							}

							err = UnzipReadZip(reader, patterns, outc)
							if err != nil {
								err = errors.Wrap(err, "Err reading zipped files")
								log.Println(err.Error())
								return
							}
						}(inc, outc, rePatterns, wg)
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

func UnzipReadZippedFile(zfile *zip.File, outc MessageChannel) (error) {
	file, err := zfile.Open()
	if err != nil {
		return errors.Wrapf(err, "Err opening zipped file %q", zfile.Name)
	}

	defer file.Close()

	err = ReadBytesSendMessages(file, outc)
	if err != nil {
		if err != io.EOF {
			return errors.Wrapf(err, "Err reading zipped file %q", zfile.Name)
		}
	}

	return nil
}

func UnzipReadZip(reader *zip.ReadCloser, patterns []*regexp.Regexp, outc MessageChannel) (error) {
	for _, zfile := range reader.File {
		for _, pattern := range patterns {
			ok := pattern.MatchString(zfile.Name)
			if ok {
				err := UnzipReadZippedFile(zfile, outc)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func NewUnzip() (Module) {
	return &Unzip{}
}

func (m *Unzip) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringArrayVar(&m.patterns, "pattern", []string{".*",}, "Read the file each time it matches a pattern.")
}
