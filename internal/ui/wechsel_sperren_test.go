// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Sperren des Wechsels der Aufnahme (ADR-0036 Ergänzung 6, Zusatz vom 24.11.2026). Übernommen aus den Gegenproben des
// Kritikers B der Sitzungs-QA 24.11. nachmittags (KB-1, KB-2, KB-4, KB-6, KB-7, KB-8) und für die Mutanten KM1 bis KM25
// der Gruppe bibliothek-ui (F-652 bis F-657, F-654).

import (
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/libview"
)

// ladeAngehalten klickt auf die Vorschau i und wartet, bis die Ladefunktion läuft und an l.tor hält.
func (l *leistenLauf) ladeAngehalten(i int) {
	l.t.Helper()
	l.e.s.Gespeichert()
	l.tor = make(chan struct{})
	l.klick(l.kachel(i))
	l.warte(func() bool { return len(l.aufrufe()) == 1 })
	if !l.e.wechsel.laedt {
		l.t.Fatal("Vorbedingung: lädt nicht")
	}
}

// F-652 (KB-1): Während die andere Aufnahme lädt, öffnet Strg+Umschalt+E keinen Export-Dialog (er schriebe sonst das
// neue Dokument unter dem für das alte gewählten Namen).
func TestWechselKeinExportWaehrendLaden(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(t.TempDir(), "kein-bus"))
	l.ladeAngehalten(0)
	l.fokusiere(l.e)
	l.taste("E", key.ModShortcut|key.ModShift)
	if l.e.exportOffen || l.e.exportAntwort != nil {
		t.Error("Export-Dialog während des Ladens geöffnet")
	}
	close(l.tor)
	l.ruhig()
	if l.e.s.D != l.neu[l.pfade[0]] {
		t.Error("ohne Export nicht gewechselt")
	}
}

// F-652: Ist beim Abholen doch ein Export-Dialog offen (an der Sperre vorbei), bleibt die offene Aufnahme; die
// geladene gibt ihre Rettungskopie frei, die Statuszeile sagt warum.
func TestWechselExportOffenBeimAbholen(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	altD := l.e.s.D
	l.ladeAngehalten(0)
	l.e.exportOffen = true
	t.Cleanup(func() { l.e.exportOffen = false })
	close(l.tor)
	l.ruhig()
	if l.e.s.D != altD {
		t.Fatal("bei offenem Export-Dialog gewechselt")
	}
	if r := l.rett[l.pfade[0]]; r == nil || r.ende != 1 {
		t.Errorf("Rettung der geladenen: %+v", r)
	}
	if !strings.Contains(l.e.meldung.Text, "export") {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
}

// F-653 (KB-2, KB-4): Während des Ladens ruht das Bearbeiten — keine Texteingabe, kein Zug, kein Bereich der
// Texterkennung; nach dem Laden wechselt das Fenster, und nichts Getipptes geht verloren. Scheitert das Laden, ist
// das Bearbeiten wieder frei.
func TestWechselBearbeitenRuhtWaehrendLaden(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.ladeAngehalten(0)
	if !l.e.s.Ruht() {
		t.Error("Bearbeiten ruht nicht")
	}
	// Texteingabe am Testtext
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugText)
	l.setle()
	f := l.e.leinwandFlaeche(l.gtx())
	q := l.e.s.Abb.ZuFenster(document.Pt(5, 20))
	l.klick(f.Min.Add(image.Pt(int(q.X), int(q.Y))))
	l.fokusiere(l.e)
	l.tippe("bc")
	if l.e.s.TextLaeuft() || l.e.s.Geaendert() {
		t.Error("Texteingabe während des Ladens angenommen")
	}
	// Zug mit dem Kasten
	l.e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	l.setle()
	a := f.Min.Add(image.Pt(f.Dx()/2, f.Dy()/2))
	l.maus(pointer.Move, a, 0)
	l.maus(pointer.Press, a, pointer.ButtonPrimary)
	l.maus(pointer.Move, a.Add(image.Pt(30, 20)), pointer.ButtonPrimary)
	l.bild()
	if l.e.s.Zieht() {
		t.Error("Zug während des Ladens angenommen")
	}
	l.maus(pointer.Release, a.Add(image.Pt(30, 20)), 0)
	if l.e.s.BeginneTextBereich() {
		t.Error("Bereich der Texterkennung während des Ladens angenommen")
	}
	close(l.tor)
	l.ruhig()
	if l.e.s.D != l.neu[l.pfade[0]] {
		t.Fatal("nicht gewechselt")
	}
	if l.e.s.Ruht() {
		t.Error("neue Sitzung ruht")
	}

	// Scheitert das Laden, ist das Bearbeiten der offenen Aufnahme wieder frei, und die Statuszeile nennt den Grund ohne
	// Paketpräfix und ohne Pfad (F-663, F-661).
	l.fehler = fmt.Errorf("fileformat: %w", &fs.PathError{Op: "open", Path: "/home/jemand/Bilder/x.png", Err: syscall.EACCES})
	l.mu.Lock()
	l.offen = nil
	l.mu.Unlock()
	l.ladeAngehalten(1)
	close(l.tor)
	l.ruhig()
	if l.e.s.Ruht() {
		t.Error("nach gescheitertem Laden ruht das Bearbeiten weiter")
	}
	if m := l.e.meldung.Text; !strings.HasSuffix(m, ": "+syscall.EACCES.Error()) || strings.Contains(m, "fileformat") || strings.Contains(m, "/") {
		t.Errorf("Meldung %q", m)
	}
}

