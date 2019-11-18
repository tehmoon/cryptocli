package main

//import (
//	"github.com/spf13/pflag"
//	"compress/gzip"
//	"io"
//	"github.com/tehmoon/errors"
//	"log"
//	"sync"
//)
//
//func init() {
//	MODULELIST.Register("gunzip", "Gunzip de-compress", NewGunzip)
//}
//
//type Gunzip struct {}
//
//func (m *Gunzip) Init(in, out chan *Message, global *GlobalFlags) (error) {
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
//
//						reader, writer := io.Pipe()
//
//						wg.Add(2)
//						go func() {
//							for payload := range inc {
//								_, err := writer.Write(payload)
//								if err != nil {
//									err = errors.Wrap(err, "Error wrinting data to pipe")
//									log.Println(err.Error())
//									break
//								}
//							}
//
//							writer.Close()
//							DrainChannel(inc, nil)
//							wg.Done()
//						}()
//
//						go func() {
//							defer wg.Done()
//							defer close(outc)
//
//							gzipReader, err := gzip.NewReader(reader)
//							if err != nil {
//								err = errors.Wrap(err, "Error initializing gunzip reader")
//								log.Println(err.Error())
//								return
//							}
//
//							err = ReadBytesSendMessages(gzipReader, outc)
//							if err != nil {
//								err = errors.Wrap(err, "Error reading gzip reader in gunzip")
//								log.Println(err.Error())
//								return
//							}
//						}()
//					}
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
//func NewGunzip() (Module) {
//	return &Gunzip{}
//}
//
//func (m *Gunzip) SetFlagSet(fs *pflag.FlagSet, args []string) {}
