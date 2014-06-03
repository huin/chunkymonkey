package permission

type Users map[string]*User

type User struct {
	Groups      []string
	Permissions []string
}
