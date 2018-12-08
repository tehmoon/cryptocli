package main

import (
	"io"
	"bufio"
)

const (
	ReaderMaxPowerSize uint = 24
	ReaderMinPowerSize uint = 10
)

func ReadBytesStep(r io.Reader, cb func([]byte)) (error) {
	var (
		err error
		i int
		power = ReaderMinPowerSize
	)

	for {
		buff := make([]byte, 2 << power)

		i, err = io.ReadFull(r, buff)
		if err != nil {
			cb(buff[:i])
			break
		}

		cb(buff)

		if i == 2<<power && power != ReaderMaxPowerSize {
			power++
		}
	}

	return err
}

func ReadDelimStep(r io.Reader, delim byte, cb func([]byte)) (error) {
	var (
		reader = bufio.NewReader(r)
		b []byte
		err error
	)

	for {
		b, err = reader.ReadBytes(delim)
		if err != nil {
			cb(b)
			break
		}

		cb(b)
	}

	return err
}
