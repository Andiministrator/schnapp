// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"image"
	"testing"

	"github.com/Andiministrator/schnapp/internal/settings"
)

// F-628: Ein Stand, den styles.ini nicht aufnähme (Strich 0, Schriftname mit Leerraum am Rand), wird keine Kachel;
// jeder angenommene Stil besteht FormatStile.
func TestEigeneSchnellstileNurSpeicherbare(t *testing.T) {
	s, _ := stilSitzung(t, image.Pt(200, 100))
	s.SetzeWerkzeug(WerkzeugKasten)
	s.Stil.Breite = 0 // so käme ein Kasten ohne Strich aus einer Datei in den Stand
	if _, r := s.FuegeSchnellstilHinzu(SchnellKasten); r != HinzuUnbrauchbar || s.PruefeSchnellstilHinzu(SchnellKasten) != HinzuUnbrauchbar {
		t.Errorf("Strich 0: %v", r)
	}
	if s.KannSchnellstilHinzufuegen(SchnellKasten) || len(s.EigeneSchnellstile()) != 0 {
		t.Errorf("Strich 0 wurde angenommen")
	}
	s.SetzeWerkzeug(WerkzeugText)
	s.Stil.Schrift = " Test Sans"
	if _, r := s.FuegeSchnellstilHinzu(SchnellText); r != HinzuUnbrauchbar {
		t.Errorf("Schriftname mit Leerraum: %v", r)
	}
	s.Stil.Schrift = "Test Sans" // ein Leerzeichen mitten im Namen ist in Ordnung
	st, r := s.FuegeSchnellstilHinzu(SchnellText)
	if r != HinzuOK {
		t.Fatalf("gültiger Name: %v", r)
	}
	if _, err := settings.FormatStile([]settings.Stil{st.Einstellung()}); err != nil {
		t.Errorf("angenommener Stil besteht FormatStile nicht: %v", err)
	}
}
