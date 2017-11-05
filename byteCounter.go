package main

import (
  "io"
)

type ByteCounter struct {
  startAt uint64
  stopAt uint64
  read uint64
  wrote uint64
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
}

func (byteCounter *ByteCounter) Read(p []byte) (int, error) {
  if byteCounter.stopAt != 0 {
    if byteCounter.read > byteCounter.stopAt {
      return 0, io.EOF
    }

    if byteCounter.read + uint64(len(p)) > byteCounter.stopAt {
      toRead := byteCounter.stopAt - byteCounter.read
      read, err := byteCounter.pipeReader.Read(p[:toRead])
      if err != nil {
        if err != io.EOF {
          return read, err
        }
      }

      return read, io.EOF
    }

    read, err := byteCounter.pipeReader.Read(p)
    byteCounter.read += uint64(read)
    return read, err
  }

  return byteCounter.pipeReader.Read(p)
}

func (byteCounter *ByteCounter) Write(data []byte) (int, error) {
  if byteCounter.wrote < byteCounter.startAt {
    if byteCounter.wrote + uint64(len(data)) > byteCounter.startAt {
      toWrite := (byteCounter.wrote + uint64(len(data))) - byteCounter.startAt

      wrote, err := byteCounter.pipeWriter.Write(data[int(uint64(len(data)) - toWrite):])
      written := uint64(wrote) + (uint64(len(data)) - toWrite)
      byteCounter.wrote += written
      return int(written), err
    }

    return len(data), nil
  }

  return byteCounter.pipeWriter.Write(data)
}

func (byteCounter *ByteCounter) Close() (error) {
  return byteCounter.pipeWriter.Close()
}

func newByteCounter(startAt, stopAt uint64) (io.ReadWriteCloser) {
  byteCounter := &ByteCounter{
    startAt: startAt,
    stopAt: stopAt,
    read: 0,
    wrote: 0,
  }

  byteCounter.pipeReader, byteCounter.pipeWriter = io.Pipe()

  return byteCounter
}
