package main

import (
	"github.com/LeakIX/wpfinger"
	"github.com/alecthomas/kong"
)

var CLI struct {
	UpdateDb wpfinger.CmdUpdateDb `name:"update" cmd:"" help:"Update WPFinger database."`
	Scan     wpfinger.CmdScan     `name:"scan" cmd:"" help:"Scan WordPress site."`
	BuildDb  wpfinger.CmdBuildDb  `name:"build-db" cmd:"" cmd:"" help:"Build WPFinger database, will take several hours."`
}

func main() {
	ctx := kong.Parse(&CLI)
	// Call the Run() method of the selected parsed command.
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
