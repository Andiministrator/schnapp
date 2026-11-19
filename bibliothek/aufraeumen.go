// SPDX-License-Identifier: GPL-3.0-or-later

package bibliothek

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

	"github.com/Andiministrator/schnapp/datei"
)

// Einstellung des automatischen Aufräumens (ADR-0016).
type Einstellung struct {
	An   bool
	Tage int // Frist in Tagen, mindestens 1
}

// Voreinstellung: an, 90 Tage.
var Voreinstellung = Einstellung{An: true, Tage: 90}

// Fehler des Aufräumens.
var (
	ErrFrist              = errors.New("bibliothek: Frist muss mindestens 1 Tag betragen")
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
	if e.Tage < 1 {
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
		v, err := b.inPapierkorb(ein, jetzt)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue // Datei fehlt: Sache des Abgleichs, nicht melden
			}
			erg.Uebersprungen = append(erg.Uebersprungen, Uebersprungen{Pfad: ein.Pfad, Grund: err})
			continue
		}
		weg[ein.Pfad] = true
		erg.Verschoben = append(erg.Verschoben, v)
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
func (b *Bibliothek) inPapierkorb(ein Eintrag, jetzt time.Time) (Verschoben, error) {
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
	gel, err := datei.LeseArbeitsdatei(f)
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
	name := basis
	for n := 2; ; n++ {
		_, e1 := os.Lstat(filepath.Join(dateien, name))
		_, e2 := os.Lstat(filepath.Join(infos, name+".trashinfo"))
		if errors.Is(e1, fs.ErrNotExist) && errors.Is(e2, fs.ErrNotExist) {
			break
		}
		if n > 10000 {
			return Verschoben{}, errors.New("bibliothek: kein freier Name im Papierkorb")
		}
		name = fmt.Sprintf("%s-%d%s", stamm, n, endung)
	}
	ziel := filepath.Join(dateien, name)
	if err := umbenenne(quelle, ziel); err != nil {
		if errors.Is(err, syscall.EXDEV) {
			return Verschoben{}, ErrAnderesDateisystem
		}
		return Verschoben{}, err
	}
	pfadUrl := (&url.URL{Path: absolut}).EscapedPath()
	info := "[Trash Info]\nPath=" + pfadUrl + "\nDeletionDate=" + jetzt.In(b.Zone).Format("2006-01-02T15:04:05") + "\n"
	err = schreibeAtomar(filepath.Join(infos, name+".trashinfo"), func(w io.Writer) error {
		_, err := io.WriteString(w, info)
		return err
	})
	if err != nil {
		// Ohne .trashinfo ließe sich die Aufnahme nicht wiederherstellen: zurücklegen, nichts löschen.
		if rueck := umbenenne(ziel, quelle); rueck != nil {
			return Verschoben{}, fmt.Errorf("%w; Rücklegen scheiterte, Datei liegt in %s: %v", err, ziel, rueck)
		}
		return Verschoben{}, err
	}
	return Verschoben{Pfad: ein.Pfad, Papierkorb: ziel}, nil
}
