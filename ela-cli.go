package main

import (
	"os"
	"sort"

	"ELAClient/cli/info"
	"ELAClient/cli/wallet"
	"ELAClient/common/log"
	"ELAClient/cli/mining"
	"github.com/urfave/cli"
	"ELAClient/cli/log_level"
)

var Version string

func init() {
	log.InitLog()
}

func main() {
	app := cli.NewApp()
	app.Name = "ela-cli"
	app.Version = Version
	app.HelpName = "ela-cli"
	app.Usage = "command line tool for ELA blockchain"
	app.UsageText = "ela-cli [global options] command [command options] [args]"
	app.HideHelp = false
	app.HideVersion = false
	//commands
	app.Commands = []cli.Command{
		*log_level.NewCommand(),
		*info.NewCommand(),
		*wallet.NewCommand(),
		*mining.NewCommand(),
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	sort.Sort(cli.FlagsByName(app.Flags))

	app.Run(os.Args)
}
