package inout

import (
	"net/http"
	"io"
	"net/url"
	"github.com/pkg/errors"
)

func copyDefaultHTTPTransport()(*http.Transport) {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if ! ok {
		return nil
	}

	transport := &http.Transport{}

	*transport = *defaultTransport

	return transport
}

func readHTTP(uri url.URL, client *http.Client) (io.ReadCloser, error) {
	user := uri.User
	uri.User = nil

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating request")
	}

	if user != nil {
		password, _ := user.Password()
		req.SetBasicAuth(user.Username(), password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Error sending request")
	}

	if resp.StatusCode < 200 && resp.StatusCode > 400 {
		return nil, errors.Errorf("Status code error %d\n", resp.StatusCode)
	}

	return resp.Body, nil
}

func writeHTTP(uri url.URL, client *http.Client, sync chan error) (io.WriteCloser, error) {
	user := uri.User
	uri.User = nil

	reader, writer := io.Pipe()

	req, err := http.NewRequest("POST", uri.String(), reader)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating request")
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	if user != nil {
		password, _ := user.Password()
		req.SetBasicAuth(user.Username(), password)
	}

	go func() {
		resp, err := client.Do(req)
		if err != nil {
			sync <- errors.Wrap(err, "Error sending request")
		}

		if resp.StatusCode < 200 && resp.StatusCode > 400 {
			sync <- errors.Errorf("Status code error %d\n", resp.StatusCode)
		}

		sync <- nil
	}()

	return writer, nil
}

func httpRedirectPolicy() (func (*http.Request, []*http.Request) (error)) {
	maxRedirect := 3

	return func (req *http.Request, via []*http.Request) (error) {
		if len(via) > 0 {
			lastReq := via[len(via) - 1]

			if req.URL.Scheme != lastReq.URL.Scheme {
				return errors.Errorf("URL scheme changed from %s to %s. Aborting.\n", lastReq.URL.Scheme, req.URL.Scheme)
			}
		}

		if len(via) == maxRedirect {
			return errors.Errorf("Max redirect reached\n")
		}

		return nil
	}
}
