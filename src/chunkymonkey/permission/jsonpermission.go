package permission

import (
	"io/ioutil"
	"json"
	"path"
	"strings"
	"os"
)

const (
	FileNameGroups = "groups.json"
	FileNameUsers  = "users.json"
)

var (
	ErrFileGroups = os.NewError("Couldn't find groups.json")
	ErrFileUsers  = os.NewError("Couldn't find users.json")
)

// This is a permission system based on two json files "groups.json" and "users.json".
// It has one world support.
type JsonPermission struct {
	users       map[string]*CachedUser
	defaultUser *CachedUser
}


// The expected folder structure is:
// ./groups.json
// ./users.yml
func LoadJsonPermission(folder string) (*JsonPermission, os.Error) {
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	var fileNameUsers string
	var fileNameGroups string
	for _, file := range files {
		fileName := path.Base(file.Name)
		if fileName == FileNameGroups {
			fileNameGroups = file.Name
		} else if fileName == FileNameUsers {
			fileNameUsers = file.Name
		}
	}
	if len(fileNameUsers) == 0 {
		return nil, ErrFileUsers
	} else if len(fileNameGroups) == 0 {
		return nil, ErrFileGroups
	}
	// Load users
	var bytesUsers []byte
	bytesUsers, err = ioutil.ReadFile(fileNameUsers)
	if err != nil {
		return nil, err
	}
	var users Users
	err = json.Unmarshal(bytesUsers, &users)
	if err != nil {
		return nil, err
	}
	// Load groups
	var bytesGroups []byte
	bytesGroups, err = ioutil.ReadFile(fileNameGroups)
	if err != nil {
		return nil, err
	}
	var groups Groups
	err = json.Unmarshal(bytesGroups, &groups)
	if err != nil {
		return nil, err
	}
	jPermission := &JsonPermission{users: make(map[string]*CachedUser)}
	// Cache users and merge groups into users
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
	// Cache default group
	defaultUser := &CachedUser{permissions: make([]string, 0)}
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
