package httpapi

import "testing"

func TestSafeAdminNext(t *testing.T) {
	cases := map[string]string{
		"":                           "/admin",
		"/admin":                     "/admin",
		"/admin/users":               "/admin/users",
		"/admin/transactions?q=a":    "/admin/transactions?q=a",
		"/admin/login":               "/admin",
		"https://evil.example/admin": "/admin",
		"//evil/admin":               "/admin",
		"/api/v1/me":                 "/admin",
	}
	for in, want := range cases {
		if got := safeAdminNext(in); got != want {
			t.Errorf("safeAdminNext(%q)=%q want %q", in, got, want)
		}
	}
}

func TestFormatRubKop(t *testing.T) {
	if got := formatRubKop(59900); got != "599 ₽" {
		t.Fatalf("599: %q", got)
	}
	if got := formatRubKop(4_990_00); got != "4 990 ₽" {
		t.Fatalf("4990: %q", got)
	}
	if got := formatRubKop(0); got != "0 ₽" {
		t.Fatalf("0: %q", got)
	}
}
