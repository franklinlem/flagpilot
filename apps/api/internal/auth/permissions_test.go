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
		if Can(role, Permission("unknown")) {
			t.Fatal("permissão desconhecida concedida")
		}
	}
}
