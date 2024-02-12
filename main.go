package main

import (
    "fmt"
    "net/http"
    "os/exec"
    "strings"
    "time" // Import the time package
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    serviceStatus = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "service_status",
            Help: "Status of services",
        },
        []string{"name", "status", "lifetime", "memory", "enabled"},
    )
)

func getServiceStatus() {
    serviceStatus.Reset()

    cmd := exec.Command("systemctl", "list-units", "--type=service", "--state=active,inactive,failed", "--no-pager", "--no-legend")
    out, err := cmd.CombinedOutput()
    if err != nil {
        fmt.Println("Error running systemctl:", err)
        fmt.Println("Output:", string(out))
        return
    }

    lines := strings.Split(string(out), "\n")
    for _, line := range lines {
        fields := strings.Fields(line)
        if len(fields) >= 5 {
            // Extract relevant information
            serviceName := fields[0]
            lifetime := fields[2]
            memory := fields[3]

            // Check if service is enabled
            enabled := "unknown"
            if len(fields) >= 6 {
                if strings.Contains(fields[5], "enabled") {
                    enabled = "True"
                } else if strings.Contains(fields[5], "disabled") {
                    enabled = "False"
                }
            }

            // Determine the status value
            var statusValue float64
            if fields[2] == "inactive" || fields[2] == "failed" {
                statusValue = 0 // Set to 0 for inactive or failed
            } else {
                statusValue = 1 // Default to 1 for active
            }

            // Clear the metric for the stopped service before updating it
            serviceStatus.DeleteLabelValues(serviceName, fields[2], lifetime, memory, enabled)

            // Set the gauge value for the service
            serviceStatus.WithLabelValues(serviceName, fields[2], lifetime, memory, enabled).Set(statusValue)
        }
    }
}


func main() {
    prometheus.MustRegister(serviceStatus)

    // Serve /metrics endpoint
    http.Handle("/metrics", promhttp.Handler())

    // Periodically update service metrics
    go func() {
        for {
            getServiceStatus()
            // Sleep for some interval before updating again
            <-time.After(1 * time.Second)
        }
    }()

    fmt.Println("Exporter listening on :8875/metrics")
    http.ListenAndServe(":8875", nil)
}
