// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

import (
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

func bild(w, h int) *image.NRGBA {
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := range b.Pix {
		b.Pix[i] = uint8(i*7 + 3)
	}
	return b
}

// ausgang legt eine Ausgangsdatei an und liefert Pfad und Kennung.
func ausgang(t *testing.T, inhalt string) (string, Kennung) {
	t.Helper()
	pfad := filepath.Join(t.TempDir(), "aufnahme.png")
	if err := os.WriteFile(pfad, []byte(inhalt), 0o600); err != nil {
		t.Fatal(err)
	}
	k, err := KennungVon(pfad)
	if err != nil {
		t.Fatal(err)
	}
	return pfad, k
}

// bearbeite ändert d: Pfeil, Text, Schwärzung, Verpixelung, Zuschnitt, Streifen.
func bearbeite(t *testing.T, d *document.Dokument) {
	t.Helper()
	schwarz := document.Schwaerzung{Farbe: color.NRGBA{A: 255}}
	schritte := []func() error{
		func() error {
			_, err := d.Hinzufuegen(document.GeraderPfeil(document.Pt(2, 2), document.Pt(30, 20), document.Strich{Breite: 3, Farbe: color.NRGBA{R: 255, A: 255}}))
			return err
		},
		func() error { return d.Schwaerzen(image.Rect(5, 5, 20, 12), schwarz) },
		func() error {
			_, err := d.Hinzufuegen(document.Text{Ort: document.Pt(10, 40), Inhalt: "Hallo", Schriftgrad: 14})
			return err
		},
		func() error { return d.Schwaerzen(image.Rect(30, 30, 50, 50), document.Schwaerzung{Block: 8}) },
		func() error { return d.SetzeZuschnitt(image.Rect(1, 1, 60, 55)) },
		func() error { return d.SetzeStreifen([]document.Streifen{{Zeilen: true, Von: 20, Bis: 25}}) },
	}
	for i, s := range schritte {
		if err := s(); err != nil {
			t.Fatalf("Schritt %d: %v", i, err)
		}
	}
}

// ADR-0029: Kopie bilden, schreiben, lesen und auf das Ausgangsbild anwenden ergibt dieselben Angaben und dieselben
// Pixel (Schwärzen ohne Mosaik ist deterministisch); Rückgängig führt auf den Ausgangsstand zurück.
func TestRundlauf(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := filepath.Join(t.TempDir(), "state", "schnapp", "rescue")
	original := bild(64, 64)
	d, err := document.Neu(original)
	if err != nil {
		t.Fatal(err)
	}
	bearbeite(t, d)
	kp, err := Erzeuge(d, k, time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(kp.Schwaerzungen) != 2 {
		t.Fatalf("%d Schwärzungen, erwartet 2", len(kp.Schwaerzungen))
	}
	if err := Schreibe(ort, kp); err != nil {
		t.Fatal(err)
	}
	gelesen, err := Lies(ort, pfad)
	if err != nil {
		t.Fatal(err)
	}
	neu, err := document.Neu(original)
	if err != nil {
		t.Fatal(err)
	}
	basis := neu.Stand()
	if err := Anwenden(neu, gelesen, nil); err != nil {
		t.Fatal(err)
	}
	if neu.Zuschnitt() != d.Zuschnitt() || len(neu.Streifen()) != 1 || len(neu.Objekte()) != 2 {
		t.Errorf("Angaben weichen ab: %v %v %d", neu.Zuschnitt(), neu.Streifen(), len(neu.Objekte()))
	}
	if string(neu.OriginalBild().Pix) != string(d.OriginalBild().Pix) {
		t.Error("Pixel nach dem Wiederholen der Schwärzungen weichen ab")
	}
	for neu.KannRueckgaengig() {
		if err := neu.Rueckgaengig(); err != nil {
			t.Fatal(err)
		}
	}
	if neu.Stand() != basis || string(neu.OriginalBild().Pix) != string(original.Pix) || len(neu.Objekte()) != 0 {
		t.Error("Rückgängig führt nicht auf den Ausgangsstand zurück")
	}
}

// Die Kopie enthält keine Pixel und keinen Pfad im Dateinamen; Ordner 0700, Datei 0600 (ADR-0029 Punkt 3).
func TestOrtRechteUndInhalt(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := filepath.Join(t.TempDir(), "rescue")
	d, err := document.Neu(bild(32, 32))
	if err != nil {
		t.Fatal(err)
	}
	bearbeite32 := func() {
		if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(3, 20), Inhalt: "x", Schriftgrad: 10}); err != nil {
			t.Fatal(err)
		}
	}
	bearbeite32()
	kp, err := Erzeuge(d, k, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := Schreibe(ort, kp); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(ort)
	if err != nil || st.Mode().Perm() != 0o700 {
		t.Errorf("Ordner %v, %v", st.Mode().Perm(), err)
	}
	f := filepath.Join(ort, Name(pfad))
	st, err = os.Stat(f)
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("Datei %v, %v", st, err)
	}
	if strings.Contains(Name(pfad), "aufnahme") {
		t.Error("Pfad im Dateinamen")
	}
	daten, _ := os.ReadFile(f)
	if strings.Contains(string(daten), "data:") || strings.Contains(string(daten), "base64") || len(daten) > 8<<10 {
		t.Errorf("Kopie trägt mehr als Angaben (%d Bytes)", len(daten))
	}
	eintraege, _ := os.ReadDir(ort)
	if len(eintraege) != 1 {
		t.Errorf("%d Einträge im Ordner, erwartet 1 (keine temporären Reste)", len(eintraege))
	}
}

