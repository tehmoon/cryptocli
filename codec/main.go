package codec

import (
  "github.com/pkg/errors"
  "io"
)

var (
  ErrCodecUnknown = errors.New("Unknown codec")
  ErrEmptyParseCodecs = errors.New("Some codecs were parsed but all were empty string")
  CodecList = []string{
    "hex",
    "binary",
    "binary_string",
    "base64",
    "gzip",
  }
)

func ParseAll(codecs []string) ([]Codec, error) {
  cds := make([]Codec, 0)

  for _, codec := range codecs {
    if codec == "" {
      continue
    }

    c, err := Parse(codec)
    if err != nil {
      return nil, err
    }

    cds = append(cds, c)
  }

  if len(cds) == 0 {
    return nil, ErrEmptyParseCodecs
  }

  return cds, nil
}

func Parse(codec string) (Codec, error) {
  switch codec {
    case "hex":
      return NewCodecHex(), nil
    case "binary":
      return NewCodecBinary(), nil
    case "binary_string":
      return NewCodecBinaryString(), nil
    case "base64":
      return NewCodecBase64(), nil
    case "gzip":
      return NewCodecGzip(), nil
    default:
      return nil, ErrCodecUnknown
  }
}

type Codec interface {
  Decoder() (CodecDecoder)
  Encoder() (CodecEncoder)
}

type CodecEncoder interface {
  io.ReadWriteCloser
  Init() (error)
}

type CodecDecoder interface {
  io.ReadWriteCloser
  Init() (error)
}
