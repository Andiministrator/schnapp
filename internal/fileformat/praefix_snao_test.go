// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/document"
)

// einmalPraefix: Der Text beginnt mit „fileformat: “, und das Präfix steht genau einmal da (ADR-0019 Nachtrag 2, F-362).
func einmalPraefix(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: kein Fehler", name)
		return
	}
	if s := err.Error(); !strings.HasPrefix(s, "fileformat: ") || strings.Count(s, "fileformat:") != 1 {
		t.Errorf("%s: Fehlertext %q ohne genau ein Präfix „fileformat: “ am Anfang", name, s)
	}
}

// F-362: Ist der Block snAO einer Arbeitsdatei kein lesbares PNG, beginnt der Fehlertext von LeseArbeitsdatei mit dem
// Paketnamen, der Blockname folgt („fileformat: snAO: …“); die Fehlerklasse bleibt erhalten.
func TestPraefixSnAO(t *testing.T) {
	d, err := document.Neu(rauschbild(8, 8, 1))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := SchreibeArbeitsdatei(&buf, d, rauschbild(8, 8, 2), Metadaten{}); err != nil {
		t.Fatal(err)
	}
	bloecke, err := leseBloecke(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	gefunden := false
	for i := range bloecke {
		if bloecke[i].typ == blockOriginal {
			bloecke[i].daten = []byte("kein PNG")
			gefunden = true
		}
	}
	if !gefunden {
		t.Fatal("Vorbedingung: kein Block snAO in der Arbeitsdatei")
	}
	var kaputt bytes.Buffer
	if err := schreibeBloecke(&kaputt, bloecke); err != nil {
		t.Fatal(err)
	}
	_, err = LeseArbeitsdatei(bytes.NewReader(kaputt.Bytes()))
	if err == nil {
		t.Fatal("Vorbedingung: beschädigter snAO-Block wird gelesen")
	}
	einmalPraefix(t, "LeseArbeitsdatei", err)
	if !strings.HasPrefix(err.Error(), "fileformat: snAO: ") {
		t.Errorf("Blockname fehlt hinter dem Präfix: %q", err)
	}
	if !IstBeschaedigt(err) || !errors.Is(err, ErrBeschaedigt) {
		t.Errorf("Fehlerklasse verloren: %v", err)
	}
	// LeseMetadaten: dieselbe Hülle (metadaten.go), hier über einen Block ohne gültigen IHDR.
	_, err = LeseMetadaten(bytes.NewReader(kaputt.Bytes()))
	einmalPraefix(t, "LeseMetadaten", err)
	if !strings.HasPrefix(err.Error(), "fileformat: snAO: ") || !IstBeschaedigt(err) {
		t.Errorf("LeseMetadaten: %q", err)
	}
}

// F-362 Teil 3: Rohe Fehler von Betriebssystem und Schreiber verlassen das Paket mit Präfix und bleiben per errors.Is
// erkennbar.
func TestPraefixPaketgrenze(t *testing.T) {
	_, err := Oeffne(filepath.Join(t.TempDir(), "gibt-es-nicht.png"))
	einmalPraefix(t, "Oeffne", err)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Oeffne: errors.Is(fs.ErrNotExist) gilt nicht für %v", err)
	}

	d, err := document.Neu(rauschbild(8, 8, 1))
	if err != nil {
		t.Fatal(err)
	}
	ausfall := errors.New("Schreiber kaputt")
	err = SchreibeArbeitsdatei(&scheiterSchreiber{err: ausfall}, d, rauschbild(8, 8, 2), Metadaten{})
	einmalPraefix(t, "SchreibeArbeitsdatei", err)
	if !errors.Is(err, ausfall) {
		t.Errorf("SchreibeArbeitsdatei: Ursache verloren: %v", err)
	}
	err = ExportierePNG(&scheiterSchreiber{err: ausfall}, rauschbild(8, 8, 3))
	einmalPraefix(t, "ExportierePNG", err)
	if !errors.Is(err, ausfall) {
		t.Errorf("ExportierePNG: Ursache verloren: %v", err)
	}
	err = ExportiereSVG(&scheiterSchreiber{err: ausfall}, d)
	einmalPraefix(t, "ExportiereSVG", err)
	if !errors.Is(err, ausfall) {
		t.Errorf("ExportiereSVG: Ursache verloren: %v", err)
	}
	einmalPraefix(t, "ExportiereJPG", ExportiereJPG(&bytes.Buffer{}, rauschbild(8, 8, 3), 0))
}

// scheiterSchreiber schlägt bei jedem Schreiben mit err fehl.
type scheiterSchreiber struct{ err error }

func (s *scheiterSchreiber) Write([]byte) (int, error) { return 0, s.err }
