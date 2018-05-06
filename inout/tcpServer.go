package inout

import (
	"net/url"
	"net"
	"github.com/tehmoon/errors"
	"io"
)

var DefaultTcpServer = &TcpServer{
	name: "tcp-server://",
	description: "Listen tcp on host:port and either read or write data",
}

type TcpServer struct{
	description string
	name string
}

func (h TcpServer) In(uri *url.URL) (Input) {
	in := &TcpServerInput{
		host: uri.Host,
		name: "tcp-server-intput",
		sync: make(chan error),
	}

	in.pipeReader, in.pipeWriter = io.Pipe()

	return in
}

func (h TcpServer) Out(uri *url.URL) (Output) {
	out := &TcpServerOutput{
		host: uri.Host,
		name: "tcp-server-output",
		sync: make(chan error),
	}

	out.pipeReader, out.pipeWriter = io.Pipe()

	return out
}

func (h TcpServer) Name() (string) {
	return h.name
}

func (h TcpServer) Description() (string) {
	return h.description
}

type TcpServerInput struct {
	host string
	name string
	l net.Listener
	sync chan error
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
}

type TcpServerOutput struct {
	host string
	name string
	l net.Listener
	sync chan error
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
}

func (in TcpServerInput) Name() (string) {
	return in.name
}

func closeTcpServerWithChan(closers []io.Closer, sync chan error, err error) {
	for _, closer := range closers {
		closer.Close()
	}

	sync <- err
}

func (in *TcpServerInput) Init() (error) {
	if in.host == "" {
		return errors.New("Host cannot be empty")
	}

	var err error

	in.l, err = net.Listen("tcp", in.host)
	if err != nil {
		return errors.Wrapf(err, "Error listening on %s", in.host)
	}

	go func(writer io.WriteCloser, l net.Listener, sync chan error) {
		conn, err := l.Accept()
		if err != nil {
			err = errors.Wrap(err, "Error accepting the connection")
			closeTcpServerWithChan([]io.Closer{conn, writer,}, sync, err)

			return
		}

		_, err = io.Copy(writer, conn)
		closeTcpServerWithChan([]io.Closer{conn, writer,}, sync, err)
	}(in.pipeWriter, in.l, in.sync)

	return nil
}

func (in *TcpServerInput) Close() (error) {
	in.l.Close()

	return <- in.sync
}

func (in *TcpServerInput) Read(p []byte) (int, error) {
	return in.pipeReader.Read(p)
}

func (out TcpServerOutput) Init() (error) {
	if out.host == "" {
		return errors.New("Host cannot be empty")
	}

	var err error

	out.l, err = net.Listen("tcp", out.host)
	if err != nil {
		return errors.Wrapf(err, "Error listening on %s", out.host)
	}

	go func(reader io.ReadCloser, l net.Listener, sync chan error) {

		conn, err := l.Accept()
		if err != nil {
			err = errors.Wrap(err, "Error accepting the connection")
			closeTcpServerWithChan([]io.Closer{reader, l,}, sync, err)

			return
		}

		_, err = io.Copy(conn, reader)
		closeTcpServerWithChan([]io.Closer{reader, l, conn,}, sync, err)
	}(out.pipeReader, out.l, out.sync)

	return nil
}

func (out TcpServerOutput) Write(data []byte) (int, error) {
	return out.pipeWriter.Write(data)
}

func (out TcpServerOutput) Close() (error) {
	out.pipeWriter.Close()

	return <- out.sync
}

func (out TcpServerOutput) Name() (string) {
	return out.name
}

func (out TcpServerOutput) Chomp(chomp bool) {}
