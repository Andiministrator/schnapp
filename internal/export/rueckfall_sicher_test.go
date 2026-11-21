// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests zum Rückfall ohne temporäre Datei (F-322): Schreibfehler stellen den alten Inhalt wieder her. Die Fehler
// entstehen über die Naht oeffneDirekt im eigenen Code, nicht von außen.

// haelfteDatei schreibt die Hälfte wirklich und meldet dann den Fehler (Medium voll, Netzlaufwerk).
type haelfteDatei struct {
	zielDatei
	fehler error
}

func (h haelfteDatei) Write(p []byte) (int, error) {
	n, _ := h.zielDatei.Write(p[:len(p)/2])
	return n, h.fehler
}

// syncFehlerDatei lässt Write gelingen, scheitert aber beim Sync.
type syncFehlerDatei struct {
	zielDatei
	fehler error
}

func (s syncFehlerDatei) Sync() error { return s.fehler }

// biegeUm ersetzt oeffneDirekt: Die ersten n Öffnungen liefern die Hülle um die echte Datei, danach die echte.
func biegeUm(t *testing.T, n int, huelle func(zielDatei) zielDatei) {
	t.Helper()
	orig := oeffneDirekt
	t.Cleanup(func() { oeffneDirekt = orig })
	aufrufe := 0
	oeffneDirekt = func(pfad string, flags int) (zielDatei, error) {
		f, err := orig(pfad, flags)
		aufrufe++
		if err == nil && aufrufe <= n {
			return huelle(f), nil
		}
		return f, err
	}
}

func zielMit(t *testing.T, inhalt []byte) string {
	t.Helper()
	ziel := filepath.Join(t.TempDir(), "bild.png")
	if err := os.WriteFile(ziel, inhalt, 0o600); err != nil {
		t.Fatal(err)
	}
	return ziel
}

func schreibeNeu(w io.Writer) error {
	_, err := w.Write([]byte("neuer Inhalt, kürzer"))
	return err
}

// TestRueckfallSchreibfehlerStelltAltWiederHer: Scheitert Write nach der Hälfte, steht der alte Inhalt bitgleich da.
func TestRueckfallSchreibfehlerStelltAltWiederHer(t *testing.T) {
	alt := []byte("\x00\x89PNG alter, deutlich längerer Inhalt\xff\x00")
	ziel := zielMit(t, alt)
	kaputt := errors.New("Medium voll")
	biegeUm(t, 1, func(f zielDatei) zielDatei { return haelfteDatei{f, kaputt} })
	err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission)
	if !errors.Is(err, kaputt) {
		t.Fatalf("Fehler %v, erwartet der Schreibfehler", err)
	}
	if errors.Is(err, ErrZielBeschaedigt) {
		t.Errorf("fälschlich als beschädigt gemeldet: %v", err)
	}
	if b, _ := os.ReadFile(ziel); !bytes.Equal(b, alt) {
		t.Errorf("Ziel %q, erwartet der alte Inhalt", b)
	}
}

// TestRueckfallSyncfehlerStelltAltWiederHer: Auch ein Sync-Fehler stellt den alten Inhalt her.
func TestRueckfallSyncfehlerStelltAltWiederHer(t *testing.T) {
	alt := []byte("alt und länger als der neue Inhalt")
	ziel := zielMit(t, alt)
	kaputt := errors.New("Sync")
	biegeUm(t, 1, func(f zielDatei) zielDatei { return syncFehlerDatei{f, kaputt} })
	if err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission); !errors.Is(err, kaputt) {
		t.Fatalf("Fehler %v", err)
	}
	if b, _ := os.ReadFile(ziel); !bytes.Equal(b, alt) {
		t.Errorf("Ziel %q", b)
	}
}

