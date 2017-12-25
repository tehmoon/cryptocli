package command

import (
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
}

var DefaultScrypt = &Scrypt{
  name: "scrypt",
  description: "Derive a key from input using the scrypt algorithm",
  usage: &flags.Usage{
    CommandLine: "[-salt-in <filetype> | -salt-length <lenght>] [-key-lenght <length>] [-rounds <rounds>]",
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
    reader , err := inout.ParseInput(command.options.saltIn)
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
  } else {
    command.salt = make([]byte, command.options.saltLen)
    read, err := io.ReadFull(cryptoRand.Reader, command.salt)
    if err != nil {
      return errors.Wrap(err, "Error generating salt")
    }

    if read != len(command.salt) {
      return errors.New("Error getting salt fully generating")
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

    buff.Write(command.salt)
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
  set.UintVar(&command.options.saltLen, "salt-length", 32, "Lenght of the generated salt in bytes. Mutualy exclusive with -salt")
  set.UintVar(&command.options.iter, "rounds", 32768, "Number of interation for scrypt. Cannot go lower than 16384")
  set.UintVar(&command.options.keyLen, "key-length", 32, "Lenght of the derivated key in bytes.")
}

func (command *Scrypt) ParseFlags(options *flags.GlobalOptions) (error) {
  if command.options.iter < 16384 {
    return errors.New("Cannot have less than 16384 iterations")
  }

  return nil
}
