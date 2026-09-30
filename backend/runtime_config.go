package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func listenAddress(value string) (string, error) {
	if value == "" {
		return "127.0.0.1:8080", nil
	}
	host, port, err := net.SplitHostPort(value)
	n, parseErr := strconv.Atoi(port)
	if err != nil || parseErr != nil || n < 1 || n > 65535 || strings.TrimSpace(host) != host || strings.ContainsAny(host, "/?#@") {
		return "", fmt.Errorf("invalid LISTEN_ADDR: expected host:port with port 1..65535")
	}
	return value, nil
}
