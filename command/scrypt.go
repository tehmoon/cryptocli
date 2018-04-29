package command

import (
	"fmt"
	"io"
	"bytes"
	"io/ioutil"
	"github.com/pkg/errors"
	"flag"
	"../flags"
	"golang.org/x/crypto/scrypt"
	"../inout"
	cryptoRand "crypto/rand"
)

type Scrypt struct {
	name string
	description string
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	usage *flags.Usage
	flagSet *flag.FlagSet
	options *ScryptOptions
	salt []byte
	sync chan error
}

type ScryptOptions struct {
	saltIn string
	iter uint
	saltLen uint
	keyLen uint
	noOutputSalt bool
}

var DefaultScrypt = &Scrypt{
	name: "scrypt",
	description: "Derive a key from input using the scrypt algorithm",
	usage: &flags.Usage{
		CommandLine: "[[-salt-in <filetype>] [ -no-output-salt]] | -salt-length <length>] [-key-lenght <length>] [-rounds <rounds>]",
		Other: "",
	},
	sync: make(chan error),
}

func (command *Scrypt) Init() (error) {
	var (
		reader io.Reader
		writer io.WriteCloser
	)

	if command.options.saltIn != "" {
		reader, err := inout.ParseInput(command.options.saltIn)
		if err != nil {
			return errors.Wrap(err, "Error parsing filetype for -salt-in")
		}

		err = reader.Init()
		if err != nil {
			return errors.Wrap(err, "Error intializing filtype for -salt-in")
		}

		command.salt, err = ioutil.ReadAll(reader)
		if err != nil {
			return errors.Wrap(err, "Error reading filetype for -salt-in")
		}

		err = reader.Close()
		if err != nil {
			return errors.Wrap(err, "Error closing filetype for -salt-in")
		}
	} else {
		command.salt = make([]byte, command.options.saltLen)
		read, err := io.ReadFull(cryptoRand.Reader, command.salt)
		if err != nil {
			return errors.Wrap(err, "Error generating salt")
		}

		if read != len(command.salt) {
			return errors.New("Error generated salt doens't match length")
		}
	}

	reader, command.pipeWriter = io.Pipe()
	command.pipeReader, writer = io.Pipe()

	go func() {
		data, err := ioutil.ReadAll(reader)
		if err != nil {
			command.sync <- errors.Wrap(err, "Error reading all input")
			return
		}

		iter := int(command.options.iter)
		keyLen := int(command.options.keyLen)

		dk, err := scrypt.Key(data, command.salt, iter, 8, 1, keyLen)
		if err != nil {
			command.sync <- errors.Wrap(err, "Error generating key")
			return
		}
		buff := new(bytes.Buffer)

		if ! command.options.noOutputSalt {
			buff.Write(command.salt)
		}

		buff.Write(dk)

		io.Copy(writer, buff)

		command.sync <- writer.Close()
	}()

	return nil
}

func (command Scrypt) Usage() (*flags.Usage) {
	return command.usage
}

func (command Scrypt) Name() (string) {
	return command.name
}

func (command Scrypt) Description() (string) {
	return command.description
}

func (command Scrypt) Read(p []byte) (int, error) {
	return command.pipeReader.Read(p)
}

func (command Scrypt) Write(data []byte) (int, error) {
	return command.pipeWriter.Write(data)
}

func (command Scrypt) Close() (error) {
	err := command.pipeWriter.Close()
	if err != nil {
		return err
	}

	return <- command.sync
}

func (command *Scrypt) SetupFlags(set *flag.FlagSet) {
	command.flagSet = set
	command.options = &ScryptOptions{}

	set.StringVar(&command.options.saltIn, "salt-in", "", "If provided use salt in hex format instead of generating a new one. Mutualy exclusive with salt-length")
	set.UintVar(&command.options.saltLen, "salt-length", 32, "Lenght of the generated salt in bytes. Mutualy exclusive with -salt-in")
	set.UintVar(&command.options.iter, "rounds", 1<<15, fmt.Sprintf("Number of interation for scrypt. Cannot go lower than %d", 1<<14))
	set.UintVar(&command.options.keyLen, "key-length", 32, "Lenght of the derivated key in bytes.")
	set.BoolVar(&command.options.noOutputSalt, "no-output-salt", false, "Don't output salt")
}

func (command *Scrypt) ParseFlags(options *flags.GlobalOptions) (error) {
	if command.options.iter < 1<<14{
		return errors.Errorf("Cannot have less than %d iterations\n", 1<<14)
	}

	if command.options.noOutputSalt && command.options.saltIn == "" {
		return errors.New("You can only set -no-output-salt with -salt-in option")
	}

	return nil
}
