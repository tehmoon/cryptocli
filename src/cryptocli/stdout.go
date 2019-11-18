package main

//import (
//	"sync"
//	"os"
//	"github.com/tehmoon/errors"
//	"github.com/spf13/pflag"
//)
//
//var stdoutMutex = struct{sync.Mutex; Init bool}{Init: false,}
//
//func init() {
//	MODULELIST.Register("stdout", "Writes to stdout", NewStdout)
//}
//
//type Stdout struct {}
//
//func (m Stdout) Init(in, out chan *Message, global *GlobalFlags) (error) {
//	stdoutMutex.Lock()
//	defer stdoutMutex.Unlock()
//	defer func() {
//		stdoutMutex.Init = false
//	}()
//
//	if stdoutMutex.Init {
//		return errors.New("Module \"stdout\" cannot be added more than once")
//	}
//
//	stdoutMutex.Init = true
//
//	go func(in, out chan *Message) {
//		wg := &sync.WaitGroup{}
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
//						wg.Add(1)
//						go func(inc, outc MessageChannel, wg *sync.WaitGroup) {
//							for payload := range inc {
//								os.Stdout.Write(payload)
//								os.Stdout.Sync()
//							}
//
//							close(outc)
//							wg.Done()
//						}(inc, outc, wg)
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
//func NewStdout() (Module) {
//	return &Stdout{}
//}
//
//func (m Stdout) SetFlagSet(fs *pflag.FlagSet, args []string) {}
