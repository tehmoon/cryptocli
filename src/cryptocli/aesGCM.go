package main

import (
	"sync"
	"github.com/spf13/pflag"
	"github.com/tehmoon/errors"
	"golang.org/x/crypto/scrypt"
	"crypto/rand"
	"io"
	"log"
	"encoding/binary"
	"crypto/aes"
	"crypto/cipher"
)

func init() {
	MODULELIST.Register("aes-gcm", "AES-GCM encryption/decryption", NewAESGCM)
}

/*
aes-gcm module will encrypt/decrypt in a streaming fashion

format:
	12 bytes salt || 8 bytes nonce || 4 bytes length || x bytes data || ...

encryption:
	- A 12 bytes salt is generated and the key is derived using scrypt with 2^18 rounds
	- We generate an 8 bytes nonce using crypto/rand.
	- Then we write that nonce to the pipeline
	- A 4 bytes counter is started
	- The counter is appended to the nonce in little endian format
	- The length of the payload is saved in little endian format and written to the pipeline
	- Then we encrypt using the derived key and the 12 bytes nonce
	- The length of the payload is added as additional data to verify
	- The counter is incremented
	- If the counter reaches 2^32 - 1, a new 8 bytes salt is generated, written to the pipeline and the counter is reset to 0

decryption:
	- The salt is read from the pipeline
  - The key gets derived using that salt and scrypt with 2^18 rounds
	- We read the next 8 bytes to get the salt
  - A counter is started and gets appened to the salt
	- We then read the next 4 bytes which will be the length of the payload
	- We read up to those 4 bytes in little endian format
	- That data gets decrypted with the 4 bytes length as additional data
	- The decrypted data is written to the pipeline
	- The counter is incremented
	- If the counter reaches 2^32 - 1, we read the next 8 bytes which will be the new salt, then the counter gets reset to 0
*/

type AESGCM struct {
	in chan *Message
	out chan *Message
	wg *sync.WaitGroup
	password []byte
	key []byte
	flags AESGCMFlags
}

type AESGCMFlags struct {
	passwordIn string
	half bool
	full bool
	keyLen int
	encrypt bool
	decrypt bool
}

