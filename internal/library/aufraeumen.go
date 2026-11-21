// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// Einstellung des automatischen Aufräumens (ADR-0016).
type Einstellung struct {
	An   bool
	Tage int // Frist in Tagen, mindestens 1
}

// Voreinstellung: an, 90 Tage.
var Voreinstellung = Einstellung{An: true, Tage: 90}

// MaxTage ist die längste einstellbare Frist (100 Jahre).
const MaxTage = 36500

// Fehler des Aufräumens.
var (
	ErrFrist              = errors.New("bibliothek: Frist muss zwischen 1 und 36500 Tagen liegen")
	ErrAnderesDateisystem = errors.New("bibliothek: Ablage und Papierkorb liegen auf verschiedenen Dateisystemen")
	ErrGeschuetzt         = errors.New("bibliothek: Datei trägt Schlagworte oder ist bearbeitet (laut Datei)")
)

// Verschoben ist eine in den Papierkorb gelegte Aufnahme.
type Verschoben struct {
	Pfad       string // relativ zur Ablage
	Papierkorb string // absoluter Pfad der Datei im Papierkorb
}

// Uebersprungen ist eine Aufnahme, die das Aufräumen erfassen wollte, aber nicht bewegt hat.
type Uebersprungen struct {
	Pfad  string
	Grund error
}

// AufraeumErgebnis ist das Ergebnis von Aufraeumen.
type AufraeumErgebnis struct {
	Verschoben    []Verschoben
	Uebersprungen []Uebersprungen
}

// Aufraeumen legt Aufnahmen in den freedesktop-Papierkorb des Nutzers, die ohne Schlagworte, nicht bearbeitet und
// älter als e.Tage sind. Aus (e.An == false) tut nichts. Gelöscht wird nie: Verschoben wird per Umbenennen; liegt
// die Ablage auf einem anderen Dateisystem als der Papierkorb, bleibt die Aufnahme liegen und wird gemeldet.
// Vor dem Verschieben prüft die Funktion Schlagworte und Bearbeitung in der Datei selbst (der Index ist nur ein
// Cache). Danach fallen die verschobenen Aufnahmen aus dem Index.
func (b *Bibliothek) Aufraeumen(jetzt time.Time, e Einstellung) (AufraeumErgebnis, error) {
	var erg AufraeumErgebnis
	if !e.An {
		return erg, nil
	}
	// Obergrenze: 100 Jahre. Darüber liefe time.Duration über und die Frist würde negativ — dann gingen auch
	// frische Aufnahmen in den Papierkorb (F-50).
	if e.Tage < 1 || e.Tage > MaxTage {
		return erg, ErrFrist
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	g, err := b.lade()
	if err != nil {
		return erg, err
	}
	frist := time.Duration(e.Tage) * 24 * time.Hour
	weg := map[string]bool{}
	for _, ein := range g.Eintraege {
		if len(ein.Schlagworte) > 0 || ein.Bearbeitet || jetzt.Sub(ein.Erstellt) <= frist {
			continue
		}
		v, err := b.inPapierkorb(ein, jetzt, frist)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue // Datei fehlt: Sache des Abgleichs, nicht melden
			}
			erg.Uebersprungen = append(erg.Uebersprungen, Uebersprungen{Pfad: ein.Pfad, Grund: err})
			continue
		}
		weg[ein.Pfad] = true
		erg.Verschoben = append(erg.Verschoben, v)
		_ = b.entferneVorschau(ein.Pfad) // keine Kopie im Cache zurücklassen (F-51)
	}
	if len(weg) == 0 {
		return erg, nil
	}
	var rest []Eintrag
	for _, ein := range g.Eintraege {
		if !weg[ein.Pfad] {
			rest = append(rest, ein)
		}
	}
	if err := b.schreibeIndex(rest); err != nil {
		return erg, fmt.Errorf("%w: %v", ErrIndex, err)
	}
	return erg, nil
}

