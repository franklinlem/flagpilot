package auth

type Permission string

const (
	ReadMembers   Permission = "members.read"
	ManageMembers Permission = "members.manage"
)

func ValidRole(role string) bool {
	switch role {
	case "Owner", "Admin", "Editor", "Viewer":
		return true
	}
	return false
}

// Deny by default: papéis e permissões desconhecidos não concedem acesso.
func Can(role string, permission Permission) bool {
	if !ValidRole(role) {
		return false
	}
	switch permission {
	case ReadMembers:
		return true
	case ManageMembers:
		return role == "Owner"
	}
	return false
}
