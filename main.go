package main

import (
    "fmt"
    "net/http"
    "os/exec"
    "strings"
    "time" // Import the time package
    "regexp"
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
    out, err := exec.Command("systemctl", "list-units", "--type=service", "--state=active", "--no-pager", "--no-legend").Output()
    if err != nil {
        fmt.Println("Error running systemctl:", err)
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

            // Check if service name matches the pattern to skip
            skipPattern := "systemd-.*\\.service"
            matched, err := regexp.MatchString(skipPattern, serviceName)
            if err != nil {
                fmt.Println("Error matching service name pattern:", err)
                continue
            }
            if matched {
                fmt.Println("Skipping service:", serviceName)
                continue
            }

            // Check if service is enabled
            enabled := "unknown"
            if len(fields) >= 6 {
                if strings.Contains(fields[5], "enabled") {
                    enabled = "True"
                } else if strings.Contains(fields[5], "disabled") {
                    enabled = "False"
                }
            }

            // Set gauge value for running services
            serviceStatus.WithLabelValues(serviceName, "running", lifetime, memory, enabled).Set(1)
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
            <-time.After(1 * time.Minute)
        }
    }()

    fmt.Println("Exporter listening on :8875/metrics")
    http.ListenAndServe(":8875", nil)
}
