package main

import (
	"sync"
	"github.com/spf13/pflag"
	"github.com/tehmoon/errors"
	"strings"
	"crypto"
	_ "golang.org/x/crypto/blake2s"
	_ "golang.org/x/crypto/blake2b"
	_ "golang.org/x/crypto/ripemd160"
	_ "golang.org/x/crypto/sha3"
)

func init() {
	MODULELIST.Register("dgst", "Dgst decode or encode", NewDgst)
}

type Dgst struct {
	hash crypto.Hash
	algo string
}

func findDgst(name string) (crypto.Hash, error) {
	var hash crypto.Hash
	name = strings.ToLower(name)

	switch name {
		case "md5":
			hash = crypto.MD5
		case "sha1":
			hash = crypto.SHA1
		case "sha256":
			hash = crypto.SHA256
		case "sha512":
			hash = crypto.SHA512
		case "sha3_224":
			hash = crypto.SHA3_224
		case "sha3_256":
			hash = crypto.SHA3_256
		case "sha3_384":
			hash = crypto.SHA3_384
		case "sha3_512":
			hash = crypto.SHA3_512
		case "blake2s_256":
			hash = crypto.BLAKE2s_256
		case "blake2b_256":
			hash = crypto.BLAKE2b_256
		case "blake2b_384":
			hash = crypto.BLAKE2b_384
		case "blake2b_512":
			hash = crypto.BLAKE2b_512
		case "ripemd160":
			hash = crypto.RIPEMD160
		default:
			return 0, errors.Errorf("Hash algorithm %q is no supported", name)
	}

	if ! hash.Available() {
		return 0, errors.Errorf("Hash algorithm %q is not linked to this binary", name)
	}

	return hash, nil
}

func (m *Dgst) Init(in, out chan *Message, global *GlobalFlags) (error) {
	var err error

	m.hash, err = findDgst(m.algo)
	if err != nil {
		return err
	}

	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		init := false
		mc := NewMessageChannel()

		out <- &Message{
			Type: MessageTypeChannel,
			Interface: mc.Callback,
		}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					if ! init {
						close(mc.Channel)
					}

					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					cb, ok := message.Interface.(MessageChannelFunc)
					if ok {
						if ! init {
							init = true
						} else {
							mc = NewMessageChannel()

							out <- &Message{
								Type: MessageTypeChannel,
								Interface: mc.Callback,
							}
						}

						hash := m.hash.New()
						wg.Add(1)

						go func() {
							mc.Start(nil)
							_, inc := cb()
							outc := mc.Channel

							for payload := range inc {
								hash.Write(payload)
							}

							outc <- hash.Sum(nil)
							close(outc)
							wg.Done()
						}()

						if ! global.MultiStreams {
							if ! init {
								close(mc.Channel)
							}
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

func NewDgst() (Module) {
	return &Dgst{}
}

func (m *Dgst) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.algo, "algo", "", "Hash algorithm to use: md5, sha1, sha256, sha512, sha3_224, sha3_256, sha3_384, sha3_512, blake2s_256, blake2b_256, blake2b_384, blake2b_512, ripemd160")
}