func TestOrt(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if o, err := Ort(env(map[string]string{"XDG_STATE_HOME": "/s", "HOME": "/h"})); err != nil || o != filepath.Join("/s", "schnapp", "rescue") {
		t.Errorf("XDG: %q %v", o, err)
	}
	if o, err := Ort(env(map[string]string{"XDG_STATE_HOME": "relativ", "HOME": "/h"})); err != nil || o != filepath.Join("/h", ".local", "state", "schnapp", "rescue") {
		t.Errorf("HOME: %q %v", o, err)
	}
	if _, err := Ort(env(nil)); !errors.Is(err, ErrKeinOrt) {
		t.Errorf("ohne beides: %v", err)
	}
}

// Kennung: jede der vier Angaben zählt (ADR-0029 Punkt 2).
func TestKennungGleich(t *testing.T) {
	_, k := ausgang(t, "datei")
	if !k.Gleich(k) {
		t.Fatal("ungleich mit sich selbst")
	}
	for name, a := range map[string]func(*Kennung){
		"Pfad":    func(x *Kennung) { x.Pfad += "x" },
		"Größe":   func(x *Kennung) { x.Groesse++ },
		"Zeit":    func(x *Kennung) { x.Zeit = x.Zeit.Add(time.Nanosecond) },
		"SHA-256": func(x *Kennung) { x.SHA256 = strings.Repeat("0", 64) },
	} {
		o := k
		a(&o)
		if k.Gleich(o) {
			t.Errorf("%s geändert, trotzdem gleich", name)
		}
	}
}

