package codec

import (
	"io"
	"net/url"
	"fmt"
	"encoding/hex"
	"github.com/tehmoon/errors"
)

var DefaultByteString = &ByteString{
	name: "byte-string",
	description: "Decode and encode in a byte string format",
}

type ByteString struct{
	name string
	description string
}

func (codec ByteString) Name() (string) {
	return codec.name
}

func (codec ByteString) Description() (string) {
	return codec.description
}

type ByteStringDecoder struct {
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
}

type ByteStringEncoder struct {
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
}

func (codec ByteString) Decoder(values url.Values) (CodecDecoder) {

	dec := &ByteStringDecoder{}

	dec.pipeReader, dec.pipeWriter = io.Pipe()
	return dec
}

func (codec ByteString) Encoder(values url.Values) (CodecEncoder) {
	enc := &ByteStringEncoder{}
	enc.pipeReader, enc.pipeWriter = io.Pipe()
	return enc
}

func (codec ByteStringDecoder) Init() (error) {
	return nil
}

func (dec ByteStringDecoder) Read(p []byte) (int, error) {
	var i int

	for i = range p {
		buff := make([]byte, 4)

		_, err := dec.pipeReader.Read(buff)
		if err != nil {
			return i, err
		}

		b, err := ByteStringDecode(buff)
		if err != nil {
			return i, err
		}

		p[i] = b
	}

	return i, nil
}

func ByteStringDecode(p []byte) (byte, error) {
	l := len(p)

	if l != 3 && l != 4 {
		return 0, errors.New("Wrong decoding size, must be either 3 or 4 bytes long")
	}

	if p[0] != '/' && p[1] != 'x' {
		return 0, errors.New("Byte to decode should start with \\x")
	}

	var (
		src = p[2:]
		dest [1]byte
	)

	if l == 3 {
		src[0] = '0'
		src[1] = p[2]
	}

	i, err := hex.Decode(dest[:], src)
	if err != nil {
		return 0, err
	}

	if i != 1 {
		return 0, io.ErrShortWrite
	}

	return dest[0], nil
}

func (dec *ByteStringDecoder) Write(data []byte) (int, error) {
	return dec.pipeWriter.Write(data)
}

func (dec ByteStringDecoder) Close() (error) {
	return dec.pipeWriter.Close()
}

func (enc ByteStringEncoder) Init() (error) {
	return nil
}

func (enc ByteStringEncoder) Read(p []byte) (int, error) {
	return enc.pipeReader.Read(p)
}

func (enc ByteStringEncoder) Write(data []byte) (int, error) {
	var (
		wrote int
		i int
		err error
	)

	for _, c := range data {
		buff := fmt.Sprintf("\\x%02x", c)
		i, err = enc.pipeWriter.Write([]byte(buff))
		if i > 0 {
			wrote++
		}

		if err != nil {
			break
		}
	}

	return wrote, err
}

func (enc ByteStringEncoder) Close() (error) {
	return enc.pipeWriter.Close()
}
