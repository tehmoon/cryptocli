package codec

import (
	"io"
	"net/url"
	"github.com/pkg/errors"
	"encoding/base32"
)

var DefaultBase32 = &Base32{
	name: "base32",
	description: "base32 decode input and base32 encode output",
}

type Base32 struct{
	name string
	description string
}

func (codec Base32) Name() (string) {
	return codec.name
}

func (codec Base32) Description() (string) {
	return codec.description
}

type Base32Decoder struct {
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	decoder io.Reader
}

type Base32Encoder struct {
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	encoder io.WriteCloser
}

func (codec Base32) Decoder(values url.Values) (CodecDecoder) {
	dec := &Base32Decoder{
	}
	dec.pipeReader, dec.pipeWriter = io.Pipe()
	dec.decoder = base32.NewDecoder(base32.StdEncoding, dec.pipeReader)
	return dec
}

func (codec Base32) Encoder(values url.Values) (CodecEncoder) {
	enc := &Base32Encoder{}
	enc.pipeReader, enc.pipeWriter = io.Pipe()
	enc.encoder = base32.NewEncoder(base32.StdEncoding, enc.pipeWriter)
	return enc
}

func (codec Base32Decoder) Init() (error) {
	return nil
}

func (dec *Base32Decoder) Read(p []byte) (int, error) {
	return dec.decoder.Read(p)
}

func (dec *Base32Decoder) Write(data []byte) (int, error) {
	return dec.pipeWriter.Write(data)
}

func (dec *Base32Decoder) Close() (error) {
	return dec.pipeWriter.Close()
}

func (ecn *Base32Encoder) Init() (error) {
	return nil
}

func (enc *Base32Encoder) Read(p []byte) (int, error) {
	return enc.pipeReader.Read(p)
}

func (enc *Base32Encoder) Write(data []byte) (int, error) {
	return enc.encoder.Write(data)
}

func (enc *Base32Encoder) Close() (error) {
	err := enc.encoder.Close()
	if err != nil {
		return errors.Wrap(err, "Error closing the base32 encoder")
	}

	return enc.pipeWriter.Close()
}
