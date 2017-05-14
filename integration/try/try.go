package try

import (
	"fmt"
	"github.com/containous/traefik/log"
	"math"
	"net/http"
	"os"
	"time"
)

const (
	// CITimeoutMultiplier is the multiplier for all timeout in the CI
	CITimeoutMultiplier = 3
	maxInterval         = 5 * time.Second
)

type timedAction func(timeout time.Duration, operation func() error) error

// Sleep pauses the current goroutine for at least the duration d.
// Deprecated: Use only when use an other Try[...] functions is not possible.
func Sleep(d time.Duration) {
	d = applyCIMultiplier(d)
	time.Sleep(d)
}

// Response is like Request, but returns the response for further
// processing at the call site.
// Conditions are not allowed since it would complicate signaling if the
// response body needs to be closed or not. Callers are expected to close on
// their own if the function returns a nil error.
func Response(req *http.Request, timeout time.Duration) (*http.Response, error) {
	return doTry(req, timeout)
}

// ResponseUntilStatusCode is like Request, but returns the response for further
// processing at the call site.
// Conditions are not allowed since it would complicate signaling if the
// response body needs to be closed or not. Callers are expected to close on
// their own if the function returns a nil error.
func ResponseUntilStatusCode(req *http.Request, timeout time.Duration, statusCode int) (*http.Response, error) {
	return doTry(req, timeout, StatusCodeIs(statusCode))
}

// GetRequest is like Do, but runs a request against the given URL and applies
// the condition on the response.
// Condition may be nil, in which case only the request against the URL must
// succeed.
func GetRequest(url string, timeout time.Duration, conditions ...Condition) error {
	resp, err := doTryGet(url, timeout, conditions...)

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	return err
}

// Request is like Do, but runs a request against the given URL and applies
// the condition on the response.
// Condition may be nil, in which case only the request against the URL must
// succeed.
func Request(req *http.Request, timeout time.Duration, conditions ...Condition) error {
	resp, err := doTry(req, timeout, conditions...)

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	return err
}

// Do repeatedly executes an operation until no error condition occurs or the
// given timeout is reached, whatever comes first.
func Do(timeout time.Duration, operation func() error) error {
	if timeout <= 0 {
		panic("timeout must be larger than zero")
	}

	interval := time.Duration(math.Ceil(float64(timeout) / 15.0))
	if interval > maxInterval {
		interval = maxInterval
	}

	timeout = applyCIMultiplier(timeout)

	var err error
	if err = operation(); err == nil {
		fmt.Println("+")
		return nil
	}
	fmt.Print("*")

	stopTimer := time.NewTimer(timeout)
	retryTick := time.NewTicker(interval)

	for {
		select {
		case <-stopTimer.C:
			fmt.Println("-")
			stopTimer.Stop()
			retryTick.Stop()
			return fmt.Errorf("try operation failed: %s", err)
		case <-retryTick.C:
			fmt.Print("*")
			if err = operation(); err == nil {
				fmt.Println("+")
				stopTimer.Stop()
				retryTick.Stop()
				return err
			}
		}
	}
}

func doTryGet(url string, timeout time.Duration, conditions ...Condition) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return doTry(req, timeout, conditions...)
}

func doTry(request *http.Request, timeout time.Duration, conditions ...Condition) (*http.Response, error) {
	return doRequest(Do, timeout, request, conditions...)
}

func doRequest(action timedAction, timeout time.Duration, request *http.Request, conditions ...Condition) (*http.Response, error) {
	var resp *http.Response
	return resp, action(timeout, func() error {
		var err error
		client := &http.Client{}

		resp, err = client.Do(request)

		for _, condition := range conditions {
			if err == nil && condition != nil {
				err := condition(resp)
				if err != nil {
					return err
				}
			}
		}

		return err
	})
}

func applyCIMultiplier(timeout time.Duration) time.Duration {
	ci := os.Getenv("CI")
	if len(ci) > 0 {
		log.Debug("Apply CI multiplier:", CITimeoutMultiplier)
		return time.Duration(float64(timeout) * CITimeoutMultiplier)
	}
	return timeout
}
