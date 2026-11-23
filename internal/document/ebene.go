// SPDX-License-Identifier: GPL-3.0-or-later

package document

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// Ebenen (ADR-0032): Jedes Objekt ist eine Ebene über dem Bild, die Reihenfolge der Objekte ist die der Ebenen (unten
// zuerst). Die Angaben einer Ebene stehen im Objekt selbst (eingebettet), damit Abzug, Rettungskopie und Format ohne
// zweite Liste auskommen.

// MaxEbenenName begrenzt den Namen einer Ebene in Zeichen (Runen).
const MaxEbenenName = 100

// Ebene sind die Angaben einer Ebene. Der Nullwert ist eine sichtbare, offene, deckende Ebene ohne eigenen Namen —
// so bleibt jedes Objekt, das ohne diese Angaben angelegt wird, wie bisher.
type Ebene struct {
	// Name ist der eigene Name; leer heißt, die Oberfläche zeigt einen Namen aus Art und Inhalt.
	Name string
	// Ausgeblendet: weder Editor noch Renderer zeichnen das Objekt, Export und Zwischenablage lassen es weg; die
	// Arbeitsdatei behält es (ADR-0032 Punkt 4).
	Ausgeblendet bool
	// Gesperrt: auf der Leinwand nicht zu treffen, ohne Griffe (Punkt 5).
	Gesperrt bool
	// Transparenz ist 100 − Deckkraft in Prozent (0 = deckend), damit der Nullwert deckend ist (Punkt 2).
	Transparenz uint8
}

// Ebenenwerte liefert die Angaben der Ebene (über die Einbettung eine Methode jedes Objekts).
func (e Ebene) Ebenenwerte() Ebene { return e }

// Deckkraft ist die Deckkraft der Ebene in Prozent (0 bis 100).
func (e Ebene) Deckkraft() int { return 100 - int(min(e.Transparenz, 100)) }

// MitDeckkraft liefert die Angaben mit der Deckkraft p (auf 0 bis 100 begrenzt).
func (e Ebene) MitDeckkraft(p int) Ebene {
	e.Transparenz = uint8(100 - max(0, min(100, p)))
	return e
}

// Bearbeitbar meldet, ob das Objekt auf der Leinwand zu greifen ist: sichtbar und nicht gesperrt.
func (e Ebene) Bearbeitbar() bool { return !e.Ausgeblendet && !e.Gesperrt }

// EbenenNameGueltig meldet, ob n als Name einer Ebene taugt: gültiges UTF-8, höchstens MaxEbenenName Zeichen, keine
// Steuerzeichen (ein Name ist eine Zeile).
func EbenenNameGueltig(n string) bool {
	if !utf8.ValidString(n) || utf8.RuneCountInString(n) > MaxEbenenName {
		return false
	}
	for _, r := range n {
		if unicode.IsControl(r) || r == 0xfffe || r == 0xffff {
			return false
		}
	}
	return true
}

func (e Ebene) pruefe() error {
	if !EbenenNameGueltig(e.Name) {
		return fmt.Errorf("document: Name der Ebene ungültig")
	}
	if e.Transparenz > 100 {
		return fmt.Errorf("document: Deckkraft der Ebene ungültig")
	}
	return nil
}

// MitEbene liefert o mit den Angaben e (gleiche Kennung). Für Ersetzen: Umbenennen, Ausblenden, Sperren, Deckkraft.
func MitEbene(o Objekt, e Ebene) Objekt {
	if o == nil {
		return nil
	}
	return o.mitEbene(e)
}

func (p Pfeil) mitEbene(e Ebene) Objekt  { p.Ebene = e; return p }
func (k Kasten) mitEbene(e Ebene) Objekt { k.Ebene = e; return k }
func (t Text) mitEbene(e Ebene) Objekt   { t.Ebene = e; return t }
func (n Nummer) mitEbene(e Ebene) Objekt { n.Ebene = e; return n }

// Stelle liefert die Lage des Objekts id in der Zeichenreihenfolge (0 = unterstes), -1, wenn es fehlt.
func (d *Dokument) Stelle(id int) int { return d.index(id) }

// umordnen bringt das Objekt von Stelle von an Stelle nach (Stellen der Liste nach dem Herausnehmen).
type umordnen struct{ von, nach int }

func (b umordnen) ausfuehren(d *Dokument)    { d.objekte = verschobenIn(d.objekte, b.von, b.nach) }
func (b umordnen) zuruecknehmen(d *Dokument) { d.objekte = verschobenIn(d.objekte, b.nach, b.von) }

// verschobenIn nimmt das Element bei von heraus und setzt es bei nach wieder ein, in einem neuen Feld.
func verschobenIn(l []Objekt, von, nach int) []Objekt {
	o := l[von]
	rest := append(append([]Objekt(nil), l[:von]...), l[von+1:]...)
	return append(rest[:nach], append([]Objekt{o}, rest[nach:]...)...)
}

// Umordnen bringt das Objekt id an Stelle nach (0 = unterstes, len-1 = oberstes; begrenzt) — ein Rückgängig-Schritt,
// ohne Wirkung, wenn es schon dort steht.
func (d *Dokument) Umordnen(id, nach int) error {
	if d.zug != nil {
		return ErrZugLaeuft
	}
	von := d.index(id)
	if von < 0 {
		return ErrUnbekannt
	}
	nach = max(0, min(nach, len(d.objekte)-1))
	if nach == von {
		return nil
	}
	d.fuehreAus(umordnen{von, nach})
	return nil
}

// FuegeEin legt o an Stelle stelle (begrenzt auf 0 bis Anzahl) ein und liefert seine neue Kennung (Doppeln: direkt
// über dem Original). Ein Rückgängig-Schritt.
func (d *Dokument) FuegeEin(o Objekt, stelle int) (int, error) {
	if o == nil {
		return 0, ErrUnbekannt
	}
	if d.zug != nil {
		return 0, ErrZugLaeuft
	}
	if len(d.objekte) >= MaxObjekte {
		return 0, fmt.Errorf("document: mehr als %d Objekte", MaxObjekte)
	}
	if err := o.pruefe(); err != nil {
		return 0, err
	}
	o = o.mitKennung(d.naechsteID)
	d.naechsteID++
	d.fuehreAus(einfuegen{o, max(0, min(stelle, len(d.objekte)))})
	return o.Kennung(), nil
}
