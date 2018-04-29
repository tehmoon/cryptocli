package inout

import (
	"net/url"
	"io"
	"github.com/tehmoon/errors"
	"crypto/rand"
)

var DefaultRand = &Rand{
	name: "rand:",
	description: "Rand is a global, shared instance of a cryptographically strong pseudo-random generator. On Linux, Rand uses getrandom(2) if available, /dev/urandom otherwise. On OpenBSD, Rand uses getentropy(2). On other Unix-like systems, Rand reads from /dev/urandom. On Windows systems, Rand uses the CryptGenRandom API. Doesn't work with output.",
}

type Rand struct {
	description string
	name string
}

func (h Rand) In(uri *url.URL) (Input) {
	return &RandInput{
		name: "rand-input",
	}
}

func (h Rand) Out(uri *url.URL) (Output) {
	return &RandOutput{
		name: "rand-output",
	}
}

func (h Rand) Name() (string) {
	return h.name
}

func (h Rand) Description() (string) {
	return h.description
}

type RandInput struct {
	name string
}

func (in *RandInput) Init() (error) {
	return nil
}

func (in RandInput) Read(p []byte) (int, error) {
	return rand.Read(p)
}

func (in RandInput) Close() (error) {
	return nil
}

func (in RandInput) Name() (string) {
	return in.name
}

type RandOutput struct {
	name string
}

func (out RandOutput) Init() (error) {
	return errors.New("Rand module doesn't support output\n")
}

func (out RandOutput) Write(data []byte) (int, error) {
	return 0, io.EOF
}

func (out RandOutput) Close() (error) {
	return nil
}

func (out RandOutput) Name() (string) {
	return out.name
}

func (out RandOutput) Chomp(chomp bool) {}
