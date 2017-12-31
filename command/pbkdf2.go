package command

import (
  "io"
  "bytes"
  "io/ioutil"
  "github.com/pkg/errors"
  "flag"
  "../flags"
  "hash"
  "crypto/sha1"
  "crypto/sha256"
  "crypto/sha512"
  "golang.org/x/crypto/pbkdf2"
  "../inout"
  cryptoRand "crypto/rand"
)

type Pbkdf2 struct {
  name string
  description string
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  usage *flags.Usage
  hash func() hash.Hash
  flagSet *flag.FlagSet
  options *Pbkdf2Options
  salt []byte
  sync chan error
}

type Pbkdf2Options struct {
  saltIn string
  iter uint
  saltLen uint
  keyLen uint
  noOutputSalt bool
}

var DefaultPbkdf2 = &Pbkdf2{
  name: "pbkdf2",
  description: "Derive a key from input using the PBKDF2 algorithm",
  usage: &flags.Usage{
    CommandLine: "[[-salt-in <filetype>} [no-output-salt]] | -salt-length <lenght>] [-key-lenght <length>] [-rounds <rounds>] [hash algorithm]",
    Other: "Hash Algorithms:\n  sha1\n  sha256\n  sha512: default",
  },
  sync: make(chan error),
}

func (command *Pbkdf2) Init() (error) {
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

    dk := pbkdf2.Key(data, command.salt, iter, keyLen, command.hash)
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

func (command Pbkdf2) Usage() (*flags.Usage) {
  return command.usage
}

func (command Pbkdf2) Name() (string) {
  return command.name
}

func (command Pbkdf2) Description() (string) {
  return command.description
}

func (command Pbkdf2) Read(p []byte) (int, error) {
  return command.pipeReader.Read(p)
}

func (command Pbkdf2) Write(data []byte) (int, error) {
  return command.pipeWriter.Write(data)
}

func (command Pbkdf2) Close() (error) {
  err := command.pipeWriter.Close()
  if err != nil {
    return err
  }

  return <- command.sync
}

func (command *Pbkdf2) SetupFlags(set *flag.FlagSet) {
  command.flagSet = set
  command.options = &Pbkdf2Options{}

  set.StringVar(&command.options.saltIn, "salt-in", "", "If provided read from <filetype> instead of generating a new one. Mutualy exclusive with salt-length")
  set.UintVar(&command.options.saltLen, "salt-length", 32, "Lenght of the generated salt in bytes. Mutualy exclusive with -salt")
  set.UintVar(&command.options.iter, "rounds", 32768, "Number of interation for pbkdf2. Cannot go lower than 8192")
  set.UintVar(&command.options.keyLen, "key-length", 32, "Lenght of the derivated key in bytes.")
  set.BoolVar(&command.options.noOutputSalt, "no-output-salt", false, "Don't output salt")
}

func (command *Pbkdf2) ParseFlags(options *flags.GlobalOptions) (error) {
  hashFunction := ""

  if command.flagSet.Parsed() {
    hashFunction = command.flagSet.Arg(0)
  }

  switch hashFunction {
    case "sha1":
      command.hash = sha1.New
    case "sha256":
      command.hash = sha256.New
    case "sha512":
      command.hash = sha512.New
    case "":
      command.hash = sha512.New
    default:
      return flags.ErrBadFlag
  }

  if command.options.iter < 8192 {
    return errors.New("Cannot have less than 8192 iterations")

  if command.options.noOutputSalt && command.options.saltIn == "" {
    return errors.New("You can only set -no-output-salt with -salt-in option")
  }

  return nil
}
