package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/enr/go-files/files"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

// Auth is auth
type Auth struct {
	Secret string
	// encrypted secret in base 64
	EncryptedSecret string `yaml:"encrypted_secret"`
	IdentityFile    string `yaml:"identity_file"`
}

// Endpoint is server
type Endpoint struct {
	Host string
	Port int
}

func (e *Endpoint) String() string {
	host := "localhost"
	if e.Host != "" {
		host = e.Host
	}
	return fmt.Sprintf("%s:%d", host, e.Port)
}

// SSHTunnelProcess implements RunpProcess.
type SSHTunnelProcess struct {
	WorkingDir string `yaml:"workdir"`
	Env        map[string]string
	Await      AwaitCondition

	User   string
	Auth   Auth
	Local  Endpoint
	Jump   Endpoint
	Target Endpoint
	// command executed to test connection to jump server
	TestCommand string `yaml:"test_command"`

	id            string
	vars          map[string]string
	secretKey     string
	preconditions []Precondition
	cmd           *SSHTunnelCommandWrapper
	stopTimeout   string
}

// ID for the sub process
func (p *SSHTunnelProcess) ID() string {
	return p.id
}

// SetPreconditions set preconditions.
func (p *SSHTunnelProcess) SetPreconditions(preconditions []Precondition) {
	p.preconditions = preconditions
}

// VerifyPreconditions check if process can be started
func (p *SSHTunnelProcess) VerifyPreconditions() error {
	var err error
	for _, p := range p.preconditions {
		err = p.Verify()
		if err != nil {
			return err
		}
	}

	if p.TestCommand != "" {
		cmdout, err := p.executeCmd(p.TestCommand)
		ui.Debugf("Test command %s :\n%s", p.TestCommand, cmdout.String())
		return err
	}
	return nil
}

func (p *SSHTunnelProcess) executeCmd(command string) (*bytes.Buffer, error) {
	config, err := p.resolveSSHCommandConfiguration()
	if err != nil {
		return nil, err
	}
	hostname := p.Jump.Host
	port := p.Jump.Port
	ui.Debugf("Test command %s on %s:%d", p.TestCommand, hostname, port)
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", hostname, port), config)
	if err != nil {
		return nil, err
	}
	session, err := conn.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run(command)
	return &stdoutBuf, nil
}

// StopTimeout duration to wait to force kill process
func (p *SSHTunnelProcess) StopTimeout() time.Duration {
	if p.stopTimeout != "" {
		d, err := time.ParseDuration(p.stopTimeout)
		if err != nil {
			return time.Duration(5) * time.Second
		}
		return d
	}
	return time.Duration(5) * time.Second
}

// SetID for the sub process
func (p *SSHTunnelProcess) SetID(id string) {
	p.id = id
}

// StartCommand ho
func (p *SSHTunnelProcess) StartCommand() (RunpCommand, error) {
	config, err := p.resolveSSHCommandConfiguration()
	if err != nil {
		return nil, err
	}
	p.cmd = &SSHTunnelCommandWrapper{
		config:        config,
		localAddress:  p.Local.String(),
		jumpAddress:   p.Jump.String(),
		targetAddress: p.Target.String(),
	}

	return p.cmd, nil
}

func (p *SSHTunnelProcess) resolveSSHCommandConfiguration() (*ssh.ClientConfig, error) {
	cliPreprocessor := newCliPreprocessor(p.vars)
	authMethods := []ssh.AuthMethod{}
	if p.Auth.IdentityFile != "" {
		aif := cliPreprocessor.process(p.Auth.IdentityFile)
		identityFile, err := resolvePath(aif, "")
		if err != nil {
			return nil, err
		}
		if !files.IsRegular(identityFile) {
			return nil, errors.New("Invalid identity file " + identityFile)
		}
		ui.Debugf("Connecting using identity file %s", identityFile)
		authMethods = append(authMethods, publicKeyFile(identityFile))
	}
	if p.Auth.Secret != "" {
		as := cliPreprocessor.process(p.Auth.Secret)
		authMethods = append(authMethods, ssh.Password(as))
		ui.Debugf("Connecting using secret")
	}
	if p.Auth.EncryptedSecret != "" {
		key := p.secretKey
		ui.WriteLinef("KEY %s", key)
		if key == "" {
			return nil, errors.New(`Missing key for "encrypted_secret"`)
		}
		secretB64 := p.Auth.EncryptedSecret
		secret, err := DecryptBase64(secretB64, key)
		if err != nil {
			return nil, err
		}
		ui.Debugf("Connecting using encrypted secret %s", secretB64)
		authMethods = append(authMethods, ssh.Password(string(secret)))
	}
	if len(authMethods) == 0 {
		return nil, errors.New("No Auth method set")
	}
	sshUser := cliPreprocessor.process(p.User)
	config := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return config, nil
}

// StopCommand ho
func (p *SSHTunnelProcess) StopCommand() RunpCommand {

	// fallito in inizializzazione
	if p.cmd == nil {
		return &SSHTunnelCommandStopper{
			cmd: &SSHTunnelCommandWrapper{},
		}
	}

	return &SSHTunnelCommandStopper{
		cmd: p.cmd,
	}
}

// Dir for the sub process
func (p *SSHTunnelProcess) Dir() string {
	return p.WorkingDir
}

// SetDir for the sub process
func (p *SSHTunnelProcess) SetDir(wd string) {
	p.WorkingDir = wd
}

// String representation of process
func (p *SSHTunnelProcess) String() string {
	return fmt.Sprintf("%T{id=%s}", p, p.ID())
}

// ShouldWait returns if the process has await set.
func (p *SSHTunnelProcess) ShouldWait() bool {
	return (p.Await.Resource != "")
}

// AwaitResource returns the await resource.
func (p *SSHTunnelProcess) AwaitResource() string {
	return p.Await.Resource
}

// AwaitTimeout returns the await timeout.
func (p *SSHTunnelProcess) AwaitTimeout() string {
	return p.Await.Timeout
}

// IsStartable always true.
func (p *SSHTunnelProcess) IsStartable() (bool, error) {
	return true, nil
}

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		ui.WriteLinef("Cannot read SSH public key file %s", file)
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		ui.WriteLinef("Cannot parse SSH public key file %s", file)
		return nil
	}
	return ssh.PublicKeys(key)
}
