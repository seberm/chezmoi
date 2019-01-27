package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	vfs "github.com/twpayne/go-vfs"
)

// initCmd represents the init command
var initCommand = &cobra.Command{
	Args:  cobra.ExactArgs(1),
	Use:   "init repo",
	Short: "Initial setup of the source directory then update the destination directory to match the target state",
	Long: `Initial setup of the source directory then update the destination directory to match the target state.

This command is supposed to run once when you want to setup your dotfiles on a
new host. It will clone the given repository into your source directory (see --source flag)
and make sure that all directory permissions are correct.

After your source directory was checked out and setup (e.g. git submodules) this
command will automatically invoke the "apply" command to update the destination
directory. You can use the --apply=false flag to prevent this from happening.
`,
	Example: `
  # Checkout from github using the public HTTPS API
  chezmoi init https://github.com/example/dotfiles.git

  # Checkout from github using your private key
  chezmoi init git@github.com:example/dotfiles.git
`,
	RunE: makeRunE(config.runInitCommand),
}

type initCommandConfig struct {
	apply bool
}

func init() {
	rootCommand.AddCommand(initCommand)

	persistentFlags := initCommand.PersistentFlags()
	persistentFlags.BoolVar(&config.init.apply, "apply", true, "update destination directory")
}

func (c *Config) runInitCommand(fs vfs.FS, args []string) error {
	vcsInfo, err := c.getVCSInfo()
	if err != nil {
		return err
	}
	if vcsInfo.cloneArgsFunc == nil {
		return fmt.Errorf("%s: cloning not supported", c.SourceVCS.Command)
	}

	mutator := c.getDefaultMutator(fs)

	if err := c.ensureSourceDirectory(fs, mutator); err != nil {
		return err
	}

	cloneArgs := vcsInfo.cloneArgsFunc(args[0], c.SourceDir)
	if err := c.run("", c.SourceVCS.Command, cloneArgs...); err != nil {
		return err
	}

	// FIXME this should be part of struct vcs
	switch filepath.Base(c.SourceVCS.Command) {
	case "git":
		if _, err := fs.Stat(filepath.Join(c.SourceDir, ".gitmodules")); err == nil {
			for _, args := range [][]string{
				[]string{"submodule", "init"},
				[]string{"submodule", "update"},
			} {
				if err := c.run(c.SourceDir, c.SourceVCS.Command, args...); err != nil {
					return err
				}
			}
		}
	}

	if c.init.apply {
		if err := c.applyArgs(fs, nil, mutator); err != nil {
			return err
		}
	}

	return nil
}
