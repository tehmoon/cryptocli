package command

// https://www.imperialviolet.org/2014/06/27/streamingencryption.html
// https://stackoverflow.com/questions/39347206/how-to-encrypt-files-with-aes256-gcm-in-golang
// https://tools.ietf.org/html/rfc5116#section-3.2

import (
  "io"
  "flag"
  "../flags"
  "github.com/tehmoon/errors"
  "encoding/binary"
  "fmt"
  cryptoRand "crypto/rand"
)


var (
  DefaultAesGcmEncrypt = &AesGcmEncrypt{
    name: "aes-gcm-encrypt",
    usage: &flags.Usage{
      CommandLine: "<-derived-salt-key-in <filetype> | -password-in <filetype>> [-salt-length int] [-block-size int] [key-length]",
      Other: "Key Length:\n  128: Use 128 bits key security\n  256: Use 256 bits key security",
    },
    description: "Encrypt and authenticate <-block-size>'s blocks of data using AES algorithm with GCM mode. Padding is not necessary so if EOF is reached, it will return less. Nonce are 8 random bytes followed byte 4 bytes which starts by 0 and are incremented. So it outputs the following <-salt-length> || nonce[:8] || (tag || encrypted data)... . To Decrypt you must read <-salt-length> if you need to derive the key, then reconstruct the nonce by taking the following 8 bytes and appending 0x00000000, then the following <-block-size> can be decrypted with GCM. For each <-block-size>'s block until EOF -- no padding -- don't forget to reuse the first 8 bytes of the nonce and increment the last 4 bytes ONLY.",
    options: &AesGcmEncryptOptions{
      NonceSize: 12,
    },
  }
)

type AesGcmEncrypt struct {
  name string
  flagSet *flag.FlagSet
  description string
  usage *flags.Usage
  reader *io.PipeReader
  writer *io.PipeWriter
  encReader *io.PipeReader
  encWriter *io.PipeWriter
  counter uint32
  sync chan error
  options *AesGcmEncryptOptions
}

type AesGcmEncryptOptions struct {
  BlockSize uint
  NonceSize int
  KeySize int
  SaltLen int
  DerivedSaltKeyIn string
  PasswordIn string
}

func (command *AesGcmEncrypt) Init() (error) {
  command.reader, command.writer = io.Pipe()
  command.encReader, command.encWriter = io.Pipe()
  command.sync = make(chan error)
  salt := make([]byte, command.options.SaltLen)
  nonce := make([]byte, command.options.NonceSize)
  key := make([]byte, command.options.KeySize)

  if command.options.DerivedSaltKeyIn != "" {
    err := ReadSaltKeyFromFiletype(salt, nil, key, command.options.DerivedSaltKeyIn)
    if err != nil {
      return errors.Wrap(err, "Error in -derived-salt-key-in")
    }
  }

  if command.options.PasswordIn != "" {
    err := DeriveKey(salt, nil, key, command.options.PasswordIn)
    if err != nil {
      return errors.Wrap(err, "Error in -password-in")
    }
  }

  _, err := io.ReadFull(cryptoRand.Reader, nonce[:8])
  if err != nil {
    return errors.Wrap(err, "Error generating nonce")
  }

  aesgcm, err := CreateAEAD(key)
  if err != nil {
    return err
  }

  go func() {
    var (
      close bool
      started bool
      err error
    )

    for {
      counter := nonce[8:]
      binary.BigEndian.PutUint32(counter, command.counter)
      buff := make([]byte, command.options.BlockSize)

      buffRead, err := io.ReadFull(command.reader, buff)
      if err != nil {
        close = true
        if buffRead == 0 {
          break
        }
      }

      if ! started {
        _, err = command.encWriter.Write(salt)
        if err != nil {
          err = errors.Wrap(err, "Failed to write nonce to out")
          break
        }

        _, err = command.encWriter.Write(nonce[:8])
        if err != nil {
          err = errors.Wrap(err, "Failed to write nonce to out")
          break
        }

        started = true
      }

      ciphertext := aesgcm.Seal(nil, nonce, buff[:buffRead], nil)
      _, err = command.encWriter.Write(ciphertext)
      if err != nil {
        err = errors.Wrap(err, "Failed to seal gcm block")
        break
      }

      // counter is going to overflow so we need a new nonce
      // Once the nonce has been written, let the counter overflows
      // so it starts at 0 again
      if command.counter + 1 == 0 {
        _, err = io.ReadFull(cryptoRand.Reader, nonce[:8])
        if err != nil {
          err = errors.Wrap(err, "Failed to generate new nonce")
          break
        }

        _, err = command.encWriter.Write(nonce[:8])
        if err != nil {
          err = errors.Wrap(err, "Error writing new nonce to writer")
          break
        }
      }

      command.counter++

      if close {
        break
      }
    }

    command.encWriter.Close()
    command.sync <- err
  }()

  return nil
}

func (command AesGcmEncrypt) Usage() (*flags.Usage) {
  return command.usage
}

func (command AesGcmEncrypt) Name() (string) {
  return command.name
}

func (command AesGcmEncrypt) Description() (string) {
  return command.description
}

func (command AesGcmEncrypt) Read(p []byte) (int, error) {
  return command.encReader.Read(p)
}

func (command AesGcmEncrypt) Write(data []byte) (int, error) {
  return command.writer.Write(data)
}

func (command AesGcmEncrypt) Close() (error) {
  command.writer.Close()
  err := <- command.sync
  if err == io.EOF {
    return nil
  }

  return err
}

func (command *AesGcmEncrypt) SetupFlags(set *flag.FlagSet) {
  command.flagSet = set

  set.IntVar(&command.options.SaltLen, "salt-length", 32, "Byte to read from -derived-salt-key-in")
  set.StringVar(&command.options.DerivedSaltKeyIn, "derived-salt-key-in", "", fmt.Sprintf("If specified, read the number of bytes from -salt-length for salt and the remaining %d for the key. Will if the key is too long or too short. Cannot be used with -password", command.options.KeySize))
  set.StringVar(&command.options.PasswordIn, "password-in", "", fmt.Sprintf("Derive a key from <filetype> using scrypt with %d rounds and a %d bytes salt. Alternatively you could use -derived-salt-key-in for more control.", 1<<20, 32))
  set.UintVar(&command.options.BlockSize, "block-size", 1<<14, "Encrypt and authenticate blocks of -block-size's length")
}

func (command *AesGcmEncrypt) ParseFlags(options *flags.GlobalOptions) (error) {
  if (command.options.DerivedSaltKeyIn == "" && command.options.PasswordIn == "") ||
     (command.options.DerivedSaltKeyIn != "" && command.options.PasswordIn != "") {
    return errors.New("One of -derived-salt-in and -password-in must be used")
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
