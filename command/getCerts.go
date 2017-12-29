package command

import (
  "encoding/pem"
  "io"
  "../flags"
  "crypto/tls"
  "github.com/tehmoon/errors"
  "flag"
  "net"
)

type GetCerts struct {
  name string
  description string
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  flagSet *flag.FlagSet
  usage *flags.Usage
  addr *net.TCPAddr
  sync chan error
}

var DefaultGetCerts = &GetCerts{
  name: "get-certs",
  description: "Establish tls connection and get the certificates. Doesn't use any input.",
  usage: &flags.Usage{
    CommandLine: "<host[:<port>]>",
    Other: "",
  },
}

func (command *GetCerts) Init() (error) {
  command.pipeReader, command.pipeWriter = io.Pipe()
  command.sync = make(chan error)

  config := &tls.Config{
    InsecureSkipVerify: true,
  }

  conn, err := tls.Dial("tcp", command.addr.String(), config)
  if err != nil {
    return errors.Wrap(err, "Error dialing tls connection")
  }

  conn.Close()

  state := conn.ConnectionState()
  certs := state.PeerCertificates

  go func() {
    var e error

    for _, cert := range certs {
      block := &pem.Block{
        Bytes: cert.Raw,
        Type: "CERTIFICATE",
      }

      err := pem.Encode(command.pipeWriter, block)
      e = errors.WrapErr(e, err)

      if err != nil {
        break
      }
    }

    e = errors.WrapErr(e, command.pipeWriter.Close())
    command.sync <- e
  }()

  return nil
}

func (command GetCerts) Usage() (*flags.Usage) {
  return command.usage
}

func (command GetCerts) Name() (string) {
  return command.name
}

func (command GetCerts) Description() (string) {
  return command.description
}

func (command GetCerts) Read(p []byte) (int, error) {
  return command.pipeReader.Read(p)
}

func (command GetCerts) Write(data []byte) (int, error) {
  return 0, io.EOF
}

func (command GetCerts) Close() (error) {
  var e error

  e = errors.WrapErr(e, <- command.sync)
  e = errors.WrapErr(e, command.pipeWriter.Close())
  return e
}

func (command *GetCerts) SetupFlags(set *flag.FlagSet) {
  command.flagSet = set
}

func (command *GetCerts) ParseFlags(options *flags.GlobalOptions) (error) {
  args := command.flagSet.Args()

  if len(args) == 0 {
    return errors.New("Address not found in the command line.")
  }

  addr, err := net.ResolveTCPAddr("tcp", args[0])
  if err != nil {
    return errors.Wrap(err, "Error parsing <host[:<port>]>")
  }

  if addr.IP.String() == "" {
    return errors.New("Address cannot be empty")
  }

  command.addr = addr
  options.Chomp = true

  return nil
}
