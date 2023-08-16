/*
Copyright © 2022 xiandan HERE xiandan-erizo@outlook.com
*/
package main

import (
	"fmt"
	"hctl/cmd"
	"os"
)

func main() {
	baseCommand := cmd.NewBaseCommand()
	if err := baseCommand.CobraCmd().Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
