package kafka

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/catalystgo/healthcheck"
)

// Checker Name is the name of the Kafka checker for
// usage in liveness/readiness probes
const CheckerName = "kafka"

// DialCheck executes TCP dial to all Kafka endpoints
// and returns an error if all endpoints returned errors.
// If at least one node is alive, it will return OK.
func DialCheck(endpoints []string, timeout time.Duration) healthcheck.Check {
	return func() error {
		if len(endpoints) == 0 {
			return errors.New("empty kafka endpoints")
		}

		var errorsList []error

		for _, ep := range endpoints {
			conn, err := net.DialTimeout("tcp", ep, timeout)
			if err != nil {
				errorsList = append(errorsList, err)
				continue
			}

			err = conn.Close()
			if err != nil {
				errorsList = append(errorsList, err)
				continue
			}

			return nil
		}

		return fmt.Errorf("%s", errorsList)
	}
}
