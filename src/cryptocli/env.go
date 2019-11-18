package main

//import (
//	"sync"
//	"github.com/spf13/pflag"
//	"github.com/tehmoon/errors"
//	"os"
//)
//
//func init() {
//	MODULELIST.Register("env", "Read an environment variable", NewEnv)
//}
//
//type Env struct {
//	v string
//}
//
//func (m Env) Init(in, out chan *Message, global *GlobalFlags) (error) {
//	if m.v == "" {
//		return errors.Errorf("Flag %q must be specified in module init", "var")
//	}
//
//	go func() {
//		outc := make(MessageChannel)
//		out <- &Message{
//			Type: MessageTypeChannel,
//			Interface: outc,
//		}
//
//		wg := &sync.WaitGroup{}
//
//		message, opened := <- in
//		if ! opened {
//			close(outc)
//			<- in
//			close(out)
//		}
//
//		switch message.Type {
//			case MessageTypeTerminate:
//				close(outc)
//				out <- message
//			case MessageTypeChannel:
//				inc, ok := message.Interface.(MessageChannel)
//
//				if ok {
//					wg.Add(1)
//					go DrainChannel(inc, wg)
//				}
//
//				outc <- []byte(os.Getenv(m.v))
//
//				close(outc)
//				out <- &Message{
//					Type: MessageTypeTerminate,
//				}
//		}
//
//		wg.Wait()
//		<- in
//		close(out)
//	}()
//
//	return nil
//}
//
//func NewEnv() (Module) {
//	return &Env{}
//}
//
//func (m *Env) SetFlagSet(fs *pflag.FlagSet, args []string) {
//	fs.StringVar(&m.v, "var", "", "Variable to read from")
//}
