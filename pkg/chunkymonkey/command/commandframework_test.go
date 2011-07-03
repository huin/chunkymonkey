package command

import (
	"testing"
)

const (
	cmdText = "give"
	desc    = "Gives x y to player z"
	usage   = "Usage text"
	message = "/give 1 64 admin"
)

func TestCommandFramework(t *testing.T) {
	NewCommandFramework("/")
	// TODO: Write test cases for CommandFramework and all commands
}
