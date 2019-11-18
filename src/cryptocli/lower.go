package main

//import (
//	"github.com/spf13/pflag"
//	"sync"
//)
//
//func init() {
//	MODULELIST.Register("lower", "Lowercase all ascii characters", NewLower)
//}
//
//type Lower struct {}
//
//func (m Lower) Init(in, out chan *Message, global *GlobalFlags) (error) {
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
//						go startLower(inc, outc, wg)
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
//func NewLower() (Module) {
//	return &Lower{}
//}
//
//func startLower(inc, outc MessageChannel, wg *sync.WaitGroup) {
//	for payload := range inc {
//		for i, b := range payload {
//			if b > 64 && b < 91 {
//				payload[i] = b + 32
//			}
//		}
//		outc <- payload
//	}
//
//	close(outc)
//	wg.Done()
//}
//
//func (m *Lower) SetFlagSet(fs *pflag.FlagSet, args []string) {}
