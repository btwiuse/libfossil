package main

import (
	"context"
	"os"

	"charm.land/fang/v2"
	"github.com/danmestas/libfossil/cli"
	_ "github.com/btwiuse/libfossil/db/driver/ncruces"
)

func main() {
	root := cli.NewRootCommand()
	if err := fang.Execute(context.Background(), root); err != nil {
		os.Exit(1)
	}
}
