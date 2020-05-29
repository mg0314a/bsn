package main

import (
	"fisco/check"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{Name: "test", Aliases: []string{"t"}, Usage: "test truffle functions", Action: check.Test},
			{Name: "full",  Aliases: []string{"full"}, Usage: "test full sequence", Action: check.TestFull},
		},
	}
	if err := app.Run(os.Args); err != nil {
		println("error:", err.Error())
		return
	}
}

func cmdTest(ctx *cli.Context) (err error) {

	return err
}
