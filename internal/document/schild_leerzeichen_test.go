// SPDX-License-Identifier: GPL-3.0-or-later

package document

import "testing"

// F-365: Eine Zeile nur aus Leerzeichen macht das Schild breiter (Vorschub), damit die Einfügemarke beim Tippen im
// Schild bleibt; Hintergrund, Rahmen und Treffer folgen, weil alle aus Text.Grenzen kommen.
func TestSchildZeileNurLeerzeichen(t *testing.T) {
	leer := Text{Ort: Pt(10, 10), Inhalt: "", Schriftgrad: 40}
	raum := Text{Ort: Pt(10, 10), Inhalt: "          ", Schriftgrad: 40}
	if raum.Grenzen().Breite() <= leer.Grenzen().Breite() {
		t.Errorf("Schild aus Leerzeichen %.1f breit, nicht breiter als das leere (%.1f)", raum.Grenzen().Breite(), leer.Grenzen().Breite())
	}
	// Gegenprobe: Mit Tinte davor zählen nachgestellte Leerzeichen (Vorschub der Zeile).
	a := Text{Ort: Pt(10, 10), Inhalt: "a", Schriftgrad: 40}
	ar := Text{Ort: Pt(10, 10), Inhalt: "a          ", Schriftgrad: 40}
	if ar.Grenzen().Breite() <= a.Grenzen().Breite() {
		t.Errorf("Schild mit Nachlauf %.1f breit, nicht breiter als ohne (%.1f)", ar.Grenzen().Breite(), a.Grenzen().Breite())
	}
}

// Eine mehrzeilige Eingabe, deren breiteste Zeile nur aus Leerzeichen besteht, wird so breit wie diese Zeile.
func TestSchildBreitesteZeileNurLeerzeichen(t *testing.T) {
	raum := Text{Ort: Pt(10, 10), Inhalt: "          ", Schriftgrad: 40}
	mehr := Text{Ort: Pt(10, 10), Inhalt: "a\n          \nb", Schriftgrad: 40}
	if mehr.Grenzen().Breite() < raum.Grenzen().Breite() {
		t.Errorf("mehrzeilig %.1f breit, schmaler als die Leerzeichenzeile allein (%.1f)", mehr.Grenzen().Breite(), raum.Grenzen().Breite())
	}
	ohne := Text{Ort: Pt(10, 10), Inhalt: "a\n\nb", Schriftgrad: 40}
	if mehr.Grenzen().Breite() <= ohne.Grenzen().Breite() {
		t.Errorf("mehrzeilig %.1f breit, nicht breiter als mit leerer Zeile (%.1f)", mehr.Grenzen().Breite(), ohne.Grenzen().Breite())
	}
	// Die Höhe ändert sich nicht: Die Zeilenhöhe zählte schon vorher für jede Zeile.
	if mehr.Grenzen().Hoehe() != ohne.Grenzen().Hoehe() {
		t.Errorf("Höhe %.1f gegen %.1f", mehr.Grenzen().Hoehe(), ohne.Grenzen().Hoehe())
	}
}
