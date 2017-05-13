package utils

import (
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/containous/traefik/log"
)

const (
	CITimeoutMultiplier = 3
	maxInterval         = 5 * time.Second
)

// TryGetResponse is like TryRequest, but returns the response for further
// processing at the call site.
// Conditions are not allowed since it would complicate signaling if the
// response body needs to be closed or not. Callers are expected to close on
// their own if the function returns a nil error.
func TryGetResponse(url string, timeout time.Duration) (*http.Response, error) {
	return tryGetResponse(url, timeout, nil)
}

// TryRequest is like Try, but runs a request against the given URL and applies
// the condition on the response.
// Condition may be nil, in which case only the request against the URL must
// succeed.
func TryRequest(url string, timeout time.Duration, condition Condition) error {
	_, err := tryGetResponse(url, timeout, condition)
	if err != nil {
		return err
	}

	return nil
}

// Try repeatedly executes an operation until no error condition occurs or the
// given timeout is reached, whatever comes first.
func Try(timeout time.Duration, operation func() error) error {
	if timeout <= 0 {
		panic("timeout must be larger than zero")
	}

	interval := time.Duration(math.Ceil(float64(timeout) / 10.0))
	if interval > maxInterval {
		interval = maxInterval
	}

	ci := os.Getenv("CI")
	if len(ci) > 0 {
		log.Println("Activate CI multiplier:", CITimeoutMultiplier)
		timeout = time.Duration(float64(timeout) * CITimeoutMultiplier)
	}

	var err error
	if err = operation(); err == nil {
		return nil
	}

	stopTime := time.Now().Add(timeout)

	for {
		if time.Now().After(stopTime) {
			fmt.Println("-")
			return fmt.Errorf("try operation failed: %s", err)
		}

		select {
		case <-time.Tick(interval):
			fmt.Print("*")
			if err = operation(); err == nil {
				fmt.Println("+")
				return nil
			}
		}
	}
}

// Sleep pauses the current goroutine for at least the duration d.
// Deprecated: Use only when other Try[...] function is not possible.
func Sleep(d time.Duration) {
	ci := os.Getenv("CI")
	if len(ci) > 0 {
		log.Println("Activate CI multiplier:", CITimeoutMultiplier)
		d = time.Duration(float64(d) * CITimeoutMultiplier)
	}
	time.Sleep(d)
}

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

func tryGetResponse(url string, timeout time.Duration, condition Condition) (*http.Response, error) {
	var resp *http.Response
	return resp, Try(timeout, func() error {
		var err error
		resp, err = http.Get(url)

		if err == nil && condition != nil {
			err = condition(resp)

			if resp.Body != nil {
				defer resp.Body.Close()
			}
		}

		return err
	})
}
