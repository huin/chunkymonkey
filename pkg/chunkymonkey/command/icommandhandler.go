package command

import (
	"os"
)

type ICommandHandler interface {
	GiveItem(int, int, int) os.Error // ID, quantity, data
	SendMessageToPlayer(string)
	BroadcastMessage(string, bool)
}
