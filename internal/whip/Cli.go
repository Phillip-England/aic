package whip

import (
	"os"

	"github.com/phillip-england/aic/internal/wherr"
)

type Cli struct {
	Source         string
	Args           map[string]*Arg
	Flags          map[string]*Arg
	Commands       map[string]CommandFactory
	DefaultFactory CommandFactory
	DefaultCmd     Cmd
	Store          map[string]any
	Cwd            string
}

func New(factory CommandFactory) (Cli, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Cli{}, wherr.Consume(wherr.Here(), err, "")
	}
	osArgs := os.Args
	flags := make(map[string]*Arg)
	args := make(map[string]*Arg)
	source := ""
	if len(osArgs) > 0 {
		source = osArgs[0]
	}
	for i, arg := range osArgs {
		if len(arg) > 1 && i > 0 && (arg[0] == '-' || (len(arg) > 2 && arg[:2] == "--")) {
			flags[arg] = &Arg{
				Position: i,
				Value:    arg,
			}
			continue
		}
		args[arg] = &Arg{
			Position: i,
			Value:    arg,
		}
	}
	cli := Cli{
		Source:   source,
		Args:     args,
		Flags:    flags,
		Commands: make(map[string]CommandFactory),
		Cwd:      cwd,
	}
	err = cli.setDefault(factory)
	if err != nil {
		return cli, wherr.Consume(wherr.Here(), err, "")
	}
	return cli, nil
}

func (cli *Cli) At(commandName string, factory CommandFactory) {
	cli.Commands[commandName] = factory
}

func (cli *Cli) setDefault(factory CommandFactory) error {
	cli.DefaultFactory = factory
	cmd, err := factory(cli)
	if err != nil {
		return wherr.Consume(wherr.Here(), err, "")
	}
	cli.DefaultCmd = cmd
	return nil
}

func (cli *Cli) Run() error {

	firstArgPosition := 1
	var firstArg string

	for _, arg := range cli.Args {
		if arg.Position == firstArgPosition {
			firstArg = arg.Value
			break
		}
	}

	if firstArg == "" {
		return cli.DefaultCmd.Execute(cli)
	}

	if factory, exists := cli.Commands[firstArg]; exists {
		cmd, err := factory(cli)
		if err != nil {
			return wherr.Consume(wherr.Here(), err, "")
		}
		return cmd.Execute(cli)
	}

	return cli.DefaultCmd.Execute(cli)
}

func (cli *Cli) FlagExists(flag string) bool {
	_, exists := cli.Flags[flag]
	return exists
}

func (cli *Cli) ArgExists(arg string) bool {
	_, exists := cli.Args[arg]
	return exists
}

func (cli *Cli) ArgGetByStr(arg string) (string, bool) {
	val, exists := cli.Args[arg]
	return val.Value, exists
}

func (cli *Cli) ArgForceByStr(arg string) (string, error) {
	val, exists := cli.Args[arg]
	if !exists {
		return "", wherr.Err(wherr.Here(), "arg %s not found", arg)
	}
	return val.Value, nil
}

func (cli *Cli) ArgGetOrDefaultValue(arg string, defaultValue string) string {
	if val, exists := cli.Args[arg]; exists {
		return val.Value
	}
	return defaultValue
}

func (cli *Cli) ArgGetByPosition(position int) (string, bool) {
	for _, arg := range cli.Args {
		if arg.Position == position {
			return arg.Value, true
		}
	}
	return "", false
}

func (cli *Cli) ArgGetByPositionForce(position int, errMsg string) (string, error) {
	arg, exists := cli.ArgGetByPosition(position)
	if !exists {
		return "", wherr.Err(wherr.Here(), "%s", errMsg)
	}
	return arg, nil
}

func (cli *Cli) ArgMorphAtPosition(position int, value string) {
	for _, arg := range cli.Args {
		if arg.Position == position {
			arg.Value = value
		}
	}
}