// inPapierkorb prüft die Datei und verschiebt sie samt .trashinfo in den Papierkorb.
func (b *Bibliothek) inPapierkorb(ein Eintrag, jetzt time.Time, frist time.Duration) (Verschoben, error) {
	quelle := filepath.Join(b.Orte.Ablage, filepath.FromSlash(ein.Pfad))
	st, err := os.Lstat(quelle)
	if err != nil {
		return Verschoben{}, err
	}
	if !st.Mode().IsRegular() {
		return Verschoben{}, fmt.Errorf("%w: keine gewöhnliche Datei", ErrPfad)
	}
	// Maßgeblich sind die Angaben der Datei selbst, nicht der Cache.
	f, err := os.Open(quelle)
	if err != nil {
		return Verschoben{}, err
	}
	gel, err := fileformat.LeseArbeitsdatei(f)
	f.Close()
	if err != nil {
		return Verschoben{}, err
	}
	if gel.Fremd {
		return Verschoben{}, ErrFremd
	}
	if len(gel.Meta.Schlagworte) > 0 || gel.Meta.Bearbeitet {
		return Verschoben{}, ErrGeschuetzt
	}
	// Auch das Alter zählt aus der Datei (Erstellt im snAP-Block, sonst Änderungszeit), nicht aus dem Index (F-54).
	erstellt := gel.Meta.Erstellt
	if erstellt.IsZero() {
		erstellt = st.ModTime()
	}
	if jetzt.Sub(erstellt) <= frist {
		return Verschoben{}, ErrGeschuetzt
	}

	dateien := filepath.Join(b.Orte.Papierkorb, "files")
	infos := filepath.Join(b.Orte.Papierkorb, "info")
	for _, o := range []string{b.Orte.Papierkorb, dateien, infos} {
		if err := os.MkdirAll(o, rechteOrdner); err != nil {
			return Verschoben{}, err
		}
	}
	absolut, err := filepath.Abs(quelle)
	if err != nil {
		return Verschoben{}, err
	}
	basis := filepath.Base(quelle)
	stamm, endung := strings.TrimSuffix(basis, filepath.Ext(basis)), filepath.Ext(basis)
	pfadUrl := (&url.URL{Path: absolut}).EscapedPath()
	inhalt := "[Trash Info]\nPath=" + pfadUrl + "\nDeletionDate=" + jetzt.In(b.Zone).Format("2006-01-02T15:04:05") + "\n"
	// Nach der freedesktop-Spezifikation reserviert die .trashinfo den Namen: zuerst exklusiv anlegen (O_EXCL),
	// dann die Datei verschieben. So überschreibt Schnapp keinen Eintrag, den ein anderes Programm gleichzeitig
	// anlegt (F-53).
	name, infoPfad := basis, ""
	for n := 2; ; n++ {
		kandidat := filepath.Join(infos, name+".trashinfo")
		_, belegt := os.Lstat(filepath.Join(dateien, name))
		if errors.Is(belegt, fs.ErrNotExist) {
			f, err := os.OpenFile(kandidat, os.O_WRONLY|os.O_CREATE|os.O_EXCL, rechteDatei)
			if err == nil {
				_, err = io.WriteString(f, inhalt)
				if err == nil {
					err = f.Sync()
				}
				if cerr := f.Close(); err == nil {
					err = cerr
				}
				if err != nil {
					entferneEigeneInfo(infos, kandidat)
					return Verschoben{}, err
				}
				infoPfad = kandidat
				break
			}
			if !errors.Is(err, fs.ErrExist) {
				return Verschoben{}, err
			}
		}
		if n > 10000 {
			return Verschoben{}, errors.New("bibliothek: kein freier Name im Papierkorb")
		}
		name = fmt.Sprintf("%s-%d%s", stamm, n, endung)
	}
	ziel := filepath.Join(dateien, name)
	if err := umbenenne(quelle, ziel); err != nil {
		// Verschieben gescheitert: nur die eben selbst angelegte Reservierung zurücknehmen; die Aufnahme bleibt liegen.
		entferneEigeneInfo(infos, infoPfad)
		if errors.Is(err, syscall.EXDEV) {
			return Verschoben{}, ErrAnderesDateisystem
		}
		return Verschoben{}, err
	}
	return Verschoben{Pfad: ein.Pfad, Papierkorb: ziel}, nil
}

// entferneEigeneInfo löscht eine .trashinfo, die dieser Lauf eben selbst exklusiv angelegt hat (Reservierung
// zurücknehmen). Nur Dateien im info-Ordner mit der Endung .trashinfo.
func entferneEigeneInfo(infos, pfad string) {
	if pfad != "" && filepath.Dir(pfad) == filepath.Clean(infos) && strings.HasSuffix(pfad, ".trashinfo") {
		os.Remove(pfad)
	}
}
