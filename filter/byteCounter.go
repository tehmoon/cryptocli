package filter

import (
	"../inout"
	"net/url"
	"io"
	"github.com/tehmoon/errors"
	"math/big"
)

var (
	DefaultByteCounter = &ByteCounter{
		name: "byte-counter",
		description: "Keep track of in and out bytes. Options: start-at=<number> stop-at=[+]<number>. Start-at option will discard the first <number> bytes. The Stop-at option will stop at byte <number>. Position <number> can be express in base16 with 0x/0X, base2 with 0b/0B or 0 for base8. If a + sign is found in the stop-at option; start-at <number> is added to stop-at <number>.",
	}
)

type ByteCounter struct {
	name string
	description string
}

func (bc ByteCounter) Name() (string) {
	return bc.name
}

func (bc ByteCounter) Description() (string) {
	return bc.description
}

func (bc ByteCounter) Filter(u *url.URL) (Filter) {
	bcf := &ByteCounterFilter{}

	if u != nil {
		bcf.qs = u.Opaque
	}

	return bcf
}

type ByteCounterFilter struct {
	qs string
	reader *io.PipeReader
	writer *io.PipeWriter
	options *ByteCounterFilterOptions
	inReader *io.PipeReader
	inWriter *io.PipeWriter
	outReader *io.PipeReader
	outWriter *io.PipeWriter
	sync chan error
}

type ByteCounterFilterOptions struct {
	StartAt int64
	StopAt int64
}

func (bcf *ByteCounterFilter) Init() (error) {
	var ok bool

	if bcf.options == nil {
		options := &ByteCounterFilterOptions{
			StartAt: -1,
			StopAt: -1,
		}

		values, err := url.ParseQuery(bcf.qs)
		if err != nil {
			return errors.Wrap(err, "Error parsing URL options")
		}

		startAt := values.Get("start-at")
		stopAt := values.Get("stop-at")

		if startAt != "" {
			options.StartAt, ok = parseBytePositionArgument(startAt)
			if ! ok {
				return errors.New("Bad number at option start-at")
			}
		}

		if stopAt != "" {
			options.StopAt, ok = parseBytePositionArgument(stopAt)
			if ! ok {
				return errors.New("Bad number at option stop-at")
			}

			if stopAt[0] == '+' {
				n1 := new(big.Int).SetInt64(options.StartAt)
				n2 := new(big.Int).SetInt64(options.StopAt)
				n := new(big.Int).Add(n1, n2)
				ok = n.IsInt64()

				if ! ok {
					return errors.Errorf("Option stop-at overflows int64")
				}

				options.StopAt = n.Int64()
			}
		}

		bcf.options = options
	}

	bcf.inReader, bcf.inWriter = io.Pipe()
	bcf.outReader, bcf.outWriter = io.Pipe()
	bcf.sync = make(chan error)
	nullReader := inout.DefaultNull.Out(nil)
	nullReader.Init()

	go func() {
		var err error

		defer func() {
			bcf.outWriter.CloseWithError(err)
			bcf.sync <- err
		}()

		if bcf.options.StartAt > 0 {
			io.CopyN(nullReader, bcf.inReader, bcf.options.StartAt)
		}

		if bcf.options.StopAt >= 0 {
			_, err = io.CopyN(bcf.outWriter, bcf.inReader, bcf.options.StopAt)
			if err == io.EOF {
				err = nil
			}
			return
		}

		_, err = io.Copy(bcf.outWriter, bcf.inReader)
	}()

	return nil
}

func (bcf ByteCounterFilter) Read(p []byte) (int, error) {
	return bcf.outReader.Read(p)
}

func (bcf ByteCounterFilter) Write(data []byte) (int, error) {
	return bcf.inWriter.Write(data)
}

func (bcf ByteCounterFilter) Close() (error) {
	bcf.inWriter.Close()

	err := <- bcf.sync
	return err
}

func parseBytePositionArgument(mark string) (int64, bool) {
	if mark[0] == '+' {
		mark = mark[1:]
	}

	i, ok := new(big.Int).SetString(mark, 0)
	if ! ok {
		return 0, false
	}

	ok = i.IsInt64()
	if ! ok {
		return 0, false
	}

	return i.Int64(), true
}
