package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	c := &http.Client{Timeout: 2 * time.Second}
	res, err := c.Get("http://127.0.0.1:8080/api/v1/health/ready")
	if err != nil {
		os.Exit(1)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
