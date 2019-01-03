package main

import (
	"io"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3"
)

func init() {
	MODULELIST.Register("s3", "Downloads or uploads a file from s3", NewS3)
}

type S3 struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
	read bool
	write bool
	bucket string
	path string
	session *session.Session
}

func (m *S3) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.bucket, "bucket", "", "Specify the bucket name")
	fs.StringVar(&m.path, "path", "", "Object path")
	fs.BoolVar(&m.read, "read", false, "Read from s3")
	fs.BoolVar(&m.write, "write", false, "Write to s3")
}

func (m *S3) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *S3) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func (m *S3) Init(global *GlobalFlags) (error) {
	if m.path == "" {
		return errors.Errorf("Path %q is missing", "--path")
	}

	if m.bucket == "" {
		return errors.Errorf("Path %q is missing", "--bucket")
	}

	if (m.read && m.write) || (! m.read && ! m.write) {
		return errors.Errorf("Specify on of %q or %q", "--read", "--write")
	}

	m.session = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
	}))

	return nil
}

func (m S3) Start() {
	m.sync.Add(1)

	options := &S3Options{
		Bucket: m.bucket,
		Path: m.path,
		Session: m.session,
	}

	go func() {
		if m.read {
			m.sync.Add(1)
			go s3ReadStartIn(m.in, m.sync)
			go s3ReadStartOut(m.out, options, m.sync)

			return
		}

		if m.write {
			go s3WriteStartIn(m.in, m.out, options, m.sync)

			return
		}

		log.Fatal("Unhandled case")
	}()
}

func (m S3) Wait() {
	m.sync.Wait()

	for range m.in {}
}

func s3ReadStartIn(in chan *Message, wg *sync.WaitGroup) {
	for range in {}

	wg.Done()
}

func s3ReadStartOut(out chan *Message, options *S3Options, wg *sync.WaitGroup) {
	downloader := s3manager.NewDownloader(options.Session, func(d *s3manager.Downloader) {
		d.Concurrency = 1
	})

	params := &s3.GetObjectInput{
		Bucket: &options.Bucket,
		Key: &options.Path,
	}

	_, err := downloader.Download(NewS3DownloadStream(out), params)
	if err != nil {
		log.Println(errors.Wrap(err, "Error reading from s3"))
	}

	close(out)
	wg.Done()
}

func s3WriteStartIn(in, out chan *Message, options *S3Options, wg *sync.WaitGroup) {
	uploader := s3manager.NewUploader(options.Session)

	reader, writer := io.Pipe()

	params := &s3manager.UploadInput{
		Bucket: &options.Bucket,
		Key: &options.Path,
		Body: reader,
	}

	go func() {
		for message := range in {
			_, err := writer.Write(message.Payload)
			if err != nil {
				break
			}
		}

		writer.Close()
	}()

	_, err := uploader.Upload(params)
	if err != nil {
		log.Println(errors.Wrap(err, "Error writing to s3"))
		reader.Close()
	}

	close(out)

	wg.Done()
}

func NewS3() (Module) {
	return &S3{
		sync: &sync.WaitGroup{},
	}
}

type S3DownloadStream struct {
	out chan *Message
	offset int64
}

func NewS3DownloadStream(out chan *Message) (*S3DownloadStream) {
	return &S3DownloadStream{
		out: out,
		offset: 0,
	}
}

func (s *S3DownloadStream) WriteAt(p []byte, off int64) (int, error) {
	if s.offset != off {
		return 0, io.EOF
	}

	buff := make([]byte, len(p))
	copy(buff, p)

	SendMessage(buff, s.out)

	s.offset += int64(len(p))

	return len(p), nil
}

type S3Options struct {
	Bucket string
	Path string
	Session *session.Session
}
