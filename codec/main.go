package codec

import (
  "github.com/pkg/errors"
  "io"
)

var (
  ErrCodecUnknown = errors.New("Unknown codec\n")
  ErrEmptyParseCodecs = errors.New("Some codecs were parsed but all were empty string\n")
  CodecList = []Codec{
    DefaultHex,
    DefaultBinary,
    DefaultBinaryString,
    DefaultBase64,
    DefaultGzip,
    DefaultHexdump,
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
      return DefaultHex, nil
    case "binary":
      return DefaultBinary, nil
    case "binary_string":
      return DefaultBinaryString, nil
    case "base64":
      return DefaultBase64, nil
    case "gzip":
      return DefaultGzip, nil
    case "hexdump":
      return DefaultHexdump, nil
    default:
      return nil, ErrCodecUnknown
  }
}

type Codec interface {
  Decoder() (CodecDecoder)
  Encoder() (CodecEncoder)
  Name() (string)
  Description() (string)
}

type CodecEncoder interface {
  io.ReadWriteCloser
  Init() (error)
}

type CodecDecoder interface {
  io.ReadWriteCloser
  Init() (error)
}
