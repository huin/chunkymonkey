package permission

import (
	"encoding/json"
	"io"
	"os"
	"strings"
)

// This is a permission system based on groups and users, with data stored in
// two json files "groups.json" and "users.json". It has one world support.
type JsonPermission struct {
	users       map[string]*CachedUser
	defaultUser *CachedUser
}

func LoadJsonPermissionFromFiles(userDefFile, groupDefFile string) (jPermission *JsonPermission, err error) {
	// Load users
	usersFile, err := os.Open(userDefFile)
	if err != nil {
		return
	}
	defer usersFile.Close()

	// Load groups
	groupsFile, err := os.Open(groupDefFile)
	if err != nil {
		return
	}
	defer groupsFile.Close()

	return LoadJsonPermission(usersFile, groupsFile)
}

func LoadJsonPermission(userReader io.Reader, groupReader io.Reader) (jPermission *JsonPermission, err error) {
	// Load users
	usersDecoder := json.NewDecoder(userReader)
	var users Users
	if err = usersDecoder.Decode(&users); err != nil {
		return
	}

	// Load groups
	groupsDecoder := json.NewDecoder(groupReader)
	var groups Groups
	if err = groupsDecoder.Decode(&groups); err != nil {
		return nil, err
	}

	jPermission = &JsonPermission{
		users: make(map[string]*CachedUser),
	}

	// Cache users and merge groups into users.
	for name, user := range users {
		permissions := make([]string, len(user.Permissions))
		for i := range user.Permissions {
			permissions[i] = user.Permissions[i]
		}
		inhPerm := getInheritance(user.Groups, groups)
		for _, perm := range inhPerm {
			permissions = append(permissions, perm)
		}
		jPermission.users[name] = &CachedUser{permissions: permissions}
	}

	// Cache default user.
	defaultUser := &CachedUser{
		permissions: make([]string, 0),
	}
	for _, group := range groups {
		if group.Default {
			for _, perm := range group.Permissions {
				defaultUser.permissions = append(defaultUser.permissions, perm)
			}
			inhPerm := getInheritance(group.Inheritance, groups)
			for _, perm := range inhPerm {
				defaultUser.permissions = append(defaultUser.permissions, perm)
			}
		}
	}
	jPermission.defaultUser = defaultUser
	return jPermission, nil
}

func getInheritance(groupList []string, groups Groups) []string {
	permList := make([]string, 0)
	for _, group := range groupList {
		for _, permission := range groups[group].Permissions {
			permList = append(permList, permission)
		}
		inhPerm := getInheritance(groups[group].Inheritance, groups)
		for _, permission := range inhPerm {
			permList = append(permList, permission)
		}
	}
	return permList
}

// Implementation of IPermissions
func (p *JsonPermission) UserPermissions(username string) IUserPermissions {
	if user, ok := p.users[username]; ok {
		return user
	}
	return p.defaultUser
}

// A JsonPermission user with chached permissions.
type CachedUser struct {
	permissions []string
}

// Implementation of IUserPermissions
func (u *CachedUser) Has(node string) bool {
	for _, p := range u.permissions {
		if p == node {
			return true
		} else if strings.HasSuffix(p, "*") && len(p) > 0 {
			if strings.HasPrefix(node, p[:len(p)-1]) {
				return true
			}
		}
	}
	return false
}
