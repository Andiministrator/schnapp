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

// sperreNehmen nimmt die Sperre zur Ausgangsdatei pfad im Ordner ort, ohne zu warten. Hält sie ein anderer, kommt
// ErrBelegt. Wurde die Sperrdatei zwischen Öffnen und Sperren entfernt (ein anderes Fenster endete, Aufräumen), sperrt
// der Deskriptor eine verwaiste Datei; dann von vorn.
func sperreNehmen(ort, pfad string) (_ *sperre, fehler error) {
	defer errprefix.Defer("rescue", &fehler)
	if err := os.MkdirAll(ort, 0o700); err != nil {
		return nil, err
	}
	name := filepath.Join(ort, sperrName(pfad))
	for range 10 {
		f, belegt, err := oeffneGesperrt(name)
		if err != nil {
			return nil, err
		}
		if belegt {
			return nil, ErrBelegt
		}
		a, err1 := f.Stat()
		b, err2 := os.Lstat(name)
		if err1 == nil && err2 == nil && os.SameFile(a, b) {
			return &sperre{pfad: pfad, f: f, name: name}, nil
		}
		_ = f.Close()
	}
	return nil, errors.New("rescue: Sperrdatei wechselt ständig")
}

// oeffneGesperrt öffnet (oder legt an) die Sperrdatei name und versucht die Sperre ohne Warten. belegt: ein anderer hält
// sie (die Datei ist dann geschlossen).
func oeffneGesperrt(name string) (_ *os.File, belegt bool, _ error) {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, false, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, true, nil
		}
		return nil, false, err
	}
	return f, false, nil
}

// frei entfernt die Sperrdatei, solange sie noch gehalten ist, und gibt die Sperre frei. Wer die Datei inzwischen
// geöffnet hat, merkt am Vergleich in sperreNehmen, dass sie verwaist ist.
func (s *sperre) frei() error {
	err := os.Remove(s.name)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	return errors.Join(err, s.f.Close())
}

// raeumeSperre entfernt die Sperrdatei name, wenn niemand sie hält (Aufräumen nach einem Absturz); ok: entfernt.
func raeumeSperre(name string) (ok bool, _ error) {
	f, belegt, err := oeffneGesperrt(name)
	if err != nil || belegt {
		return false, err
	}
	return true, (&sperre{f: f, name: name}).frei()
}
