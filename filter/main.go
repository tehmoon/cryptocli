package filter

import (
  "io"
  "net/url"
  "github.com/tehmoon/errors"
  "strings"
)

type Filter interface {
  io.ReadWriteCloser
  Init() (error)
}

type Initializer interface {
  Name() (string)
  Description() (string)
  Filter(*url.URL) (Filter)
}

var (
  FilterList []Initializer = []Initializer{
    DefaultPem,
  }
  ErrFilterParseEmpty = errors.New("empty filter")
  ErrFilterNotImplemented = errors.New("filter not implemented")
)

func Parse(filter string) (Filter, error) {
  if filter == "" {
    return nil, ErrFilterParseEmpty
  }

  uri, err := url.Parse(filter)
  if err != nil {
    return nil, errors.Wrap(err, "Error parsing filter to URL")
  }

  switch uri.Scheme {
    case "pem":
      return DefaultPem.Filter(uri), nil
  }

  switch filter {
    case "pem":
      return DefaultPem.Filter(nil), nil
  }

  return nil, ErrFilterNotImplemented
}

func ParseAll(str string) ([]Filter, error) {
  fs := strings.Split(str, ",")

  if len(fs) == 0 {
    return nil, ErrFilterParseEmpty
  }

  filters := make([]Filter, 0)
  for _, f := range fs {
    filter, err := Parse(f)
    if err != nil {
      return nil, errors.Wrapf(err, "Error parsing filter %s", f)
    }

    filters = append(filters, filter)
  }

  return filters, nil
}
