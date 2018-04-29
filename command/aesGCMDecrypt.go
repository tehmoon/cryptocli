package command

// https://www.imperialviolet.org/2014/06/27/streamingencryption.html
// https://stackoverflow.com/questions/39347206/how-to-encrypt-files-with-aes256-gcm-in-golang
// https://tools.ietf.org/html/rfc5116#section-3.2

import (
	"io"
	"flag"
	"../flags"
	"github.com/tehmoon/errors"
	"fmt"
	"crypto/cipher"
	"encoding/binary"
)

var (
	DefaultAesGcmDecrypt = &AesGcmDecrypt{
		name: "aes-gcm-decrypt",
		usage: &flags.Usage{
			CommandLine: "<-direved-salt-key-in <filetype> | -password-in <filetype>> [-salt-length int] [-block-size int] [key-lenth]",
			Other: "Key Length:\n  128: Use 128 bits key security\n  256: Use 256 bits key security",
		},
		description: "Decrypt and verify authentication of <-block-size>'s block of data using AES algorithm with GCM mode. Nonce of 8 bytes is read after reading the salt to derived the key. Then we append 4 bytes number to the nonce every block starting at 0. Only the first 8 bytes of the nonce is reused. By default it uses scrypt to derive the key but if you want to use your own KDF, aes-gcm-decrypt  will read the salt up to -salt-length then set the environment variable SALT to the hex salt value so you can execute your KDF using the pipe: inout module. If you do that, the salt is expected to be found prepended to the key.",
		options: &AesGcmDecryptOptions{
			NonceSize: 12,
		},
	}
)

type AesGcmDecrypt struct {
	name string
	flagSet *flag.FlagSet
	description string
	usage *flags.Usage
	counter uint32
	sync chan error
	options *AesGcmDecryptOptions
	encReader *io.PipeReader
	encWriter *io.PipeWriter
	reader *io.PipeReader
	writer *io.PipeWriter
}

type AesGcmDecryptOptions struct {
	BlockSize uint
	NonceSize int
	KeySize int
	SaltLen int
	DerivedSaltKeyIn string
	PasswordIn string
}

func (command *AesGcmDecrypt) Init() (error) {
	command.reader, command.writer = io.Pipe()
	command.encReader, command.encWriter = io.Pipe()
	command.sync = make(chan error)

	go func() {
		var (
			salt []byte = make([]byte, command.options.SaltLen)
			nonce []byte = make([]byte, command.options.NonceSize)
			key []byte = make([]byte, command.options.KeySize)
			plaintext []byte
			read int
			counter []byte = nonce[8:]
			err error
			aesgcm cipher.AEAD
		)

		defer func() {
			command.writer.CloseWithError(err)
			command.sync <- err
		}()

		_, err = command.encReader.Read(salt)
		if err != nil {
			err = errors.Wrapf(err, "Error reading salt of length %d", len(salt))
			return
		}

		if command.options.PasswordIn != "" {
			err = DeriveKey(salt, salt, key, command.options.PasswordIn)
			if err != nil {
				err = errors.Wrap(err, "Error to derive key from -password-in")
				return
			}
		}

		if command.options.DerivedSaltKeyIn != "" {
			err = ReadSaltKeyFromFiletype(salt, salt, key, command.options.DerivedSaltKeyIn)
			if err != nil {
				err = errors.Wrap(err, "Error to derive key from -derive-salt-key-in")
				return
			}
		}

		read, err = command.encReader.Read(nonce[:8])
		if err != nil {
			err = errors.Wrap(err, "Error in reading nonce")
			return
		}

		if read != 8 {
			err = errors.Errorf("Couldn't fully read 8 bytes nonce")
			return
		}

		aesgcm, err = CreateAEAD(key)
		if err != nil {
			return
		}

		buff := make([]byte, uint64(command.options.BlockSize) + uint64(aesgcm.Overhead()))

		LOOP: for {
			read, err = io.ReadFull(command.encReader, buff)
			if err != nil {
				switch err {
					case io.EOF:
						err = nil
						break LOOP
					case io.ErrUnexpectedEOF:
						err = nil
					default:
						err = errors.Wrap(err, "Error reading encrypted data")
						break LOOP
				}
			}

			binary.BigEndian.PutUint32(counter, command.counter)

			plaintext, err = aesgcm.Open(nil, nonce, buff[:read], nil)
			if err != nil {
				err = errors.Wrap(err, "Error decrypting data")
				break
			}

			_, err = command.writer.Write(plaintext)
			if err != nil {
				err = errors.Wrap(err, "Error writing plaintext data")
				break
			}

			// Same thing as for encryption, if counter is about to overflow
			// we read the next 8 bytes which are the new nonce and let
			// the counter overflow so it resets to 0 naturally
			if command.counter + 1 == 0 {
				read, err = command.encReader.Read(nonce[:8])
				if err != nil {
					err = errors.Wrap(err, "Error in reading new nonce")
					return
				}

				if read != 8 {
					err = errors.Errorf("Couldn't fully read 8 bytes nonce")
					return
				}
			}

			command.counter++
		}
	}()

	return nil
}

func (command AesGcmDecrypt) Read(p []byte) (int, error) {
	return command.reader.Read(p)
}

func (command AesGcmDecrypt) Write(data []byte) (int, error) {
	return command.encWriter.Write(data)
}

func (command AesGcmDecrypt) Description() (string) {
	return command.description
}

func (command AesGcmDecrypt) Name() (string) {
	return command.name
}

func (command AesGcmDecrypt) Usage() (*flags.Usage) {
	return command.usage
}

func (command AesGcmDecrypt) Close() (error) {
	command.encWriter.Close()

	return <- command.sync
}

func (command *AesGcmDecrypt) SetupFlags(set *flag.FlagSet) {
	command.flagSet = set

	set.IntVar(&command.options.SaltLen, "salt-length", 32, "Byte to read from -derived-salt-key-in")
	set.StringVar(&command.options.DerivedSaltKeyIn, "derived-salt-key-in", "", fmt.Sprintf("If specified, read the number of bytes from -salt-length for salt and the remaining %d for the key. Will if the key is too long or too short. Cannot be used with -password", command.options.KeySize))
	set.StringVar(&command.options.PasswordIn, "password-in", "", fmt.Sprintf("Derive a key from <filetype> using scrypt with %d rounds and a %d bytes salt. Alternatively you could use -derived-salt-key-in for more control.", 1<<20, 32))
	set.UintVar(&command.options.BlockSize, "block-size", 1<<14, "Encrypt and authenticate blocks of -block-size's length")
}

func (command *AesGcmDecrypt) ParseFlags(options *flags.GlobalOptions) (error) {
	if (command.options.DerivedSaltKeyIn == "" && command.options.PasswordIn == "") ||
		 (command.options.DerivedSaltKeyIn != "" && command.options.PasswordIn != "") {
		return errors.New("One of -derived-salt-key in and -password-in must be used")
	}

	if command.options.SaltLen > 64 {
		return errors.New("Option -salt-length of more than 64 bytes is weird, but feel free to reach out to have it increased")
	}

	if command.options.BlockSize > 1<<32 || command.options.BlockSize == 0 {
		return errors.Errorf("Option -block-size must be between %d and %d", 1, 1<<32)
	}

	switch keySize := command.flagSet.Arg(0); keySize {
		case "":
			command.options.KeySize = 32
		case "128":
			command.options.KeySize = 16
		case "256":
			command.options.KeySize = 32
		default:
			return errors.Errorf("Key size is of %s is not supported", keySize)
	}

	return nil
}
