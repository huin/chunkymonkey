package permission

type IPermissions interface {
	UserPermissions(username string) IUserPermissions
}

// IUserPermissions represents the permissions of a user.
type IUserPermissions interface {
	Has(node string) bool
}
