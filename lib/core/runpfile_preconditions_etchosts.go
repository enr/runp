package core

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

var defaultEtcHostsReader = func() ([]byte, error) {
	ehp := `/etc/hosts`
	if runtime.GOOS == "windows" {
		ehp = `c:\Windows\System32\Drivers\etc\hosts`
	}
	return os.ReadFile(ehp)
}

// EtcHostsPrecondition verify /etc/hosts contains given host.
type EtcHostsPrecondition struct {
	Contains map[string][]string

	etcHostsReader func() ([]byte, error)
}

// Verify ...
func (p *EtcHostsPrecondition) Verify() PreconditionVerifyResult {
	if p.etcHostsReader == nil {
		p.etcHostsReader = defaultEtcHostsReader
	}
	etcHosts, err := p.etcHostsReader()
	if err != nil {
		return PreconditionVerifyResult{
			Vote:    Stop,
			Reasons: []string{err.Error()},
		}
	}
	hosts, err := parseEtcHosts(etcHosts)
	for k, values := range p.Contains {
		mapped, exist := hosts[k]
		if !exist {
			return PreconditionVerifyResult{
				Vote:    Stop,
				Reasons: []string{fmt.Sprintf(`No mapping found for "%s"`, k)},
			}
		}
		for _, v := range values {
			if sliceContains(mapped, v) {
				continue
			}
			return PreconditionVerifyResult{
				Vote:    Stop,
				Reasons: []string{fmt.Sprintf(`Hosts mapping "%s" %v does not contain "%s"`, k, mapped, v)},
			}
		}
	}
	return PreconditionVerifyResult{
		Vote:    Proceed,
		Reasons: []string{},
	}
}

// IsSet ...
func (p *EtcHostsPrecondition) IsSet() bool {
	return len(p.Contains) > 0
}

func parseEtcHosts(hostsFileContent []byte) (map[string][]string, error) {
	hostsMap := map[string][]string{}
	for _, line := range strings.Split(strings.Trim(string(hostsFileContent), " \t\r\n"), "\n") {
		line = strings.Replace(strings.Trim(line, " \t"), "\t", " ", -1)
		if len(line) == 0 || line[0] == ';' || line[0] == '#' {
			continue
		}
		pieces := strings.SplitN(line, " ", 2)
		if len(pieces) > 1 && len(pieces[0]) > 0 {
			if names := strings.Fields(pieces[1]); len(names) > 0 {
				if _, ok := hostsMap[pieces[0]]; ok {
					hostsMap[pieces[0]] = append(hostsMap[pieces[0]], names...)
				} else {
					hostsMap[pieces[0]] = names
				}
			}
		}
	}
	return hostsMap, nil
}
