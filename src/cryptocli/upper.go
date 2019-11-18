package main

//import (
//	"github.com/spf13/pflag"
//	"sync"
//)
//
//func init() {
//	MODULELIST.Register("upper", "Uppercase all ascii characters", NewUpper)
//}
//
//type Upper struct {}
//
//func (m Upper) Init(in, out chan *Message, global *GlobalFlags) (error) {
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
//						go startUpper(inc, outc, wg)
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
//func NewUpper() (Module) {
//	return &Upper{}
//}
//
//func startUpper(inc, outc MessageChannel, wg *sync.WaitGroup) {
//	for payload := range inc {
//		for i, b := range payload {
//			if b > 96 && b < 123 {
//				payload[i] = b - 32
//			}
//		}
//
//		outc <- payload
//	}
//
//	close(outc)
//	wg.Done()
//}
//
//func (m *Upper) SetFlagSet(fs *pflag.FlagSet, args []string) {}