func (m *AESGCM) Init(in, out chan *Message, global *GlobalFlags) (err error) {
	if (! m.flags.decrypt && ! m.flags.encrypt) || (m.flags.decrypt && m.flags.encrypt) {
		return errors.Errorf("One of %q or %q is required in aes module", "encrypt", "decrypt")
	}

	if (! m.flags.half && ! m.flags.full) || (m.flags.half && m.flags.full) {
		return errors.Errorf("One of %q or %q ir required in aes module", "128", "256")
	}

	if m.flags.passwordIn == "" {
		return errors.Errorf("Flag %q cannot be empty in aes module", "password-pipe")
	}

	m.password, err = ReadAllPipeline(m.flags.passwordIn)
	if err != nil {
		return errors.Wrapf(err, "Error reading password from %q flag in aes module", "password-pipe")
	}

	if len(m.password) == 0 {
		return errors.New("Password is empty in aes module")
	}

	m.flags.keyLen = 128
	if m.flags.full {
		m.flags.keyLen = 256
	}

	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					inc, ok := message.Interface.(MessageChannel)
					if ok {
						outc := make(MessageChannel)

						out <- &Message{
							Type: MessageTypeChannel,
							Interface: outc,
						}
						wg.Add(1)
						if m.flags.encrypt {
							go startAesGCMEncrypt(inc, outc, m, wg)
						}	else {
							go startAesGCMDecrypt(inc, outc, m, wg)
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

func startAesGCMDecrypt(inc, outc MessageChannel, m *AESGCM, wg *sync.WaitGroup) {
	defer wg.Done()
	defer DrainChannel(inc, nil)
	defer close(outc)

	salt := make([]byte, 12)
	reader := NewMessageReader(inc)
	defer reader.Close()
	_, err := io.ReadFull(reader, salt)
	if err != nil {
		err = errors.Wrap(err, "Error reading salt in aes module")
		log.Println(err.Error())
		return
	}

	key, err := AESGCMDeriveKey(salt, m.password, m.flags.keyLen / 8)
	if err != nil {
		err = errors.Wrap(err, "Error derivating key in aes module")
		log.Println(err.Error())
		return
	}

	aead, err := NewAESAEAD(key)
	if err != nil {
		err = errors.Wrap(err, "Error creating aead object")
		log.Println(err.Error())
		return
	}

	nonceLength := 8
	l := make([]byte, 4)

	nonce, err := NewAESNonce(nonceLength, reader)
	if err != nil {
		err = errors.Wrap(err, "Error generating nonce in aes module")
		log.Println(err.Error())
		return
	}

	for {
		_, err = io.ReadFull(reader, l)
		if err != nil {
			err = errors.Wrap(err, "Error reading length in aes module")
			log.Println(err.Error())
			return
		}

		i := binary.LittleEndian.Uint32(l)

		payload := make([]byte, i + 16 /* 16 is the tag */)
		_, err = io.ReadFull(reader, payload)
		if err != nil {
			err = errors.Wrap(err, "Error reading encrypted payload in aes module")
			log.Println(err.Error())
			return
		}

		plaintext, err := aead.Open(nil, nonce.Nonce(), payload, l)
		if err != nil {
			err = errors.Wrap(err, "Error decrypting the payload")
			log.Println(err.Error())
			return
		}

		outc <- plaintext

		_, err = nonce.Increment()
		if err != nil {
			err = errors.Wrap(err, "Error incrementing nonce in aes module")
			log.Println(err.Error())
			return
		}
	}
}

func startAesGCMEncrypt(inc, outc MessageChannel, m *AESGCM, wg *sync.WaitGroup) {
	defer wg.Done()
	defer DrainChannel(inc, nil)
	defer close(outc)

	salt := make([]byte, 12)
	err := AESGCMGenerateSalt(salt)
	if err != nil {
		err = errors.Wrap(err, "Error generating salt in aes module")
		log.Println(err.Error())
		return
	}

	outc <- salt

	key, err := AESGCMDeriveKey(salt, m.password, m.flags.keyLen / 8)
	if err != nil {
		err = errors.Wrap(err, "Error derivating key in aes module")
		log.Println(err.Error())
		return
	}

	aead, err := NewAESAEAD(key)
	if err != nil {
		err = errors.Wrap(err, "Error creating aead object")
		log.Println(err.Error())
		return
	}

	nonceLength := 8
	nonce, err := NewAESNonce(nonceLength, rand.Reader)
	if err != nil {
		err = errors.Wrap(err, "Error generating nonce in aes module")
		log.Println(err.Error())
		return
	}

	outc <- nonce.Nonce()[:nonceLength]

	l := make([]byte, 4)

	for payload := range inc {
		if len(payload) == 0 {
			continue
		}

		binary.LittleEndian.PutUint32(l, uint32(len(payload)))

		payload = aead.Seal(nil, nonce.Nonce(), payload, l)
		outc <- append(l, payload...)

		rotate, err := nonce.Increment()
		if err != nil {
			err = errors.Wrap(err, "Error incrementing nonce in aes module")
			log.Println(err.Error())
			break
		}

		if rotate {
			outc <- nonce.Nonce()[:nonceLength]
		}
	}
}

func NewAESGCM() (Module) {
	return &AESGCM{
		flags: AESGCMFlags{},
	}
}

func (m *AESGCM) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.BoolVar(&m.flags.encrypt, "encrypt", false, "Encrypt")
	fs.BoolVar(&m.flags.decrypt, "decrypt", false, "Decrypt")
	fs.BoolVar(&m.flags.half, "128", true, "128 bits key")
	fs.BoolVar(&m.flags.full, "256", false, "256 bits key")
	fs.StringVar(&m.flags.passwordIn, "password-in", "", "Pipeline definition to set the password")
}

func AESGCMDeriveKey(salt, password []byte, l int) (k []byte, err error) {
	k, err = scrypt.Key(password, salt, 1<<18, 8, 1, l)
	if err != nil {
		return nil, err
	}

	return k, nil
}

func AESGCMGenerateSalt(salt []byte) (err error) {
	_, err = io.ReadFull(rand.Reader, salt)

	return err
}

type AESNonce struct {
	nonce []byte
	counter uint32
	reader io.Reader
}

func NewAESNonce(l int, reader io.Reader) (nonce *AESNonce, err error) {
	if l < 0 {
		return nil, errors.New("Nonce length cannot be negative")
	}

	nonce = &AESNonce{
		counter: 0,
		nonce: make([]byte, l + 4),
		reader: reader,
	}

	_, err = io.ReadFull(reader, nonce.nonce[:l])
	if err != nil {
		return nil, errors.Wrap(err, "Error reading nonce from reader")
	}

	return nonce, nil
}

func (n *AESNonce) Increment() (rotate bool, err error) {
	l := len(n.nonce) - 4
	rotate = false

	if n.counter == (1<<32) - 1 {
		_, err = io.ReadFull(n.reader, n.nonce[:l])
		if err != nil {
			return false, errors.Wrap(err, "Error reading from reader")
		}

		rotate = true
	}

	n.counter++
	binary.LittleEndian.PutUint32(n.nonce[l:], n.counter)

	return rotate, nil
}

func (n AESNonce) Nonce() (nonce []byte) {
	return n.nonce
}

func NewAESAEAD(key []byte) (aead cipher.AEAD, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "Error generating aes.NewCipher")
	}

	return cipher.NewGCM(block)
}
