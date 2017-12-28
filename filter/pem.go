package filter

import (
  "strings"
  "net/url"
  "io"
  "github.com/tehmoon/errors"
  "bytes"
  "encoding/pem"
  "strconv"
)

var (
  DefaultPem = &Pem{
    name: "pem",
    description: "Filter PEM objects. Options: type=<PEM type> start-at=<number> stop-at=<number>. Type will filter only PEM objects with this type. Start-at will discard the first <number> PEM objects. Stop-at will stop at PEM object <number>.",
  }
  ErrPemFilterAlreadyStarted = errors.New("Pem filter has already been Init(). Call Reset()")
  ErrPemFilterLeftBytes = errors.New("Pem filter still has some bytes to decode")
)

type Pem struct {
  name string
  description string
}

func (pem Pem) Name() (string) {
  return pem.name
}

func (pem Pem) Description() (string) {
  return pem.description
}

func (pem Pem) Filter(u *url.URL) (Filter) {
  f := &PemFilter{
    started: false,
  }

  if u != nil {
    f.qs = u.Opaque
  }

  return f
}

type PemFilter struct {
  reader *io.PipeReader
  writer *io.PipeWriter
  started bool
  rest []byte
  buff *bytes.Buffer
  qs string
  options *PemFilterOptions
  counter uint64
}

type PemFilterOptions struct {
  Type string
  StartAt uint64
  StopAt uint64
}

func (pf *PemFilter) Init() (error) {
  if pf.started {
    return ErrPemFilterAlreadyStarted
  }

  if pf.options == nil {
    options := &PemFilterOptions{}

    values, err := url.ParseQuery(pf.qs)
    if err != nil {
      return errors.Wrap(err, "Error parsing URL options")
    }

    t := values.Get("type")
    options.Type = strings.ToLower(t)

    startAt := values.Get("start-at")
    stopAt := values.Get("stop-at")

    if startAt != "" {
      options.StartAt, err = strconv.ParseUint(startAt, 10, 64)
      if err != nil {
        return errors.Wrap(err, "Error parsing start-at value")
      }
    }

    if stopAt != "" {
      options.StopAt, err = strconv.ParseUint(stopAt, 10, 64)
      if err != nil {
        return errors.Wrap(err, "Error parsing stop-at value")
      }

      if options.StopAt == 0 {
        return errors.New("stop-at options cannot be 0")
      }
    }

    pf.options = options
  }

  pf.reader, pf.writer = io.Pipe()
  pf.buff = bytes.NewBuffer(nil)

  return nil
}

func (pf *PemFilter) Read(p []byte) (int, error) {
  return pf.reader.Read(p)
}

func (pf *PemFilter) Write(data []byte) (int, error) {
  var block *pem.Block

  pf.buff.Write(data)

  rest := pf.buff.Bytes()

  for {
    block, rest = pem.Decode(rest)
    if block != nil {
      if pf.options.Type == strings.ToLower(block.Type) || pf.options.Type == "" {
        if pf.counter >= pf.options.StartAt && (pf.counter < pf.options.StopAt || pf.options.StopAt == 0) {
          err := pem.Encode(pf.writer, block)
          if err != nil {
            return len(data), err
          }
        }
      }

      pf.buff.Reset()
      pf.counter++
      continue
    }

    _, err := pf.buff.Write(rest)
    if err != nil {
      return len(data), err
    }

    break
  }

  return len(data), nil
}

func (pf *PemFilter) Close() (error) {
  if pf.writer == nil {
    return nil
  }

  return pf.writer.Close()
}
