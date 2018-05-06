package inout

import (
	"net/url"
	"net"
	"github.com/tehmoon/errors"
	"io"
)

var DefaultTcp = &Tcp{
	name: "tcp://",
	description: "Connect to tcp server and either read or write data",
}

type Tcp struct{
	description string
	name string
}

func (h Tcp) In(uri *url.URL) (Input) {
	in := &TcpInput{
		host: uri.Host,
		name: "tcp-intput",
		sync: make(chan error),
	}

	in.pipeReader, in.pipeWriter = io.Pipe()

	return in
}

func (h Tcp) Out(uri *url.URL) (Output) {
	out := &TcpOutput{
		host: uri.Host,
		name: "tcp-output",
		sync: make(chan error),
	}

	out.pipeReader, out.pipeWriter = io.Pipe()

	return out
}

func (h Tcp) Name() (string) {
	return h.name
}

func (h Tcp) Description() (string) {
	return h.description
}

type TcpInput struct {
	host string
	name string
	conn net.Conn
	sync chan error
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
}

type TcpOutput struct {
	host string
	name string
	conn net.Conn
	sync chan error
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
}

func (in TcpInput) Name() (string) {
	return in.name
}

func (in *TcpInput) Init() (error) {
	if in.host == "" {
		return errors.New("Host cannot be empty")
	}

	var err error

	in.conn, err = net.Dial("tcp", in.host)
	if err != nil {
		return errors.Wrapf(err, "Error dialing host %s", in.host)
	}

	go func(writer io.WriteCloser, reader io.ReadCloser, sync chan error) {
		_, err := io.Copy(writer, reader)
		writer.Close()

		sync <- err
	}(in.pipeWriter, in.conn, in.sync)

	return nil
}

func (in *TcpInput) Close() (error) {
	in.conn.Close()

	return <- in.sync
}

func (in *TcpInput) Read(p []byte) (int, error) {
	return in.pipeReader.Read(p)
}

func (out TcpOutput) Init() (error) {
	if out.host == "" {
		return errors.New("Host cannot be empty")
	}

	var err error

	out.conn, err = net.Dial("tcp", out.host)
	if err != nil {
		return errors.Wrapf(err, "Error dialing host %s", out.host)
	}

	go func(writer io.WriteCloser, reader io.ReadCloser, sync chan error) {
		_, err := io.Copy(writer, reader)
		writer.Close()
		reader.Close()

		sync <- err
	}(out.conn, out.pipeReader, out.sync)

	return nil
}

func (out TcpOutput) Write(data []byte) (int, error) {
	return out.pipeWriter.Write(data)
}

func (out TcpOutput) Close() (error) {
	out.pipeWriter.Close()

	return <- out.sync
}

func (out TcpOutput) Name() (string) {
	return out.name
}

func (out TcpOutput) Chomp(chomp bool) {}
