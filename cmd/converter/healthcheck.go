package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// cmdHealthcheck performs a lightweight HTTP health check against the local
// server. It is designed to be used as the Docker HEALTHCHECK command, so
// the container image does not need curl, wget, or any other external tool.
// This allows us to use a scratch base image (zero attack surface).
//
// Usage:  converter healthcheck [port]   (default 8080)
// Exit 0 = healthy, Exit 1 = unhealthy.
func cmdHealthcheck(args []string) {
	port := "8080"
	if len(args) > 0 {
		port = args[0]
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost:" + port + "/api/info")
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck failed: HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}

	os.Exit(0)
}
