package main

//import (
//	"sync"
//	"github.com/spf13/pflag"
//	"io"
//	"encoding/base64"
//	"log"
//	"github.com/tehmoon/errors"
//)
//
//func init() {
//	MODULELIST.Register("base64", "Base64 decode or encode", NewBase64)
//}
//
//type Base64 struct {
//	decode bool
//	encode bool
//}
//
//func (m Base64) Init(in, out chan *Message, global *GlobalFlags) (error) {
//	if (m.decode && m.encode) || (! m.decode && ! m.encode) {
//		return errors.Errorf("One of %q and %q is required", "encode", "decode")
//	}
//
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
//						if m.decode {
//							startBase64Decode(inc, outc, wg)
//							continue
//						}
//
//						if m.encode {
//							startBase64Encode(inc, outc, wg)
//							continue
//						}
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
//func startBase64Decode(inc, outc MessageChannel, wg *sync.WaitGroup) {
//	reader, writer := io.Pipe()
//	b64 := base64.NewDecoder(base64.StdEncoding, reader)
//
//	wg.Add(2)
//	go func() {
//		for payload := range inc {
//			_, err := writer.Write(payload)
//			if err != nil {
//				err = errors.Wrap(err, "Error writing data to pipe in base64")
//				log.Println(err.Error())
//				break
//			}
//		}
//
//		writer.Close()
//		DrainChannel(inc, nil)
//		wg.Done()
//	}()
//
//	go func() {
//		err := ReadBytesSendMessages(b64, outc)
//		if err != nil {
//			err = errors.Wrap(err, "Error reading base64 reader in base64")
//			log.Println(err.Error())
//		}
//
//		close(outc)
//		wg.Done()
//	}()
//}
//
//func startBase64Encode(inc, outc MessageChannel, wg *sync.WaitGroup) {
//	reader, writer := io.Pipe()
//
//	wg.Add(2)
//	go func() {
//		b64w := base64.NewEncoder(base64.StdEncoding, writer)
//
//		for payload := range inc {
//			_, err := b64w.Write(payload)
//			if err != nil {
//				break
//			}
//		}
//
//		b64w.Close()
//		writer.Close()
//
//		DrainChannel(inc, nil)
//		wg.Done()
//	}()
//
//	go func() {
//		err := ReadBytesSendMessages(reader, outc)
//		if err != nil {
//			err = errors.Wrap(err, "Errors in base64 encode")
//			log.Println(err.Error())
//		}
//
//		reader.Close()
//		close(outc)
//		wg.Done()
//	}()
//}
//
//func NewBase64() (Module) {
//	return &Base64{}
//}
//
//func (m *Base64) SetFlagSet(fs *pflag.FlagSet, args []string) {
//	fs.BoolVar(&m.decode, "decode", false, "Base64 decode")
//	fs.BoolVar(&m.encode, "encode", false, "Base64 encode")
//}
