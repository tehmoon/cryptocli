package main

import (
	"encoding/json"
	"time"
	"sync"
	"github.com/spf13/pflag"
	"log"
	"github.com/tehmoon/errors"
	"github.com/robertkrimen/otto"
	"io"
	"regexp"
	jsParser "github.com/robertkrimen/otto/parser"
)

func init() {
	MODULELIST.Register("pwn", "Start a javascript VM to control input/output", NewPwn)
}

type Pwn struct {
	jsFilePipe string
}

func (m *Pwn) Init(in, out chan *Message, global *GlobalFlags) (error) {
	content, err := ReadAllPipeline(m.jsFilePipe)
	if err != nil {
		return errors.Wrapf(err, "Error reading the javascript content from %q flag", "file-pipe")
	}

	js, err := jsParser.ParseFile(nil, "", content, 0)
	if err != nil {
		return errors.Wrap(err, "Error compiling javascript")
	}


	init := false
	outc := make(MessageChannel)

	out <- &Message{
		Type: MessageTypeChannel,
		Interface: outc,
	}

	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					if ! init {
						close(outc)
					}
					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					inc, ok := message.Interface.(MessageChannel)
					if ok {
						if ! init {
							init = true
						} else {
							outc = make(MessageChannel)

							out <- &Message{
								Type: MessageTypeChannel,
								Interface: outc,
							}
						}

						vm := otto.New()
						_, err = vm.Run(js)
						if err != nil {
							err = errors.Wrap(err, "Unexpected error running file-pipe javascript")
							log.Println(err.Error())
							if ! global.MultiStreams {
								out <- &Message{Type: MessageTypeTerminate,}
								break LOOP
							}
						}

						wg.Add(1)
						go PwnHandler(m, inc, outc, vm, wg)

						if ! global.MultiStreams {
							wg.Wait()
							out <- &Message{Type: MessageTypeTerminate,}
							break LOOP
						}
					}
			}
		}

		wg.Wait()
		// Last message will signal the closing of the channel
		<- in
		close(out)
	}(in, out)

	return nil
}

