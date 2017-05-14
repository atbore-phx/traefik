package try

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// Condition is a retry condition function.
// It receives a response, and returns an error
// if the response failed the condition.
type Condition func(*http.Response) error

// BodyContains returns a retry condition function.
// The condition returns an error if the request body does not contain the given
// string.
func BodyContains(s string) Condition {
	return func(res *http.Response) error {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %s", err)
		}

		if !strings.Contains(string(body), s) {
			return fmt.Errorf("could not find '%s' in body '%s'", s, string(body))
		}
		return nil
	}
}

// StatusCodeIs returns a retry condition function.
// The condition returns an error if the given response's status code is not the
// given HTTP status code.
func StatusCodeIs(status int) Condition {
	return func(res *http.Response) error {
		if res.StatusCode != status {
			return fmt.Errorf("got status code %d, wanted %d", res.StatusCode, status)
		}
		return nil
	}
}
