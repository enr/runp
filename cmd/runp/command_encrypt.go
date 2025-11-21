package main

import (
	"os"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

func doEncrypt(c *cli.Context) error {
	if c.Args().Len() != 1 {
		return exitErrorf(3, "Secret parameter is required")
	}

	kev := c.String(`key-env`)
	key := c.String(`key`)
	if kev != "" && key != "" {
		return exitErrorf(3, "key and key-env used: they are mutually exclusive")
	}
	if kev != "" {
		ev := os.Getenv(kev)
		if ev == "" {
			return exitErrorf(3, `key-env "%s" empty`, kev)
		}
		key = ev
	}
	if key == "" {
		ui.WriteLinef("No key set, a random value will be used")
		key = core.RandomKey()
	}
	ui.Debugf("Secret encrypted using key %s", key)
	plain := c.Args().First()
	secret, err := core.EncryptToBase64([]byte(plain), key)
	if err != nil {
		return exitErrorf(3, "Encryption failed: %v", err)
	}
	ui.WriteLinef("Secret: %s", secret)
	return nil
}
