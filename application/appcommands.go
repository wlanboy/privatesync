package application

import (
	"errors"
	"flag"
	"fmt"
	"io"

	store "github.com/wlanboy/privatesync/store"
)

const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

/*RunCommands parses the arguments (without program name), runs the command and returns the exit code*/
func (sync *Sync) RunCommands(args []string) int {
	if len(args) == 0 {
		showHelp(sync.ErrOut)
		return exitUsage
	}
	cmd := args[0]
	switch cmd {
	case "help", "-h", "-help", "--help":
		showHelp(sync.Out)
		return exitOK
	case "crypt", "decrypt":
	default:
		fmt.Fprintf(sync.ErrOut, "unknown command %q\n\n", cmd)
		showHelp(sync.ErrOut)
		return exitUsage
	}

	var opts options
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(sync.ErrOut)
	configPath := fs.String("config", "", "path of the settings file")
	if cmd == "crypt" {
		fs.BoolVar(&opts.Prune, "prune", false, "remove encrypted files whose source file was deleted")
	} else {
		fs.BoolVar(&opts.Force, "force", false, "overwrite local files that are newer than the encrypted copy")
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(sync.ErrOut, "unexpected arguments: %v\n", fs.Args())
		return exitUsage
	}

	if err := sync.execute(cmd, *configPath, opts); err != nil {
		fmt.Fprintln(sync.ErrOut, "error:", err)
		return exitError
	}
	return exitOK
}

func (sync *Sync) execute(cmd, configPath string, opts options) error {
	if configPath == "" {
		path, legacy, err := store.DefaultSettingsPath()
		if err != nil {
			return err
		}
		if legacy {
			fmt.Fprintf(sync.ErrOut, "note: using %s from the current directory; move it to the user config directory (see README)\n", path)
		}
		configPath = path
	}

	sync.Storage = &store.Storage{SettingsPath: configPath}
	if err := sync.Storage.Load(sync.Prompter, sync.Out); err != nil {
		return err
	}
	settings := sync.Storage.Settings

	workdir, err := store.ExpandPath(settings.WorkPath)
	if err != nil {
		return err
	}
	syncdir, err := store.ExpandPath(settings.SyncPath)
	if err != nil {
		return err
	}
	if err := validatePaths(workdir, syncdir); err != nil {
		return err
	}

	key, err := deriveKey(sync.Storage.PlainSecret, settings.Salt)
	if err != nil {
		return err
	}

	var st stats
	if cmd == "crypt" {
		st, err = cryptTree(workdir, syncdir, key, opts, sync.Out)
	} else {
		st, err = decryptTree(workdir, syncdir, key, opts, sync.Out)
	}
	printStats(sync.Out, cmd, st)
	if err != nil {
		return err
	}
	if st.Failed > 0 {
		return fmt.Errorf("%d file(s) failed", st.Failed)
	}
	return nil
}

func printStats(out io.Writer, cmd string, st stats) {
	fmt.Fprintf(out, "%s: %d written, %d unchanged", cmd, st.Written, st.Unchanged)
	if st.Conflicts > 0 {
		fmt.Fprintf(out, ", %d skipped (local newer)", st.Conflicts)
	}
	if st.Removed > 0 {
		fmt.Fprintf(out, ", %d removed", st.Removed)
	}
	if st.Ignored > 0 {
		fmt.Fprintf(out, ", %d ignored", st.Ignored)
	}
	if st.Failed > 0 {
		fmt.Fprintf(out, ", %d failed", st.Failed)
	}
	fmt.Fprintln(out)
}

func showHelp(out io.Writer) {
	fmt.Fprintln(out, "Usage: privatesync <command> [flags]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "commands:")
	fmt.Fprintln(out, "  crypt    encrypt files from work directory to sync directory")
	fmt.Fprintln(out, "           --prune   remove encrypted files whose source file was deleted")
	fmt.Fprintln(out, "  decrypt  decrypt files from sync directory to work directory")
	fmt.Fprintln(out, "           --force   overwrite local files that are newer than the encrypted copy")
	fmt.Fprintln(out, "  help     show this help")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "common flags:")
	fmt.Fprintln(out, "  --config <file>  settings file (default: <user config dir>/privatesync/syncsettings.json)")
}
