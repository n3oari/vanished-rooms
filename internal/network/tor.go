package network

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/proxy"
)

func NewTorClient() (*http.Client, error) {
	torProxy := "127.0.0.1:9050"

	dialer, err := proxy.SOCKS5("tcp", torProxy, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}, nil
}

func CheckTorConnection(client *http.Client) {
	fmt.Println("Checking identity on TOR network...")
	resp, err := client.Get("https://check.torproject.org/api/ip")
	if err != nil {
		log.Printf("[!!!]: TOR does not appear to be routing correctly: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Print("Your TOR exit IP is: ")
	io.Copy(os.Stdout, resp.Body)
	fmt.Println()
}
