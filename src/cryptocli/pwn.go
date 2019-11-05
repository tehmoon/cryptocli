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
	"bufio"
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
	pr, pw := io.Pipe()
	buffer := bufio.NewReader(pr)

	wg.Add(1)
	go func(inc MessageChannel, writer *io.PipeWriter, wg *sync.WaitGroup, cancel chan struct{}) {
		defer wg.Done()
		defer DrainChannel(inc, nil)

		LOOP: for {
			select {
				case <- cancel:
					break LOOP
				case payload, opened := <- inc:
					if ! opened {
						break LOOP
					}

					_, err := writer.Write(payload)
					if err != nil {
						err = errors.Wrap(err, "Error writing to pipe")
						log.Println(err.Error())
						break LOOP
					}
			}
		}

		writer.Close()
	}(inc, pw, wg, cancel)

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

	vm.Set("readline", func(call otto.FunctionCall) otto.Value {
		line := ""

		for {
			l, still, err := buffer.ReadLine()
			if err != nil && err != io.EOF {
				err = errors.Wrapf(err, "Error reading from pipe in %s\n", call.CallerLocation())
				log.Println(err.Error())
				return otto.UndefinedValue()
			}

			line += string(l)
			if ! still {
				break
			}
		}

		val, _ := otto.ToValue(line)

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
		i, err := io.ReadFull(buffer, payload)
		if err != nil {
			err = errors.Wrapf(err, "Error reading from pipe in %s\n", call.CallerLocation())
			log.Println(err.Error())
			return otto.UndefinedValue()
		}

		val, _ := otto.ToValue(string(payload[:i]))

		return val
	})

	wg.Add(1)
	go func(outc MessageChannel, vm *otto.Otto, reader *io.PipeReader, wg *sync.WaitGroup, cancel chan struct{}) {
		defer wg.Done()
		defer close(outc)
		defer pr.Close()
		defer close(cancel)

		_, err := vm.Call("start", nil)
		if err != nil {
			err = errors.Wrap(err, "Error calling start function")
			log.Println(err.Error())
			return
		}
	}(outc, vm, pr, wg, cancel)
}

func NewPwn() (Module) {
	return &Pwn{}
}

func (m *Pwn) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.jsFilePipe, "file-pipe", "", "Read content of the file from a pipeline. IE: `\"read-file --path test.js\"`")
}
