package main

import (
	"fmt"
	"os"

	"github.com/enr/runp/lib/core"
	"github.com/urfave/cli/v2"
)

func doConfigGet(c *cli.Context) error {
	key := c.Args().First()
	if key == "" {
		return exitErrorf(3, "Usage: runp config get <key>\n\nSupported keys:\n%s", core.SupportedSettingKeysHelp())
	}
	value, err := core.GetSettingValue(key)
	if err != nil {
		return exitErrorf(3, "%s", err.Error())
	}
	fmt.Fprintf(os.Stdout, "%s\n", value)
	return nil
}

func doConfigSet(c *cli.Context) error {
	args := c.Args()
	if args.Len() < 2 {
		return exitErrorf(3, "Usage: runp config set <key> <value>\n\nSupported keys:\n%s", core.SupportedSettingKeysHelp())
	}
	key := args.Get(0)
	value := args.Get(1)
	if err := core.SetSettingValue(key, value); err != nil {
		return exitErrorf(3, "%s", err.Error())
	}
	path, _ := core.EnvironmentSettingsPath()
	ui.WriteLinef("Set %s = %q  (%s)", key, value, path)
	return nil
}

func doConfigShow(c *cli.Context) error {
	path, err := core.EnvironmentSettingsPath()
	if err != nil {
		return exitErrorf(3, "Failed to resolve settings path: %s", err.Error())
	}

	ui.WriteLinef("Settings file: %s", path)
	ui.WriteLinef("")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			ui.WriteLinef("(file does not exist — no settings have been customized)")
			ui.WriteLinef("")
			ui.WriteLinef("Defaults:")
			for _, meta := range settingsMeta() {
				ui.WriteLinef("  %-20s  %q", meta.key, meta.def)
			}
			return nil
		}
		return exitErrorf(3, "Failed to read settings file: %s", err.Error())
	}

	fmt.Fprint(os.Stdout, string(data))
	return nil
}

type keyEntry struct {
	key string
	def string
}

func settingsMeta() []keyEntry {
	m := core.SupportedSettingKeys()
	out := make([]keyEntry, 0, len(m))
	for k, v := range m {
		out = append(out, keyEntry{key: k, def: v.Default})
	}
	return out
}
