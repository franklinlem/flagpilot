package auth

type Permission string

const (
	ReadProjects   Permission = "projects.read"
	ManageProjects Permission = "projects.manage"
	ReadMembers    Permission = "members.read"
	ManageMembers  Permission = "members.manage"
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
	case ReadMembers, ReadProjects:
		return true
	case ManageProjects:
		return role == "Owner" || role == "Admin"
	case ManageMembers:
		return role == "Owner"
	}
	return false
}