// Wiederherstellen: passende Kopie wird angewendet; nach einer Änderung der Datei nicht, die Kopie bleibt liegen.
func TestWiederherstellen(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	s := NeueSicherung(ort, k)
	s.Sichere(d)
	s.Ende()

	neu, _ := document.Neu(bild(64, 64))
	w := NeueSicherung(ort, k).Wiederherstellen(neu, nil)
	if !w.Angewendet || w.Gesichert.IsZero() || len(neu.Objekte()) != 2 {
		t.Fatalf("nicht angewendet: %+v", w)
	}

	if err := os.WriteFile(pfad, []byte("anders"), 0o600); err != nil {
		t.Fatal(err)
	}
	k2, _ := KennungVon(pfad)
	neu2, _ := document.Neu(bild(64, 64))
	s2 := NeueSicherung(ort, k2)
	w = s2.Wiederherstellen(neu2, nil)
	if w.Angewendet || !w.Liegt || !errors.Is(w.Grund, ErrAnders) || len(neu2.Objekte()) != 0 {
		t.Errorf("trotz geänderter Datei: %+v", w)
	}
	// V7-1: Die nicht angewendete Kopie liegt beiseite; weder Sichere noch Verwirf dieser Sitzung berühren sie.
	if w.Beiseite == "" || !eigenerName(w.Beiseite) {
		t.Fatalf("nicht beiseitegelegt: %q", w.Beiseite)
	}
	bearbeite(t, neu2)
	s2.Sichere(neu2)
	s2.Ende()
	s2.Verwirf()
	if _, err := os.Stat(filepath.Join(ort, w.Beiseite)); err != nil {
		t.Errorf("beiseitegelegte Kopie weg: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ort, Name(pfad))); !os.IsNotExist(err) {
		t.Errorf("eigene Kopie der Sitzung nach Verwirf noch da: %v", err)
	}
}

// V7-1: Eigene Namen beim Aufräumen, auch für beiseitegelegte Kopien.
func TestEigenerName(t *testing.T) {
	h := strings.Repeat("a1", 32)
	for n, soll := range map[string]bool{
		h + ".json": true, h + ".123.alt.json": true, ".schreibt-9": true,
		h + ".alt.json": false, h + ".12x.alt.json": false, "fremd.json": false, strings.Repeat("G", 64) + ".json": false,
	} {
		if eigenerName(n) != soll {
			t.Errorf("%q: %v", n, !soll)
		}
	}
}

// Eine Kopie, die nicht passt (andere Bildgröße) oder die Prüfung nicht besteht, ändert das Dokument nicht.
func TestAnwendenLaesstDokumentBeiFehler(t *testing.T) {
	_, k := ausgang(t, "datei")
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	kp, _ := Erzeuge(d, k, time.Now())

	klein, _ := document.Neu(bild(32, 32))
	if err := Anwenden(klein, &kp, nil); !errors.Is(err, ErrGroesse) || klein.KannRueckgaengig() {
		t.Errorf("andere Größe: %v", err)
	}
	gleich, _ := document.Neu(bild(64, 64))
	vorher := string(gleich.OriginalBild().Pix)
	nein := errors.New("zu teuer")
	if err := Anwenden(gleich, &kp, func(*document.Dokument) error { return nein }); !errors.Is(err, nein) {
		t.Errorf("Prüfung: %v", err)
	}
	if gleich.KannRueckgaengig() || string(gleich.OriginalBild().Pix) != vorher {
		t.Error("Dokument trotz abgelehnter Prüfung geändert")
	}
	kp.Schwaerzungen[0].Farbe = "schwarz"
	if err := Anwenden(gleich, &kp, nil); !errors.Is(err, ErrFormat) || gleich.KannRueckgaengig() {
		t.Errorf("kaputte Farbe: %v", err)
	}
}

// Lies: fehlend, fremde oder neuere Fassung, kaputtes JSON, unbekanntes Feld, falscher Pfad.
func TestLiesGrenzen(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	if _, err := Lies(ort, pfad); !errors.Is(err, ErrKeine) {
		t.Errorf("fehlend: %v", err)
	}
	schreib := func(s string) {
		if err := os.WriteFile(filepath.Join(ort, Name(pfad)), []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	faelle := map[string]struct {
		json string
		soll error
	}{
		"neuer":     {`{"format":2,"datei":{"pfad":"` + pfad + `"}}`, ErrNeuer},
		"kaputt":    {`{"format":1,`, ErrFormat},
		"unbekannt": {`{"format":1,"datei":{"pfad":"` + pfad + `"},"pixel":"x"}`, ErrFormat},
		"Pfad":      {`{"format":1,"datei":{"pfad":"/anders"}}`, ErrFormat},
		"Fassung 0": {`{"datei":{"pfad":"` + pfad + `"}}`, ErrFormat},
	}
	for name, f := range faelle {
		schreib(f.json)
		if _, err := Lies(ort, pfad); !errors.Is(err, f.soll) || !strings.HasPrefix(err.Error(), "rescue: ") {
			t.Errorf("%s: %v, erwartet %v mit Präfix", name, err, f.soll)
		}
	}
	_ = k
}

// Sicherung: Verwirf löscht; NeueBasis löscht die Kopie der alten Datei; eine nach Verwirf fertige ältere Kopie
// schreibt nicht mehr.
func TestSicherungVerwirfUndNeueBasis(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	d, _ := document.Neu(bild(64, 64))
	bearbeite(t, d)
	s := NeueSicherung(ort, k)
	s.Sichere(d)
	s.Ende()
	if _, err := os.Stat(filepath.Join(ort, Name(pfad))); err != nil {
		t.Fatalf("nicht geschrieben: %v", err)
	}
	s.Verwirf()
	if _, err := os.Stat(filepath.Join(ort, Name(pfad))); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("nach Verwirf noch da: %v", err)
	}
	s.Sichere(d)
	s.Ende()
	pfad2, k2 := ausgang(t, "neu")
	s.NeueBasis(k2)
	if _, err := os.Stat(filepath.Join(ort, Name(pfad))); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Kopie der alten Basis noch da: %v", err)
	}
	s.Sichere(d)
	s.Ende()
	kp, err := Lies(ort, pfad2)
	if err != nil || !kp.Datei.Gleich(k2) {
		t.Errorf("neue Basis: %v %+v", err, kp)
	}
}

// Viele Sicherungen hintereinander: Am Ende steht die jüngste (keine ältere überschreibt sie).
func TestSicherungReihenfolge(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	d, _ := document.Neu(bild(64, 64))
	s := NeueSicherung(ort, k)
	for i := 0; i < 30; i++ {
		if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(5, 20), Inhalt: "t", Schriftgrad: 10}); err != nil {
			t.Fatal(err)
		}
		s.Sichere(d)
	}
	s.Ende()
	kp, err := Lies(ort, pfad)
	if err != nil {
		t.Fatal(err)
	}
	neu, _ := document.Neu(bild(64, 64))
	if err := Anwenden(neu, kp, nil); err != nil || len(neu.Objekte()) != 30 {
		t.Errorf("jüngste Kopie nicht zuletzt: %d Objekte, %v", len(neu.Objekte()), err)
	}
}

