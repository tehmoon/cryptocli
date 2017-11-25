package codec

import (
  "github.com/pkg/errors"
  "io"
  "strings"
  "net/url"
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

type CodecValues struct {
  Codec Codec
  Values url.Values
}

func ParseAll(codecs []string) ([]CodecValues, error) {
  cvs := make([]CodecValues, 0)

  for _, codec := range codecs {
    if codec == "" {
      continue
    }

    cv, err := Parse(codec)
    if err != nil {
      return nil, err
    }

    cvs = append(cvs, *cv)
  }

  if len(cvs) == 0 {
    return nil, ErrEmptyParseCodecs
  }

  return cvs, nil
}

func Parse(codecStr string) (*CodecValues, error) {
  parts := strings.Split(codecStr, ":")
  codec := parts[0]

  var values url.Values

  if len(parts) > 1 {
    qs := strings.Join(parts[1:], ":")

    var err error
    values, err = url.ParseQuery(qs)
    if err != nil {
      return nil, errors.Wrapf(err, "Error parsing query string value: %s", qs)
    }
  }

  switch codec {
    case "hex":
      return &CodecValues{Codec: DefaultHex, Values: values,}, nil
    case "binary":
      return &CodecValues{Codec: DefaultBinary, Values: values,}, nil
    case "binary_string":
      return &CodecValues{Codec: DefaultBinaryString, Values: values,}, nil
    case "base64":
      return &CodecValues{Codec: DefaultBase64, Values: values,}, nil
    case "gzip":
      return &CodecValues{Codec: DefaultGzip, Values: values,}, nil
    case "hexdump":
      return &CodecValues{Codec: DefaultHexdump, Values: values,}, nil
    default:
      return nil, ErrCodecUnknown
  }
}

type Codec interface {
  Decoder(url.Values) (CodecDecoder)
  Encoder(url.Values) (CodecEncoder)
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
