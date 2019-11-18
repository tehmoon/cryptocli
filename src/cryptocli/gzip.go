package main

//import (
//	"sync"
//	"github.com/spf13/pflag"
//	"compress/gzip"
//	"bytes"
//)
//
//func init() {
//	MODULELIST.Register("gzip", "Gzip compress", NewGzip)
//}
//
//type Gzip struct {}
//
//func (m Gzip) Init(in, out chan *Message, global *GlobalFlags) (error) {
//	go func(in, out chan *Message) {
//		wg := &sync.WaitGroup{}
//
//		LOOP: for message := range in {
//			switch message.Type {
//				case MessageTypeTerminate:
//					wg.Wait()
//					out <- message
//					break LOOP
//				case MessageTypeChannel:
//					inc, ok := message.Interface.(MessageChannel)
//					if ok {
//						outc := make(MessageChannel)
//
//						out <- &Message{
//							Type: MessageTypeChannel,
//							Interface: outc,
//						}
//						wg.Add(1)
//						go func() {
//							buff := bytes.NewBuffer(nil)
//							gzipWriter := gzip.NewWriter(buff)
//
//							go func() {
//								for payload := range inc {
//									_, err := gzipWriter.Write(payload)
//									if err != nil {
//										break
//									}
//
//									gzipWriter.Flush()
//
//									outc <- CopyResetBuffer(buff)
//								}
//
//								close(outc)
//								DrainChannel(inc, nil)
//								wg.Done()
//							}()
//						}()
//					}
//
//			}
//		}
//
//		wg.Wait()
//		// Last message will signal the closing of the channel
//		<- in
//		close(out)
//	}(in, out)
//
//	return nil
//}
//
//func NewGzip() (Module) {
//	return &Gzip{}
//}
//
//func (m *Gzip) SetFlagSet(fs *pflag.FlagSet, args []string) {}
