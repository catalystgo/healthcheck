# Healthcheck 🩺

This package provides a powerful interface for managing liveness and readiness checks within your Kubernetes deployments. Ensuring the health and readiness of your applications is crucial for maintaining their reliability and availability in production environments. With this package, you can easily integrate healthcheck handlers into your Kubernetes deployments to monitor the status of your applications.

### Contribution:
Contributions are welcome! If you encounter any issues or have suggestions for improvements, please feel free to open an issue or submit a pull request.

Let's keep our applications healthy and ready to serve! 💪🚀

---

### Checker:

There are 2 types of k8s checks probes

- **Liveness Checks**: Monitor the health of your application instances to ensure continuous operation.
- **Readiness Checks**: Determine when application instances are ready to serve traffic to users.

### Documentation:

For detailed documentation on how to use the Healthcheck Package for Kubernetes, please refer to the [external documentation](https://your-documentation-link-here.com).

### Example Code:

```go
package main

import (
    "net/http"
    "github.com/catalystgo/healthcheck"
)

func main() {
    // Create a new healthcheck handler
    handler := healthcheck.NewHandler()

    // Add liveness and readiness checks
    handler.AddLivenessCheck("database", func() error {
        // Check database connection
        // Return nil if connection is successful, otherwise return an error
    })

    handler.AddReadinessCheck("cache", func() error {
        // Check cache availability
        // Return nil if cache is available, otherwise return an error
    })

    // Add check error handler (optional)
    handler.AddCheckErrorHandler(func(name string, err error) {
        // Handle check error
        // Log the error or take appropriate action
    })

    // Serve healthcheck endpoints
    http.Handle(healthcheck.LivenessHandlerPath, handler)
    http.Handle(healthcheck.ReadinessHandlerPath, handler)
    http.ListenAndServe(":8080", nil)
}
```
