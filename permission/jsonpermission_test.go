package permission

import (
	"strings"
	"testing"
)

const (
	testUsersJson = `{
		"agon": {
			"groups": ["admin"],
			"permissions": [
				"server.web.*",
				"admin.*",
				"server.status"
			]
		},
		"huin": {
			"groups": ["admin"],
			"permissions": [
				"server.*",
				"admin.*"
			]
		},
		"defaulty": {
			"groups": ["default"]
		},
		"griefy": {
			"groups": ["banned"]
		}
	}`
	testGroupsJson = `{
		"basic": {
			"permissions": [
				"login"
			]
		},
		"default": {
			"default": true,
			"inheritance": ["basic"],
			"permissions": [
				"user.commands.help",
				"user.commands.me",
				"world.build"
			]
		},
		"admin": {
			"inheritance": ["default"],
			"permissions": [
				"admin.commands.give",
				"world.*"
			]
		},
		"banned": {
			"inheritance": [],
			"permissions": []
		}
	}`
)

func testLoadPermission() (permissions IPermissions) {
	usersReader := strings.NewReader(testUsersJson)
	groupsReader := strings.NewReader(testGroupsJson)

	permissions, err := LoadJsonPermission(usersReader, groupsReader)
	if err != nil {
		panic(err)
	}

	return
}

func TestJsonPermission(t *testing.T) {
	perm := testLoadPermission()

	type Test struct {
		username    string
		permission  string
		expectedHas bool
	}

	tests := []Test{
		// Check user permissions
		{"agon", "server.status", true},
		// Check user permissions from groups
		{"agon", "admin.commands.give", true},
		// Check if user has no permission
		{"huin", "this.node.does.not.exist.tm", false},
		// Check group inheritance.
		{"huin", "user.commands.me", true},
		// Wildcard check
		{"huin", "server.stop", true},
		// Default player permissions.
		{"newbie", "login", true},
		{"newbie", "admin.commands.give", false},
		// Banned player should have no permissions.
		{"griefy", "login", false},
		{"griefy", "world.build", false},
	}

	for i := range tests {
		test := &tests[i]
		result := perm.UserPermissions(test.username).Has(test.permission)
		if test.expectedHas != result {
			var msg string
			if test.expectedHas {
				msg = "User %s should have node %s"
			} else {
				msg = "User %s should *not* have node %s"
			}
			t.Errorf(msg, test.username, test.permission)
		}
	}
}

func Benchmark_PermissionMatchExact(b *testing.B) {
	perm := testLoadPermission()

	userPerm := perm.UserPermissions("defaulty")

	b.ResetTimer()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		userPerm.Has("user.commands.me")
	}
}

func Benchmark_PermissionMatchWildcard(b *testing.B) {
	perm := testLoadPermission()

	userPerm := perm.UserPermissions("huin")

	b.ResetTimer()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		userPerm.Has("world.foo")
	}
}
