package store

import "testing"

func TestUserRolesCRUD(t *testing.T) {
	s := newLibraryStore(t)
	adminID, err := s.CreateFirstAdmin("admin", "hash")
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.UserByID(adminID)
	if err != nil || u.Role != "admin" {
		t.Fatalf("first admin role = %q err=%v", u.Role, err)
	}

	viewerID, err := s.CreateUser("viewer1", "hash", "viewer")
	if err != nil {
		t.Fatal(err)
	}
	users, err := s.Users()
	if err != nil || len(users) != 2 {
		t.Fatalf("users = %#v err=%v", users, err)
	}

	if err := s.SetUserRole(viewerID, "admin"); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CountAdmins(); n != 2 {
		t.Fatalf("admins = %d, want 2", n)
	}
	if u, _ := s.UserByID(viewerID); u.Role != "admin" {
		t.Fatalf("role update failed: %q", u.Role)
	}

	if err := s.UpdateUserPassword(viewerID, "newhash"); err != nil {
		t.Fatal(err)
	}
	if u, _ := s.UserByID(viewerID); u.PasswordHash != "newhash" {
		t.Fatalf("password not updated")
	}

	if err := s.DeleteUser(viewerID); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CountAdmins(); n != 1 {
		t.Fatalf("admins after delete = %d, want 1", n)
	}
}
