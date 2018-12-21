package main

import (
	"sync"
	"github.com/spf13/pflag"
	"hash"
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
	in chan *Message
	out chan *Message
	hash hash.Hash
	algo string
	wg *sync.WaitGroup
}

func findDgst(name string) (hash.Hash, error) {
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
			return nil, errors.Errorf("Hash algorithm %q is no supported", name)
	}

	if ! hash.Available() {
		return nil, errors.Errorf("Hash algorithm %q is not linked to this binary", name)
	}

	return hash.New(), nil
}

func (m *Dgst) Init(global *GlobalFlags) (error) {
	var err error

	m.hash, err = findDgst(m.algo)
	if err != nil {
		return err
	}

	return nil
}

func (m Dgst) Start() {
	m.wg.Add(1)

	go func() {
		for message := range m.in {
			m.hash.Write(message.Payload)
		}

		SendMessage(m.hash.Sum(nil), m.out)
		close(m.out)

		m.wg.Done()
	}()
}

func (m Dgst) Wait() {
	m.wg.Wait()

	for range m.in {}
}

func (m *Dgst) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Dgst) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewDgst() (Module) {
	return &Dgst{
		wg: &sync.WaitGroup{},
	}
}

func (m *Dgst) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.algo, "algo", "", "Hash algorithm to use: md5, sha1, sha256, sha512, sha3_224, sha3_256, sha3_384, sha3_512, blake2s_256, blake2b_256, blake2b_384, blake2b_512, ripemd160")
}
