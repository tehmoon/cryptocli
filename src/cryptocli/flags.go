package main

import (
	"github.com/spf13/pflag"
	"os"
	"log"
	"fmt"
	"github.com/tehmoon/errors"
	"io/ioutil"
)

type Flags struct {
	Modules *Modules
	Global GlobalFlags
}

type GlobalFlags struct {
	Std bool
}

func NewFlags() (*Flags) {
	return &Flags{
		Modules: NewModules(),
		Global: GlobalFlags{},
	}
}

func SetRootUsage(fs *pflag.FlagSet) (func ()) {
	return func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: [options] -- <module> [options] -- <module> [options] -- ...\n", os.Args[0])
		fs.PrintDefaults()

		fmt.Fprintln(os.Stderr, MODULELIST.Help())
	}
}

// Return new flagset from arguments and skip any errors
// This is just to initialize the next flagset to parse.
func ParseArgsQuiet(args []string) (*pflag.FlagSet) {
	fs := pflag.NewFlagSet("module", pflag.ContinueOnError)
	fs.SetOutput(ioutil.Discard)
	fs.ParseErrorsWhitelist.UnknownFlags = true

	fs.Parse(args)

	return fs
}

func ParseFlags() (*Flags, error) {
	flags := NewFlags()

	root := pflag.NewFlagSet("root", pflag.ContinueOnError)
	root.BoolVar(&flags.Global.Std, "std", false, "Read from stdin and writes to stdout instead of setting both modules")
	root.Usage = SetRootUsage(root)
	err := root.Parse(os.Args[1:])
	if err != nil {
		if err == pflag.ErrHelp {
			os.Exit(2)
		}

		return nil, err
	}

	remaining := root.ArgsLenAtDash()
	if remaining == -1 {
		return flags, nil
	}

	err = ParseRootRemainingArgs(flags.Modules, remaining, root)
	if err != nil {
		return nil, err
	}

	return flags, nil
}

// Exit if flag is requested
func ParseModuleArgs(name string, module Module, args []string) (error) {
	fs := pflag.NewFlagSet(fmt.Sprintf("module %q", name), pflag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	module.SetFlagSet(fs)

	err := fs.Parse(args)
	if err != nil {
		if err == pflag.ErrHelp {
			os.Exit(2)
		}
	}

	return err
}

// Parse the rest of the arguments and populate modules
func ParseRootRemainingArgs(modules *Modules, remaining int, root *pflag.FlagSet) (error) {
	for i := 0;; i++ {
		args := root.Args()[remaining:]
		if len(args) < 1 {
			break
		}

		name := args[0]

		if name != "--" {
			module, err := MODULELIST.Find(name)
			if err != nil {
				return errors.Wrapf(err, "Could not find module %q", name)
			}

			moduleArgs := make([]string, 0)

			if len(args) > 1 {
				moduleArgs = args[1:]
			}

			err = ParseModuleArgs(name, module, moduleArgs)
			if err != nil {
				log.Fatal(errors.Wrapf(err, "Error parsing flags for module %q", name))
			}

			modules.Register(module)
		}

		root = ParseArgsQuiet(args)

		remaining = root.ArgsLenAtDash()
		if remaining < 0 {
			break
		}
	}

	return nil
}

func SanetizeFlags(fs *pflag.FlagSet) ([]string) {
	args := fs.Args()

	if len(args) == 0 {
		return args
	}

	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			args = args[:i]
			break
		}
	}

	return args
}
