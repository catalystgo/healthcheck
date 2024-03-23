package misc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/catalystgo/healthcheck"
)

const (
	DNSResolveSuffix = "_dns_resolve"
	TCPDialSuffix    = "_tcp_dial"
	HTTPGetSuffix    = "_http_get"

	GoroutinesCount = "goroutines_threshold"
)

// DNSResolveCheck returns a checker checking that the host can resolve
// to at least one IP address during the timeout.
func DNSResolveCheck(host string, timeout time.Duration) healthcheck.Check {
	resolver := net.Resolver{}
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		addrs, err := resolver.LookupHost(ctx, host)
		if err != nil {
			return err
		}
		if len(addrs) < 1 {
			return fmt.Errorf("could not resolve host")
		}
		return nil
	}
}

// TCPDialCheck returns a Check that checks the TCP connection to
// the provided endpoint.
func TCPDialCheck(addr string, timeout time.Duration) healthcheck.Check {
	return func() error {
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			return err
		}
		return conn.Close()
	}
}

// HTTPGetCheck returns a checker that executes an HTTP GET request to the specified
// URL. The check fails if the request is timed out or returns any code but 200 OK.
func HTTPGetCheck(url string, timeout time.Duration) healthcheck.Check {
	client := http.Client{
		Timeout: timeout,
		// never follow redirects
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return func() error {
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("returned status %d", resp.StatusCode)
		}
		return nil
	}
}

// GoroutineCountCheck returns a checker that fails if
// too many goroutines are running (this may mean a resource leak).
func GoroutineCountCheck(threshold int) healthcheck.Check {
	return func() error {
		count := runtime.NumGoroutine()
		if count > threshold {
			return fmt.Errorf("too many goroutines (%d > %d)", count, threshold)
		}
		return nil
	}
}
