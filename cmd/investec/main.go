package main

import (
	"os"

	"github.com/yashiels/investec/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
