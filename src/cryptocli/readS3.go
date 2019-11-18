package main

//import (
//	"github.com/aws/aws-sdk-go/aws/session"
//	"github.com/aws/aws-sdk-go/service/s3/s3manager"
//	"github.com/tehmoon/errors"
//	"github.com/aws/aws-sdk-go/service/s3"
//	"github.com/spf13/pflag"
//	"sync"
//	"log"
//	"path"
//	"io"
//)
//
//func init() {
//	MODULELIST.Register("read-s3", "Read a file from s3", NewReadS3)
//}
//
//type ReadS3 struct {
//	bucket string
//	path string
//}
//
//func (m *ReadS3) SetFlagSet(fs *pflag.FlagSet, args []string) {
//	fs.StringVar(&m.bucket, "bucket", "", "Specify the bucket name")
//	fs.StringVar(&m.path, "path", "", "Object path")
//}
//
//func (m *ReadS3) Init(in, out chan *Message, global *GlobalFlags) (err error) {
//	if m.path == "" {
//		return errors.Errorf("Path %q is missing", "--path")
//	}
//
//	if m.bucket == "" {
//		return errors.Errorf("Path %q is missing", "--bucket")
//	}
//
//	session := session.Must(session.NewSessionWithOptions(session.Options{
//		SharedConfigState: session.SharedConfigEnable,
//	}))
//
//	s3options := &S3Options{
//		Bucket: m.bucket,
//		Path: path.Clean(m.path),
//		Session: session,
//	}
//
//	go func(m *ReadS3, in, out chan *Message) {
//		wg := &sync.WaitGroup{}
//
//		init := false
//		outc := make(MessageChannel)
//
//		out <- &Message{
//			Type: MessageTypeChannel,
//			Interface: outc,
//		}
//
//		LOOP: for message := range in {
//			switch message.Type {
//				case MessageTypeTerminate:
//					if ! init {
//						close(outc)
//					}
//					wg.Wait()
//					out <- message
//					break LOOP
//				case MessageTypeChannel:
//					inc, ok := message.Interface.(MessageChannel)
//					if ok {
//						if ! init {
//							init = true
//						} else {
//							outc = make(MessageChannel)
//
//							out <- &Message{
//								Type: MessageTypeChannel,
//								Interface: outc,
//							}
//						}
//
//						wg.Add(2)
//						go DrainChannel(inc, wg)
//						go ReadS3StartOut(outc, s3options, wg)
//
//						if ! global.MultiStreams {
//							if ! init {
//								close(outc)
//							}
//							wg.Wait()
//							out <- &Message{Type: MessageTypeTerminate,}
//							break LOOP
//						}
//					}
//			}
//		}
//
//		wg.Wait()
//		// Last message will signal the closing of the channel
//		<- in
//		close(out)
//	}(m, in, out)
//
//	return nil
//}
//
//func ReadS3StartOut(outc MessageChannel, options *S3Options, wg *sync.WaitGroup) {
//	defer wg.Done()
//	defer close(outc)
//
//	downloader := s3manager.NewDownloader(options.Session, func(d *s3manager.Downloader) {
//		d.Concurrency = 1
//	})
//
//	params := &s3.GetObjectInput{
//		Bucket: &options.Bucket,
//		Key: &options.Path,
//	}
//
//	_, err := downloader.Download(NewS3DownloadStream(outc), params)
//	if err != nil {
//		err = errors.Wrap(err, "Error reading from s3")
//		log.Println(err.Error())
//		return
//	}
//}
//
//type S3DownloadStream struct {
//	outc MessageChannel
//	offset int64
//}
//
//func NewS3DownloadStream(outc MessageChannel) (*S3DownloadStream) {
//	return &S3DownloadStream{
//		outc: outc,
//		offset: 0,
//	}
//}
//
//func (s *S3DownloadStream) WriteAt(p []byte, off int64) (int, error) {
//	if s.offset != off {
//		return 0, io.EOF
//	}
//
//	buff := make([]byte, len(p))
//	copy(buff, p)
//
//	s.outc <- buff
//
//	s.offset += int64(len(p))
//
//	return len(p), nil
//}
//
//func NewReadS3() (Module) {
//	return &ReadS3{}
//}