// F-653: Läuft beim Abholen doch eine Texteingabe oder ein Zug (an der Ruhe vorbei), bleibt die offene Aufnahme.
func TestWechselTextOderZugBeimAbholen(t *testing.T) {
	for _, fall := range []string{"text", "zug"} {
		t.Run(fall, func(t *testing.T) {
			l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
			altD := l.e.s.D
			l.ladeAngehalten(0)
			l.e.s.SetzeRuhe(false) // an der Ruhe vorbei
			if fall == "text" {
				l.textEingabeStarten()
			} else {
				l.e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
				l.setle()
				f := l.e.leinwandFlaeche(l.gtx())
				a := f.Min.Add(image.Pt(f.Dx()/2, f.Dy()/2))
				l.maus(pointer.Move, a, 0)
				l.maus(pointer.Press, a, pointer.ButtonPrimary)
				l.maus(pointer.Move, a.Add(image.Pt(30, 20)), pointer.ButtonPrimary)
				l.bild()
				if !l.e.s.Zieht() {
					t.Fatal("Vorbedingung: kein Zug")
				}
			}
			if l.e.s.Geaendert() {
				t.Fatal("Vorbedingung: Stand schon geändert (Prüfung über Geaendert)")
			}
			close(l.tor)
			l.ruhig()
			if l.e.s.D != altD {
				t.Errorf("Wechsel trotz laufendem %s", fall)
			}
		})
	}
}

// KM1, KM2: Bei offenem Export-Dialog oder laufender Texterkennung wechselt ein Klick nicht, die Statuszeile sagt warum.
func TestWechselAnfrageExportOderOCR(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	altD := l.e.s.D
	l.e.exportOffen = true
	l.klick(l.kachel(0))
	l.ruhig()
	if len(l.aufrufe()) != 0 || l.e.s.D != altD || !strings.Contains(l.e.meldung.Text, "export") {
		t.Errorf("Export offen: geladen %v, Meldung %q", l.aufrufe(), l.e.meldung.Text)
	}
	l.e.exportOffen = false
	l.e.ocr.laeuft = true
	l.klick(l.kachel(0))
	l.ruhig()
	l.e.ocr.laeuft = false
	if len(l.aufrufe()) != 0 || l.e.s.D != altD || !strings.Contains(l.e.meldung.Text, "Text recognition") {
		t.Errorf("Texterkennung: geladen %v, Meldung %q", l.aufrufe(), l.e.meldung.Text)
	}
}

