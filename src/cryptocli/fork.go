package main

import (
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	"log"
	"os/exec"
	"os"
	"io"
)

func init() {
	MODULELIST.Register("fork", "Start a program and attach stdin and stdout to the pipeline", NewFork)
}

type Fork struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
	fs *pflag.FlagSet
	cmd *exec.Cmd
	stdin io.WriteCloser
	stdout io.ReadCloser
}

func (m *Fork) SetFlagSet(fs *pflag.FlagSet) {
	fs.SetInterspersed(false)
	m.fs = fs
}

func (m *Fork) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Fork) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func (m *Fork) Init(global *GlobalFlags) (error) {
	args := SanetizeFlags(m.fs)

	if len(args) == 0 {
		return errors.New("No argument specified in fork module")
	}


	m.cmd = exec.Command(args[0], args[1:]...)
	m.cmd.Env = make([]string, 0)

	var err error
	m.stdin, m.stdout, err = forkPipeStd(m.cmd)
	if err != nil {
		return err
	}

	log.Printf("Executing %q with %v in fork module\n", args[0], args[1:])

	return nil
}

// Create os.Pipe() and attach them to the cmd.Stdin and cmd.Stdout
// Return the other side of the pipe in stdin(writer) and stdout(reader) order.
func forkPipeStd(cmd *exec.Cmd) (stdin io.WriteCloser, stdout io.ReadCloser, err error) {
	cmd.Stdin, stdin, err = os.Pipe()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error creating pipes for stdin in fork module")
	}

	stdout, cmd.Stdout, err = os.Pipe()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error creating pipes for stdout in fork module")
	}

	return stdin, stdout, nil
}

func (m Fork) Start() {
	m.sync.Add(1)

	go func() {
		err := ReadBytesSendMessages(m.stdout, m.out)
		if err != nil {
			log.Println(errors.Wrap(err, "Error reading in fork module"))
		}

		close(m.out)
	}()

	go func() {
		err := m.cmd.Run()
		if err != nil {
			log.Println(errors.Wrap(err, "Error waiting for command in fork module"))
		}

		m.stdout.Close()
		m.sync.Done()
	}()

	go func() {
		for message := range m.in {
			_, err := m.stdin.Write(message.Payload)
			if err != nil {
				log.Println(errors.Wrap(err, "Error writing in for module"))
				break
			}
		}

		m.stdin.Close()
	}()
}

func (m Fork) Wait() {
	m.sync.Wait()

	for range m.in {}
}

func NewFork() (Module) {
	return &Fork{
		sync: &sync.WaitGroup{},
	}
}
