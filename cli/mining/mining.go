package mining

import (
	. "ELAClient/cli/common"
	"ELAClient/rpc"
	"errors"

	"github.com/urfave/cli"
	"strconv"
)

func miningAction(c *cli.Context) (err error) {
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
		resp, _ := rpc.Call("togglecpumining", isMining)
		FormatOutput(resp)
		return nil
	}

	if num := c.String("discrete"); num != "" {
		number, err := strconv.ParseInt(num, 10, 16)
		if err != nil || number < 1 {
			return errors.New("[number] must be a positive integer")
		}
		resp, _ := rpc.Call("discretemining", number)
		FormatOutput(resp)
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
			PrintError(c, err, "mining")
			return cli.NewExitError("", 1)
		},
	}
}
