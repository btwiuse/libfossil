package main

import (
	"github.com/alecthomas/kong"
	"github.com/danmestas/libfossil/cli"
	_ "github.com/btwiuse/libfossil/db/driver/ncruces"
)

type CLI struct {
	cli.Globals
	cli.RepoCmd
}

func main() {
	var c CLI
	ctx := kong.Parse(&c,
		kong.Name("libfossil"),
		kong.Description("Fossil-compatible repository tool (pure Go)"),
		kong.UsageOnError(),
	)
	err := ctx.Run(&c.Globals)
	ctx.FatalIfErrorf(err)
}