// Aufräumen: nur eigene Namen, nur älter als die Haltezeit.
func TestAufraeumen(t *testing.T) {
	ort := t.TempDir()
	jetzt := time.Now()
	alt := jetzt.Add(-Haltezeit - time.Hour)
	dateien := map[string]time.Time{
		Name("/a"):                       alt,
		Name("/b"):                       jetzt,
		".schreibt-123":                  alt,
		"fremd.json":                     alt,
		strings.Repeat("z", 64) + ".txt": alt,
	}
	for n, z := range dateien {
		p := filepath.Join(ort, n)
		if err := os.WriteFile(p, []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, z, z); err != nil {
			t.Fatal(err)
		}
	}
	n, err := Aufraeumen(ort, jetzt)
	if err != nil || n != 2 {
		t.Errorf("%d entfernt, %v; erwartet 2", n, err)
	}
	for _, bleibt := range []string{Name("/b"), "fremd.json", strings.Repeat("z", 64) + ".txt"} {
		if _, err := os.Stat(filepath.Join(ort, bleibt)); err != nil {
			t.Errorf("%s fehlt: %v", bleibt, err)
		}
	}
	if n, err := Aufraeumen(filepath.Join(ort, "fehlt"), jetzt); n != 0 || err != nil {
		t.Errorf("fehlender Ordner: %d %v", n, err)
	}
}
