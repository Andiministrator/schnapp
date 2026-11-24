// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Andiministrator/schnapp/internal/errprefix"
)

// Sperre einer Sitzung (F-413): Solange ein Editor eine Ausgangsdatei offen hat, hält er eine flock-Sperre auf
// <SHA-256 des Pfads>.lock im Ordner der Rettungskopien. Ein zweites Fenster auf derselben Datei bekommt sie nicht und
// lässt die Kopie des ersten in Ruhe: kein Anwenden, kein Beiseitelegen, kein Schreiben, kein Löschen. Stürzt der
// Halter ab, gibt der Kern die Sperre frei; das nächste Fenster wendet die Kopie dann wie gewohnt an. Die Sperre steht
// nicht in der Kopie (ihr Format bleibt), und sie braucht keinen Blick auf fremde Prozesse.

// sperrEndung ist die Endung der Sperrdatei.
const sperrEndung = ".lock"

// ErrBelegt meldet, dass ein anderes Fenster dieselbe Ausgangsdatei offen hat und ihre Rettungskopie führt.
var ErrBelegt = errors.New("rescue: die Datei ist in einem anderen Fenster geöffnet, keine Rettungskopie für dieses")

// sperrName ist der Dateiname der Sperre zur Ausgangsdatei pfad.
func sperrName(pfad string) string { return strings.TrimSuffix(Name(pfad), ".json") + sperrEndung }

// sperre ist eine gehaltene Sperre.
type sperre struct {
	pfad string // Ausgangsdatei
	f    *os.File
	name string // voller Pfad der Sperrdatei
}

// errKeinFlock meldet ein Dateisystem, das flock nicht kann (ENOLCK, ENOTSUP/EOPNOTSUPP): Nur dann läuft eine Sitzung
// ohne Sperre weiter (F-440). Jeder andere Fehler beim Öffnen oder Sperren (Symlink am Sperrnamen, Verzeichnis, fehlende
// Rechte, ...) macht sie stumm: Wer die Sperre nicht prüfen kann, wendet keine Kopie an und schreibt und löscht keine.
var errKeinFlock = errors.New("rescue: Dateisystem kann keine flock-Sperre")

// Nähte für Tests (F-440): das Sperren selbst und ein Haken zwischen Sperren und Prüfen der Sperrdatei.
var (
	flockFn    = syscall.Flock
	sperrHaken = func(name string) {}
)

// sperreNehmen nimmt die Sperre zur Ausgangsdatei pfad im Ordner ort, ohne zu warten. Hält sie ein anderer, kommt
// ErrBelegt, kann das Dateisystem kein flock, errKeinFlock. Wurde die Sperrdatei zwischen Öffnen und Sperren entfernt
// (ein anderes Fenster endete, Aufräumen), sperrt der Deskriptor eine verwaiste Datei; dann von vorn.
func sperreNehmen(ort, pfad string) (_ *sperre, fehler error) {
	defer errprefix.Defer("rescue", &fehler)
	if err := os.MkdirAll(ort, 0o700); err != nil {
		return nil, err
	}
	name := filepath.Join(ort, sperrName(pfad))
	for range 10 {
		f, belegt, err := oeffneGesperrt(name, true)
		if err != nil {
			return nil, err
		}
		if belegt {
			return nil, ErrBelegt
		}
		if gleicheDatei(f, name) {
			return &sperre{pfad: pfad, f: f, name: name}, nil
		}
		_ = f.Close()
	}
	return nil, errors.New("rescue: Sperrdatei wechselt ständig")
}

// gleicheDatei meldet, ob der offene Deskriptor f und der Name name dieselbe reguläre Datei sind.
func gleicheDatei(f *os.File, name string) bool {
	a, err1 := f.Stat()
	b, err2 := os.Lstat(name)
	return err1 == nil && err2 == nil && a.Mode().IsRegular() && os.SameFile(a, b)
}

// oeffneGesperrt öffnet (mit anlegen: oder legt an) die Sperrdatei name und versucht die Sperre ohne Warten. belegt: ein
// anderer hält sie (die Datei ist dann geschlossen). O_NOFOLLOW weist einen Symlink am Namen ab, O_NONBLOCK verhindert
// das Hängen an einer Pipe; was keine reguläre Datei ist, ist ein Fehler.
func oeffneGesperrt(name string, anlegen bool) (_ *os.File, belegt bool, _ error) {
	flags := os.O_RDWR | syscall.O_NOFOLLOW | syscall.O_NONBLOCK
	if anlegen {
		flags |= os.O_CREATE
	}
	f, err := os.OpenFile(name, flags, 0o600)
	if err != nil {
		return nil, false, err
	}
	if st, err := f.Stat(); err != nil || !st.Mode().IsRegular() {
		_ = f.Close()
		return nil, false, errors.Join(err, errors.New("rescue: Sperrdatei ist keine reguläre Datei"))
	}
	if err := flockFn(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		switch {
		case errors.Is(err, syscall.EWOULDBLOCK):
			return nil, true, nil
		case errors.Is(err, syscall.ENOLCK), errors.Is(err, syscall.ENOTSUP), errors.Is(err, syscall.EOPNOTSUPP):
			return nil, false, errors.Join(errKeinFlock, err)
		}
		return nil, false, err
	}
	sperrHaken(name)
	return f, false, nil
}

// frei entfernt die Sperrdatei, solange sie noch gehalten ist und der Name noch dieselbe Datei meint (os.SameFile:
// entfernt wird nur, was geprüft ist), und gibt die Sperre frei. Wer die Datei inzwischen geöffnet hat, merkt am
// Vergleich in sperreNehmen, dass sie verwaist ist.
func (s *sperre) frei() error {
	var err error
	if gleicheDatei(s.f, s.name) {
		err = os.Remove(s.name)
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
	}
	return errors.Join(err, s.f.Close())
}

// raeumeSperre entfernt die Sperrdatei name, wenn niemand sie hält (Aufräumen nach einem Absturz); ok: entfernt. Sie
// legt nichts an; ist der Name nach dem Sperren nicht mehr dieselbe Datei, bleibt er unberührt.
func raeumeSperre(name string) (ok bool, _ error) {
	f, belegt, err := oeffneGesperrt(name, false)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil || belegt {
		return false, err
	}
	if !gleicheDatei(f, name) {
		return false, f.Close()
	}
	err = os.Remove(name)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	return err == nil, errors.Join(err, f.Close())
}

// sperreGehalten meldet, ob ein anderer die Sperrdatei name hält (F-702). Sie legt nichts an; fehlt die Datei oder kann
// das Dateisystem kein flock, hält niemand. Jeder andere Fehler wird gemeldet, der Aufrufer fasst dann nichts an.
func sperreGehalten(name string) (bool, error) {
	f, belegt, err := oeffneGesperrt(name, false)
	switch {
	case errors.Is(err, os.ErrNotExist), errors.Is(err, errKeinFlock):
		return false, nil
	case err != nil:
		return false, err
	case belegt:
		return true, nil
	}
	return false, f.Close() // freigegeben, die Datei bleibt (nur ihr Halter oder raeumeSperre entfernt sie)
}
