package main

import (
	"encoding/json"
	"time"
	"sync"
	"github.com/spf13/pflag"
	"log"
	"github.com/tehmoon/errors"
	"github.com/robertkrimen/otto"
	"github.com/robertkrimen/otto/ast"
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


	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		init := false
		mc := NewMessageChannel()

		out <- &Message{
			Type: MessageTypeChannel,
			Interface: mc.Callback,
		}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					if ! init {
						close(mc.Channel)
					}

					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					cb, ok := message.Interface.(MessageChannelFunc)
					if ok {
						if ! init {
							init = true
						} else {
							mc = NewMessageChannel()

							out <- &Message{
								Type: MessageTypeChannel,
								Interface: mc.Callback,
							}
						}

						wg.Add(1)
						go PwnHandler(m, cb, mc, js, wg)

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

type PwnPipeFlags struct {
	MaxConcurrentStreams int
	MultiStreams bool
}

func PwnHandler(m *Pwn, cb MessageChannelFunc, mc *MessageChannel, js *ast.Program, wg *sync.WaitGroup) {
	defer wg.Done()

	mc.Start(nil)
	metadata, inc := cb()
	outc := mc.Channel

	vm := otto.New()
	_, err := vm.Run(js)
	if err != nil {
		err = errors.Wrap(err, "Unexpected error running file-pipe javascript")
		log.Println(err.Error())
		close(outc)
		DrainChannel(inc, nil)
		return
	}

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

		flags := &PwnPipeFlags{
			MaxConcurrentStreams: 1,
			MultiStreams: false,
		}

		third, _ := call.Argument(2).Export()
		if third != nil {
			v, ok := third.(map[string]interface{})
			if ! ok {
				err = errors.Wrapf(err, "error casting argument to oject in %s\n", call.CallerLocation())
				log.Println(err.Error())
				return otto.UndefinedValue()
			}

			if arg, ok := v["max-concurrent-streams"]; ok {
				maxConcurrentStreams, ok := arg.(int64)
				if ! ok {
					err = errors.Wrapf(err, "error casting max-concurrent-streams to int in %s\n", call.CallerLocation())
					log.Println(err.Error())
					return otto.UndefinedValue()
				}

				flags.MaxConcurrentStreams = int(maxConcurrentStreams)

				if flags.MaxConcurrentStreams < 1 {
					err = errors.Errorf("Flag %q cannot be less than 1\n", "max-concurrent-streams", call.CallerLocation())
					log.Println(err.Error())
					return otto.UndefinedValue()
				}
			}

			if arg, ok := v["multi-streams"]; ok {
				flags.MultiStreams, ok = arg.(bool)
				if ! ok {
					err = errors.Wrapf(err, "error casting multi-streams to bool in %s\n", call.CallerLocation())
					log.Println(err.Error())
					return otto.UndefinedValue()
				}
			}
		}

		pin, pout, _, err := InitPipeline(pipe, &GlobalFlags{
			MaxConcurrentStreams: flags.MaxConcurrentStreams,
			MultiStreams: flags.MultiStreams,
		})
		if err != nil {
			err = errors.Wrapf(err, "Error init pipeline in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		wg := &sync.WaitGroup{}
		init := false

		pmc := NewMessageChannel()
		pout <- &Message{
			Type: MessageTypeChannel,
			Interface: pmc.Callback,
		}

		LOOP: for {
			select {
				case message, opened := <- pin:
					if ! opened {
						if ! init {
							close(pmc.Channel)
						}
						break LOOP
					}

					switch message.Type {
						case MessageTypeTerminate:
							if ! init {
								close(pmc.Channel)
							}
							wg.Wait()
							pout <- message
							break LOOP
						case MessageTypeChannel:
							pcb, ok := message.Interface.(MessageChannelFunc)
							if ok {
								if ! init {
									init = true
								} else {
									pmc = NewMessageChannel()
									pout <- &Message{
										Type: MessageTypeChannel,
										Interface: pmc.Callback,
									}
								}

								pmc.Start(nil)
								pmeta, pinc := pcb()
								poutc := pmc.Channel

								if callback != "undefined" {
									oldReader := reader
									reader = NewChannelReader(pinc)

									oldOutc := outc
									outc = poutc

									_, err := vm.Call(callback, nil, pmeta)
									if err != nil {
										err = errors.Wrap(err, "Error calling callback function")
										log.Println(err.Error())
									}

									close(outc)
									reader.Close()
									wg.Wait()
									outc = oldOutc
									reader = oldReader
									if ! flags.MultiStreams {
										pout <- &Message{Type: MessageTypeTerminate,}
										break LOOP
									}

									continue LOOP
								}
								wg.Add(2)
								go func(pinc, outc chan []byte, wg *sync.WaitGroup) {
									for payload := range pinc {
										outc <- payload
									}
									wg.Done()
								}(pinc, outc, wg)
								go func(reader *ChannelReader, poutc chan []byte, wg *sync.WaitGroup) {
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
								}(reader, pmc.Channel, wg)
								if ! flags.MultiStreams {
									wg.Wait()
									pout <- &Message{Type: MessageTypeTerminate,}
									break LOOP
								}
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
			err = errors.Wrapf(err, "Error casting argument %T to oject in %s\n", first, call.CallerLocation())
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
	go func(outc chan []byte, metadata map[string]interface{}, vm *otto.Otto, wg *sync.WaitGroup, cancel chan struct{}) {
		defer wg.Done()
		defer close(outc)
		defer close(cancel)

		_, err := vm.Call("start", nil, metadata)
		if err != nil {
			err = errors.Wrap(err, "Error calling start function")
			log.Println(err.Error())
			return
		}
	}(outc, metadata, vm, wg, cancel)
}

func NewPwn() (Module) {
	return &Pwn{}
}

func (m *Pwn) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.jsFilePipe, "file-pipe", "", "Read content of the file from a pipeline. IE: `\"read-file --path test.js\"`")
}
