// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// ohneLinks baut ein Dateisystem ohne harte Links nach; vorLink läuft im Link-Versuch (etwa: das Ziel entsteht gerade).
func ohneLinks(t *testing.T, vorLink func(neu string)) {
	t.Helper()
	alt := linkNaht
	linkNaht = func(_, neu string) error {
		if vorLink != nil {
			vorLink(neu)
		}
		return &os.LinkError{Op: "link", Err: unix.EPERM}
	}
	t.Cleanup(func() { linkNaht = alt })
}

func schreibeHallo(w io.Writer) error { _, err := io.WriteString(w, "neu"); return err }

// TestOhneErsetzenOhneLinks: Ohne harte Links ersetzt SchreibeOhneErsetzen ein Ziel nie, auch eines, das erst während
// des Anlegens entsteht (F-494: RENAME_NOREPLACE statt Prüfen und Umbenennen); ein neues Ziel entsteht.
func TestOhneErsetzenOhneLinks(t *testing.T) {
	ordner := t.TempDir()
	ziel := filepath.Join(ordner, "z.png")
	ohneLinks(t, func(neu string) {
		if err := os.WriteFile(neu, []byte("kam dazwischen"), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	if err := SchreibeOhneErsetzen(ziel, schreibeHallo); !errors.Is(err, ErrZielVorhanden) {
		t.Fatalf("Ziel entstand im Wettlauf: %v, erwartet ErrZielVorhanden", err)
	}
	if b, _ := os.ReadFile(ziel); string(b) != "kam dazwischen" {
		t.Errorf("Ziel ersetzt: %q", b)
	}
	if rest, _ := filepath.Glob(filepath.Join(ordner, ".schnapp-export-*")); len(rest) != 0 {
		t.Errorf("temporäre Datei blieb: %v", rest)
	}

	neu := filepath.Join(ordner, "neu.png")
	ohneLinks(t, nil)
	if err := SchreibeOhneErsetzen(neu, schreibeHallo); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(neu); string(b) != "neu" {
		t.Errorf("neues Ziel: %q", b)
	}
}

// TestOhneErsetzenOhneRenameat2: Kann das Dateisystem auch RENAME_NOREPLACE nicht, gilt der bestmögliche Weg: ein
// vorhandenes Ziel bleibt.
func TestOhneErsetzenOhneRenameat2(t *testing.T) {
	ordner := t.TempDir()
	ziel := filepath.Join(ordner, "z.png")
	if err := os.WriteFile(ziel, []byte("alt"), 0o600); err != nil {
		t.Fatal(err)
	}
	ohneLinks(t, nil)
	alt := umbenennenOhneErsetzen
	umbenennenOhneErsetzen = func(string, string) error { return &os.LinkError{Op: "renameat2", Err: unix.EINVAL} }
	t.Cleanup(func() { umbenennenOhneErsetzen = alt })
	if err := SchreibeOhneErsetzen(ziel, schreibeHallo); !errors.Is(err, ErrZielVorhanden) {
		t.Fatalf("%v, erwartet ErrZielVorhanden", err)
	}
	if b, _ := os.ReadFile(ziel); string(b) != "alt" {
		t.Errorf("Ziel ersetzt: %q", b)
	}
	neu := filepath.Join(ordner, "neu.png")
	if err := SchreibeOhneErsetzen(neu, schreibeHallo); err != nil {
		t.Fatal(err)
	}
}
