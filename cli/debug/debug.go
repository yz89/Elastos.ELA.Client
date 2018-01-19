package debug

import (
	"fmt"

	"ELAClient/rpc"

	"github.com/urfave/cli"
)

func debugAction(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	if level := c.Int("level"); level != -1 {
		result, err := rpc.CallAndUnmarshal("setdebuginfo", level)
		if err != nil {
			fmt.Println("error: set debug info failed, ", err)
			return err
		}
		fmt.Println(result)
		return nil
	}

	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{Name: "debug",
		Usage: "blockchain node debugging",
		Description: "With ela-cli debug, you could debug blockchain node.",
		ArgsUsage: "[args]",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "level, l",
				Usage: "log level 0-6",
				Value: -1,
			},
		},
		Action: debugAction,
		OnUsageError: func(c *cli.Context, err error, isSubCommand bool) error {
			return cli.NewExitError(err, 1)
		},
	}
}
