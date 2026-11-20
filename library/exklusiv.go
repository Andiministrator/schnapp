// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// verknuepfe ist os.Link; Tests ersetzen es, um einen Fehlschlag (kein Hardlink-Support) nachzubauen.
var verknuepfe = os.Link

// maxNamenszusatz begrenzt die Suche nach einem freien Namen beim Ablegen.
const maxNamenszusatz = 10000

// schreibeTemp schreibt den Inhalt in eine neue temporäre Datei (0600) im Ordner von ziel und synchronisiert sie.
// Zurück kommt deren Pfad; bei einem Fehler ist sie schon wieder entfernt.
func schreibeTemp(ziel string, schreibe func(io.Writer) error) (tmpPfad string, fehler error) {
	ordner := filepath.Dir(ziel)
	tmp, err := os.CreateTemp(ordner, tempPraefix+"*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	geschlossen := false
	defer func() {
		if fehler == nil {
			return
		}
		if !geschlossen {
			tmp.Close()
		}
		entferneEigeneTemp(ordner, name)
	}()
	if err := tmp.Chmod(rechteDatei); err != nil {
		return "", err
	}
	if err := schreibe(tmp); err != nil {
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		return "", err
	}
	geschlossen = true
	if err := tmp.Close(); err != nil {
		return "", err
	}
	return name, nil
}

// ErrTempRest: Die Aufnahme ist abgelegt (oder gespeichert), aber eine temporäre Datei von Schnapp mit demselben
// Inhalt ließ sich nicht entfernen. Nach dem Link-Weg ist so ein Rest ein zweiter Name derselben Aufnahme; er zeigt
// nach einem späteren Schwärzen weiter den ungeschwärzten Stand (F-274, Kernprinzip 2; ADR-0022 Nachtrag 2). Der Aufrufer meldet es,
// die Aufnahme selbst bleibt gültig.
var ErrTempRest = errors.New("bibliothek: temporäre Datei nicht entfernt")

// entferneTemp ist os.Remove für temporäre Dateien; Tests ersetzen es, um ein scheiterndes Entfernen nachzubauen.
var entferneTemp = os.Remove

// tempAlter: Erst temporäre Dateien, die älter sind, gelten als liegen geblieben. Ein Schreibvorgang dauert
// Sekunden; eine Minute Abstand schützt die gerade laufende Ablage eines zweiten Schnapp-Prozesses.
const tempAlter = time.Minute

// entferneEigeneTemp löscht eine temporäre Datei dieses Laufs: nur im genannten Ordner und mit eigenem Präfix.
// Liefert den Fehler (fehlende Datei zählt nicht).
func entferneEigeneTemp(ordner, name string) error {
	if filepath.Dir(name) != filepath.Clean(ordner) || !strings.HasPrefix(filepath.Base(name), tempPraefix) {
		return nil
	}
	if err := entferneTemp(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// syncOrdner schreibt die Verzeichniseinträge eines Ordners auf den Datenträger (nach Link und rename, F-274 c).
// Best Effort: Nicht jedes Dateisystem kann das, und die Datei selbst ist schon gesichert.
func syncOrdner(ordner string) {
	f, err := os.Open(ordner)
	if err != nil {
		return
	}
	_ = f.Sync()
	f.Close()
}

// raeumeTempReste entfernt liegen gebliebene temporäre Dateien von Schnapp (Präfix tempPraefix) im Ordner: erstens
// jede mit demselben Inode wie eine der Dateien in gleich, ohne Altersgrenze (nach dem Link-Weg ist sie ein zweiter
// Name der Aufnahme, die gleich ersetzt oder entfernt wird; ihre Änderungszeit ist die der Aufnahme, also oft jünger
// als eine Minute oder, bei verstellter Uhr, in der Zukunft, Z-1); zweitens jede, die älter als tempAlter ist. Nur
// reguläre Dateien im genannten Ordner, nie ein Symlink, nie eine Aufnahme (anderer Name), nie ein Vorschaubild. Die
// Altersgrenze schützt die gerade laufende Ablage eines zweiten Schnapp-Prozesses. Gewählt statt „nur Reste mit
// gleichem Inode wie eine Aufnahme“: Auch ein Rest mit halb geschriebenem Inhalt trägt Bildpunkte einer Aufnahme, und
// die Inode-Prüfung fände ihn nicht.
//
// Reste, die danach bleiben, weil sie eine Zeit weit in der Zukunft tragen, werden nicht gelöscht, aber als
// ErrTempRest gemeldet; ebenso, wenn meldeJunge gesetzt ist, die zu jungen, und Reste, deren Entfernen scheitert.
// meldeJunge gilt für den Ordner der Aufnahme (Speichern, Entfernen): Ein junger fremder Rest dort ist halb
// geschrieben oder gehört einem zweiten Prozess. Im Vorschau-Ordner bleibt es aus, denn dort schreibt jedes Ablegen
// eines zweiten Prozesses kurz eine solche Datei. Das Aufräumen läuft für die übrigen weiter.
func raeumeTempReste(ordner string, jetzt time.Time, meldeJunge bool, gleich ...os.FileInfo) (entfernt int, fehler error) {
	eintraege, err := os.ReadDir(ordner)
	if err != nil {
		return 0, nil // kein Ordner, nichts zu räumen
	}
	for _, e := range eintraege {
		if !e.Type().IsRegular() || !strings.HasPrefix(e.Name(), tempPraefix) {
			continue
		}
		fi, err := e.Info()
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		eigen := false
		for _, g := range gleich {
			if g != nil && os.SameFile(fi, g) {
				eigen = true
			}
		}
		if !eigen && jetzt.Sub(fi.ModTime()) < tempAlter {
			grund := "zu jung"
			zukunft := fi.ModTime().After(jetzt.Add(tempAlter))
			if zukunft {
				grund = "Änderungszeit in der Zukunft"
			}
			if meldeJunge || zukunft {
				fehler = errors.Join(fehler, fmt.Errorf("%w: %s: %s, nicht entfernt", ErrTempRest, e.Name(), grund))
			}
			continue
		}
		if err := entferneEigeneTemp(ordner, filepath.Join(ordner, e.Name())); err != nil {
			fehler = errors.Join(fehler, fmt.Errorf("%w: %s: %v", ErrTempRest, e.Name(), err))
			continue
		}
		entfernt++
	}
	return entfernt, fehler
}

// keinHardlink meldet Fehler, die heißen „dieses Dateisystem oder diese Stelle kann keine Hardlinks“ — im Gegensatz
// zu EEXIST (Name belegt) und echten E/A-Fehlern.
func keinHardlink(err error) bool {
	return errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EOPNOTSUPP) ||
		errors.Is(err, syscall.EXDEV) || errors.Is(err, syscall.ENOSYS) || errors.Is(err, syscall.EMLINK) ||
		errors.Is(err, syscall.EACCES)
}

// legeExklusivAn bringt die fertige temporäre Datei tmp unter dem Namen ziel an, ohne je eine vorhandene Datei zu
// überschreiben. Ist der Name belegt, kommt fs.ErrExist zurück (der Aufrufer nimmt den nächsten Namen); tmp bleibt
// dann unberührt. Bei Erfolg ist tmp entfernt und ziel vollständig da.
//
// Weg 1 (Regelfall): os.Link(tmp, ziel). Das Anlegen des Namens ist atomar und scheitert mit EEXIST, wenn er
// belegt ist — auch gegen einen anderen Prozess. Der Inhalt steht vollständig unter dem Namen, sobald dieser sichtbar
// wird.
//
// Weg 2 (Rückfall, wenn das Dateisystem keine Hardlinks kann, etwa FAT/exFAT oder manche Netzfreigaben): ziel wird
// mit O_CREATE|O_EXCL als leerer Platzhalter reserviert (ebenfalls atomar und prozessübergreifend), danach ersetzt
// rename die Reservierung durch tmp. Überschrieben wird nur die eigene Reservierung. Preis: Zwischen Reservierung und
// rename ist unter dem Namen kurz eine leere Datei sichtbar, und ein Absturz genau dort lässt sie liegen — der
// Neuaufbau überspringt sie als beschädigt, eine Aufnahme geht nicht verloren, denn diese ist noch in tmp. Gewählt
// statt rename ohne Reservierung, weil das eine fremde Datei ersetzen könnte (F-48).
func legeExklusivAn(tmp, ziel string) error {
	err := verknuepfe(tmp, ziel)
	if err == nil {
		// Ab hier ist tmp ein zweiter Name der Aufnahme (F-274): Scheitert das Entfernen, ist sie abgelegt, aber der
		// Rest bleibt — das darf der Aufrufer nicht verschlucken (Kernprinzip 2; ADR-0022 Nachtrag 2).
		ordner := filepath.Dir(ziel)
		rest := entferneEigeneTemp(ordner, tmp)
		syncOrdner(ordner)
		if rest != nil {
			return fmt.Errorf("%w: %v", ErrTempRest, rest)
		}
		return nil
	}
	if errors.Is(err, fs.ErrExist) {
		return fs.ErrExist
	}
	if !keinHardlink(err) {
		return err
	}
	f, err := os.OpenFile(ziel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, rechteDatei)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fs.ErrExist
		}
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(ziel) // die eigene, eben angelegte Reservierung
		return err
	}
	if err := umbenenne(tmp, ziel); err != nil {
		os.Remove(ziel) // die eigene Reservierung; die Aufnahme bleibt in tmp, das der Aufrufer entfernt
		return err
	}
	syncOrdner(filepath.Dir(ziel))
	return nil
}

// schreibeNeu schreibt eine neue Datei unter dem ersten freien Namen aus namen und liefert diesen (Index in namen).
// Die Datei wird einmal vollständig in eine temporäre Datei geschrieben, dann exklusiv angelegt (legeExklusivAn).
// Ein vorhandener Name wird nie überschrieben. Bei einem Fehler bleibt nichts zurück als fremde Dateien.
func schreibeNeu(ordner string, name func(n int) string, schreibe func(io.Writer) error) (gewaehlt string, fehler error) {
	tmp, err := schreibeTemp(filepath.Join(ordner, "x"), schreibe)
	if err != nil {
		return "", err
	}
	defer func() {
		if fehler != nil {
			entferneEigeneTemp(ordner, tmp)
		}
	}()
	for n := 1; ; n++ {
		if n > maxNamenszusatz {
			return "", fmt.Errorf("bibliothek: kein freier Name in %q", ordner)
		}
		dn := name(n)
		err := legeExklusivAn(tmp, filepath.Join(ordner, dn))
		if err == nil {
			return dn, nil
		}
		if errors.Is(err, ErrTempRest) {
			return dn, err // die Datei liegt unter dn; der Rest wird gemeldet (der Aufrufer behandelt ihn)
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	}
}
