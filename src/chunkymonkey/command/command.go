package command

import (
	"chunkymonkey/gamerules"
)

// A CommandCallback takes the name of the player invoking the command, any
// text supplied after the trigger for the command, and an interface via which
// game-wide 'actions' can be taken.
type CommandCallback func(player gamerules.IPlayerClient, args string, game gamerules.IGame)

type Command struct {
	Trigger     string          // The initial text eg. "give".
	Description string          // A description of what the command does.
	Usage       string          // A usage string for the command.
	Callback    CommandCallback // This function will be called if a Message begins with the CommandPrefix and the Trigger.
}

func NewCommand(trigger, desc, usage string, callback CommandCallback) *Command {
	return &Command{Trigger: trigger, Description: desc, Usage: usage, Callback: callback}
}
