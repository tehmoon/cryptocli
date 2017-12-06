package inout

import (
  "net/url"
  "os"
)

var DefaultFile File = File{
  name: "file://",
  description: "Read from a file or write to a file. Default when no <filetype> is specified. Truncate output file unless OUTFILENOTRUNC=1 in environment variable.",
}

type File struct {
  description string
  name string
}

func (f File) In(uri *url.URL) (Input) {
  return &FileInput{
    path: uri.Path,
    name: "file-input",
  }
}

func (f File) Out(uri *url.URL) (Output) {
  return &FileOutput{
    path: uri.Path,
    name: "file-output",
  }
}

func (f File) Name() (string) {
  return f.name
}

func (f File) Description() (string) {
  return f.description
}

type FileInput struct {
  path string
  file *os.File
  name string
}

func (in *FileInput) Init() (error) {
  file, err := os.OpenFile(in.path, os.O_RDONLY, 000)
  if err != nil {
    return err
  }

  in.file = file

  return nil
}

func (in *FileInput) Read(p []byte) (int, error) {
  return in.file.Read(p)
}

func (in *FileInput) Close() (error) {
  return in.file.Close()
}

func (in FileInput) Name() (string) {
  return in.name
}

type FileOutput struct {
  path string
  file *os.File
  name string
}

func (out *FileOutput) Init() (error) {
  flags := os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC

  if notrunc := os.Getenv("OUTFILENOTRUNC"); notrunc == "1" {
    flags = flags ^ os.O_TRUNC
  }

  file, err := os.OpenFile(out.path, flags, 0640)
  if err != nil {
    return err
  }

  out.file = file

  return nil
}

func (out *FileOutput) Write(data []byte) (int, error) {
  return out.file.Write(data)
}

func (out *FileOutput) Close() (error) {
  return out.file.Close()
}

func (out FileOutput) Name() (string) {
  return out.name
}

func (out FileOutput) Chomp(chomp bool) {}
