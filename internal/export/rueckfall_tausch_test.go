// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// Tests zu F-330: Der Rückfall kürzt nur die Datei, die er gesichert hat. Der Tausch entsteht über die Naht
// oeffneDirekt im eigenen Code.

// vorOeffnen führt aktion unmittelbar vor dem ersten Öffnen des Ziels aus (Tausch zwischen Sichern und Öffnen).
func vorOeffnen(t *testing.T, aktion func()) {
	t.Helper()
	orig := oeffneDirekt
	t.Cleanup(func() { oeffneDirekt = orig })
	einmal := false
	oeffneDirekt = func(pfad string, st os.FileInfo) (zielDatei, error) {
		if !einmal {
			einmal = true
			aktion()
		}
		return orig(pfad, st)
	}
}

// TestRueckfallTauschVorOeffnen: Wird der Name zwischen Sichern und Öffnen gegen eine andere reguläre Datei getauscht,
// bleibt sie unverändert und der Fehler der temporären Datei kommt zurück.
func TestRueckfallTauschVorOeffnen(t *testing.T) {
	ziel := zielMit(t, []byte("alt"))
	fremd := filepath.Join(filepath.Dir(ziel), "fremd.png")
	inhalt := []byte("fremder Inhalt, der bleiben muss")
	if err := os.WriteFile(fremd, inhalt, 0o600); err != nil {
		t.Fatal(err)
	}
	vorOeffnen(t, func() {
		if err := os.Rename(fremd, ziel); err != nil {
			t.Fatal(err)
		}
	})
	if err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission); !errors.Is(err, os.ErrPermission) {
		t.Errorf("Fehler %v, erwartet die Ursache", err)
	}
	if got, err := os.ReadFile(ziel); err != nil || !bytes.Equal(got, inhalt) {
		t.Errorf("Ersatzdatei verändert: %q, %v", got, err)
	}
}

// tauschBeiSchreibfehler tauscht das Ziel, nachdem das Schreiben halb gelang und scheiterte.
type tauschBeiSchreibfehler struct {
	haelfteDatei
	tausche func()
}

func (x tauschBeiSchreibfehler) Write(p []byte) (int, error) {
	n, err := x.haelfteDatei.Write(p)
	if err != nil {
		x.tausche() // nur nach dem gescheiterten Schreiben, nicht beim Wiederherstellen
	}
	return n, err
}

// TestRueckfallTauschNachSchreibfehler: Wird das Ziel nach einem Schreibfehler getauscht, überschreibt die
// Wiederherstellung die fremde Datei nicht; sie schreibt den alten Inhalt über denselben Deskriptor in genau die
// gesicherte Datei zurück (F-338), und gemeldet wird der Schreibfehler.
func TestRueckfallTauschNachSchreibfehler(t *testing.T) {
	ziel := zielMit(t, []byte("alt"))
	gesichert := filepath.Join(filepath.Dir(ziel), "gesichert.png") // zweiter Name der Originaldatei
	if err := os.Link(ziel, gesichert); err != nil {
		t.Fatal(err)
	}
	fremd := filepath.Join(filepath.Dir(ziel), "fremd.png")
	inhalt := []byte("fremder Inhalt")
	if err := os.WriteFile(fremd, inhalt, 0o600); err != nil {
		t.Fatal(err)
	}
	kaputt := errors.New("Platte voll")
	biegeUm(t, 1, func(f zielDatei) zielDatei {
		return tauschBeiSchreibfehler{haelfteDatei{f, kaputt, scheitert(1)}, func() {
			if err := os.Rename(fremd, ziel); err != nil {
				t.Fatal(err)
			}
		}}
	})
	err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission)
	if !errors.Is(err, kaputt) || errors.Is(err, ErrZielBeschaedigt) {
		t.Errorf("Fehler %v, erwartet der Schreibfehler", err)
	}
	if got, _ := os.ReadFile(ziel); !bytes.Equal(got, inhalt) {
		t.Errorf("Ersatzdatei verändert: %q", got)
	}
	if got, _ := os.ReadFile(gesichert); string(got) != "alt" {
		t.Errorf("gesicherte Datei nicht wiederhergestellt: %q", got)
	}
}

// TestRueckfallFifoBlockiertNicht: Ein untergeschobenes FIFO am Zielnamen blockiert das Öffnen nicht (O_NONBLOCK).
func TestRueckfallFifoBlockiertNicht(t *testing.T) {
	ziel := zielMit(t, []byte("alt"))
	vorOeffnen(t, func() {
		if err := os.Remove(ziel); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(ziel, 0o600); err != nil {
			t.Fatal(err)
		}
	})
	fertig := make(chan error, 1)
	go func() { fertig <- schreibeDirekt(ziel, schreibeNeu, os.ErrPermission) }()
	select {
	case err := <-fertig:
		if !errors.Is(err, os.ErrPermission) {
			t.Errorf("Fehler %v, erwartet die Ursache", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Rückfall blockiert an einem FIFO")
	}
}
