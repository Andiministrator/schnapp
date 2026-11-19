// SPDX-License-Identifier: GPL-3.0-or-later

// Regressionstests zu den Kritiker-Befunden der Sitzungs-QA Sprint 3 (F-50 bis F-55), aus den Proben des Kritikers
// übernommen. Externes Testpaket: geprüft wird nur die öffentliche Schnittstelle.
package library_test

import (
	"errors"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/library"
)

var jetzt = time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)

func neu(t *testing.T) *library.Bibliothek {
	t.Helper()
	heim := t.TempDir()
	o, err := library.OrteAus(func(k string) string {
		if k == "HOME" {
			return heim
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	b := library.Neu(o)
	b.Zone = time.UTC
	b.Jetzt = func() time.Time { return jetzt }
	return b
}

func ablegen(t *testing.T, b *library.Bibliothek, erstellt time.Time) library.Eintrag {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	img.Set(0, 0, color.NRGBA{1, 2, 3, 255})
	d, err := document.Neu(img)
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: erstellt})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// F-50: Eine riesige Frist („nie“) läuft nicht über, sondern ist ein Fehler; die Obergrenze selbst gilt noch.
func TestFristObergrenze(t *testing.T) {
	b := neu(t)
	ablegen(t, b, jetzt.Add(-time.Hour))
	erg, err := b.Aufraeumen(jetzt, library.Einstellung{An: true, Tage: 200000})
	if !errors.Is(err, library.ErrFrist) || len(erg.Verschoben) != 0 {
		t.Errorf("Frist 200000: %+v %v", erg, err)
	}
	if _, err := b.Aufraeumen(jetzt, library.Einstellung{An: true, Tage: 36500}); err != nil {
		t.Errorf("Frist 36500: %v", err)
	}
}

// F-51: Das Vorschaubild verschwindet mit der Aufnahme — beim Aufräumen und beim Abgleich.
func TestVorschauVerschwindetMitAufnahme(t *testing.T) {
	b := neu(t)
	e := ablegen(t, b, jetzt.Add(-100*24*time.Hour))
	vp := b.VorschauPfad(e.Pfad)
	if _, err := os.Stat(vp); err != nil {
		t.Fatalf("keine Vorschau: %v", err)
	}
	erg, err := b.Aufraeumen(jetzt, library.Voreinstellung)
	if err != nil || len(erg.Verschoben) != 1 {
		t.Fatalf("Aufräumen: %+v %v", erg, err)
	}
	if _, err := os.Stat(vp); err == nil {
		t.Error("Vorschau bleibt nach dem Aufräumen liegen")
	}

	f := ablegen(t, b, jetzt)
	vf := b.VorschauPfad(f.Pfad)
	pfad := b.Orte.Ablage + string(os.PathSeparator) + f.Pfad
	ziel := b.Orte.Ablage + string(os.PathSeparator) + "weg.png.bak"
	if err := os.Rename(pfad, ziel); err != nil { // „im Dateimanager verschoben“
		t.Fatal(err)
	}
	if n, err := b.Abgleich(); err != nil || n != 1 {
		t.Fatalf("Abgleich: %d %v", n, err)
	}
	if _, err := os.Stat(vf); err == nil {
		t.Error("Vorschau bleibt nach dem Abgleich liegen")
	}
}

// F-52: Der Neuaufbau schreibt eine fehlende Vorschau neu.
func TestNeuaufbauHeiltVorschau(t *testing.T) {
	b := neu(t)
	e := ablegen(t, b, jetzt)
	vp := b.VorschauPfad(e.Pfad)
	if err := os.Remove(vp); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Neuaufbau(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(vp); err != nil {
		t.Errorf("Neuaufbau schreibt keine Vorschau: %v", err)
	}
}

// F-54: Ein falsches Datum im Index (Cache) bringt eine frische Aufnahme nicht in den Papierkorb.
func TestAlterAusDerDatei(t *testing.T) {
	b := neu(t)
	e := ablegen(t, b, jetzt)
	roh, err := os.ReadFile(b.Orte.Index)
	if err != nil {
		t.Fatal(err)
	}
	gefaelscht := strings.Replace(string(roh), e.Erstellt.Format(time.RFC3339), "2020-01-01T00:00:00Z", 1)
	if gefaelscht == string(roh) {
		t.Fatalf("Datum nicht im Index gefunden: %s", roh)
	}
	if err := os.WriteFile(b.Orte.Index, []byte(gefaelscht), 0o600); err != nil {
		t.Fatal(err)
	}
	erg, err := b.Aufraeumen(jetzt, library.Voreinstellung)
	if err != nil {
		t.Fatal(err)
	}
	if len(erg.Verschoben) != 0 || len(erg.Uebersprungen) != 1 || !errors.Is(erg.Uebersprungen[0].Grund, library.ErrGeschuetzt) {
		t.Errorf("frische Aufnahme nach Index-Datum behandelt: %+v", erg)
	}
}

// F-55: Vorhandene eigene Ordner werden auf 0700 gebracht.
func TestOrdnerRechteImBestand(t *testing.T) {
	b := neu(t)
	for _, o := range []string{b.Orte.Ablage, b.Orte.Vorschau} {
		if err := os.MkdirAll(o, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(o, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ablegen(t, b, jetzt)
	for _, o := range []string{b.Orte.Ablage, b.Orte.Vorschau} {
		if st, err := os.Stat(o); err != nil || st.Mode().Perm() != 0o700 {
			t.Errorf("%s: %v %v", o, st.Mode().Perm(), err)
		}
	}
}
