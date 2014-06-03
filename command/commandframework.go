package command

import (
	"errors"
	"strings"

	"chunkymonkey/gamerules"
)

var ErrCmdExists = errors.New("The command already exists.")

// The CommandFramework handles all message based commands.
// It uses channels to safly handle multiple calls.
type CommandFramework struct {
	prefix string // The command prefix befor every command.
	cmds   map[string]*Command
}

// Creates a new CommandFramework and starts the update process.
func NewCommandFramework(prefix string) *CommandFramework {
	cf := &CommandFramework{prefix: prefix}
	cmds := getCommands()
	commandHelp := NewCommand(helpCmd, helpDesc, helpUsage, func(player gamerules.IPlayerClient, msg string, game gamerules.IGame) {
		cmdHelp(player, msg, cf, game)
	})
	cmds[helpCmd] = commandHelp
	cmds[helpShortCmd] = commandHelp
	cf.cmds = cmds
	return cf
}

func (cf *CommandFramework) Prefix() string {
	return cf.prefix
}

func (cf *CommandFramework) Commands() map[string]*Command {
	return cf.cmds
}

func (cf *CommandFramework) Process(player gamerules.IPlayerClient, message string, game gamerules.IGame) {
	if len(message) < 2 || message[0:len(cf.prefix)] != cf.prefix {
		return
	}
	attr := strings.Split(message, " ")
	trigger := attr[0][1:]
	if cmd, ok := cf.cmds[trigger]; ok {
		cmd.Callback(player, message, game)
	}
}
