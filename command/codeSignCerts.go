package command

import (
	"encoding/pem"
	"encoding/binary"
	"encoding/asn1"
	"debug/pe"
	"os"
	"github.com/tehmoon/pkcs7"
	"io"
	"../flags"
	"github.com/tehmoon/errors"
	"flag"
)

type CodeSignCerts struct {
	name string
	description string
	usage *flags.Usage

	sync chan error

	inReader, outReader *io.PipeReader
	inWriter, outWriter *io.PipeWriter

	file *os.File
}

var ErrCodeSignCertsNoCertificateTableFound = errors.New("No Certificate Table found")

var DefaultCodeSignCerts = &CodeSignCerts{
	name: "code-sign-certs",
	description: "Parse a windows PE file and extract certificate in PEM form. It will bufferize the executable to disk instead of memory.",
	usage: &flags.Usage{
		CommandLine: "",
		Other: "",
	},
}

func (command *CodeSignCerts) Init() (error) {
	command.inReader, command.inWriter = io.Pipe()
	command.outReader, command.outWriter = io.Pipe()

	f, err := CreateTempFile()
	if err != nil {
		return err
	}

	command.sync = make(chan error)

	go StartCodeSignCerts(command.inReader, f, command.outWriter, command.sync)

	return nil
}

type CodeSignCertsCertificateTable struct {
	Identifier asn1.ObjectIdentifier
	RawCertificates asn1.RawValue
}

func (command CodeSignCerts) Usage() (*flags.Usage) {
	return command.usage
}

func (command CodeSignCerts) Name() (string) {
	return command.name
}

func (command CodeSignCerts) Description() (string) {
	return command.description
}

func (command CodeSignCerts) Read(p []byte) (int, error) {
	return command.outReader.Read(p)
}

func (command CodeSignCerts) Write(data []byte) (int, error) {
	return command.inWriter.Write(data)
}

func (command CodeSignCerts) Close() (error) {
	var e error

	e = errors.WrapErr(e, command.inWriter.Close())
	e = errors.WrapErr(e, <- command.sync)
	e = errors.WrapErr(e, command.outWriter.Close())

	return e
}

func (command *CodeSignCerts) SetupFlags(set *flag.FlagSet) {}

func (command *CodeSignCerts) ParseFlags(options *flags.GlobalOptions) (error) {
	options.Chomp = true

	return nil
}

func CodeSignCertsParseHeader64(header *pe.OptionalHeader64, file io.ReaderAt) ([]byte, *asn1.ObjectIdentifier, error) {
	return CodeSignCertsExtractCerts(header.DataDirectory, file)
}

func CodeSignCertsParseHeader32(header *pe.OptionalHeader32, file io.ReaderAt) ([]byte, *asn1.ObjectIdentifier, error) {
	return CodeSignCertsExtractCerts(header.DataDirectory, file)
}

func CodeSignCertsExtractCerts(dds [16]pe.DataDirectory, file io.ReaderAt) ([]byte, *asn1.ObjectIdentifier, error) {
	for i, dd := range dds {
		if i == 4 {
			if dd.VirtualAddress == 0 {
				continue
			}

			buff, err := ExtractReadAt(file, 4, int64(dd.VirtualAddress))
			if err != nil {
				return nil, nil, errors.Wrap(err, "Error reading certificate table's size")
			}

			start := dd.VirtualAddress + 8
			stop := start + binary.LittleEndian.Uint32(buff)

			buff, err = ExtractReadAt(file, int(stop - start - 8), int64(start))
			if err != nil {
				return nil, nil, errors.Wrap(err, "Error reading certificate table first entry")
			}

			ct := &CodeSignCertsCertificateTable{}
			_, err = asn1.Unmarshal(buff, ct)
			if err != nil {
				return nil, nil, errors.Wrap(err, "Error unmarshaling certificate table from ASN.1")
			}

			return buff, &ct.Identifier, nil
		}
	}

	return nil, nil, ErrCodeSignCertsNoCertificateTableFound
}

func StartCodeSignCerts(reader io.Reader, file *os.File, writer io.WriteCloser, sync chan error) {
	_, err := io.Copy(file, reader)
	if err != nil {
		sync <- errors.Wrap(err, "Error copying init reader to temporary file")
		return
	}

	f, err := pe.NewFile(file)
	if err != nil {
		file.Close()

		if err == io.EOF {
			sync <- nil
			return
		}

		sync <- errors.Wrap(err, "Error opening Portable Executable file")
		return
	}
	defer f.Close()

	var (
		buff []byte
		ob *asn1.ObjectIdentifier
	)

	switch header := f.OptionalHeader.(type) {
		case *pe.OptionalHeader32:
			buff, ob, err = CodeSignCertsParseHeader32(header, file)
		case *pe.OptionalHeader64:
			buff, ob, err = CodeSignCertsParseHeader64(header, file)
	}

	if err != nil {
		sync <- err
		return
	}

	err = CodeSignCertsParseCertificateTable(buff, ob, writer)
	if err != nil {
		if err != ErrCodeSignCertsNoCertificateTableFound {
			sync <- err
			return
		}

		sync <- nil
		return
	}

	sync <- nil
}

func CodeSignCertsParseCertificateTable(buff []byte, ob *asn1.ObjectIdentifier, writer io.Writer) (error) {
	if ob.Equal(asn1.ObjectIdentifier{1,2,840,113549,1,7,2}) {
		p7, err := pkcs7.Parse(buff)
		if err != nil {
			return errors.Wrap(err, "Error parsing Certificate Table entry to PKCS7")
		}

		for _, cert := range p7.Certificates {
			block := &pem.Block{
				Type: "CERTIFICATE",
				Bytes: cert.Raw,
			}

			err := pem.Encode(writer, block)
			if err != nil {
				return errors.Wrap(err, "Error encoding certificate to PEM")
			}
		}

		return nil
	}

	return errors.Errorf("Certificate Table's entry type of %s is not supported\n", ob.String())
}
