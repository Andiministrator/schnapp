// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// ADR-0023 Nachtrag 6, docs/format.md 7.1: Die Kantenform steht als s:kante an s:ansicht und kommt beim Lesen zurück,
// ohne Rückgängig-Schritt und ohne den Stand zu ändern; jede Form im Rundlauf, auch in den Angaben der Rettungskopie.
func TestKanteRundlauf(t *testing.T) {
	for _, k := range []document.Kante{document.KanteAbriss, document.KanteWelle, document.KanteZacken} {
		d, err := document.Neu(testbild(40, 30))
		if err != nil {
			t.Fatal(err)
		}
		if err := d.SetzeKante(k); err != nil {
			t.Fatal(err)
		}
		daten := schreibe(t, d, Metadaten{})
		if !bytes.Contains(entpackeSVG(t, daten), []byte(`<s:ansicht s:zuschnitt="0 0 40 30" s:kante="`+k.Name()+`">`)) {
			t.Errorf("%v: s:kante fehlt im snAP-Block", k)
		}
		g, err := LeseArbeitsdatei(bytes.NewReader(daten))
		if err != nil {
			t.Fatal(err)
		}
		if g.Dokument.Kante() != k || g.Dokument.KannRueckgaengig() {
			t.Errorf("%v: gelesen %v, Rückgängig möglich %v", k, g.Dokument.Kante(), g.Dokument.KannRueckgaengig())
		}
		a, err := LeseAngaben(bytes.NewReader(AngabenSVG(d)))
		if err != nil || a.Kante != k {
			t.Errorf("%v: Angaben %v %v", k, a, err)
		}
	}
}

// Fehlt s:kante (Dateien vor Nachtrag 6) oder ist der Wert unbekannt (eine neuere Form), gilt zigzag; die Datei ist
// nicht beschädigt (Regel 3 in docs/format.md 11), und LeseMetadaten urteilt gleich.
func TestKanteFehltOderUnbekannt(t *testing.T) {
	png, svg := beispielSVG(t, "view")
	alt := `s:kante="wave"`
	if !strings.Contains(svg, alt) {
		t.Fatalf("Beispiel view ohne %s", alt)
	}
	for name, neu := range map[string]string{
		"fehlt":              ``,
		"unbekannt":          `s:kante="torn-paper"`,
		"Großschreibung":     `s:kante="Wave"`,
		"leer":               `s:kante=""`,
		"anderer Namensraum": `kante="wave"`,
	} {
		s := strings.Replace(svg, alt, neu, 1)
		daten := pngMitSVG(t, png, s)
		g, err := LeseArbeitsdatei(bytes.NewReader(daten))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if g.Dokument.Kante() != document.KanteZacken || g.Dokument.KannRueckgaengig() {
			t.Errorf("%s: Kante %v, Rückgängig möglich %v, Soll zigzag ohne Schritt", name, g.Dokument.Kante(),
				g.Dokument.KannRueckgaengig())
		}
		if _, err := LeseMetadaten(bytes.NewReader(daten)); err != nil {
			t.Errorf("%s: LeseMetadaten %v", name, err)
		}
		a, err := LeseAngaben(strings.NewReader(s))
		if err != nil || a.Kante != document.KanteZacken {
			t.Errorf("%s: Angaben %v %v", name, a, err)
		}
	}
}

// Ein fremdes PNG (ohne Schnapp-Blöcke) bekommt die Voreinstellung des Dokuments; die Form aus der Einstellung setzt
// erst cmd/schnapp (ADR-0023 Nachtrag 6).
func TestKanteFremdesBild(t *testing.T) {
	var buf bytes.Buffer
	if err := ExportierePNG(&buf, testbild(20, 10)); err != nil {
		t.Fatal(err)
	}
	g, err := LeseArbeitsdatei(&buf)
	if err != nil || !g.Fremd {
		t.Fatalf("%v %v", g, err)
	}
	if g.Dokument.Kante() != document.KanteAbriss {
		t.Errorf("fremdes Bild: %v, Soll tear", g.Dokument.Kante())
	}
}
