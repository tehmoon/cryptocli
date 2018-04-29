package inout

import (
	"net/url"
	"os"
	"net/http"
	"crypto/tls"
	"io"
)

var DefaultHttps = &Https{
	name: "https://",
	description: "Get https url or post the output to https. Use INHTTPSNOVERIFY=1 and/or OUTHTTPSNOVERIFY=1 environment variables to disable certificate check. Max redirects count is 3. Will fail if scheme changes.",
}

type Https struct{
	description string
	name string
}

func (h Https) In(uri *url.URL) (Input) {
	in := &HttpsInput{
		uri: uri,
		name: "https-input",
	}

	in.pipeReader, in.pipeWriter = io.Pipe()

	return in
}

func (h Https) Out(uri *url.URL) (Output) {
	out := &HttpsOutput{
		uri: uri,
		name: "https-output",
		sync: make(chan error),
	}

	out.pipeReader, out.pipeWriter = io.Pipe()

	return out
}

func (h Https) Name() (string) {
	return h.name
}

func (h Https) Description() (string) {
	return h.description
}

type HttpsInput struct {
	uri *url.URL
	name string
	body io.ReadCloser
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
}

type HttpsOutput struct {
	uri *url.URL
	name string
	sync chan error
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
}

func (in HttpsInput) Name() (string) {
	return in.name
}

func (in *HttpsInput) Init() (error) {
	transport := copyDefaultHTTPTransport()
	client := &http.Client{
		Transport: transport,
		CheckRedirect: httpRedirectPolicy(),
	}

	if transport != nil {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: false,
		}

		if verifyTLS := os.Getenv("INHTTPSNOVERIFY"); verifyTLS == "1" {
			transport.TLSClientConfig.InsecureSkipVerify = true
		}
	}

	body, err := readHTTP(*in.uri, client)
	if err != nil {
		return err
	}

	in.body = body

	go func() {
		io.Copy(in.pipeWriter, in.body)

		in.pipeWriter.Close()
	}()

	return nil
}

func (in *HttpsInput) Close() (error) {
	return in.body.Close()
}

func (in *HttpsInput) Read(p []byte) (int, error) {
	return in.pipeReader.Read(p)
}

func (out HttpsOutput) Init() (error) {
	transport := copyDefaultHTTPTransport()
	client := &http.Client{
		Transport: transport,
		CheckRedirect: httpRedirectPolicy(),
	}

	if transport != nil {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: false,
		}

		if verifyTLS := os.Getenv("OUTHTTPSNOVERIFY"); verifyTLS == "1" {
			transport.TLSClientConfig.InsecureSkipVerify = true
		}
	}

	writer, err := writeHTTP(*out.uri, client, out.sync)
	if err != nil {
		return err
	}

	go func() {
		io.Copy(writer, out.pipeReader)
		writer.Close()
	}()

	return nil
}

func (out HttpsOutput) Write(data []byte) (int, error) {
	return out.pipeWriter.Write(data)
}

func (out HttpsOutput) Close() (error) {
	out.pipeWriter.Close()

	return <- out.sync
}

func (out HttpsOutput) Name() (string) {
	return out.name
}

func (out HttpsOutput) Chomp(chomp bool) {}
