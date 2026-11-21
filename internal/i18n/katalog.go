// SPDX-License-Identifier: GPL-3.0-or-later

package i18n

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"sync/atomic"
)

// Katalog hält die Übersetzungen einer Sprache. Der Nullwert und nil sind gültig und übersetzen nichts
// (Ausgangssprache Englisch).
type Katalog struct {
	Sprache string
	regel   Mehrzahlregel
	texte   map[schluessel][]string
}

type schluessel struct{ kontext, id string }

// NeuerKatalog baut einen Katalog aus gelesenen PO-Einträgen. Unscharfe Einträge und leere Übersetzungen werden
// ausgelassen; der Kopfeintrag liefert die Mehrzahlregel (ohne Kopf: Regel des Englischen).
func NeuerKatalog(sprache string, eintraege []Eintrag) (*Katalog, error) {
	k := &Katalog{Sprache: sprache, regel: englischeRegel, texte: map[schluessel][]string{}}
	for _, e := range eintraege {
		if e.ID == "" && e.Kontext == "" {
			if r, ok := kopffeld(e, "Plural-Forms"); ok {
				regel, err := leseMehrzahlregel(r)
				if err != nil {
					return nil, fmt.Errorf("i18n: %s: %w", sprache, err)
				}
				k.regel = regel
			}
			continue
		}
		if e.Unscharf || len(e.Texte) == 0 {
			continue
		}
		vollstaendig := true
		for _, t := range e.Texte {
			if t == "" {
				vollstaendig = false
			}
		}
		if vollstaendig {
			k.texte[schluessel{e.Kontext, e.ID}] = e.Texte
		}
	}
	return k, nil
}

// kopffeld liest ein Feld wie "Plural-Forms: …" aus dem Kopfeintrag.
func kopffeld(kopf Eintrag, name string) (string, bool) {
	if len(kopf.Texte) == 0 {
		return "", false
	}
	for _, zeile := range strings.Split(kopf.Texte[0], "\n") {
		n, w, ok := strings.Cut(zeile, ":")
		if ok && strings.EqualFold(strings.TrimSpace(n), name) {
			return strings.TrimSpace(w), true
		}
	}
	return "", false
}

// T übersetzt text; fehlt die Übersetzung, kommt text zurück.
func (k *Katalog) T(text string) string { return k.TK("", text) }

// TK übersetzt text im Kontext kontext (msgctxt), etwa wenn dasselbe englische Wort zwei Bedeutungen hat.
func (k *Katalog) TK(kontext, text string) string {
	if k != nil {
		if t, ok := k.texte[schluessel{kontext, text}]; ok {
			return t[0]
		}
	}
	return text
}

// N wählt die Mehrzahlform für n: einzahl und mehrzahl sind die englischen Formen. Formatieren mit n bleibt beim
// Aufrufer (fmt.Sprintf).
func (k *Katalog) N(einzahl, mehrzahl string, n int) string {
	if k != nil {
		if t, ok := k.texte[schluessel{"", einzahl}]; ok {
			if i := k.regel.Index(n); i < len(t) {
				return t[i]
			}
		}
	}
	if englischeRegel.Index(n) == 0 {
		return einzahl
	}
	return mehrzahl
}

// Sprachen nennt die Sprachen, für die dateien eine PO-Datei "<sprache>.po" enthält, sortiert.
func Sprachen(dateien fs.FS) ([]string, error) {
	namen, err := fs.Glob(dateien, "*.po")
	if err != nil {
		return nil, fmt.Errorf("i18n: %w", err)
	}
	var res []string
	for _, n := range namen {
		res = append(res, strings.TrimSuffix(path.Base(n), ".po"))
	}
	sort.Strings(res)
	return res, nil
}

// Lade liest "<sprache>.po" aus dateien. Für die Ausgangssprache gibt es einen leeren Katalog.
func Lade(dateien fs.FS, sprache string) (*Katalog, error) {
	if sprache == Ausgangssprache {
		return &Katalog{Sprache: sprache, regel: englischeRegel}, nil
	}
	f, err := dateien.Open(sprache + ".po")
	if err != nil {
		return nil, fmt.Errorf("i18n: %w", err)
	}
	defer f.Close()
	eintraege, err := lesePO(f)
	if err != nil {
		return nil, fmt.Errorf("i18n: %s.po: %w", sprache, err)
	}
	return NeuerKatalog(sprache, eintraege)
}

// aktiv ist der Katalog hinter den Paketfunktionen T, TK und N.
var aktiv atomic.Pointer[Katalog]

// Setze macht k zum Katalog der Paketfunktionen T, TK und N.
func Setze(k *Katalog) { aktiv.Store(k) }

// Starte wählt die Sprache aus Einstellung und Umgebung des Programms, lädt sie und setzt sie aktiv. Scheitert das
// Laden, bleibt Englisch aktiv und der Fehler kommt zurück; das Programm läuft trotzdem weiter.
func Starte(dateien fs.FS, einstellung string) (string, error) {
	verfuegbar, err := Sprachen(dateien)
	if err != nil {
		return Ausgangssprache, err
	}
	s := Waehle(os.Getenv, einstellung, verfuegbar)
	k, err := Lade(dateien, s)
	if err != nil {
		Setze(nil)
		return Ausgangssprache, err
	}
	Setze(k)
	return s, nil
}

// T übersetzt text mit dem aktiven Katalog.
func T(text string) string { return aktiv.Load().T(text) }

// TK übersetzt text im Kontext kontext mit dem aktiven Katalog.
func TK(kontext, text string) string { return aktiv.Load().TK(kontext, text) }

// N wählt die Mehrzahlform für n mit dem aktiven Katalog.
func N(einzahl, mehrzahl string, n int) string { return aktiv.Load().N(einzahl, mehrzahl, n) }
