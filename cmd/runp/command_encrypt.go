package main

import (
	"io"
	"os"
	"strings"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

func doEncrypt(c *cli.Context) error {
	plain, err := resolveSecret(c)
	if err != nil {
		return err
	}

	kev := c.String(`key-env`)
	key := c.String(`key`)
	if kev != "" && key != "" {
		return exitErrorf(exitCodeArg, "Options --key and --key-env are mutually exclusive")
	}
	if kev != "" {
		ev := os.Getenv(kev)
		if ev == "" {
			return exitErrorf(exitCodeArg, "Environment variable %s is empty", kev)
		}
		key = ev
	}
	if key == "" {
		ui.WriteLinef("No encryption key provided, generating random key")
		key = core.RandomKey()
	}
	ui.Debugf("Encrypting secret using key: %s", key)
	secret, err := core.EncryptToBase64([]byte(plain), key)
	if err != nil {
		return exitErrorf(exitCodeExec, "Encryption operation failed: %v", err)
	}
	ui.WriteLinef("Encrypted secret: %s", secret)
	return nil
}

// resolveSecret returns the plaintext secret from either the positional
// argument or stdin (when stdin is a pipe). Returns an error when neither
// source provides a value.
func resolveSecret(c *cli.Context) (string, error) {
	switch c.Args().Len() {
	case 1:
		return c.Args().First(), nil
	case 0:
		fi, err := os.Stdin.Stat()
		if err != nil || (fi.Mode()&os.ModeCharDevice) != 0 {
			return "", exitErrorf(exitCodeArg,
				"Secret value required: pass it as an argument or pipe it via stdin\n  Example: echo mysecret | runp encrypt --key-env RUNP_KEY")
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", exitErrorf(exitCodeExec, "Failed to read secret from stdin: %v", err)
		}
		plain := strings.TrimRight(string(data), "\r\n")
		if plain == "" {
			return "", exitErrorf(exitCodeArg, "Secret value is empty: stdin contained no data")
		}
		return plain, nil
	default:
		return "", exitErrorf(exitCodeArg, "Too many arguments: expected one secret value, got %d", c.Args().Len())
	}
}
