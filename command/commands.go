package command

import (
	"fmt"
	"strconv"
	"strings"

	"chunkymonkey/gamerules"
	. "chunkymonkey/types"
	"log"
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

func cmdSay(player gamerules.IPlayerClient, message string, cmdHandler gamerules.IGame) {
	args := strings.Split(message, " ")
	if len(args) < 2 {
		player.EchoMessage(sayUsage)
		return
	}
	msg := strings.Join(args[1:], " ")
	cmdHandler.BroadcastMessage("§d" + msg)
}

// tp player1 player2

const tpCmd = "tp"
const tpUsage = "tp <player1> <player2>"
const tpDesc = "Teleports player1 to player2."

func cmdTp(player gamerules.IPlayerClient, message string, cmdHandler gamerules.IGame) {
	args := strings.Split(message, " ")
	if len(args) < 3 {
		player.EchoMessage(tpUsage)
		return
	}

	teleportee := cmdHandler.PlayerByName(args[1])
	destination := cmdHandler.PlayerByName(args[2])
	if teleportee == nil {
		msg := fmt.Sprintf("'%s' is not logged in", args[1])
		player.EchoMessage(msg)
		return
	}
	if destination == nil {
		msg := fmt.Sprintf("'%s' is not logged in", args[2])
		player.EchoMessage(msg)
		return
	}

	pos, look := destination.PositionLook()

	// TODO: Remove this hack or figure out what needs to happen instead
	pos.Y += 1.63

	teleportee.EchoMessage(fmt.Sprintf("Hold still! You are being teleported to %s", args[2]))
	msg := fmt.Sprintf("Teleporting %s to %s at (%.2f, %.2f, %.2f)", args[1], args[2], pos.X, pos.Y, pos.Z)
	log.Printf("Message: %s", msg)
	player.EchoMessage(msg)

	teleportee.SetPositionLook(pos, look)
}

// /kill
const killCmd = "kill"
const killUsage = "kill"
const killDesc = "Inflicts damage to self. Useful when lost or stuck."

func cmdKill(player gamerules.IPlayerClient, message string, cmdHandler gamerules.IGame) {
	// TODO inflict damage to player
	player.EchoMessage(msgNotImplemented)
}

// /tell player message
const tellCmd = "tell"
const tellUsage = "tell <player> <message>"
const tellDesc = "Tells a player a message."

func cmdTell(player gamerules.IPlayerClient, message string, cmdHandler gamerules.IGame) {
	args := strings.Split(message, " ")
	if len(args) < 3 {
		player.EchoMessage(tellUsage)
		return
	}
	/* TODO Get player to send message, too
	player := args[1]
	message := strings.Join(args[2:], " ")
	*/
	player.EchoMessage(msgNotImplemented)
}

const helpShortCmd = "?"
const helpCmd = "help"
const helpUsage = "help|?"
const helpDesc = "Shows a list of all commands."
const msgUnknownCommand = "Command not available."

func cmdHelp(player gamerules.IPlayerClient, message string, cmdFramework *CommandFramework, cmdHandler gamerules.IGame) {
	args := strings.Split(message, " ")
	if len(args) > 2 {
		player.EchoMessage(helpUsage)
		return
	}
	cmds := cmdFramework.Commands()
	if len(args) == 2 {
		cmd := args[1]
		if command, ok := cmds[cmd]; ok {
			player.EchoMessage("Command: " + cmdFramework.Prefix() + command.Trigger)
			player.EchoMessage("Usage: " + command.Usage)
			player.EchoMessage("Description: " + command.Description)
			return
		}
		player.EchoMessage(msgUnknownCommand)
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
	player.EchoMessage(resp)
}

const giveCmd = "give"
const giveUsage = "give <player> <item ID> [<quantity> [<data>]]"
const giveDesc = "Gives x amount of y items to player."

func cmdGive(player gamerules.IPlayerClient, message string, cmdHandler gamerules.IGame) {
	args := strings.Split(message, " ")
	if len(args) < 3 || len(args) > 5 {
		player.EchoMessage(giveUsage)
		return
	}
	args = args[1:]

	// Check to make sure this is a valid player name (at the moment command
	// is being run, the player may log off before command completes).
	target := cmdHandler.PlayerByName(args[0])

	if target == nil {
		msg := fmt.Sprintf("'%s' is not logged in", args[0])
		player.EchoMessage(msg)
		return
	}

	itemNum, err := strconv.Atoi(args[1])
	itemType, ok := cmdHandler.ItemTypeById(itemNum)
	if err != nil || !ok {
		msg := fmt.Sprintf("'%s' is not a valid item id", args[1])
		player.EchoMessage(msg)
		return
	}

	quantity := 1
	if len(args) >= 3 {
		quantity, err = strconv.Atoi(args[2])
		if err != nil {
			player.EchoMessage(giveUsage)
			return
		}

		if quantity > 512 {
			msg := "Cannot give more than 512 items at once"
			player.EchoMessage(msg)
			return
		}
	}

	data := 0
	if len(args) >= 4 {
		data, err = strconv.Atoi(args[2])
		if err != nil {
			player.EchoMessage(giveUsage)
			return
		}
	}

	// Perform the actual give
	msg := fmt.Sprintf("Giving %d of '%s' to %s", quantity, itemType.Name, args[0])
	player.EchoMessage(msg)

	maxStack := int(itemType.MaxStack)

	for quantity > 0 {
		count := quantity
		if count > maxStack {
			count = maxStack
		}

		item := gamerules.Slot{
			ItemTypeId: itemType.Id,
			Count:      ItemCount(count),
			Data:       ItemData(data),
		}
		target.GiveItem(item)
		quantity -= count
	}

	if player != target {
		msg = fmt.Sprintf("%s gave you %d of '%s'", player, quantity, itemType.Name)
		target.EchoMessage(msg)
	}
}
