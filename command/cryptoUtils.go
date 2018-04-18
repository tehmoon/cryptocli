package command

import (
  "fmt"
  "github.com/tehmoon/errors"
  "io"
  "../inout"
  "io/ioutil"
  "bytes"
  "crypto/cipher"
  "crypto/aes"
)

// Read len(saltDst) from saltSrc, set to nil if you don't have a salt yet -- generating password
// saltDst and saltSrc must be the same if you read from saltSrc
func DeriveKey(saltDst, saltSrc, key []byte, filetype string) (error) {
  reader, err := inout.ParseInput(filetype)
  if err != nil {
    return errors.Wrap(err, "Error parsing filetype")
  }

  err = reader.Init()
  if err != nil {
    return errors.Wrap(err, "Error initializing filetype")
  }

  password := bytes.NewBuffer(nil)

  _, err = io.Copy(password, reader)
  if err != nil {
    return errors.Wrap(err, "Error reading filetype data")
  }

  err = reader.Close()
  if err != nil {
    return errors.Wrap(err, "Error closing filetype")
  }

  s := &Scrypt{
    sync: make(chan error),
    options: &ScryptOptions{
      keyLen: uint(len(key)),
      iter: 1<<20,
      saltLen: uint(len(saltDst)),
    },
  }

  if saltSrc != nil {
    s.options.saltIn = fmt.Sprintf("hex:%x", saltSrc)
  }

  err = s.Init()
  if err != nil {
    return errors.Wrap(err, "Error initializing -password filetype")
  }

  _, err = s.Write(password.Bytes())
  if err != nil {
    return errors.Wrap(err, "Error derivating a key from -password")
  }

  derivedKey := bytes.NewBuffer(nil)
  sync := make(chan error)

  go func() {
    _, err := io.Copy(derivedKey, s)
    sync <- err
  }()

  err = s.Close()
  if err != nil {
    return errors.Wrap(err, "Error closing the derived key function")
  }

  err = <- sync
  if err != nil {
    return errors.Wrap(err, "Error reading from the derived key function")
  }

  return ReadSaltKey(saltDst, key, derivedKey)
}

// Init filetype as inout.In then fill up saltDst and key buffers
// if saltSrc is not nil, saltDst must point to saltSrc and saltSrc is read.
// If filetype is pipe:, saltSrc is passed as SALT environment variable
// is set.
func ReadSaltKeyFromFiletype(saltDst, saltSrc, key []byte, filetype string) (error) {
  reader, err := inout.ParseInput(filetype)
  if err != nil {
    return errors.Wrap(err, "Error parsing filetype")
  }

  if saltSrc != nil {
    if pipeInput, ok := reader.(*inout.PipeInput); ok {
      pipeInput.Env = append(pipeInput.Env, "SALT=" + string(saltSrc))
    }
  }

  err = reader.Init()
  if err != nil {
    return errors.Wrap(err, "Error initializing filetype")
  }

  err = ReadSaltKey(saltDst, key, reader)
  if err != nil {
    return err
  }

  // Discard the rest of the reader and error if more had to be read
  i, err := io.Copy(ioutil.Discard, reader)
  if err != nil {
    return errors.Wrap(err, "Error draining filetype")
  }

  if i > 0 {
    return errors.New("Error after reading derived key you have remaining bytes to read")
  }

  err = reader.Close()
  if err != nil {
    return errors.Wrap(err, "Error closing filetype")
  }

  return nil
}

// Read from io.Reader and fill up salt and key buffers
func ReadSaltKey(salt, key []byte, reader io.Reader) (error) {
  _, err := io.ReadFull(reader, salt) /* will err if salt is not full */
  if err != nil {
    return errors.Wrap(err, "Error reading salt")
  }

  _, err = io.ReadFull(reader, key) /* fill up key after draining salt */
  if err != nil {
    return errors.Wrap(err, "Error reading key")
  }

  return nil
}

func CreateAEAD(key []byte) (cipher.AEAD, error) {
  block, err := aes.NewCipher(key)
  if err != nil {
    return nil, errors.Wrap(err, "Error creating block from key")
  }

  aesgcm, err := cipher.NewGCM(block)
  if err != nil {
    return nil, errors.Wrap(err, "Error creating AEAD from block")
  }

  return aesgcm, nil
}
