package command

import (
	"testing"

	"gomock.googlecode.com/hg/gomock"

	"chunkymonkey/gamerules"
	"testmatcher"
)

func TestCommandFramework(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockGame := gamerules.NewMockIGame(mockCtrl)
	mockPlayer := gamerules.NewMockIPlayerClient(mockCtrl)
	mockOther := gamerules.NewMockIPlayerClient(mockCtrl)

	cf := NewCommandFramework("/")

	mockGame.EXPECT().BroadcastMessage("§dthis is a broadcast")
	cf.Process(mockPlayer, "/say this is a broadcast", mockGame)

	mockGame.EXPECT().PlayerByName("thePlayer").Return(mockPlayer)
	mockGame.EXPECT().ItemTypeById(1)
	mockPlayer.EXPECT().EchoMessage("Giving 64 of 1 to thePlayer")
	mockPlayer.EXPECT().GiveItemAtPosition("thePlayer", gamerules.Slot{1, 64, 0})
	cf.Process(mockPlayer, "/give thePlayer 1 64", mockGame)

	mockGame.EXPECT().PlayerByName("otherPlayer")
	mockPlayer.EXPECT().EchoMessage("'otherPlayer' is not logged in")
	cf.Process(mockPlayer, "/give otherPlayer 1 64", mockGame)

	mockGame.EXPECT().PlayerByName("otherPlayer").Return(mockOther)
	mockGame.EXPECT().ItemTypeById(1).Return(gamerules.ItemType{}, false)
	mockPlayer.EXPECT().EchoMessage("'1' is not a valid item id")
	cf.Process(mockPlayer, "/give otherPlayer 1 64", mockGame)

	mockGame.EXPECT().PlayerByName("otherPlayer").Return(mockOther)
	mockGame.EXPECT().ItemTypeById(1)
	mockPlayer.EXPECT().EchoMessage("Cannot give more than 512 items at once")
	cf.Process(mockPlayer, "/give otherPlayer 1 513", mockGame)

	mockPlayer.EXPECT().EchoMessage(&testmatcher.StringPrefix{"Commands:"})
	cf.Process(mockPlayer, "/help", mockGame)

	gomock.InOrder(
		mockPlayer.EXPECT().EchoMessage("Command: /help"),
		mockPlayer.EXPECT().EchoMessage("Usage: help|?"),
		mockPlayer.EXPECT().EchoMessage("Description: Shows a list of all commands."),
	)
	cf.Process(mockPlayer, "/help help", mockGame)
}
