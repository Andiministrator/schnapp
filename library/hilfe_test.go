// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
)

// umgebung baut eine eigene Umgebung: HOME im Testverzeichnis, alle XDG-Variablen leer (also Voreinstellungen).
// Die echte Umgebung wird nie gelesen.
func umgebung(t *testing.T, zusatz map[string]string) (func(string) string, string) {
	t.Helper()
	heim := t.TempDir()
	m := map[string]string{"HOME": heim}
	for k, v := range zusatz {
		m[k] = v
	}
	return func(k string) string { return m[k] }, heim
}

// neueBibliothek legt eine Bibliothek in einem eigenen Testverzeichnis an, mit fester Zeitzone UTC+2 und fester
// Uhr.
func neueBibliothek(t *testing.T) (*Bibliothek, string) {
	t.Helper()
	env, heim := umgebung(t, nil)
	o, err := OrteAus(env)
	if err != nil {
		t.Fatal(err)
	}
	b := Neu(o)
	b.Zone = time.FixedZone("T", 2*3600)
	b.Jetzt = func() time.Time { return time.Date(2026, 11, 19, 12, 35, 12, 0, time.UTC) }
	return b, heim
}

// muster liefert ein Bild mit Struktur (Schachbrett), damit Verkleinern etwas zu zeigen hat.
func muster(w, h int) *image.NRGBA {
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBA{200, 30, 30, 255}
			if (x/8+y/8)%2 == 0 {
				c = color.NRGBA{20, 20, 220, 255}
			}
			b.SetNRGBA(x, y, c)
		}
	}
	return b
}

func neuesDokument(t *testing.T, w, h int) *document.Dokument {
	t.Helper()
	d, err := document.Neu(muster(w, h))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// ablegen legt ein 40x30-Bild mit den Metadaten ab.
func ablegen(t *testing.T, b *Bibliothek, m fileformat.Metadaten) Eintrag {
	t.Helper()
	d := neuesDokument(t, 40, 30)
	e, err := b.Ablegen(d, d.Original(), m)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func zeit(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func schreibe(t *testing.T, pfad, inhalt string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pfad), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pfad, []byte(inhalt), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ersetzeUmbenenne tauscht umbenenne für die Dauer des Tests (Attrappe für einen Fehlschlag im eigenen Code).
func ersetzeUmbenenne(t *testing.T, f func(von, nach string) error) {
	t.Helper()
	alt := umbenenne
	umbenenne = f
	t.Cleanup(func() { umbenenne = alt })
}

// ersetzeVerknuepfe tauscht verknuepfe für die Dauer des Tests (Attrappe: Dateisystem ohne Hardlinks o. ä.).
func ersetzeVerknuepfe(t *testing.T, f func(von, nach string) error) {
	t.Helper()
	alt := verknuepfe
	verknuepfe = f
	t.Cleanup(func() { verknuepfe = alt })
}
