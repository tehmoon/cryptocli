package main

import (
  "io"
)

type ByteCounter struct {
  startAt uint64
  stopAt uint64
  diff uint64
  read uint64
  wrote uint64
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
}

func (byteCounter *ByteCounter) Read(p []byte) (int, error) {
  if byteCounter.diff <= uint64(0) {
    return 0, io.EOF
  }

  if uint64(len(p)) <= byteCounter.diff {
    read, err := byteCounter.pipeReader.Read(p)
    if err != nil {
      return read, err
    }

    byteCounter.diff -= uint64(read)

    return read, nil
  }

  read, err := byteCounter.pipeReader.Read(p[:byteCounter.diff])
  if err != nil {
    return read, err
  }

  byteCounter.diff -= uint64(read)

  return read, nil
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
    diff: stopAt - startAt,
    read: 0,
    wrote: 0,
  }

  byteCounter.pipeReader, byteCounter.pipeWriter = io.Pipe()

  return byteCounter
}
