package main

//import (
//	"sync"
//	"github.com/spf13/pflag"
//	"github.com/tehmoon/errors"
//	"log"
//	"encoding/hex"
//)
//
//func init() {
//	MODULELIST.Register("hex", "Hex encoding/decoding", NewHex)
//}
//
//type Hex struct {
//	encode bool
//	decode bool
//}
//
//func (m *Hex) Init(in, out chan *Message, global *GlobalFlags) (error) {
//	if (m.encode && m.decode) || (! m.encode && ! m.decode) {
//		return errors.Errorf("One of %q and %q must be provided", "encode", "decode")
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
//						wg.Add(1)
//						if m.encode {
//							go startHexEncode(inc, outc, wg)
//							continue
//						}
//
//						go startHexDecode(inc, outc, wg)
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
//// TODO: limit the buffer size because it allocates * 2 right now.
//func startHexEncode(inc, outc MessageChannel, wg *sync.WaitGroup) {
//	for payload := range inc {
//		buff := make([]byte, hex.EncodedLen(len(payload)))
//		hex.Encode(buff, payload)
//
//		outc <- buff
//	}
//
//	close(outc)
//	wg.Done()
//}
//
//func startHexDecode(inc, outc MessageChannel, wg *sync.WaitGroup) {
//	var (
//		crumb byte
//		set bool
//		buff []byte
//	)
//
//	for payload := range inc {
//		l := len(payload)
//
//		// If we have data, there are 4 states:
//		//	- we have even number of bytes and no bytes from the previous message
//		//	- even number of bytes and one byte left from the previous message
//		//	- odd number of bytes and no bytes from the previous message
//		//	- odd number of bytes and one byte left from the previous message
//		if l != 0 {
//			if l % 2 == 0 && ! set {
//			} else if l % 2 == 0 && set {
//				payload = append([]byte{crumb,}, payload[:l - 1]...)
//				crumb = payload[l - 1]
//
//			} else if l % 2 != 0 && set {
//				payload = append([]byte{crumb,}, payload[:]...)
//				set = false
//
//			} else {
//				crumb = payload[l - 1]
//				payload = payload[:l - 1]
//
//				set = true
//			}
//
//			buff = make([]byte, hex.DecodedLen(len(payload)))
//			_, err := hex.Decode(buff, payload)
//			if err != nil {
//				err = errors.Wrap(err, "Error decoding hex")
//				log.Println(err.Error())
//				break
//			}
//
//			outc <- buff
//		}
//	}
//
//	close(outc)
//	DrainChannel(inc, nil)
//	wg.Done()
//}
//
//func NewHex() (Module) {
//	return &Hex{}
//}
//
//func (m *Hex) SetFlagSet(fs *pflag.FlagSet, args []string) {
//	fs.BoolVar(&m.encode, "encode", false, "Hexadecimal encode")
//	fs.BoolVar(&m.decode, "decode", false, "Hexadecimal decode")
//}
