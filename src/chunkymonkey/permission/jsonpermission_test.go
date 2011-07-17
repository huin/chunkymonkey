package permission

import (
	"testing"
)

func TestJsonPermission(t *testing.T) {
	perm, err := LoadJsonPermission("./")
	if err != nil {
		t.Fatalf("Error while loading JsonPermission: %s", err)
	}
	// Check User permissions
	if perm.UserPermissions("agon").Has("server.status") == false {
		t.Error("User agon should have node server.status.")
	}
	// Check User permissions from groups
	if perm.UserPermissions("agon").Has("admin.commands.give") == false {
		t.Error("User agon should have node admin.commands.give through the admin group.")
	}
	// Check if User has no permission
	if perm.UserPermissions("huin").Has("this.node.does.not.exist.tm") {
		t.Error("User huin should not have this.node.does.not.exist.tm as a permission node.")
	}
	// Wildcard check
	if perm.UserPermissions("huin").Has("server.stop") == false { // huin has "server.*", means he has permission for "server.stop"
		t.Error("User huin should have permission for server.stop.")
	}
}
