package main

import (
	"io"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"bytes"
	"text/template"
	"path"
)

func init() {
	MODULELIST.Register("write-s3", "uploads a file to s3", NewWriteS3)
}

type WriteS3 struct {
	bucket string
	path string
	pathTmpl *template.Template
	bucketTmpl *template.Template
	session *session.Session
}

func (m *WriteS3) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.bucket, "bucket", "", "Specify the bucket name using metadata")
	fs.StringVar(&m.path, "path", "", "Object path using metadata")
}

func (m *WriteS3) Init(in, out chan *Message, global *GlobalFlags) (err error) {
	if m.path == "" {
		return errors.Errorf("Path %q is missing", "--path")
	}

	if m.bucket == "" {
		return errors.Errorf("Path %q is missing", "--bucket")
	}

	m.pathTmpl, err = template.New("root").Parse(m.path)
	if err != nil {
		return errors.Wrap(err, "Error parsing template for \"--path\" flag")
	}

	m.bucketTmpl, err = template.New("root").Parse(m.bucket)
	if err != nil {
		return errors.Wrap(err, "Error parsing template for \"--bucket\" flag")
	}

	m.session = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

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

							mc.Start(map[string]interface{}{
								"path": m.path,
								"bucket": m.bucket,
							})
							metadata, inc := cb()

							buff := bytes.NewBuffer(make([]byte, 0))
							err := m.pathTmpl.Execute(buff, metadata)
							if err != nil {
								err = errors.Wrap(err, "Error executing template path")
								log.Println(err.Error())
								close(mc.Channel)
								DrainChannel(inc, nil)
								return
							}

							p := path.Clean(string(buff.Bytes()[:]))
							buff.Reset()

							err = m.bucketTmpl.Execute(buff, metadata)
							if err != nil {
								err = errors.Wrap(err, "Error executing template bucket")
								log.Println(err.Error())
								close(mc.Channel)
								DrainChannel(inc, nil)
								return
							}

							b := string(buff.Bytes()[:])
							buff.Reset()

							outc := mc.Channel

							options := &S3Options{
								Bucket: b,
								Path: p,
								Session: m.session,
							}

							s3WriteStartIn(inc, outc, options)
						}()

						if ! global.MultiStreams {
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

func s3WriteStartIn(inc, outc chan []byte, options *S3Options) {
	uploader := s3manager.NewUploader(options.Session)

	reader, writer := io.Pipe()

	params := &s3manager.UploadInput{
		Bucket: &options.Bucket,
		Key: &options.Path,
		Body: reader,
	}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func(inc chan []byte, writer *io.PipeWriter, wg *sync.WaitGroup) {
		defer wg.Done()

		for message := range inc {
			_, err := writer.Write(message)
			if err != nil {
				err = errors.Wrap(err, "Error writing to s3")
				log.Println(err.Error())
				writer.CloseWithError(err)
				DrainChannel(inc, nil)
				return
			}
		}

		writer.Close()
	}(inc, writer, wg)

	_, err := uploader.Upload(params)
	if err != nil {
		err = errors.Wrap(err, "Error writing to s3")
		log.Println(err.Error())
		reader.Close()
	}

	close(outc)

	wg.Wait()
}

func NewWriteS3() (Module) {
	return &WriteS3{}
}

type S3Options struct {
	Bucket string
	Path string
	Session *session.Session
}
