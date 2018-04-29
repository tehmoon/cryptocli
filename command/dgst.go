package command

import (
	"io"
	"crypto/subtle"
	"flag"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"io/ioutil"
	"../flags"
	"github.com/tehmoon/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
	"../inout"
)

type Dgst struct {
	name string
	description string
	hash hash.Hash
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	flagSet *flag.FlagSet
	usage *flags.Usage
	checksum []byte
	options DgstOptions
}

type DgstOptions struct {
	checksumIn string
}

var DefaultDgst = &Dgst{
	name: "dgst",
	description: "Hash the content of stdin",
	usage: &flags.Usage{
		CommandLine: "[-checksum-in <filetype>] <hash algorithm>",
		Other: "Hash Algorithms:\n	sha1\n	sha256\n	sha512\n	blake2b-256\n  blake2b-384\n	blake2b-512\n  sha3-224\n  sha3-256\n  sha3-384\n  sha3-512",
	},
	options: DgstOptions{},
}

func (command *Dgst) Init() (error) {
	command.pipeReader, command.pipeWriter = io.Pipe()

	if command.options.checksumIn != "" {
		reader, err := inout.ParseInput(command.options.checksumIn)
		if err != nil {
			return errors.Wrap(err, "Error parsing filtype for -checksum-in")
		}

		err = reader.Init()
		if err != nil {
			return errors.Wrap(err, "Error initializing filetype for -checksum-in")
		}

		command.checksum, err = ioutil.ReadAll(reader)
		if err != nil {
			return errors.Wrap(err, "Error reading filetype for -checksum-in")
		}

		err = reader.Close()
		if err != nil {
			return errors.Wrap(err, "Erro closing filetype for -checksum-in")
		}
	}

	return nil
}

func (command Dgst) Usage() (*flags.Usage) {
	return command.usage
}

func (command Dgst) Name() (string) {
	return command.name
}

func (command Dgst) Description() (string) {
	return command.description
}

func (command Dgst) Read(p []byte) (int, error) {
	return command.pipeReader.Read(p)
}

func (command Dgst) Write(data []byte) (int, error) {
	return command.hash.Write(data)
}

func (command Dgst) Close() (error) {
	sum := command.hash.Sum(nil)

	if command.checksum != nil {
		ok := subtle.ConstantTimeCompare(sum, command.checksum)
		if ok != 1 {
			return errors.New("Checksums don't match!! ABORTING")
		}

		return command.pipeWriter.Close()
	}

	_, err := command.pipeWriter.Write(sum)
	if err != nil {
		return err
	}

	return command.pipeWriter.Close()
}

func (command *Dgst) SetupFlags(set *flag.FlagSet) {
	command.flagSet = set

	command.flagSet.StringVar(&command.options.checksumIn, "checksum-in", "", "Checksum to verify against. Outputs an error if doesn't match. Doesn't output checksum.")
}

func (command *Dgst) ParseFlags(options *flags.GlobalOptions) (error) {
	hashFunction := ""

	if command.flagSet.Parsed() {
		hashFunction = command.flagSet.Arg(0)

		if command.options.checksumIn != "" {
			options.Chomp = true
		}
	}

	switch hashFunction {
		case "sha1":
			command.hash = sha1.New()
		case "sha256":
			command.hash = sha256.New()
		case "sha512":
			command.hash = sha512.New()
		case "blake2b-256":
			command.hash, _  = blake2b.New(blake2b.Size256, nil)
		case "blake2b-384":
			command.hash, _ = blake2b.New(blake2b.Size384, nil)
		case "blake2b-512":
			command.hash, _ = blake2b.New(blake2b.Size, nil)
		case "sha3-224":
			command.hash = sha3.New224()
		case "sha3-256":
			command.hash = sha3.New256()
		case "sha3-384":
			command.hash = sha3.New384()
		case "sha3-512":
			command.hash = sha3.New512()
		case "":
			command.hash = sha512.New()
		default:
			return flags.ErrBadFlag
	}

	return nil
}
