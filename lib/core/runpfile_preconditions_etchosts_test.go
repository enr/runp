package core

import (
	"testing"
)

const etcHosts01 = `
# Copyright (c) 1993-2009 Microsoft Corp.
#
# This is a sample HOSTS file used by Microsoft TCP/IP for Windows.
#
# This file contains the mappings of IP addresses to host names. Each
# entry should be kept on an individual line. The IP address should
# be placed in the first column followed by the corresponding host name.
# The IP address and the host name should be separated by at least one
# space.
#
# Additionally, comments (such as these) may be inserted on individual
# lines or following the machine name denoted by a '#' symbol.
#
# For example:
#
#      102.54.94.97     rhino.acme.com          # source server
#       38.25.63.10     x.acme.com              # x client host

# localhost name resolution is handled within DNS itself.
#   127.0.0.1       localhost
#   ::1             localhost
# Added by Docker Desktop
# 192.168.22.134 host.docker.internal
# 192.168.22.134 gateway.docker.internal

127.0.0.1 private-host-01
127.0.0.1 private-host-02
# To allow the same kube context to work on the host and the container:
# 127.0.0.1 kubernetes.docker.internal
# End of section
127.0.0.1 domain-01
192.168.2.2				domain-03
102.54.99.99     rhino.acme.com
`

func TestEtcHostsOk(t *testing.T) {

	wanted := map[string][]string{}
	wanted[`127.0.0.1`] = []string{
		`private-host-01`,
		`private-host-02`,
		`domain-01`,
	}
	wanted[`192.168.2.2`] = []string{
		`domain-03`,
	}
	wanted[`102.54.99.99`] = []string{
		`rhino.acme.com`,
	}

	ui = CreateMainLogger(" ", 6, "%s> ", true, false)

	sut := &EtcHostsPrecondition{
		Contains: wanted,
		etcHostsReader: func() ([]byte, error) {
			return []byte(etcHosts01), nil
		},
	}
	res := sut.Verify()
	if res.Vote != Proceed {
		t.Errorf("Expected Proceed but got %v: %v", res.Vote, res.Reasons)
	}
}

func TestEtcHostsKo(t *testing.T) {

	wanted := map[string][]string{}
	wanted[`127.0.0.1`] = []string{
		`no.such.host`,
		// correct but not enough to satisfy precondition
		`domain-01`,
	}
	wanted[`192.168.22.134`] = []string{
		`host.docker.internal`,
	}

	ui = CreateMainLogger(" ", 6, "%s> ", true, false)

	sut := &EtcHostsPrecondition{
		Contains: wanted,
		etcHostsReader: func() ([]byte, error) {
			return []byte(etcHosts01), nil
		},
	}
	res := sut.Verify()
	if res.Vote != Stop {
		t.Errorf("Expected Stop but got %v", res.Vote)
	}
}
