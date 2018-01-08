package mining

import (
	"ELAClient/rpc"
	"errors"

	"github.com/urfave/cli"
	"strconv"
	"fmt"
)

func miningAction(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	if action := c.String("toggle"); action != "" {
		var isMining bool
		if action == "start" || action == "START" {
			isMining = true
		} else if action == "stop" || action == "STOP" {
			isMining = false
		} else {
			return errors.New("toggle argument must be [start, stop]")
		}
		result, err := rpc.CallAndUnmarshal("togglecpumining", isMining)
		if err != nil {
			return err
		}

		fmt.Println(result)
		return nil
	}

	if num := c.String("mine"); num != "" {
		number, err := strconv.ParseInt(num, 10, 16)
		if err != nil || number < 1 {
			return errors.New("[number] must be a positive integer")
		}
		result, err := rpc.CallAndUnmarshal("discretemining", number)
		if err != nil {
			return err
		}

		fmt.Println(result)
		return nil
	}

	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "mining",
		Usage:       "toggle cpu mining.",
		Description: "With ela-cli mining, you could toggle cpu mining, or manual mine blocks.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "toggle, t",
				Usage: "use --toggle [start, stop] to toggle cpu mining",
			},
			cli.StringFlag{
				Name:  "mine, m",
				Usage: "user --mine [number] to manual mine the given number of blocks",
			},
		},
		Action: miningAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			return cli.NewExitError(err, 1)
		},
	}
}
