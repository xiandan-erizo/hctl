/*
Copyright © 2022 xiandan HERE xiandan-erizo@outlook.com
*/
package cmd

// NewBaseCommand cmd struct
func NewBaseCommand() *BaseCommand {
	cli := NewCli()
	baseCmd := &BaseCommand{
		command: cli.rootCmd,
	}
	baseCmd.AddCommands(
		&StatsComand{}, // version command
		&DumpCommand{},
		// &TopComand{},
	)

	return baseCmd
}
