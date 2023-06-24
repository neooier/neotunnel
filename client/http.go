package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"

	"nhooyr.io/websocket"
)

func newHTTPCLient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return tls.Dial(network, cdnAddr, &tls.Config{ServerName: fakeHost /*RootCAs: pool*/})
			},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial(network, cdnAddr)
			},
		},
	}
}

func NewWSConnection() (*websocket.Conn, error) {
	var u = url.URL{Scheme: "wss", Host: realHost, Path: "/proxy"}
	c, _, err := websocket.Dial(context.TODO(), u.String(),
		&websocket.DialOptions{
			HTTPClient: newHTTPCLient(),
			HTTPHeader: http.Header{
				"Auth": []string{"1"},
			},
		})
	return c, err
}
