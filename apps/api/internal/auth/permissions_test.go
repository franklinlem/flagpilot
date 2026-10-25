package auth

import "testing"

func TestRolePolicy(t *testing.T) {
	for _, role := range []string{"Owner", "Admin", "Editor", "Viewer", "", "Root", "owner"} {
		read := role == "Owner" || role == "Admin" || role == "Editor" || role == "Viewer"
		if Can(role, ReadMembers) != read {
			t.Fatalf("leitura inesperada: %s", role)
		}
		if Can(role, ManageMembers) != (role == "Owner") {
			t.Fatalf("gestão inesperada: %s", role)
		}
		if Can(role, ReadProjects) != read || Can(role, ManageProjects) != (role == "Owner" || role == "Admin") {
			t.Fatalf("projetos: permissão inesperada %s", role)
		}
		if Can(role, ReadFlags) != read || Can(role, ManageFlags) != (role == "Owner" || role == "Admin" || role == "Editor") {
			t.Fatalf("flags: permissão inesperada %s", role)
		}
		if Can(role, Permission("unknown")) {
			t.Fatal("permissão desconhecida concedida")
		}
	}
}
