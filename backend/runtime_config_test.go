package main

import "testing"

func TestListenAddress(t *testing.T) {
	for _, value := range []string{"", "127.0.0.1:8080", ":9000", "0.0.0.0:80", "[::1]:8080"} {
		got, err := listenAddress(value)
		if err != nil {
			t.Fatal(value, err)
		}
		if value == "" && got != "127.0.0.1:8080" {
			t.Fatal(got)
		}
	}
	for _, value := range []string{"8080", "localhost:0", "localhost:65536", "localhost:abc", "http://localhost:80", " user:80", "localhost:80/path"} {
		if _, err := listenAddress(value); err == nil {
			t.Fatal("accepted", value)
		}
	}
}
