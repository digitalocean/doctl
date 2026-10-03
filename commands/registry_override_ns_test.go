package commands

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression test for https://github.com/digitalocean/doctl/issues/1749.
//
// Every display subcommand under `registry repository`, `registry
// garbage-collection`, `registries repository` and `registries
// garbage-collection` carries an overrideNS so the two command trees do not
// share viper keys. The --format and --no-header flags are bound inside
// CmdBuilder, so the override must be in place before CmdBuilder binds them;
// otherwise the flags land under the unprefixed namespace while Display reads
// them from the overridden one and both flags are silently ignored.
func TestRegistryDisplayCommandsHonorFormatFlags(t *testing.T) {
	trees := []struct {
		desc     string
		build    func() *Command
		ns       string
		commands map[string]string
	}{
		{
			desc:  "registry repository",
			build: Repository,
			ns:    "registry.repository",
			commands: map[string]string{
				"list":           "Name",
				"list-v2":        "Name",
				"list-tags":      "Tag",
				"list-manifests": "Digest",
			},
		},
		{
			desc:  "registry garbage-collection",
			build: GarbageCollection,
			ns:    "registry.garbage-collection",
			commands: map[string]string{
				"start":      "UUID",
				"get-active": "UUID",
				"list":       "UUID",
			},
		},
		{
			desc:  "registries repository",
			build: RegistriesRepository,
			ns:    "registries.repository",
			commands: map[string]string{
				"list":           "Name",
				"list-v2":        "Name",
				"list-tags":      "Tag",
				"list-manifests": "Digest",
			},
		},
		{
			desc:  "registries garbage-collection",
			build: RegistriesGarbageCollection,
			ns:    "registries.garbage-collection",
			commands: map[string]string{
				"start":      "UUID",
				"get-active": "UUID",
				"list":       "UUID",
			},
		},
	}

	for _, tree := range trees {
		root := tree.build()
		seen := map[string]bool{}
		for _, child := range root.ChildCommands() {
			sample, ok := tree.commands[child.Name()]
			if !ok {
				continue
			}
			seen[child.Name()] = true

			require.NotNil(t, child.Flags().Lookup("format"), "%s %s should accept --format", tree.desc, child.Name())
			require.NotNil(t, child.Flags().Lookup("no-header"), "%s %s should accept --no-header", tree.desc, child.Name())
			require.NoError(t, child.Flags().Set("format", sample))
			require.NoError(t, child.Flags().Set("no-header", "true"))

			// Display resolves these under cmdNS(child); if the flags were
			// bound anywhere else the user's --format/--no-header is lost.
			assert.Equal(t, sample, viper.GetString(tree.ns+"."+child.Name()+".format"),
				"%s %s: --format was not bound under the runtime namespace", tree.desc, child.Name())
			assert.True(t, viper.GetBool(tree.ns+"."+child.Name()+".no-header"),
				"%s %s: --no-header was not bound under the runtime namespace", tree.desc, child.Name())
		}
		for name := range tree.commands {
			assert.True(t, seen[name], "%s should have a %q subcommand", tree.desc, name)
		}
	}
}
