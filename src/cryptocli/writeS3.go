package main

import (
	"io"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func init() {
	MODULELIST.Register("write-s3", "uploads a file to s3", NewWriteS3)
}

type WriteS3 struct {
	bucket string
	path string
	session *session.Session
}

func (m *WriteS3) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.bucket, "bucket", "", "Specify the bucket name")
	fs.StringVar(&m.path, "path", "", "Object path")
}

func (m *WriteS3) Init(in, out chan *Message, global *GlobalFlags) (error) {
	if m.path == "" {
		return errors.Errorf("Path %q is missing", "--path")
	}

	if m.bucket == "" {
		return errors.Errorf("Path %q is missing", "--bucket")
	}

	m.session = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	options := &S3Options{
		Bucket: m.bucket,
		Path: m.path,
		Session: m.session,
	}

	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		outc := make(MessageChannel)

		out <- &Message{
			Type: MessageTypeChannel,
			Interface: outc,
		}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					inc, ok := message.Interface.(MessageChannel)
					if ok {
						wg.Add(1)
						go s3WriteStartIn(inc, outc, options, wg)

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

	return nil
}

func s3WriteStartIn(inc, outc MessageChannel, options *S3Options, wg *sync.WaitGroup) {
	uploader := s3manager.NewUploader(options.Session)

	reader, writer := io.Pipe()

	params := &s3manager.UploadInput{
		Bucket: &options.Bucket,
		Key: &options.Path,
		Body: reader,
	}

	wg.Add(1)
	go func(inc MessageChannel, writer *io.PipeWriter, wg *sync.WaitGroup) {
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

	wg.Done()
}

func NewWriteS3() (Module) {
	return &WriteS3{}
}

type S3Options struct {
	Bucket string
	Path string
	Session *session.Session
}