// KM16: Ist das Fenster zum Schließen freigegeben (erzwungen) oder ein Signal vorgemerkt, nimmt die Leiste keinen
// Wechsel mehr an.
func TestWechselAnfrageNachFreigabe(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	l.e.erzwungen = true
	l.e.wechselAnfrage(l.pfade[0])
	l.e.erzwungen = false
	if l.e.wechsel.ziel != "" || l.e.wechsel.laedt {
		t.Error("Wechsel trotz Freigabe zum Schließen angenommen")
	}
}

// KM5: Endet das Speichern vor dem Wechsel mit Schließen (Anfrage oder Signal), entfällt der Wechsel.
func TestWechselNachSpeichernSchliesst(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	for _, ende := range []canvas.Ende{{Schliessen: true, Schluss: canvas.SchlussZu}, {Schluss: canvas.SchlussHinweis}} {
		l.e.wechsel.ziel = l.pfade[0]
		l.e.wechselNachSpeichern(speicherErgebnis{}, ende)
		if l.e.wechsel.ziel != "" || l.e.wechsel.laedt {
			t.Errorf("Ende %+v: Wechsel nicht verworfen", ende)
		}
	}
	l.ruhig()
	if a := l.aufrufe(); len(a) != 0 {
		t.Errorf("geladen: %v", a)
	}
}

// KM6: Ist das Fenster beim Abholen zum Schließen freigegeben, bleibt die offene Aufnahme; die geladene gibt ihre
// Rettungskopie frei.
func TestWechselAbholenNachFreigabe(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	altD := l.e.s.D
	l.ladeAngehalten(0)
	l.e.erzwungen = true
	close(l.tor)
	l.ruhig()
	l.e.erzwungen = false
	if l.e.s.D != altD {
		t.Fatal("gewechselt, obwohl das Fenster zum Schließen freigegeben ist")
	}
	if r := l.rett[l.pfade[0]]; r == nil || r.ende != 1 {
		t.Errorf("Rettung der geladenen: %+v", r)
	}
}

// KM3, KM25: Wird das Fenster während des Ladens zerstört, gibt die geladene Aufnahme ihre Rettungskopie frei.
func TestWechselZerstoeren(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.ladeAngehalten(0)
	close(l.tor)
	_ = l.e.zerstoert(nil)
	if r := l.rett[l.pfade[0]]; r == nil || r.ende != 1 {
		t.Errorf("Rettung der geladenen nach dem Zerstören: %+v", r)
	}
	if l.e.wechsel.laedt {
		t.Error("Wechsel nach dem Zerstören nicht abgeräumt")
	}
}

// KM13, KM14, KM17, KM23: uebernimm räumt den Zustand ab, der am alten Dokument hängt — Lupenbilder, Kanten der
// Nähte, eine laufende Texterkennung und die Meldung „Opening…“.
func TestUebernimmRaeumtAb(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 60, 20)))
	if err != nil {
		t.Fatal(err)
	}
	l.e.lupe.bilder = map[lupenSchluessel]lupenBild{{id: 1}: {gueltig: true}}
	l.e.naht.gueltig = true
	abgebrochen := false
	l.e.ocr.laeuft, l.e.ocr.abbruch = true, func() { abgebrochen = true }
	l.e.melde("Opening…", false)
	l.e.uebernimm(Geoeffnet{Titel: "neu.png", Dokument: d, Rettung: &leistenRettung{}})
	if l.e.lupe.bilder != nil {
		t.Error("Lupenbilder der alten Aufnahme bleiben")
	}
	if l.e.naht.gueltig {
		t.Error("Kanten der alten Aufnahme gelten weiter")
	}
	if !abgebrochen || l.e.ocr.laeuft {
		t.Error("Texterkennung der alten Aufnahme läuft weiter")
	}
	if l.e.meldung.Text != "" {
		t.Errorf("Meldung bleibt stehen: %q", l.e.meldung.Text)
	}
}

