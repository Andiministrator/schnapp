// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Sperre des Index gegen einen zweiten Prozess (F-686, ADR-0016 Nachtrag 6): b.mu serialisiert nur innerhalb eines
// Prozesses. Jedes Lesen-Ändern-Schreiben des Index (schreibeIndex: Abgleich, Speichern, InPapierkorb, Entfernen,
// SetzeSchlagworte, Aufraeumen, Neuaufbau) und jedes Anhängen (haengeAn) läuft zusätzlich unter einer flock-Sperre auf
// der Datei `<Index>.lock` neben dem Index. Sonst konnte ein Prozess die Zeile verschlucken, die ein anderer zwischen
// Lesen und Umbenennen angehängt hatte. Lesen allein (Laden, Suchen) sperrt nicht: Der Index wird nur per Umbenennen
// ersetzt oder am Ende ergänzt, ein Leser sieht also immer einen ganzen Stand.
//
// Die Sperrdatei bleibt liegen (0600, Ordner wie der Index); sie wird nie gelöscht, damit kein zweiter Prozess eine
// verwaiste Datei sperrt. Kann das Dateisystem kein flock (ENOLCK, ENOTSUP/EOPNOTSUPP), läuft es ohne Sperre weiter
// wie bisher. Hält ein anderer Prozess die Sperre länger als sperrFrist, kommt errIndexBelegt; der Aufrufer meldet es
// wie jeden Indexfehler (ErrIndex).

// sperrFrist ist die längste Wartezeit auf die Sperre des Index. Der Neuaufbau hält sie nur für Ablaufen, Metadaten und
// Schreiben, nicht für das Heilen der Vorschaubilder (F-691; 10 000 Aufnahmen rund 0,8 s, BenchmarkNeuaufbau10000).
// Wird die Frist doch einmal überschritten, scheitert nur die Indexzeile, die Aufnahme bleibt (der nächste Neuaufbau
// holt sie).
const sperrFrist = 10 * time.Second

// sperrTakt ist der Abstand der Versuche, solange ein anderer Prozess die Sperre hält.
const sperrTakt = 5 * time.Millisecond

// errIndexBelegt: Ein anderer Prozess hielt die Sperre des Index länger als sperrFrist.
var errIndexBelegt = errors.New("library: Index von einem anderen Prozess gesperrt")

// flockFn ist syscall.Flock; Tests ersetzen es, um ein Dateisystem ohne flock nachzubauen.
var flockFn = syscall.Flock

// sperrPfad ist die Sperrdatei neben dem Index.
func (b *Bibliothek) sperrPfad() string { return b.Orte.Index + ".lock" }

// sperreIndex nimmt die Sperre des Index und liefert die Freigabe. Läuft unter b.mu; nie verschachtelt aufrufen (ein
// zweiter Deskriptor desselben Prozesses wartete auf den ersten).
func (b *Bibliothek) sperreIndex() (frei func(), _ error) {
	if err := eigenerOrdner(filepath.Dir(b.Orte.Index)); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(b.sperrPfad(), os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, rechteDatei)
	if err != nil {
		return nil, err
	}
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() {
		f.Close()
		return nil, errors.Join(err, errors.New("library: Sperrdatei des Index ist keine reguläre Datei"))
	}
	fd := int(f.Fd())
	ende := time.Now().Add(sperrFrist)
	for {
		err := flockFn(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		switch {
		case err == nil:
			return func() {
				_ = flockFn(fd, syscall.LOCK_UN)
				f.Close()
			}, nil
		case errors.Is(err, syscall.ENOLCK), errors.Is(err, syscall.ENOTSUP), errors.Is(err, syscall.EOPNOTSUPP):
			f.Close()
			return func() {}, nil // kein flock auf diesem Dateisystem: ohne Sperre weiter
		case errors.Is(err, syscall.EWOULDBLOCK), errors.Is(err, syscall.EINTR):
			if time.Now().After(ende) {
				f.Close()
				return nil, errIndexBelegt
			}
			time.Sleep(sperrTakt)
		default:
			f.Close()
			return nil, err
		}
	}
}

// unterSperre führt f unter der Sperre des Index aus. Läuft unter b.mu.
func (b *Bibliothek) unterSperre(f func() error) error {
	frei, err := b.sperreIndex()
	if err != nil {
		return err
	}
	defer frei()
	return f()
}
