// SPDX-License-Identifier: GPL-3.0-or-later

package selection

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestAbbildungPixelgenau(t *testing.T) {
	a := NeueAbbildung(image.Pt(1920, 1080), image.Pt(1920, 1080))
	if a.Zoom != 1 || a.Versatz != (Punkt{}) {
		t.Fatalf("gleich groß muss 1:1 sein: %+v", a)
	}
	if p := a.Pixel(Punkt{100.7, 50.2}, image.Pt(1920, 1080)); p != image.Pt(100, 50) {
		t.Errorf("Pixel unter Zeiger: %v", p)
	}
}

func TestAbbildungEingepasst(t *testing.T) {
	// Erster Frame in falscher Skala (gio-wayland.md): Fenster doppelt so groß wie das Bild.
	a := NeueAbbildung(image.Pt(3840, 1600), image.Pt(7680, 3200))
	if a.Zoom != 2 || a.Versatz != (Punkt{}) {
		t.Fatalf("doppelt: %+v", a)
	}
	// Seitenverhältnis anders: mittig mit Rand.
	a = NeueAbbildung(image.Pt(100, 100), image.Pt(300, 200))
	if a.Zoom != 2 || a.Versatz != (Punkt{50, 0}) {
		t.Fatalf("mittig: %+v", a)
	}
	b := a.ZuBild(a.ZuFenster(Punkt{10, 20}))
	if b != (Punkt{10, 20}) {
		t.Errorf("Hin und zurück: %v", b)
	}
	if r := a.RechteckImFenster(image.Rect(0, 0, 100, 100)); r != image.Rect(50, 0, 250, 200) {
		t.Errorf("RechteckImFenster: %v", r)
	}
	if p := a.Pixel(Punkt{0, 0}, image.Pt(100, 100)); p != image.Pt(0, 0) {
		t.Errorf("Rand links wird beschnitten: %v", p)
	}
	if p := a.Pixel(Punkt{299, 199}, image.Pt(100, 100)); p != image.Pt(99, 99) {
		t.Errorf("Rand rechts wird beschnitten: %v", p)
	}
}

func ziehe(z *Zustand, a, b Punkt) Ergebnis {
	z.Gedrueckt(a)
	z.Bewegt(Punkt{(a.X + b.X) / 2, (a.Y + b.Y) / 2})
	return z.Losgelassen(b)
}

func TestBereichAufziehen(t *testing.T) {
	z := Neu(ModusBereich, image.Pt(200, 100))
	z.SetzeAbbildung(image.Pt(200, 100))
	e := ziehe(z, Punkt{10.5, 20.5}, Punkt{50.5, 60.5})
	if !e.Fertig || e.Bereich != image.Rect(10, 20, 51, 61) {
		t.Fatalf("Bereich: %+v (Start- und Endpixel gehören dazu)", e)
	}
	// Rückwärts gezogen ergibt dasselbe.
	e = ziehe(z, Punkt{50.5, 60.5}, Punkt{10.5, 20.5})
	if e.Bereich != image.Rect(10, 20, 51, 61) {
		t.Errorf("rückwärts: %v", e.Bereich)
	}
	// Über den Rand hinaus wird beschnitten.
	e = ziehe(z, Punkt{150, 50}, Punkt{400, -30})
	if e.Bereich != image.Rect(150, 0, 200, 51) {
		t.Errorf("beschnitten: %v", e.Bereich)
	}
}

func TestKlickImBereichModusWaehltNichts(t *testing.T) {
	z := Neu(ModusBereich, image.Pt(200, 100))
	z.Gedrueckt(Punkt{10, 10})
	z.Bewegt(Punkt{12, 11}) // unter der Klick-Schwelle
	if e := z.Losgelassen(Punkt{12, 11}); e.Fertig {
		t.Fatalf("Klick darf im Bereich-Modus nichts wählen: %+v", e)
	}
	if _, ok := z.Rahmen(); ok {
		t.Error("kein Rahmen nach Klick")
	}
}

