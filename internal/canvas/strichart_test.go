// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0021 Nachtrag 25 Punkte 7 und 8: Ohne Auswahl gilt die Strichart für folgende Pfeile, Kästen und Lupen; mit
// gewähltem Objekt ändert sie es als einen Rückgängig-Schritt und merkt sie fürs Werkzeug. Ein Schnellstil lässt sie
// stehen. Seit ADR-0041 merkt jedes Werkzeug seine eigene.
func TestStrichartWerkzeugUndAuswahl(t *testing.T) {
	s, _, _ := sitzung(t)
	if s.Strichart() != document.StrichVoll {
		t.Fatalf("Voreinstellung %v", s.Strichart())
	}
	vor := s.D.Stand()
	if s.SetzeStrichart(document.StrichGestrichelt) || s.D.Stand() != vor {
		t.Error("ohne Auswahl ein Rückgängig-Schritt")
	}
	if s.SetzeStrichart(document.Strichart(7)) || s.Stil.Strichart != document.StrichGestrichelt {
		t.Error("ungültige Strichart angenommen")
	}
	for _, w := range []Werkzeug{WerkzeugPfeil, WerkzeugKasten, WerkzeugLupe} {
		s.SetzeWerkzeug(w)
		s.SetzeStrichart(document.StrichGestrichelt) // seit ADR-0041 je Werkzeug
		if !s.Seitenleiste().Strich {
			t.Errorf("Werkzeug %v ohne Strich in der Leiste", w)
		}
		zieh(s, f(20, 20), f(120, 80))
		o := letztes(t, s)
		var art document.Strichart
		switch v := o.(type) {
		case document.Pfeil:
			art = v.Strich.Art
		case document.Kasten:
			art = v.Strich.Art
		case document.Lupe:
			art = v.Strich.Art
		default:
			t.Fatalf("Werkzeug %v legte %T an", w, o)
		}
		if art != document.StrichGestrichelt {
			t.Errorf("neues %T: Strichart %v", o, art)
		}
	}
	// Mit Auswahl: ein Schritt, rückgängig zurück; das Werkzeug merkt die Wahl.
	s.SetzeWerkzeug(WerkzeugAuswahl)
	k := letztes(t, s).(document.Lupe)
	s.Auswahl = k.ID
	vor = s.D.Stand()
	if !s.SetzeStrichart(document.StrichGepunktet) || s.D.Stand() == vor {
		t.Fatal("SetzeStrichart änderte die gewählte Lupe nicht")
	}
	if s.Strichart() != document.StrichGepunktet || s.Stil.Strichart != document.StrichGepunktet {
		t.Errorf("Leiste %v, Stil %v", s.Strichart(), s.Stil.Strichart)
	}
	if s.SetzeStrichart(document.StrichGepunktet) {
		t.Error("dieselbe Art ist ein Schritt")
	}
	s.Rueckgaengig()
	if s.Strichart() != document.StrichGestrichelt {
		t.Errorf("nach Rückgängig %v", s.Strichart())
	}
	// Ein Schnellstil ändert Farbe und Breite, nicht die Strichart.
	if !s.WendeSchnellstilAn(Schnellstil{Art: SchnellLupe, Farbe: Beschriftungsfarben[1], Breite: 12}) {
		t.Fatal("Schnellstil nicht angewandt")
	}
	if l := letztes(t, s).(document.Lupe); l.Strich.Art != document.StrichGestrichelt || l.Strich.Breite != 12 {
		t.Errorf("nach Schnellstil %+v", l.Strich)
	}
	// Ein Objekt ohne Strich ändert sich nicht.
	s.SetzeWerkzeug(WerkzeugNummer)
	s.Druecken(f(185, 90), TastePrimaer)
	s.Loslassen(f(185, 90))
	s.SetzeWerkzeug(WerkzeugAuswahl)
	n, ok := letztes(t, s).(document.Nummer)
	if !ok {
		t.Fatalf("keine Nummer: %T", letztes(t, s))
	}
	s.Auswahl = n.ID
	vor = s.D.Stand()
	if s.SetzeStrichart(document.StrichVoll) || s.D.Stand() != vor {
		t.Error("Nummer bekam eine Strichart")
	}
}
