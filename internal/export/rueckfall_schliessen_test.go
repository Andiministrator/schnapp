// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

// F-331, F-344: Ein Close-Fehler nach erfolgreichem Schreiben und Sync im Rückfall ist kein Exportfehler: der neue
// Inhalt steht vollständig. Der Fehler geht nur in die Diagnose. Er entsteht über die Naht oeffneDirekt im eigenen Code
// (Hülle um die echte Datei).

// schliessFehlerDatei schließt die echte Datei und meldet dennoch einen Fehler (etwa NFS: Schreibfehler erst beim Close).
type schliessFehlerDatei struct {
	zielDatei
	fehler error
}

func (s schliessFehlerDatei) Close() error {
	s.zielDatei.Close()
	return s.fehler
}

func TestRueckfallCloseFehlerNachSyncIstErfolg(t *testing.T) {
	ziel := zielMit(t, []byte("alt und länger als der neue Inhalt"))
	kaputt := errors.New("Close")
	biegeUm(t, 1, func(f zielDatei) zielDatei { return schliessFehlerDatei{f, kaputt} })
	if err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission); err != nil {
		t.Fatalf("Fehler %v, erwartet Erfolg: der Inhalt steht nach dem Sync vollständig", err)
	}
	if b, _ := os.ReadFile(ziel); string(b) != "neuer Inhalt, kürzer" {
		t.Errorf("Ziel %q", b)
	}
}

// F-344 (a): Scheitert schon das erste Truncate dauerhaft, ist die Datei unberührt. Die Meldung nennt die Ursache und
// nicht „beschädigt“.
type truncNie struct {
	zielDatei
	fehler error
}

func (t truncNie) Truncate(int64) error { return t.fehler }

func TestRueckfallTruncateDauerhaftNichtBeschaedigt(t *testing.T) {
	alt := []byte("alt und länger als der neue Inhalt")
	ziel := zielMit(t, alt)
	kaputt := errors.New("Truncate verboten")
	biegeUm(t, 1, func(f zielDatei) zielDatei { return truncNie{f, kaputt} })
	err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission)
	if !errors.Is(err, kaputt) {
		t.Fatalf("Fehler %v, erwartet die Ursache des Kürzens", err)
	}
	if errors.Is(err, ErrZielBeschaedigt) {
		t.Errorf("fälschlich als beschädigt gemeldet, die Datei ist unberührt: %v", err)
	}
	if b, _ := os.ReadFile(ziel); !bytes.Equal(b, alt) {
		t.Errorf("Ziel %q, erwartet der alte Inhalt", b)
	}
}

// ab scheitert ab dem k-ten Aufruf von Truncate, Seek, Write oder Sync dauerhaft (Medium weg).
type abDatei struct {
	zielDatei
	k      int
	n      *int
	fehler error
}

func (a abDatei) zaehle() error {
	*a.n++
	if *a.n >= a.k {
		return a.fehler
	}
	return nil
}

func (a abDatei) Truncate(g int64) error {
	if err := a.zaehle(); err != nil {
		return err
	}
	return a.zielDatei.Truncate(g)
}

func (a abDatei) Seek(o int64, w int) (int64, error) {
	if err := a.zaehle(); err != nil {
		return 0, err
	}
	return a.zielDatei.Seek(o, w)
}

func (a abDatei) Write(p []byte) (int, error) {
	if err := a.zaehle(); err != nil {
		return 0, err
	}
	return a.zielDatei.Write(p)
}

func (a abDatei) Sync() error {
	if err := a.zaehle(); err != nil {
		return err
	}
	return a.zielDatei.Sync()
}

// F-344: Für jeden Fehlerort gilt: Erfolg heißt neuer Inhalt; steht noch der alte Inhalt da, wird nicht „beschädigt“
// gemeldet; steht etwas anderes da, wird „beschädigt“ gemeldet.
func TestRueckfallJederFehlerortMeldetRichtig(t *testing.T) {
	const alt, neu = "alter Inhalt, der deutlich länger ist als der neue", "neuer Inhalt, kürzer"
	kaputt := errors.New("kaputt")
	for k := 1; k <= 10; k++ {
		ziel := zielMit(t, []byte(alt))
		n := 0
		biegeUm(t, 1, func(f zielDatei) zielDatei { return abDatei{f, k, &n, kaputt} })
		err := schreibeDirekt(ziel, schreibeNeu, os.ErrPermission)
		b, _ := os.ReadFile(ziel)
		got := string(b)
		switch {
		case err == nil && got != neu:
			t.Errorf("k=%d: Erfolg gemeldet, Inhalt %q", k, got)
		case err != nil && got == alt && errors.Is(err, ErrZielBeschaedigt):
			t.Errorf("k=%d: beschädigt gemeldet, der alte Inhalt steht unversehrt: %v", k, err)
		case err != nil && got != alt && !errors.Is(err, ErrZielBeschaedigt):
			t.Errorf("k=%d: Inhalt %q verändert, aber nicht als beschädigt gemeldet: %v", k, got, err)
		}
	}
}
