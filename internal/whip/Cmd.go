package whip

type Cmd interface {
	Execute(cli *Cli) error
}

type CommandFactory func(cli *Cli) (Cmd, error)
