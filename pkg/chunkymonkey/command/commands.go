package command

import (
	"strconv"
	"strings"
)

func getCommands() map[string]*Command {
	cmds := map[string]*Command{}
	cmds[sayCmd] = NewCommand(sayCmd, sayDesc, sayUsage, cmdSay)
	cmds[tpCmd] = NewCommand(tpCmd, tpDesc, tpUsage, cmdTp)
	cmds[killCmd] = NewCommand(killCmd, killDesc, killUsage, cmdKill)
	cmds[tellCmd] = NewCommand(tellCmd, tellDesc, tellUsage, cmdTell)
	cmds[giveCmd] = NewCommand(giveCmd, giveDesc, giveUsage, cmdGive)
	return cmds
}

const msgNotImplemented = "We are sorry. This command is not yet implemented."
const msgUnknownItem = "Unknown item ID"
// say message 
const sayCmd = "say"
const sayUsage = "say <message>"
const sayDesc = "Broadcasts a message to all players without showing a player name. The message is colored pink."

func cmdSay(message string, cmdHandler ICommandHandler) {
	cmdParts := strings.Split(message, " ", -1)
	if len(cmdParts) < 2 {
		cmdHandler.SendMessageToPlayer(sayUsage)
		return
	}
	msg := strings.Join(cmdParts[1:], " ")
	cmdHandler.BroadcastMessage("§d"+msg, true)
}

// tp player1 player2 

const tpCmd = "tp"
const tpUsage = "tp <player1> <player2>"
const tpDesc = "Teleports player1 to player2."

func cmdTp(message string, cmdHandler ICommandHandler) {
	cmdParts := strings.Split(message, " ", -1)
	if len(cmdParts) < 3 {
		cmdHandler.SendMessageToPlayer(tpUsage)
		return
	}
	cmdHandler.SendMessageToPlayer(msgNotImplemented)
	// TODO implement teleporting
}


// /kill 
const killCmd = "kill"
const killUsage = "kill"
const killDesc = "Inflicts damage to self. Useful when lost or stuck."

func cmdKill(message string, cmdHandler ICommandHandler) {
	// TODO inflict damage to player
	cmdHandler.SendMessageToPlayer(msgNotImplemented)
}

// /tell player message 
const tellCmd = "tell"
const tellUsage = "tell <player> <message>"
const tellDesc = "Tells a player a message."

func cmdTell(message string, cmdHandler ICommandHandler) {
	cmdParts := strings.Split(message, " ", -1)
	if len(cmdParts) < 3 {
		cmdHandler.SendMessageToPlayer(tellUsage)
		return
	}
	/* TODO Get player to send message, too
	player := cmdParts[1]
	message := strings.Join(cmdParts[2:], " ")
	*/
	cmdHandler.SendMessageToPlayer(msgNotImplemented)
}

const helpShortCmd = "?"
const helpCmd = "help"
const helpUsage = "help|?"
const helpDesc = "Shows a list of all commands."
const msgUnknownCommand = "Command not available."

func cmdHelp(message string, cmdFramework *CommandFramework, cmdHandler ICommandHandler) {
	cmdParts := strings.Split(message, " ", -1)
	if len(cmdParts) > 2 {
		cmdHandler.SendMessageToPlayer(helpUsage)
		return
	}
	cmds := cmdFramework.Commands()
	if len(cmdParts) == 2 {
		cmd := cmdParts[1]
		if command, ok := cmds[cmd]; ok {
			cmdHandler.SendMessageToPlayer("Command: " + cmdFramework.Prefix() + command.Trigger)
			cmdHandler.SendMessageToPlayer("Usage: " + command.Usage)
			cmdHandler.SendMessageToPlayer("Description: " + command.Description)
			return
		}
		cmdHandler.SendMessageToPlayer(msgUnknownCommand)
		return
	}
	var resp string
	if len(cmds) == 0 {
		resp = "No commands available."
	} else {
		resp = "Commands:"
		for trigger, _ := range cmds {
			resp += " " + trigger + ","
		}
		resp = resp[:len(resp)-1]
	}
	cmdHandler.SendMessageToPlayer(resp)
}

const giveCmd = "give"
const giveUsage = "give <item ID> [<quantity> [<data>]]"
const giveDesc = "Gives x amount of y items to player."

func cmdGive(message string, cmdHandler ICommandHandler) {
	cmdParts := strings.Split(message, " ", -1)
	if len(cmdParts) < 2 || len(cmdParts) > 4 {
		cmdHandler.SendMessageToPlayer(giveUsage)
		return
	}
	cmdParts = cmdParts[1:]

	// TODO Check for item IDs which could break the client
	// TODO First argument should be player to receive item. Right now it just
	// gives it to the current player.
	itemId, err := strconv.Atoi(cmdParts[0])
	if err != nil {
		cmdHandler.SendMessageToPlayer(giveUsage)
		return
	}

	quantity := 1
	if len(cmdParts) >= 2 {
		quantity, err = strconv.Atoi(cmdParts[1])
		if err != nil {
			cmdHandler.SendMessageToPlayer(giveUsage)
			return
		}
	}

	data := 0
	if len(cmdParts) >= 3 {
		data, err = strconv.Atoi(cmdParts[2])
		if err != nil {
			cmdHandler.SendMessageToPlayer(giveUsage)
			return
		}
	}

	cmdHandler.GiveItem(itemId, quantity, data)
}
