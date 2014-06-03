package permission

type Groups map[string]*Group

type Group struct {
	Default     bool
	Permissions []string
	Inheritance []string
}
