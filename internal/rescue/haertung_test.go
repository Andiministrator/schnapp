// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

// Tests aus Verify 7 (Fable, V7-8, F-385): Sie tragen die Aussagen, die die Mutationsprobe scripts/mutanten/rescue.json
// prüft (Aufräumen, Grenzen beim Lesen, Probelauf am Abzug, Verwerfen gegen laufende Schreibaufträge).

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// R7: Aufräumen fasst weder Symlinks noch Ordner mit eigenem Namen noch fremde Namen an; das Ziel eines Symlinks bleibt.
func TestAufraeumenSymlinkUndOrdner(t *testing.T) {
	ort := t.TempDir()
	ziel := filepath.Join(t.TempDir(), "wichtig.txt")
	if err := os.WriteFile(ziel, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	altZeit := time.Now().Add(-2 * Haltezeit)
	if err := os.Chtimes(ziel, altZeit, altZeit); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(ort, Name("/irgendwo"))
	if err := os.Symlink(ziel, link); err != nil {
		t.Skip(err)
	}
	fremd := filepath.Join(ort, "notizen.json")
	unter := filepath.Join(ort, fmt.Sprintf("%064d.json", 0))
	if err := os.WriteFile(fremd, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(unter, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{fremd, unter} {
		if err := os.Chtimes(p, altZeit, altZeit); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := Aufraeumen(ort, time.Now()); n != 0 || err != nil {
		t.Errorf("entfernt %d, %v; erwartet 0", n, err)
	}
	for _, p := range []string{ziel, link, fremd, unter} {
		if _, err := os.Lstat(p); err != nil {
			t.Errorf("%s fehlt: %v", filepath.Base(p), err)
		}
	}
}

// R13, R15: Eine zu große Datei und zu viele Schwärzungen werden nicht gelesen.
func TestLiesZuGross(t *testing.T) {
	pfad, _ := ausgang(t, "datei")
	ort := t.TempDir()
	datei := filepath.Join(ort, Name(pfad))
	f, err := os.Create(datei)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxDatei + 1); err != nil { // dünn besetzt, belegt keinen Platz
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Lies(ort, pfad); !errors.Is(err, ErrZuGross) {
		t.Errorf("Datei über %d Bytes: %v, erwartet ErrZuGross", maxDatei, err)
	}

	eine := `{"bereich":[0,0,2,2],"farbe":"#000000ff"}`
	json := func(n int) string {
		return `{"format":1,"datei":{"pfad":"` + pfad + `"},"schwaerzungen":[` +
			strings.TrimSuffix(strings.Repeat(eine+",", n), ",") + `],"angaben":""}`
	}
	if err := os.WriteFile(datei, []byte(json(maxSchwaerzungen)), 0o600); err != nil {
		t.Fatal(err)
	}
	if kp, err := Lies(ort, pfad); err != nil || len(kp.Schwaerzungen) != maxSchwaerzungen {
		t.Fatalf("an der Grenze: %v", err)
	}
	if err := os.WriteFile(datei, []byte(json(maxSchwaerzungen+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Lies(ort, pfad); !errors.Is(err, ErrZuGross) {
		t.Errorf("%d Schwärzungen: %v, erwartet ErrZuGross", maxSchwaerzungen+1, err)
	}
}

// R12, R19: Präparierte Schwärzungen, die document.Schwaerzen oder das Farbformat ablehnt, lassen das Dokument
// unverändert, auch wenn eine gültige davor steht (Probelauf am Abzug).
func TestAnwendenPraeparierteSchwaerzungen(t *testing.T) {
	_, k := ausgang(t, "datei")
	for _, s := range []Schwaerzung{
		{Bereich: [4]int{-5, -5, -1, -1}, Farbe: "#000000ff"},         // außerhalb
		{Bereich: [4]int{0, 0, 10, 10}, Farbe: "#00000000"},           // durchsichtig
		{Bereich: [4]int{0, 0, 10, 10}, Block: 3, Farbe: "#000000ff"}, // Block zu klein
		{Bereich: [4]int{0, 0, 20, 20}, Block: 8, Mosaik: 65, Farbe: "#000000ff"},
		{Bereich: [4]int{0, 0, 20, 20}, Block: 8, Mosaik: -1, Farbe: "#000000ff"},
		{Bereich: [4]int{0, 0, 10, 10}, Farbe: "schwarz"},
		{Bereich: [4]int{0, 0, 10, 10}, Farbe: "#000000ff0"}, // eine Stelle zu lang (Sscanf läse deckendes Schwarz)
	} {
		d, _ := document.Neu(bild(64, 64))
		vorher := string(d.OriginalBild().Pix)
		kp, _ := Erzeuge(d, k, time.Now())
		kp.Schwaerzungen = []Schwaerzung{{Bereich: [4]int{2, 2, 6, 6}, Farbe: "#000000ff"}, s}
		if err := Anwenden(d, &kp, nil); err == nil {
			t.Errorf("%+v angewendet", s)
		}
		if d.KannRueckgaengig() || string(d.OriginalBild().Pix) != vorher {
			t.Errorf("%+v: Dokument verändert", s)
		}
	}
	// Riesige Bereiche werden aufs Bild begrenzt, eine riesige Blockgröße stürzt nicht ab (V7-4).
	d, _ := document.Neu(bild(64, 64))
	kp, _ := Erzeuge(d, k, time.Now())
	kp.Schwaerzungen = []Schwaerzung{
		{Bereich: [4]int{math.MinInt, math.MinInt, math.MaxInt, math.MaxInt}, Farbe: "#000000ff"},
		{Bereich: [4]int{1, 1, 40, 40}, Block: math.MaxInt, Farbe: "#000000ff"},
	}
	if err := Anwenden(d, &kp, nil); err != nil {
		t.Logf("riesige Werte: %v", err)
	}
}

// S1b, S7: Nach Verwirf schreibt keine vorher begonnene Kopie mehr. Die Schreibaufträge laufen in Goroutinen; über viele
// Durchgänge kommt der Fall, dass Verwirf vor dem Schreiben dran ist, sicher vor. Mit -race laufen lassen.
func TestSicherungVerwirfGegenLaufendeAuftraege(t *testing.T) {
	pfad, k := ausgang(t, "datei")
	ort := t.TempDir()
	d, _ := document.Neu(bild(32, 32))
	s := NeueSicherung(ort, k)
	for i := 0; i < 150; i++ {
		if _, err := d.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "y", Schriftgrad: 12}); err != nil {
			t.Fatal(err)
		}
		s.Sichere(d)
		s.Sichere(d)
		s.Verwirf()
		s.Ende()
		if _, err := os.Stat(filepath.Join(ort, Name(pfad))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Durchgang %d: Kopie nach Verwirf wieder da (%v)", i, err)
		}
	}
	// NeueBasis aus einer anderen Goroutine (Speicher-Goroutine gegen Ereignisschleife).
	k2 := k
	k2.Pfad = pfad + ".neu"
	for i := 0; i < 150; i++ {
		s = NeueSicherung(ort, k)
		s.Sichere(d)
		fertig := make(chan struct{})
		go func() { s.NeueBasis(k2); close(fertig) }()
		s.Sichere(d)
		<-fertig
		s.Ende()
		if _, err := os.Stat(filepath.Join(ort, Name(pfad))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Durchgang %d: Kopie zur alten Ausgangsdatei liegt nach NeueBasis (%v)", i, err)
		}
		s.Verwirf()
		s.Ende() // Fenster zu: Sperre frei für den nächsten Durchgang (F-413)
	}
	if reste, _ := os.ReadDir(ort); len(reste) != 0 {
		t.Errorf("%d Reste im Ordner", len(reste))
	}
}