func TestMonitorModus(t *testing.T) {
	z := Neu(ModusMonitor, image.Pt(200, 100))
	z.Bewegt(Punkt{5, 5})
	if !z.Hervorgehoben() {
		t.Error("Monitor unter dem Zeiger muss hervorgehoben sein")
	}
	z.Verlassen()
	if z.Hervorgehoben() {
		t.Error("ohne Zeiger nicht hervorgehoben")
	}
	z.Gedrueckt(Punkt{5, 5})
	e := z.Losgelassen(Punkt{6, 6})
	if !e.Fertig || e.Bereich != image.Rect(0, 0, 200, 100) {
		t.Fatalf("Klick nimmt den ganzen Monitor: %+v", e)
	}
	// Ziehen wählt doch einen Bereich, während des Ziehens nicht hervorgehoben.
	z.Gedrueckt(Punkt{10, 10})
	z.Bewegt(Punkt{30, 30})
	if z.Hervorgehoben() {
		t.Error("beim Ziehen nicht hervorgehoben")
	}
	if r, ok := z.Rahmen(); !ok || r != image.Rect(10, 10, 31, 31) {
		t.Errorf("Rahmen beim Ziehen: %v %v", r, ok)
	}
	e = z.Losgelassen(Punkt{30, 30})
	if !e.Fertig || e.Bereich != image.Rect(10, 10, 31, 31) {
		t.Errorf("Ziehen im Monitor-Modus: %+v", e)
	}
}

func TestAbgebrochenUndLoslassenOhneDruecken(t *testing.T) {
	z := Neu(ModusMonitor, image.Pt(200, 100))
	if e := z.Losgelassen(Punkt{5, 5}); e.Fertig {
		t.Fatal("Loslassen ohne Drücken (Druck kam auf einem anderen Monitor) wählt nichts")
	}
	z.Gedrueckt(Punkt{10, 10})
	z.Bewegt(Punkt{50, 50})
	z.Abgebrochen()
	if e := z.Losgelassen(Punkt{50, 50}); e.Fertig {
		t.Fatal("nach Abbruch kein Ergebnis")
	}
}

func TestGanz(t *testing.T) {
	z := Neu(ModusBereich, image.Pt(64, 32))
	if e := z.Ganz(); !e.Fertig || e.Bereich != image.Rect(0, 0, 64, 32) {
		t.Errorf("Ganz: %+v", e)
	}
}

func TestLupe(t *testing.T) {
	z := Neu(ModusBereich, image.Pt(1000, 800))
	z.SetzeAbbildung(image.Pt(1000, 800))
	kante := LupePixel * LupeFaktor
	l := z.LupeAn(Punkt{100.5, 100.5}, image.Pt(1000, 800))
	if l.Mitte != image.Pt(100, 100) || l.Quelle != image.Rect(92, 92, 109, 109) {
		t.Errorf("Quelle: %+v", l)
	}
	if l.Ziel != image.Rect(124, 124, 124+kante, 124+kante) {
		t.Errorf("Ziel rechts unten: %v", l.Ziel)
	}
	// Am rechten unteren Rand weicht sie nach links oben aus.
	l = z.LupeAn(Punkt{990, 790}, image.Pt(1000, 800))
	if l.Ziel != image.Rect(990-24-kante, 790-24-kante, 990-24, 790-24) {
		t.Errorf("Ziel ausgewichen: %v", l.Ziel)
	}
	// Am Bildrand ragt die Quelle hinaus, die Mitte bleibt das Randpixel.
	l = z.LupeAn(Punkt{0, 0}, image.Pt(1000, 800))
	if l.Mitte != image.Pt(0, 0) || l.Quelle.Min != image.Pt(-8, -8) {
		t.Errorf("Rand: %+v", l)
	}
}

func TestMassText(t *testing.T) {
	if s := MassText(image.Rect(10, 20, 650, 500)); s != "640 × 480" {
		t.Errorf("MassText: %q", s)
	}
	if s := PositionText(image.Pt(3, 4)); s != "3, 4" {
		t.Errorf("PositionText: %q", s)
	}
}