// KM24: Nach dem eigenen Speichern vergisst die Leiste das Bild der offenen Aufnahme im Speicher (sonst zeigte sie
// nach einer Schwärzung weiter das alte Bild, Kernprinzip 2).
func TestLeisteNachSpeichernVergisst(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	rel, ok := l.b.Relativ(l.pfade[1]) // die offene
	if !ok {
		t.Fatal("Vorbedingung")
	}
	k := &l.e.unten
	l.warte(func() bool { _, z := k.v.Bild(rel); return z != libview.Laedt })
	vorher := k.v.Belegt()
	if vorher == 0 {
		t.Fatal("Vorbedingung: kein Bild im Speicher")
	}
	l.e.leisteNachSpeichern()
	if k.v.Belegt() >= vorher {
		t.Errorf("Bild der offenen Aufnahme nicht vergessen (belegt %d, vorher %d)", k.v.Belegt(), vorher)
	}
}

// F-656 (KB-6): Kommt vorn eine Aufnahme dazu, bleibt der Tastaturfokus bei derselben Aufnahme, nicht an der Stelle.
func TestLeisteFokusNachNeulesen(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.fokusiere(&l.e.unten.knoepfe[2])
	l.setle()
	if !l.fokus(&l.e.unten.knoepfe[2]) {
		t.Fatal("Vorbedingung: kein Fokus auf der dritten Vorschau")
	}
	vorher := l.e.unten.l.Eintraege()[2].Pfad
	legeAufnahmeAb(t, l.b, color.NRGBA{G: 90}, time.Date(2026, 11, 25, 8, 0, 0, 0, time.UTC), false) // neueste, Stelle 0
	l.e.leisteNeuLesen()
	l.warte(func() bool { return !l.e.unten.indexLaeuft && len(l.e.unten.l.Eintraege()) == 4 })
	l.setle()
	stelle := -1
	for i := range 4 {
		if l.fokus(&l.e.unten.knoepfe[i]) {
			stelle = i
		}
	}
	if stelle < 0 {
		t.Fatal("kein Fokus mehr in der Leiste")
	}
	if nachher := l.e.unten.l.Eintraege()[stelle].Pfad; nachher != vorher {
		t.Errorf("Fokus saß auf %s, nach dem Neulesen auf %s", vorher, nachher)
	}
	// Ohne Fokus in der Leiste zieht das Neulesen ihn nicht hinein.
	l.fokusiere(l.e)
	l.setle()
	legeAufnahmeAb(t, l.b, color.NRGBA{B: 90}, time.Date(2026, 11, 26, 8, 0, 0, 0, time.UTC), false)
	l.e.leisteNeuLesen()
	l.warte(func() bool { return !l.e.unten.indexLaeuft && len(l.e.unten.l.Eintraege()) == 5 })
	l.setle()
	if !l.fokus(l.e) {
		t.Error("Neulesen zieht den Fokus in die Leiste")
	}
}

// F-657 (KB-7): Ein Klick auf die grün umrandete offene Aufnahme lädt sie nicht neu, auch wenn ihr Pfad über einen
// Verweis führt.
func TestLeisteKlickAufOffeneUeberVerweis(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	verweis := filepath.Join(t.TempDir(), "bilder")
	if err := os.Symlink(l.b.Orte.Ablage, verweis); err != nil {
		t.Fatal(err)
	}
	rel, ok := l.b.Relativ(l.pfade[1])
	if !ok {
		t.Fatal("Vorbedingung")
	}
	ueber := filepath.Join(verweis, filepath.FromSlash(rel))
	l.e.pfad = func() string { return ueber }
	l.e.leisteAktuell()
	l.setle()
	if !l.e.unten.l.IstAktuell(1) {
		t.Fatal("Vorbedingung: über den Verweis nicht grün")
	}
	altD := l.e.s.D
	l.klick(l.kachel(1))
	l.ruhig()
	if a := l.aufrufe(); len(a) != 0 || l.e.s.D != altD {
		t.Errorf("Klick auf die offene Aufnahme lädt sie neu: %v", a)
	}
	// Gegenprobe: eine andere wechselt.
	l.klick(l.kachel(0))
	l.ruhig()
	if a := l.aufrufe(); len(a) != 1 {
		t.Errorf("andere Aufnahme: %v", a)
	}
}
