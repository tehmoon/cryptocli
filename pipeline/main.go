package pipeline

import (
  "io"
  "fmt"
  "github.com/tehmoon/errors"
  "sync"
  "os"
)

type State uint8

const (
  StateNew State = iota
  StateInit
  StateRunning
  StateClosed
)

var (
  ErrBadState = errors.New("Bad state")
)

type Element interface {
  Init() (error)
  Read([]byte) (int, error)
  Write([]byte) (int, error)
  Close() (error)
//  Reset() (error)
}

type Pipeliner interface {
  Init() (error)
  Read([]byte) (int, error)
  Write([]byte) (int, error)
  Close() (error)
//  Reset() (error)
  Add(Element) (error)
}

type Pipeline struct {
  elems []Element
  reader *io.PipeReader
  writer *io.PipeWriter
  state State
}

func New() (Pipeliner) {
  return &Pipeline{}
}

func (p *Pipeline) Add(e Element) (error) {
  if p.state != StateNew {
    return errors.New("You can't add more decoders after calling Init()")
  }

  p.elems = append(p.elems, e)

  return nil
}

func initElements(elems []Element) (error) {
  wg := &sync.WaitGroup{}

  for _, elem := range elems {
    wg.Add(1)

    go func(elem Element, wg *sync.WaitGroup) {
      err := elem.Init()
      if err != nil {
        elem.Close()
        return
      }

      wg.Done()
    }(elem, wg)
  }

  wg.Wait()

  return nil
}

func ioCopyElements(input io.Reader, output io.WriteCloser, elems []Element) (error) {
  next := input

  for _, elem := range elems {
    go func(elem Element, reader io.Reader) {
      _, err := io.Copy(elem, reader)
      if err != nil {
        fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error in copying element %T in pipeline", reader).Error())
        os.Exit(1)
      }

      elem.Close()
    }(elem, next)

    next = elem
  }

  go func(writer io.WriteCloser, reader io.Reader) {
    _, err := io.Copy(writer, reader)
    if err != nil {
      fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error in copying element %T in pipeline", writer).Error())
      os.Exit(1)
    }

    writer.Close()
  }(output, next)

  return nil
}

func (p *Pipeline) Init() (error) {
  if p.state != StateNew {
    return errors.New("You must reset the pipeline before calling Init() again")
  }

  p.state = StateInit

  var (
    output *io.PipeReader
    input *io.PipeWriter
  )

  output, p.writer = io.Pipe()
  p.reader, input= io.Pipe()

  err := initElements(p.elems)
  if err != nil {
    return errors.Wrap(err, "Error in intializing the elements")
  }

  err = ioCopyElements(output, input, p.elems)
  if err != nil {
    return errors.Wrap(err, "Error in iocopy the elements")
  }

  p.state = StateRunning

  return nil
}

func (p *Pipeline) Close() (error) {
  return p.writer.Close()
}

func (p *Pipeline) Write(data []byte) (int, error) {
  return p.writer.Write(data)
}

func (p *Pipeline) Read(data []byte) (int, error) {
  return p.reader.Read(data)
}

//func (p *Pipeline) Reset() (error) {
//  return nil
//}
