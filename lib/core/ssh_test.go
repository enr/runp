//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package core

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"dev.justinjudd.org/justin/easyssh"
	"golang.org/x/crypto/ssh"
)

/*
Utility functions to test SSH connections
*/

func testListener(e Endpoint, t *testing.T) (net.Listener, string, int) {
	listener, err := net.Listen("tcp", e.String())
	if err != nil {
		t.Fatal(err)
	}

	host, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	return listener, host, port
}

func sshServerConfig(user, secret, key string) (*ssh.ServerConfig, error) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == user && string(pass) == secret {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
	}
	keyBytes, err := ioutil.ReadFile(key)
	if err != nil {
		return nil, err
	}
	hostKey, _ := ssh.ParsePrivateKey(keyBytes)
	config.AddHostKey(hostKey)
	return config, nil
}

func startSSHServer(listener net.Listener, config *ssh.ServerConfig, t *testing.T) {
	s := &easyssh.Server{}
	s.Config = config

	handler := easyssh.NewStandardSSHServerHandler()
	channelHandler := easyssh.NewChannelsMux()

	r := easyssh.NewGlobalMultipleRequestsMux()

	r.HandleRequestFunc(easyssh.RemoteForwardRequest, easyssh.TCPIPForwardRequest)
	channelHandler.HandleChannel(easyssh.SessionRequest, easyssh.SessionHandler())

	channelHandler.HandleChannel(easyssh.DirectForwardRequest, easyssh.DirectPortForwardHandler())
	handler.MultipleChannelsHandler = channelHandler

	s.Handler = handler
	go func() {
		s.Serve(listener)
	}()
}

func httpServer(e Endpoint, stubResponse string, t *testing.T) *http.Server {
	ts := &http.Server{
		Addr: e.String(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, stubResponse)
		}),
		ReadTimeout:    2 * time.Second,
		WriteTimeout:   2 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		ts.ListenAndServe()

	}()
	return ts
}
