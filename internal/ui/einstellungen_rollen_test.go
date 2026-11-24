// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"

	"gioui.org/io/key"
)

// einstellungenLauf öffnet die Tafel bei der Fenstergröße f.
func einstellungenLauf(t *testing.T, f image.Point) *menueLauf {
	t.Helper()
	l := neuerMenueLauf(t, f)
	l.e.einstellungenOeffnen(l.gtx())
	l.setle()
	if !l.e.einst.offen {
		t.Fatalf("%v: Tafel nicht offen", f)
	}
	return l
}

// inhaltsRand ist die Fläche der Tafel zwischen Kopf und Fuß in Fensterpixeln.
func inhaltsRand(l *menueLauf) image.Rectangle {
	tf := l.e.einst.rahmen.tafel
	return image.Rect(tf.Min.X, tf.Min.Y+tafelKopfDp, tf.Max.X, tf.Max.Y-tafelFussDp)
}

// Alle Felder jedes Reiters sind bis hinunter zur Mindestgröße des Fensters erreichbar: Per Tab bekommt jedes Feld den
// Fokus und liegt dann ganz in der Fläche über dem Fuß (die Tafel rollt zum Fokus, F-709).
func TestEinstellungenFelderPerTabErreichbar(t *testing.T) {
	for _, f := range []image.Point{{1200, 800}, {800, 600}, {640, 480}, {480, 320}} {
		l := einstellungenLauf(t, f)
		for r := range tafelReiter(reiterAnzahl) {
			l.e.einst.waehleReiter(r)
			l.setle()
			for _, z := range zeilenDes(r) {
				for range len(l.e.einst.ziele()) { // höchstens eine Runde
					if l.e.einst.ziel() == z {
						break
					}
					l.taste(key.NameTab, 0)
				}
				if l.e.einst.ziel() != z {
					t.Fatalf("Fenster %v Reiter %d: Feld %d bekommt per Tab keinen Fokus", f, r, z)
				}
				fr, flaeche := l.e.einst.rahmen.feld[z], inhaltsRand(l)
				if fr.Empty() || !fr.In(flaeche) {
					t.Errorf("Fenster %v Reiter %d: Feld %s.%s mit Fokus liegt bei %v, sichtbar ist %v (rollen %d von %d)",
						f, r, tafelZeilen[z].abschnitt, tafelZeilen[z].schluessel, fr, flaeche, l.e.einst.roll.rollen, l.e.einst.roll.rollMax)
				}
			}
		}
	}
}

// Der Fuß (Speichern, Abbrechen) bleibt beim Rollen an seinem Platz; Mausrad und Bild auf/ab rollen den Inhalt, ein
// Reiterwechsel beginnt oben.
func TestEinstellungenRollenHaeltDenFuss(t *testing.T) {
	l := einstellungenLauf(t, image.Pt(640, 480))
	t0 := &l.e.einst
	k := &t0.roll
	// Reiter mit den meisten Zeilen: dort rollt der Inhalt bei 640×480 sicher.
	gross := reiterAllgemein
	for r := range tafelReiter(reiterAnzahl) {
		if len(zeilenDes(r)) > len(zeilenDes(gross)) {
			gross = r
		}
	}
	t0.waehleReiter(gross)
	l.setle()
	if k.rollMax <= 0 {
		t.Fatalf("Vorbedingung: Reiter %d rollt bei 640×480 nicht (rollMax %d)", gross, k.rollMax)
	}
	ok, ab := t0.rahmen.ok, t0.rahmen.abbrechen
	flaeche := inhaltsRand(l)
	l.radSchritte(mitte(flaeche), 3, 15)
	if k.rollen <= 0 {
		t.Errorf("Mausrad: rollen %d, Soll > 0", k.rollen)
	}
	l.taste(key.NamePageUp, 0)
	l.taste(key.NamePageUp, 0)
	if k.rollen != 0 {
		t.Errorf("Bild auf bis zum Anfang: rollen %d, Soll 0", k.rollen)
	}
	vorher := k.rollen
	l.taste(key.NamePageDown, 0)
	if k.rollen <= vorher {
		t.Errorf("Bild ab: rollen %d, vorher %d", k.rollen, vorher)
	}
	l.taste(key.NamePageDown, 0)
	l.taste(key.NamePageDown, 0)
	l.taste(key.NamePageDown, 0)
	l.taste(key.NamePageDown, 0)
	if k.rollen != k.rollMax {
		t.Errorf("Bild ab bis zum Anschlag: rollen %d, Soll %d", k.rollen, k.rollMax)
	}
	l.taste(key.NamePageUp, 0)
	if k.rollen >= k.rollMax {
		t.Errorf("Bild auf: rollen %d, hat sich nicht bewegt", k.rollen)
	}
	if t0.rahmen.ok != ok || t0.rahmen.abbrechen != ab {
		t.Errorf("Der Fuß wandert beim Rollen: Save %v→%v, Cancel %v→%v", ok, t0.rahmen.ok, ab, t0.rahmen.abbrechen)
	}
	if !t0.offen {
		t.Error("Bild auf/ab hat die Tafel geschlossen")
	}
	l.taste(key.NamePageDown, 0)
	if k.rollen == 0 {
		t.Fatal("Vorbedingung: Bild ab rollt nicht")
	}
	t0.waehleReiter(gross)
	l.setle()
	if k.rollen != 0 {
		t.Errorf("Reiterwechsel: rollen %d, Soll 0", k.rollen)
	}
}

// Ein Klick auf ein gerolltes Feld trifft dieses Feld (die Zeigerlage folgt dem Versatz).
func TestEinstellungenKlickAufGerolltesFeld(t *testing.T) {
	l := einstellungenLauf(t, image.Pt(640, 480))
	t0 := &l.e.einst
	gross := reiterAllgemein
	for r := range tafelReiter(reiterAnzahl) {
		if len(zeilenDes(r)) > len(zeilenDes(gross)) {
			gross = r
		}
	}
	t0.waehleReiter(gross)
	l.setle()
	letzte := zeilenDes(gross)[len(zeilenDes(gross))-1]
	l.taste(key.NameEnd, 0) // Ende gehört der Tafel nicht: bleibt ohne Wirkung
	for range 8 {
		l.taste(key.NamePageDown, 0)
	}
	fr := t0.rahmen.feld[letzte]
	if !fr.In(inhaltsRand(l)) {
		t.Fatalf("Vorbedingung: letztes Feld %v nicht sichtbar in %v nach dem Rollen ans Ende", fr, inhaltsRand(l))
	}
	l.klick(mitte(fr))
	if got := t0.ziel(); got != letzte {
		t.Errorf("Klick auf das gerollte Feld %v: Fokus auf %d, Soll %d", fr, got, letzte)
	}
}
