package inout

import (
	"net/url"
	"github.com/tehmoon/errors"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
)

var (
	DefaultS3 = &S3{
		name: "s3://",
		description: "Either upload or download from s3.",
	}
	ErrInoutS3MissingBucket = errors.New("Bucket in url is missing")
	ErrInoutS3MissingKey = errors.New("Key in url is missing")
)

type S3 struct {
	name string
	description string
}

func (s S3) In(u *url.URL) (Input) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
	}))

	return &S3Input{
		name: "s3-input",
		session: sess,
		url: u,
	}
}

func (s S3) Out(u *url.URL) (Output) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
	}))

	return &S3Output{
		name: "s3-output",
		session: sess,
		url: u,
	}
}

func (s S3) Name() (string) {
	return s.name
}

func (s S3) Description() (string) {
	return s.description
}

type S3Input struct {
	url *url.URL
	name string
	session *session.Session
	streamer *S3DownloadStream
	sync chan error
}

func (in *S3Input) Init() (error) {
	var err error

	bucket, key, err := S3FromUrl(in.url)
	if err != nil {
		return err
	}

	in.sync = make(chan error)
	in.streamer = NewS3DownloadStream()

	downloader := s3manager.NewDownloader(in.session, func(d *s3manager.Downloader) {
		d.Concurrency = 1
	})

	dlParams := &s3.GetObjectInput{
		Bucket: &bucket,
		Key: &key,
	}

	go func() {
		_, err := downloader.Download(in.streamer, dlParams)
		in.streamer.Close()
		in.sync <- err
	}()

	return nil
}

func (in S3Input) Read(p []byte) (int, error) {
	return in.streamer.Read(p)
}

func (in S3Input) Close() (error) {
	return <- in.sync
}

func (in S3Input) Name() (string) {
	return in.name
}

type S3Output struct {
	session *session.Session
	name string
	url *url.URL
	reader *io.PipeReader
	writer *io.PipeWriter
	sync chan error
}

func (out *S3Output) Init() (error) {
	bucket, key, err := S3FromUrl(out.url)
	if err != nil {
		return err
	}

	uploader := s3manager.NewUploader(out.session)

	out.sync = make(chan error)
	out.reader, out.writer = io.Pipe()

	upParams := &s3manager.UploadInput{
		Bucket: &bucket,
		Key:		&key,
		Body:		out.reader,
	}

	go func() {
		closeErr := io.EOF
		_, err := uploader.Upload(upParams)
		if err != nil {
			closeErr = err
		}

		out.reader.CloseWithError(closeErr)
		out.sync <- err
	}()

	return nil
}

func (out S3Output) Close() (error) {
	out.writer.Close()
	return <- out.sync
}

func (out S3Output) Write(data []byte) (int, error) {
	return out.writer.Write(data)
}

func (out S3Output) Name() (string) {
	return out.name
}

func (out S3Output) Chomp(chomp bool) {}

func S3FromUrl(u *url.URL) (string, string, error) {
	bucket := u.Host
	key := u.Path

	if bucket == "" {
		return "", "", ErrInoutS3MissingBucket
	}

	if key == "" || key == "/" {
		return "", "", ErrInoutS3MissingKey
	}

	return bucket, key, nil
}

type S3DownloadStream struct {
	reader *io.PipeReader
	writer *io.PipeWriter
	offset int64
}

func NewS3DownloadStream() (*S3DownloadStream) {
	reader, writer := io.Pipe()

	return &S3DownloadStream{
		reader: reader,
		writer: writer,
		offset: 0,
	}
}

func (s S3DownloadStream) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *S3DownloadStream) WriteAt(p []byte, off int64) (int, error) {
	if s.offset != off {
		return 0, io.EOF
	}

	n, err := s.writer.Write(p)
	if err != nil {
		return n, err
	}

	s.offset += int64(n)

	return n, nil
}

func (s S3DownloadStream) Close() (error) {
	return s.writer.Close()
}
