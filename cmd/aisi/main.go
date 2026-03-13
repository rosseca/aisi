package main

import (
	"os"

	"github.com/rosseca/aisi/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
