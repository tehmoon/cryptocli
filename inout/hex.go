package inout

import (
	"net/url"
	"io"
	"github.com/tehmoon/errors"
	"encoding/hex"
	"bytes"
)

var DefaultHex = &Hex{
	name: "hex:",
	description: "Decode hex value and use it for input. Doesn't work for output",
}

type Hex struct {
	description string
	name string
}

func (h Hex) In(uri *url.URL) (Input) {
	return &HexInput{
		hexString: uri.Opaque,
		name: "hex-input",
	}
}

func (h Hex) Out(uri *url.URL) (Output) {
	return &HexOutput{
		name: "hex-output",
	}
}

func (h Hex) Name() (string) {
	return h.name
}

func (h Hex) Description() (string) {
	return h.description
}

type HexInput struct {
	name string
	hexString string
	reader io.Reader
}

func (in *HexInput) Init() (error) {
	buff, err := hex.DecodeString(in.hexString)
	if err != nil {
		return errors.Wrap(err, "Error decoding hex")
	}

	in.reader = bytes.NewBuffer(buff)

	return nil
}

func (in HexInput) Read(p []byte) (int, error) {
	return in.reader.Read(p)
}

func (in HexInput) Close() (error) {
	return nil
}

func (in HexInput) Name() (string) {
	return in.name
}

type HexOutput struct {
	name string
}

func (out HexOutput) Init() (error) {
	return errors.New("Hex module doesn't support output\n")
}

func (out HexOutput) Write(data []byte) (int, error) {
	return 0, io.EOF
}

func (out HexOutput) Close() (error) {
	return nil
}

func (out HexOutput) Name() (string) {
	return out.name
}

func (out HexOutput) Chomp(chomp bool) {}
