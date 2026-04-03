package main

import (
	"os"

	"github.com/a2aproject/a2a-go/v2/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