func TestMassPlatz(t *testing.T) {
	f := image.Pt(1000, 800)
	s := image.Pt(100, 24)
	if p := MassPlatz(image.Rect(100, 100, 300, 300), s, f); p != image.Pt(100, 308) {
		t.Errorf("unter dem Rahmen: %v", p)
	}
	if p := MassPlatz(image.Rect(100, 600, 300, 790), s, f); p != image.Pt(100, 600-8-24) {
		t.Errorf("über dem Rahmen: %v", p)
	}
	if p := MassPlatz(image.Rect(950, 0, 1000, 800), s, f); p != image.Pt(900, 8) {
		t.Errorf("innen, im Fenster: %v", p)
	}
}

func umgebung(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestLetzterPfad(t *testing.T) {
	if p := LetzterPfad(umgebung(map[string]string{"HOME": "/h", "XDG_STATE_HOME": "/s"})); p != "/s/schnapp/last-area.json" {
		t.Errorf("XDG_STATE_HOME: %q", p)
	}
	if p := LetzterPfad(umgebung(map[string]string{"HOME": "/h", "XDG_STATE_HOME": "rel"})); p != "/h/.local/state/schnapp/last-area.json" {
		t.Errorf("relatives XDG_STATE_HOME gilt nicht: %q", p)
	}
	if p := LetzterPfad(umgebung(map[string]string{})); p != "" {
		t.Errorf("ohne HOME: %q", p)
	}
}

func TestLetzterSpeichernLaden(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "state", "schnapp", "last-area.json")
	if _, err := LadeLetzten(pfad); !errors.Is(err, ErrKeinLetzter) {
		t.Fatalf("fehlende Datei: %v", err)
	}
	l := Letzter{Monitor: "Ausgabe-1", Groesse: image.Pt(1920, 1080), Skala: 1, Bereich: image.Rect(10, 20, 110, 220)}
	if err := SpeichereLetzten(pfad, l); err != nil {
		t.Fatal(err)
	}
	g, err := LadeLetzten(pfad)
	if err != nil || g != l {
		t.Fatalf("geladen %+v, %v", g, err)
	}
	for p, soll := range map[string]os.FileMode{pfad: 0o600, filepath.Dir(pfad): 0o700} {
		if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != soll {
			t.Errorf("Rechte %s: %v (soll %v)", filepath.Base(p), fi.Mode().Perm(), soll)
		}
	}
	// Ungültig: Bereich ragt aus dem Bild — wird weder geschrieben noch geladen.
	if err := SpeichereLetzten(pfad, Letzter{Monitor: "x", Groesse: image.Pt(10, 10), Bereich: image.Rect(0, 0, 20, 5)}); err == nil {
		t.Error("ungültiger Bereich wurde gespeichert")
	}
	if err := os.WriteFile(pfad, []byte(`{"monitor":"x","size":{"X":10,"Y":10},"area":{"Min":{"X":0,"Y":0},"Max":{"X":20,"Y":5}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LadeLetzten(pfad); !errors.Is(err, ErrKeinLetzter) {
		t.Errorf("ungültige Datei: %v", err)
	}
	if err := os.WriteFile(pfad, []byte("kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LadeLetzten(pfad); !errors.Is(err, ErrKeinLetzter) {
		t.Errorf("kaputte Datei: %v", err)
	}
}

// F-83: Wer zieht und zum Startpunkt zurückkehrt, klickt — kein 1×1-Bereich in der Bibliothek.
func TestZurueckZumStartIstKlick(t *testing.T) {
	z := Neu(ModusBereich, image.Pt(200, 100))
	z.Gedrueckt(Punkt{50, 50})
	z.Bewegt(Punkt{80, 70}) // über der Schwelle
	if e := z.Losgelassen(Punkt{51, 51}); e.Fertig {
		t.Fatalf("Bereich-Modus: zurück zum Start darf nichts wählen: %+v", e)
	}
	z = Neu(ModusMonitor, image.Pt(200, 100))
	z.Gedrueckt(Punkt{50, 50})
	z.Bewegt(Punkt{80, 70})
	if e := z.Losgelassen(Punkt{50, 50}); !e.Fertig || e.Bereich != image.Rect(0, 0, 200, 100) {
		t.Fatalf("Monitor-Modus: zurück zum Start ist ein Klick auf den Monitor: %+v", e)
	}
	// Knapp über der Schwelle ist ein Bereich.
	z = Neu(ModusBereich, image.Pt(200, 100))
	if e := ziehe(z, Punkt{50, 50}, Punkt{54, 50}); !e.Fertig || e.Bereich != image.Rect(50, 50, 55, 51) {
		t.Errorf("4 px gezogen: %+v", e)
	}
}

// F-84: Fenster größer als das Bild (eingepasst, mittig): Rahmen, Pixel und Lupe rechnen in Bildpixeln.
func TestEingepasstRahmenUndLupe(t *testing.T) {
	z := Neu(ModusBereich, image.Pt(100, 100))
	z.SetzeAbbildung(image.Pt(300, 200)) // Zoom 2, Versatz 50,0
	z.Gedrueckt(Punkt{60, 20})           // Bild 5,10
	z.Bewegt(Punkt{120, 60})             // Bild 35,30
	if r, ok := z.Rahmen(); !ok || r != image.Rect(5, 10, 36, 31) {
		t.Errorf("Rahmen eingepasst: %v %v", r, ok)
	}
	// Druck im Rand links neben dem Bild wird aufs Bild beschnitten.
	z = Neu(ModusBereich, image.Pt(100, 100))
	z.SetzeAbbildung(image.Pt(300, 200))
	if e := ziehe(z, Punkt{10, 10}, Punkt{70, 30}); !e.Fertig || e.Bereich != image.Rect(0, 5, 11, 16) {
		t.Errorf("Rand beschnitten: %+v", e)
	}
	l := z.LupeAn(Punkt{150, 100}, image.Pt(300, 200)) // Bild 50,50
	if l.Mitte != image.Pt(50, 50) || l.Quelle != image.Rect(42, 42, 59, 59) {
		t.Errorf("Lupe eingepasst: %+v", l)
	}
}

// Verlassen während des Ziehens beendet das Ziehen nicht (der Zeiger kann über den Rand hinaus).
func TestVerlassenWaehrendZiehen(t *testing.T) {
	z := Neu(ModusBereich, image.Pt(200, 100))
	z.Gedrueckt(Punkt{10, 10})
	z.Bewegt(Punkt{40, 40})
	z.Verlassen()
	if e := z.Losgelassen(Punkt{60, 50}); !e.Fertig || e.Bereich != image.Rect(10, 10, 61, 51) {
		t.Errorf("Loslassen nach Verlassen: %+v", e)
	}
}

func TestLetzterRechteUndOrt(t *testing.T) {
	if err := SpeichereLetzten("", Letzter{Monitor: "x", Groesse: image.Pt(2, 2), Skala: 1, Bereich: image.Rect(0, 0, 1, 1)}); !errors.Is(err, ErrKeinZustandsordner) {
		t.Errorf("ohne Pfad: %v", err)
	}
	if _, err := LadeLetzten(""); !errors.Is(err, ErrKeinZustandsordner) {
		t.Errorf("Laden ohne Pfad (F-100): %v", err)
	}
	ordner := filepath.Join(t.TempDir(), "schnapp")
	if err := os.Mkdir(ordner, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SpeichereLetzten(filepath.Join(ordner, "last-area.json"), Letzter{Monitor: "x", Groesse: image.Pt(2, 2), Skala: 1, Bereich: image.Rect(0, 0, 1, 1)}); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(ordner); err != nil || fi.Mode().Perm() != 0o700 {
		t.Errorf("vorhandener Ordner 0755 wird auf 0700 gebracht: %v", fi.Mode().Perm())
	}
}