// TestRueckfallLeereAltdatei: War das Ziel leer, bleibt es nach dem Fehler leer (keine halbe neue Datei). Ein
// vorhandenes Ziel ist Voraussetzung des Rückfalls; „keine Altdatei“ heißt hier: nichts zum Wiederherstellen.
func TestRueckfallLeereAltdatei(t *testing.T) {
	ziel := zielMit(t, nil)
	kaputt := errors.New("Medium voll")
	biegeUm(t, 1, func(f zielDatei) zielDatei { return haelfteDatei{f, kaputt} })
	if err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission); !errors.Is(err, kaputt) {
		t.Fatalf("Fehler %v", err)
	}
	if st, err := os.Stat(ziel); err != nil || st.Size() != 0 {
		t.Errorf("Ziel nicht leer: %v %v", st, err)
	}
}

// TestRueckfallFehlendesZielBleibtFehlend: Ohne vorhandenes Ziel entsteht im Rückfall nichts.
func TestRueckfallFehlendesZielBleibtFehlend(t *testing.T) {
	ziel := filepath.Join(t.TempDir(), "gibtsnicht.png")
	if err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission); !errors.Is(err, os.ErrPermission) {
		t.Errorf("Fehler %v, erwartet die Ursache", err)
	}
	if _, err := os.Lstat(ziel); err == nil {
		t.Error("Ziel entstanden")
	}
}

// TestRueckfallWiederherstellungScheitertMeldetBeschaedigt: Scheitert auch das Zurückschreiben, sagt der Fehler es.
func TestRueckfallWiederherstellungScheitertMeldetBeschaedigt(t *testing.T) {
	ziel := zielMit(t, []byte("alter, längerer Inhalt"))
	kaputt := errors.New("Medium voll")
	biegeUm(t, 2, func(f zielDatei) zielDatei { return haelfteDatei{f, kaputt} })
	err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission)
	if !errors.Is(err, ErrZielBeschaedigt) {
		t.Fatalf("Fehler %v, erwartet ErrZielBeschaedigt", err)
	}
	if !strings.Contains(err.Error(), "Medium voll") {
		t.Errorf("Ursache fehlt im Text: %v", err)
	}
}

// TestRueckfallZuGrosseAltdatei: Ist die Altdatei zu groß für die Sicherung, wird gar nicht direkt geschrieben.
func TestRueckfallZuGrosseAltdatei(t *testing.T) {
	ziel := zielMit(t, []byte("alt"))
	if err := os.Truncate(ziel, MaxRueckfallAlt+1); err != nil {
		t.Skip("Sparse-Datei nicht möglich:", err)
	}
	orig := oeffneDirekt
	t.Cleanup(func() { oeffneDirekt = orig })
	aufgerufen := false
	oeffneDirekt = func(string, int) (zielDatei, error) { aufgerufen = true; return nil, errors.New("nein") }
	if err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission); !errors.Is(err, os.ErrPermission) {
		t.Errorf("Fehler %v, erwartet die Ursache", err)
	}
	if aufgerufen {
		t.Error("Ziel trotz fehlender Sicherung geöffnet")
	}
}

// TestRueckfallFolgtKeinemLink: Ein Link wird weder geöffnet noch gelesen (O_NOFOLLOW), auch wenn er zwischen Lstat
// und Öffnen untergeschoben würde.
func TestRueckfallFolgtKeinemLink(t *testing.T) {
	ordner := t.TempDir()
	echt := filepath.Join(ordner, "echt.png")
	if err := os.WriteFile(echt, []byte("echt"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(ordner, "link.png")
	if err := os.Symlink(echt, link); err != nil {
		t.Fatal(err)
	}
	if f, err := oeffneDirekt(link, os.O_WRONLY|os.O_TRUNC); err == nil {
		f.Close()
		t.Error("oeffneDirekt folgte dem Link")
	}
	if st, err := os.Stat(echt); err == nil {
		if _, err := leseAlt(link, st); err == nil {
			t.Error("leseAlt folgte dem Link")
		}
	}
	if err := schreibeDirekt(link, schreibeNeu, os.ErrPermission); !errors.Is(err, os.ErrPermission) {
		t.Errorf("Fehler %v, erwartet die Ursache", err)
	}
	if b, _ := os.ReadFile(echt); string(b) != "echt" {
		t.Errorf("Linkziel verändert: %q", b)
	}
}
