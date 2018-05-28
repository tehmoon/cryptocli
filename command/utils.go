package command

import (
	"os"
	"io"
	"io/ioutil"
	"github.com/tehmoon/errors"
)

func CreateTempFile() (*os.File, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, errors.Wrap(err, "Error creating temporary file")
	}

	err = os.Remove(f.Name())
	if err != nil {
		f.Close()
		return nil, errors.Wrap(err, "Error removing temporary file")
	}

	return f, nil
}

func ExtractReadAt(reader io.ReaderAt, l int, off int64) ([]byte, error) {
	buff := make([]byte, l)

	read, err := reader.ReadAt(buff, off)
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
	}

	if read != len(buff) {
		return nil, io.ErrShortWrite
	}

	return buff, nil
}
