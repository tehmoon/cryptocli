package inout

import (
	"net/url"
	"strings"
	ivyParse "robpike.io/ivy/parse"
	ivyRun "robpike.io/ivy/run"
	ivyScan "robpike.io/ivy/scan"
	ivyConfig "robpike.io/ivy/config"
	ivyExec "robpike.io/ivy/exec"
	"github.com/tehmoon/errors"
	"io"
)

var DefaultMath = &Math{
	name: "math:",
	description: "Evaluate an expression using robpike.io/ivy. Doesn't support output.",
}

type Math struct {
	description string
	name string
}

func (m Math) In(uri *url.URL) (Input) {
	return &MathInput{
		expr: uri.Opaque,
		name: "math-input",
	}
}

func (m Math) Out(uri *url.URL) (Output) {
	return &MathOutput{
		name: "math-output",
	}
}

func (m Math) Name() (string) {
	return m.name
}

func (m Math) Description() (string) {
	return m.description
}

type MathInput struct {
	expr string
	reader io.Reader
	name string
}

func (in *MathInput) Init() (error) {
	var writer io.WriteCloser

	if in.expr == "" {
		return errors.New("Math filetype is missing expression")
	}

	in.reader, writer = io.Pipe()

	conf := &ivyConfig.Config{}
	conf.SetFormat("")
	conf.SetMaxDigits(1e9)
	conf.SetOrigin(1)
	conf.SetPrompt("")
	conf.SetOutput(writer)

	context := ivyExec.NewContext(conf)

	scanner := ivyScan.New(context, "", strings.NewReader(in.expr))
	parser := ivyParse.NewParser("", scanner, context)

	go func () {
		ivyRun.Run(parser, context, false)
		writer.Close()
	}()

	return nil
}

func (in *MathInput) Read(p []byte) (int, error) {
	i, err := in.reader.Read(p)
	if i > 0 && p[i - 1] == '\n' {
		i--
	}

	if err != nil {
		return i, err
	}

	return i, err
}

func (in *MathInput) Close() (error) {
	return nil
}

func (in MathInput) Name() (string) {
	return in.name
}

type MathOutput struct {
	name string
}

func (out *MathOutput) Init() (error) {
	return errors.New("Math module doesn't support output\n")
}

func (out *MathOutput) Write(data []byte) (int, error) {
	return 0, io.EOF
}

func (out *MathOutput) Close() (error) {
	return nil
}

func (out MathOutput) Name() (string) {
	return out.name
}

func (out MathOutput) Chomp(chomp bool) {}
