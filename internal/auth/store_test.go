package auth

import (
	"testing"
)

func redirectHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestSaveAndLoadToken_Roundtrip(t *testing.T) {
	redirectHome(t)

	token := "ghp_testtoken123"
	if err := SaveToken(token); err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error: %v", err)
	}
	if got != token {
		t.Errorf("LoadToken() = %q, want %q", got, token)
	}
}

func TestLoadToken_NoFile_ReturnsEmpty(t *testing.T) {
	redirectHome(t)

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty token, got %q", got)
	}
}

func TestSaveToken_TrimsWhitespace(t *testing.T) {
	redirectHome(t)

	if err := SaveToken("  ghp_padded  \n"); err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}
	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() error: %v", err)
	}
	if got != "ghp_padded" {
		t.Errorf("expected trimmed token, got %q", got)
	}
}

func TestDeleteToken_RemovesFile(t *testing.T) {
	redirectHome(t)

	if err := SaveToken("ghp_todelete"); err != nil {
		t.Fatalf("SaveToken() error: %v", err)
	}
	if err := DeleteToken(); err != nil {
		t.Fatalf("DeleteToken() error: %v", err)
	}

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken() after delete error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty token after delete, got %q", got)
	}
}

func TestDeleteToken_NonExistent_NoError(t *testing.T) {
	redirectHome(t)

	if err := DeleteToken(); err != nil {
		t.Errorf("DeleteToken() on missing file should return nil, got: %v", err)
	}
}
