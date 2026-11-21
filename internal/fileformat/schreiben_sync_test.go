// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"errors"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// langsamerSchreiber bildet ein langsames Medium nach: Write kehrt sofort zurück (Seitencache), Sync braucht
// syncDauer auf der Ersatzuhr. Er merkt sich, wie viele Pulse zwischen zwei Sync-Aufrufen kamen und wie lange der
// letzte Fortschritt bei jedem Sync-Ende zurücklag.
type langsamerSchreiber struct {
	uhr       *time.Time
	syncDauer time.Duration
	syncs     int
	groesse   int
	pulseEnde []uint64 // Pulszähler am Ende jedes Syncs
	pulseAnf  []uint64 // Pulszähler am Anfang jedes Syncs
	luecken   []time.Duration
	syncFehl  error
}

func (s *langsamerSchreiber) Write(b []byte) (int, error) { s.groesse += len(b); return len(b), nil }

func (s *langsamerSchreiber) Sync() error {
	s.pulseAnf = append(s.pulseAnf, saveprogress.Pulse())
	s.syncs++
	if s.syncFehl != nil {
		return s.syncFehl
	}
	*s.uhr = s.uhr.Add(s.syncDauer)
	s.luecken = append(s.luecken, saveprogress.Jetzt().Sub(saveprogress.Zuletzt()))
	s.pulseEnde = append(s.pulseEnde, saveprogress.Pulse())
	return nil
}

func ersatzuhr(t *testing.T) *time.Time {
	t.Helper()
	jetzt := saveprogress.Basis().Add(time.Hour)
	alt := saveprogress.SetzeUhr(func() time.Time { return jetzt })
	t.Cleanup(func() { saveprogress.SetzeUhr(alt) })
	return &jetzt
}

// F-313: Auf einem langsamen Medium vergeht die Zeit im Sync, nicht im Write. Ein Schreiber mit Sync wird je Stück
// synchronisiert, und nach jedem Sync folgt ein Puls — die Wartezeit ist in Stücke zerlegt, jedes unter der Frist,
// auch wenn der Sync insgesamt ein Mehrfaches der Frist (15 s) braucht.
func TestSchreibenSynchronisiertJeStueckUndPulst(t *testing.T) {
	uhr := ersatzuhr(t)
	d, bild := rauschDokument(t)
	s := &langsamerSchreiber{uhr: uhr, syncDauer: 10 * time.Second}
	if err := SchreibeArbeitsdatei(s, d, bild, Metadaten{}); err != nil {
		t.Fatal(err)
	}
	stuecke := (s.groesse + schreibStueck - 1) / schreibStueck
	if stuecke < 2 || s.syncs != stuecke {
		t.Fatalf("%d Syncs bei %d Stücken (Datei %d Bytes), erwartet einer je Stück und mindestens zwei", s.syncs, stuecke, s.groesse)
	}
	if gesamt := time.Duration(s.syncs) * s.syncDauer; gesamt < 30*time.Second {
		t.Fatalf("Sync-Gesamtdauer %v, die Probe braucht mindestens das Doppelte der Frist", gesamt)
	}
	for i, l := range s.luecken {
		if l >= 15*time.Second {
			t.Errorf("Sync %d: letzter Fortschritt %v her, das Speichern gälte als hängend", i+1, l)
		}
	}
	// Zwischen dem Ende eines Syncs und dem Anfang des nächsten: der Puls nach dem Sync und der des nächsten Stücks.
	for i := 1; i < len(s.pulseAnf); i++ {
		if s.pulseAnf[i]-s.pulseEnde[i-1] < 2 {
			t.Errorf("zwischen Sync %d und %d nur %d Puls(e), erwartet der Puls nach dem Sync und der des nächsten Stücks",
				i, i+1, s.pulseAnf[i]-s.pulseEnde[i-1])
		}
	}
}

// F-313: Der Puls nach dem letzten Sync fehlt nicht (sonst läge zwischen letztem Stück und Rückkehr kein Fortschritt).
func TestSchreibenPulstNachDemLetztenSync(t *testing.T) {
	uhr := ersatzuhr(t)
	d, bild := rauschDokument(t)
	s := &langsamerSchreiber{uhr: uhr, syncDauer: time.Second}
	if err := SchreibeArbeitsdatei(s, d, bild, Metadaten{}); err != nil {
		t.Fatal(err)
	}
	if nach := saveprogress.Pulse() - s.pulseEnde[len(s.pulseEnde)-1]; nach < 1 {
		t.Errorf("kein Puls nach dem letzten Sync")
	}
}

// F-313: Scheitert ein Sync, bricht das Schreiben ab und liefert genau diesen Fehler.
func TestSchreibenBrichtBeiSyncFehlerAb(t *testing.T) {
	uhr := ersatzuhr(t)
	d, bild := rauschDokument(t)
	platte := errors.New("Medium weg")
	s := &langsamerSchreiber{uhr: uhr, syncFehl: platte}
	if err := SchreibeArbeitsdatei(s, d, bild, Metadaten{}); !errors.Is(err, platte) {
		t.Errorf("Rückgabe %v, erwartet %v", err, platte)
	}
	if s.syncs != 1 {
		t.Errorf("%d Sync-Aufrufe, erwartet 1 (nach dem Fehler nicht weiter)", s.syncs)
	}
}