func PwnHandler(m *Pwn, inc, outc MessageChannel, vm *otto.Otto, wg *sync.WaitGroup) {
	defer wg.Done()

	cancel := make(chan struct{})
	reader := NewChannelReader(inc)

	vm.Set("log", func(call otto.FunctionCall) otto.Value {
		first, err := call.Argument(0).ToString()
		if err != nil {
			err = errors.Wrapf(err, "Error casting first argument to string in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		log.Printf("%s: %s\n", call.CallerLocation(), first)

		return otto.UndefinedValue()
	})

	vm.Set("regexp", func(call otto.FunctionCall) otto.Value {
		str, err := call.Argument(0).ToString()
		if err != nil {
			err = errors.Wrapf(err, "Error casting first argument to string in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		src, err := call.Argument(1).ToString()
		if err != nil {
			err = errors.Wrapf(err, "Error casting second argument to string in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		repl, err := call.Argument(2).ToString()
		if err != nil {
			err = errors.Wrapf(err, "Error casting third argument to string in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		re, err := regexp.Compile(str)
		if err != nil {
			err = errors.Wrapf(err, "Error compiling regexp %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		val, _ := otto.ToValue(re.ReplaceAllString(src, repl))
		return val
	})

	vm.Set("readFromPipe", func(call otto.FunctionCall) otto.Value {
		pipe, err := call.Argument(0).ToString()
		if err != nil {
			err = errors.Wrapf(err, "Error casting first argument to string in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		data, err := ReadAllPipeline(pipe)
		if err != nil {
			err = errors.Wrap(err, "Error reading from pipeline")
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		val, _ := otto.ToValue(string(data[:]))
		return val
	})

	vm.Set("writeToPipe", func(call otto.FunctionCall) otto.Value {
		pipe, err := call.Argument(0).ToString()
		if err != nil {
			err = errors.Wrapf(err, "Error casting first argument to string in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		data, err := call.Argument(1).ToString()
		if err != nil {
			err = errors.Wrapf(err, "Error casting second argument to string in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		err = WriteToPipeline(pipe, []byte(data))
		if err != nil {
			err = errors.Wrap(err, "Error writing to pipeline")
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		return otto.UndefinedValue()
	})

	vm.Set("pipe", func(call otto.FunctionCall) otto.Value {
		pipe, err := call.Argument(0).ToString()
		if err != nil {
			err = errors.Wrapf(err, "Error casting first argument to string in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		callback, _ := call.Argument(1).ToString()

		pin, pout, _, err := InitPipeline(pipe, &GlobalFlags{
			MaxConcurrentStreams: 1,
		})
		if err != nil {
			err = errors.Wrapf(err, "Error init pipeline in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		wg := &sync.WaitGroup{}

		poutc := make(MessageChannel)
		pout <- &Message{
			Type: MessageTypeChannel,
			Interface: poutc,
		}

		LOOP: for {
			select {
				case message, opened := <- pin:
					if ! opened {
						break LOOP
					}

					switch message.Type {
						case MessageTypeTerminate:
							wg.Wait()
							pout <- message
							break LOOP
						case MessageTypeChannel:
							pinc, ok := message.Interface.(MessageChannel)
							if ok {
								if callback != "undefined" {
									oldReader := reader
									reader = NewChannelReader(pinc)

									oldOutc := outc
									outc = poutc

									_, err := vm.Call(callback, nil)
									if err != nil {
										err = errors.Wrap(err, "Error calling callback function")
										log.Println(err.Error())
									}

									close(outc)
									reader.Close()
									wg.Wait()
									pout <- &Message{Type: MessageTypeTerminate,}
									outc = oldOutc
									reader = oldReader
									break LOOP
								}
								wg.Add(2)
								go func(pinc, outc MessageChannel, wg *sync.WaitGroup) {
									for payload := range pinc {
										outc <- payload
									}
									wg.Done()
								}(pinc, outc, wg)
								go func(reader *ChannelReader, poutc MessageChannel, wg *sync.WaitGroup) {
									defer wg.Done()
									defer close(poutc)
									for {
										message, err := reader.ReadMessage()
										if err != nil {
											if err == io.EOF {
												return
											}
											err = errors.Wrap(err, "Error reading from channel message")
											log.Println(err.Error())
											return
										}
										poutc <- message
									}
								}(reader, poutc, wg)
								wg.Wait()
								pout <- &Message{Type: MessageTypeTerminate,}
								break LOOP
							}
					}
			}
		}

		<- pin
		close(pout)

		return otto.UndefinedValue()
	})

	vm.Set("fromJSON", func(call otto.FunctionCall) otto.Value {
		first, err := call.Argument(0).ToString()
		if err != nil {
			err = errors.Wrapf(err, "Error casting first argument to string in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		var v interface{}
		err = json.Unmarshal([]byte(first), &v)
		if err != nil {
			err = errors.Wrapf(err, "Error unmarshaling string from json in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		val, err := vm.ToValue(v)
		if err != nil {
			err = errors.Wrapf(err, "Error casting value to otto.Value in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		return val
	})

	vm.Set("toJSON", func(call otto.FunctionCall) otto.Value {
		first, err := call.Argument(0).Export()
		if err != nil {
			err = errors.Wrapf(err, "Error exporting argument in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		object, ok := first.(map[string]interface{})
		if ! ok {
			err = errors.Wrapf(err, "Error casting argument to oject in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		payload, err := json.Marshal(object)
		if err != nil {
			err = errors.Wrapf(err, "Error marshaling object to json in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		val, _ := otto.ToValue(string(payload[:]))

		return val
	})

	vm.Set("sleep", func(call otto.FunctionCall) otto.Value {
		first, err := call.Argument(0).ToInteger()
		if err != nil {
			err = errors.Wrapf(err, "Error casting first argument to integer in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		if first < 0 {
			err = errors.New("Sleep time cannot be negative")
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		time.Sleep(time.Duration(first) * time.Second)

		return otto.UndefinedValue()
	})

	vm.Set("write", func(call otto.FunctionCall) otto.Value {
		first, err := call.Argument(0).ToString()
		if err != nil {
			err = errors.Wrapf(err, "Error casting first argument to integer in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		outc <- []byte(first)

		return otto.TrueValue()
	})

	vm.Set("readMessage", func(call otto.FunctionCall) otto.Value {
		message, err := reader.ReadMessage()
		if err != nil {
				err = errors.Wrapf(err, "Error reading from pipe in %s\n", call.CallerLocation())
				log.Println(err.Error())
				return otto.UndefinedValue()
		}

		val, _ := otto.ToValue(string(message[:]))

		return val
	})

	vm.Set("readline", func(call otto.FunctionCall) otto.Value {
		line, err := reader.ReadLine()
		if err != nil {
				err = errors.Wrapf(err, "Error reading from pipe in %s\n", call.CallerLocation())
				log.Println(err.Error())
				return otto.UndefinedValue()
		}

		val, _ := otto.ToValue(string(line[:]))

		return val
	})

	vm.Set("read", func(call otto.FunctionCall) otto.Value {
		first, err := call.Argument(0).ToInteger()
		if err != nil {
			err = errors.Wrapf(err, "Error casting first argument to integer in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		payload := make([]byte, first)
		n, err := reader.Read(payload)
		if err != nil && err != io.EOF {
			err = errors.Wrapf(err, "Error reading from pipe in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		val, _ := otto.ToValue(string(payload[:n]))

		return val
	})

	wg.Add(1)
	go func(outc MessageChannel, vm *otto.Otto, wg *sync.WaitGroup, cancel chan struct{}) {
		defer wg.Done()
		defer close(outc)
		defer close(cancel)

		_, err := vm.Call("start", nil)
		if err != nil {
			err = errors.Wrap(err, "Error calling start function")
			log.Println(err.Error())
			return
		}
	}(outc, vm, wg, cancel)
}

func NewPwn() (Module) {
	return &Pwn{}
}

func (m *Pwn) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.jsFilePipe, "file-pipe", "", "Read content of the file from a pipeline. IE: `\"read-file --path test.js\"`")
}
