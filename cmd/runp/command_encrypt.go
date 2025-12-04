package main

import (
	"os"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

func doEncrypt(c *cli.Context) error {
	if c.Args().Len() != 1 {
		return exitErrorf(3, "Secret value parameter is required")
	}

	kev := c.String(`key-env`)
	key := c.String(`key`)
	if kev != "" && key != "" {
		return exitErrorf(3, "Options --key and --key-env are mutually exclusive")
	}
	if kev != "" {
		ev := os.Getenv(kev)
		if ev == "" {
			return exitErrorf(3, "Environment variable %s is empty", kev)
		}
		key = ev
	}
	if key == "" {
		ui.WriteLinef("No encryption key provided, generating random key")
		key = core.RandomKey()
	}
	ui.Debugf("Encrypting secret using key: %s", key)
	plain := c.Args().First()
	secret, err := core.EncryptToBase64([]byte(plain), key)
	if err != nil {
		return exitErrorf(3, "Encryption operation failed: %v", err)
	}
	ui.WriteLinef("Encrypted secret: %s", secret)
	return nil
}
