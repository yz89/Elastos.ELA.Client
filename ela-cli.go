package main

import (
	"os"
	"sort"
	"runtime/pprof"
	"os/signal"

	"github.com/elastos/Elastos.ELA.Client/cli/info"
	"github.com/elastos/Elastos.ELA.Client/cli/wallet"
	"github.com/elastos/Elastos.ELA.Client/cli/mine"
	"github.com/elastos/Elastos.ELA.Client/common/log"
	cliLog "github.com/elastos/Elastos.ELA.Client/cli/log"
	"github.com/urfave/cli"
)

var Version string

func init() {
	log.InitLog()
}

func main() {
	fm, err := os.OpenFile("./mem.out", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Error(err)
	}
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
		*cliLog.NewCommand(),
		*info.NewCommand(),
		*wallet.NewCommand(),
		*mine.NewCommand(),
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	sort.Sort(cli.FlagsByName(app.Flags))

	// Handle interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Trace("ela-cli shutting down...")
			os.Exit(0)
		}
	}()

	app.Run(os.Args)
	pprof.WriteHeapProfile(fm)
	fm.Close()
}
