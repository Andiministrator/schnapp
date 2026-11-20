// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/capture"
	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/selection"
	"github.com/Andiministrator/schnapp/settings"
	"github.com/Andiministrator/schnapp/ui"
)

// testbild: jedes Pixel trägt seine Lage (R = x, G = y) und eine Kennung des Monitors (B).
func testbild(w, h int, kennung uint8) *image.NRGBA {
	b := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: kennung, A: 255})
		}
	}
	return b
}

// quelleAttrappe ersetzt KWin: keine echte Aufnahme.
type quelleAttrappe struct {
	bilder      map[string]*image.NRGBA
	aktiv       *image.NRGBA
	fenster     *image.NRGBA
	fensterErr  error
	aufrufe     []string
	geschlossen bool
	skala       float64 // 0 = 1
}

func (q *quelleAttrappe) Bildschirm(_ context.Context, name string, _ capture.Optionen) (*capture.Bild, error) {
	q.aufrufe = append(q.aufrufe, "Bildschirm "+name)
	b, ok := q.bilder[name]
	if !ok {
		return nil, errors.New("CaptureScreen: Invalid screen requested")
	}
	skala := q.skala
	if skala == 0 {
		skala = 1
	}
	return &capture.Bild{Pixel: b, Skala: skala, Bildschirm: name}, nil
}

func (q *quelleAttrappe) AktiverBildschirm(context.Context, capture.Optionen) (*capture.Bild, error) {
	q.aufrufe = append(q.aufrufe, "Aktiv")
	return &capture.Bild{Pixel: q.aktiv, Skala: 1}, nil
}

func (q *quelleAttrappe) Fenster(context.Context, capture.Optionen) (*capture.Bild, error) {
	q.aufrufe = append(q.aufrufe, "Fenster")
	if q.fensterErr != nil {
		return nil, q.fensterErr
	}
	return &capture.Bild{Pixel: q.fenster, Skala: 1}, nil
}

func (q *quelleAttrappe) Close() error { q.geschlossen = true; return nil }

type lauf struct {
	q          *quelleAttrappe
	an         anschluesse
	overlayMod []selection.Modus
	overlayErg ui.OverlayErgebnis
	overlayErr error
	abgelegt   []*document.Dokument
	ablegenErr error
}

func neuerLauf(t *testing.T) *lauf {
	l := &lauf{q: &quelleAttrappe{
		bilder:  map[string]*image.NRGBA{"A": testbild(4, 3, 1), "B": testbild(6, 4, 2)},
		aktiv:   testbild(5, 5, 3),
		fenster: testbild(2, 2, 4),
	}}
	l.an = anschluesse{
		quelle: func() (capture.Quelle, error) { return l.q, nil },
		monitore: func() ([]ui.Monitor, error) {
			return []ui.Monitor{{Name: "A", Breite: 4, Hoehe: 3}, {Name: "B", Breite: 6, Hoehe: 4}}, nil
		},
		overlay: func(b []ui.MonitorBild, m selection.Modus) (ui.OverlayErgebnis, error) {
			l.overlayMod = append(l.overlayMod, m)
			if len(b) != 2 || b[0].Name != "A" || b[1].Name != "B" {
				t.Errorf("Overlay bekommt %d Bilder", len(b))
			}
			return l.overlayErg, l.overlayErr
		},
		ablegen: func(d *document.Dokument) (string, error) {
			if l.ablegenErr != nil {
				return "", l.ablegenErr
			}
			l.abgelegt = append(l.abgelegt, d)
			return "/ablage/2026-11/datei.png", nil
		},
		letzterPfad: filepath.Join(t.TempDir(), "state", "schnapp", "last-area.json"),
	}
	return l
}

func (l *lauf) fuehre(t Auftrag) (int, string, string) {
	var aus, fehler bytes.Buffer
	code := l.an.Ausfuehren(t, &aus, &fehler)
	return code, aus.String(), fehler.String()
}

// pruefeAbgelegt: genau ein Dokument, dessen Original genau der Bereich des Monitorbilds mit der Kennung ist.
func pruefeAbgelegt(t *testing.T, l *lauf, kennung uint8, bereich image.Rectangle) {
	t.Helper()
	if len(l.abgelegt) != 1 {
		t.Fatalf("%d Dokumente abgelegt", len(l.abgelegt))
	}
	o := l.abgelegt[0].Original()
	if o.Bounds() != image.Rect(0, 0, bereich.Dx(), bereich.Dy()) {
		t.Fatalf("Original %v, Soll Größe von %v (nur der gewählte Bereich)", o.Bounds(), bereich)
	}
	for y := 0; y < bereich.Dy(); y++ {
		for x := 0; x < bereich.Dx(); x++ {
			c := o.NRGBAAt(x, y)
			if c.R != uint8(bereich.Min.X+x) || c.G != uint8(bereich.Min.Y+y) || c.B != kennung {
				t.Fatalf("Pixel %d,%d = %v, Soll aus Monitor %d bei %d,%d", x, y, c, kennung, bereich.Min.X+x, bereich.Min.Y+y)
			}
		}
	}
}

func TestBereichUeberOverlay(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	l.overlayErg = ui.OverlayErgebnis{Monitor: 1, Bereich: image.Rect(1, 1, 4, 3)}
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusBereich})
	if code != exitOK || aus != "/ablage/2026-11/datei.png\n" || fehler != "" {
		t.Fatalf("Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if strings.Join(l.q.aufrufe, ",") != "Bildschirm A,Bildschirm B" || !l.q.geschlossen {
		t.Errorf("erst alle Monitore aufnehmen, dann wählen: %v, geschlossen %v", l.q.aufrufe, l.q.geschlossen)
	}
	if len(l.overlayMod) != 1 || l.overlayMod[0] != selection.ModusBereich {
		t.Errorf("Overlay-Modus %v", l.overlayMod)
	}
	pruefeAbgelegt(t, l, 2, image.Rect(1, 1, 4, 3))
	letzter, err := selection.LadeLetzten(l.an.letzterPfad)
	if err != nil || letzter != (selection.Letzter{Monitor: "B", Groesse: image.Pt(6, 4), Skala: 1, Bereich: image.Rect(1, 1, 4, 3)}) {
		t.Errorf("letzter Bereich %+v, %v", letzter, err)
	}
}

// kaputterSchreiber scheitert bei jedem Schreiben (etwa stdout auf einer vollen Platte).
type kaputterSchreiber struct{}

func (kaputterSchreiber) Write([]byte) (int, error) { return 0, errors.New("Schreiben gescheitert") }

// F-180: Lässt sich der Pfad nicht auf stdout schreiben, steht das auf stderr; der Exit-Code bleibt 0, weil die
// Aufnahme in der Bibliothek liegt.
func TestPfadNichtGeschrieben(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	l.overlayErg = ui.OverlayErgebnis{Monitor: 0, Bereich: image.Rect(0, 0, 2, 2)}
	var fehler bytes.Buffer
	code := l.an.Ausfuehren(Auftrag{Modus: ModusBereich}, kaputterSchreiber{}, &fehler)
	soll := "Could not write the path to standard output: Schreiben gescheitert\n"
	if code != exitOK || fehler.String() != soll {
		t.Errorf("Code %d, fehler %q, Soll 0 und %q", code, fehler.String(), soll)
	}
	if len(l.abgelegt) != 1 {
		t.Errorf("%d Aufnahmen abgelegt, Soll 1", len(l.abgelegt))
	}
}

func TestBildschirmPerKlickUndAbbruch(t *testing.T) {
	l := neuerLauf(t)
	l.overlayErr = ui.ErrAbgebrochen
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusBildschirm, Ziel: Bildschirmziel{Wahl: settings.WahlKlick}})
	if code != exitAbgebrochen || aus != "" || fehler != "" {
		t.Fatalf("Esc: Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if len(l.overlayMod) != 1 || l.overlayMod[0] != selection.ModusMonitor {
		t.Errorf("--screen mit pick muss das Overlay im Monitor-Modus öffnen: %v", l.overlayMod)
	}
	if len(l.abgelegt) != 0 {
		t.Error("nach Abbruch nichts ablegen")
	}
	if _, err := selection.LadeLetzten(l.an.letzterPfad); !errors.Is(err, selection.ErrKeinLetzter) {
		t.Error("nach Abbruch keinen Bereich merken")
	}
}

func TestOverlayErgebnisAusserhalb(t *testing.T) {
	i18n.Setze(nil)
	for _, e := range []ui.OverlayErgebnis{
		{Monitor: 2, Bereich: image.Rect(0, 0, 1, 1)},
		{Monitor: 0, Bereich: image.Rect(0, 0, 5, 1)}, // A ist nur 4 breit
		{Monitor: 0, Bereich: image.Rectangle{}},
	} {
		l := neuerLauf(t)
		l.overlayErg = e
		code, _, fehler := l.fuehre(Auftrag{Modus: ModusBereich})
		if code != exitFehler || !strings.Contains(fehler, "outside the captured image") || len(l.abgelegt) != 0 {
			t.Errorf("%+v: Code %d, fehler %q", e, code, fehler)
		}
	}
}

func TestBildschirmMitNamen(t *testing.T) {
	l := neuerLauf(t)
	code, _, _ := l.fuehre(Auftrag{Modus: ModusBildschirm, Ziel: Bildschirmziel{Name: "A", Wahl: settings.WahlAlle}})
	if code != exitOK || strings.Join(l.q.aufrufe, ",") != "Bildschirm A" || len(l.overlayMod) != 0 {
		t.Fatalf("Name geht vor der Einstellung: Code %d, Aufrufe %v, Overlay %v", code, l.q.aufrufe, l.overlayMod)
	}
	pruefeAbgelegt(t, l, 1, image.Rect(0, 0, 4, 3))
	if letzter, err := selection.LadeLetzten(l.an.letzterPfad); err != nil || letzter.Monitor != "A" {
		t.Errorf("letzter Bereich %+v, %v", letzter, err)
	}
	// Unbekannter Name: Fehler von KWin, nichts abgelegt.
	l = neuerLauf(t)
	if code, _, fehler := l.fuehre(Auftrag{Modus: ModusBildschirm, Ziel: Bildschirmziel{Name: "X"}}); code != exitFehler || !strings.Contains(fehler, "Invalid screen") {
		t.Errorf("unbekannter Name: Code %d, fehler %q", code, fehler)
	}
}

func TestBildschirmUnterZeigerUndAlle(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	code, _, _ := l.fuehre(Auftrag{Modus: ModusBildschirm, Ziel: Bildschirmziel{Wahl: settings.WahlZeiger}})
	if code != exitOK || strings.Join(l.q.aufrufe, ",") != "Aktiv" || len(l.overlayMod) != 0 {
		t.Fatalf("pointer: Code %d, Aufrufe %v", code, l.q.aufrufe)
	}
	pruefeAbgelegt(t, l, 3, image.Rect(0, 0, 5, 5))

	l = neuerLauf(t)
	code, _, fehler := l.fuehre(Auftrag{Modus: ModusBildschirm, Ziel: Bildschirmziel{Wahl: settings.WahlAlle}})
	if code != exitFehler || !strings.Contains(fehler, "all monitors") || len(l.q.aufrufe) != 0 {
		t.Errorf("all: Code %d, fehler %q, Aufrufe %v", code, fehler, l.q.aufrufe)
	}
}

func TestFenster(t *testing.T) {
	l := neuerLauf(t)
	if code, _, _ := l.fuehre(Auftrag{Modus: ModusFenster}); code != exitOK {
		t.Fatalf("Code %d", code)
	}
	pruefeAbgelegt(t, l, 4, image.Rect(0, 0, 2, 2))
	if _, err := selection.LadeLetzten(l.an.letzterPfad); !errors.Is(err, selection.ErrKeinLetzter) {
		t.Error("Fensteraufnahme merkt keinen Bereich (--repeat wiederholt Bereiche)")
	}
	l = neuerLauf(t)
	l.q.fensterErr = capture.ErrAbgebrochen
	if code, aus, fehler := l.fuehre(Auftrag{Modus: ModusFenster}); code != exitAbgebrochen || aus != "" || fehler != "" {
		t.Errorf("abgebrochene Fensterwahl: Code %d, aus %q, fehler %q", code, aus, fehler)
	}
}

func TestWiederholen(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	code, _, fehler := l.fuehre(Auftrag{Modus: ModusWiederholen})
	if code != exitFehler || !strings.Contains(fehler, "No area to repeat yet") || len(l.q.aufrufe) != 0 {
		t.Fatalf("ohne letzten Bereich: Code %d, fehler %q, Aufrufe %v", code, fehler, l.q.aufrufe)
	}
	if err := selection.SpeichereLetzten(l.an.letzterPfad, selection.Letzter{Monitor: "B", Groesse: image.Pt(6, 4), Skala: 1, Bereich: image.Rect(2, 0, 5, 2)}); err != nil {
		t.Fatal(err)
	}
	if code, _, _ = l.fuehre(Auftrag{Modus: ModusWiederholen}); code != exitOK || strings.Join(l.q.aufrufe, ",") != "Bildschirm B" {
		t.Fatalf("wiederholen: Code %d, Aufrufe %v", code, l.q.aufrufe)
	}
	pruefeAbgelegt(t, l, 2, image.Rect(2, 0, 5, 2))
	// Monitor hat jetzt eine andere Größe: nicht blind ausschneiden.
	l.q.bilder["B"] = testbild(8, 8, 2)
	if code, _, fehler = l.fuehre(Auftrag{Modus: ModusWiederholen}); code != exitFehler || !strings.Contains(fehler, "has changed.") {
		t.Errorf("Größe geändert: Code %d, fehler %q", code, fehler)
	}
}

// F-85: Gleicher Name, gleiche Größe, aber andere Skala: Meldung statt Aufnahme auf falschem Inhalt.
func TestWiederholenAndereSkala(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	if err := selection.SpeichereLetzten(l.an.letzterPfad, selection.Letzter{Monitor: "B", Groesse: image.Pt(6, 4), Skala: 1, Bereich: image.Rect(2, 0, 5, 2)}); err != nil {
		t.Fatal(err)
	}
	l.q.skala = 2
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusWiederholen})
	if code != exitFehler || aus != "" || !strings.Contains(fehler, "The monitor of the last area has changed. Select a new area with --region.") {
		t.Errorf("andere Skala: Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	if len(l.abgelegt) != 0 {
		t.Errorf("%d Dokumente trotz geänderter Skala abgelegt", len(l.abgelegt))
	}
	// Gegenprobe: gleiche Skala (mit Rundungsrest) nimmt auf.
	l.q.skala = 1.0000001
	if code, _, fehler = l.fuehre(Auftrag{Modus: ModusWiederholen}); code != exitOK {
		t.Errorf("gleiche Skala: Code %d, fehler %q", code, fehler)
	}
}

func TestAblegenUndQuelleScheitern(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	l.ablegenErr = errors.New("Platte voll")
	if code, aus, fehler := l.fuehre(Auftrag{Modus: ModusFenster}); code != exitFehler || aus != "" || !strings.Contains(fehler, "Could not save the screenshot: Platte voll") {
		t.Errorf("Ablegen: Code %d, aus %q, fehler %q", code, aus, fehler)
	}
	l = neuerLauf(t)
	l.an.quelle = func() (capture.Quelle, error) { return nil, errors.New("kein Session-Bus") }
	if code, _, fehler := l.fuehre(Auftrag{Modus: ModusBereich}); code != exitFehler || !strings.Contains(fehler, "kein Session-Bus") {
		t.Errorf("Quelle: Code %d, fehler %q", code, fehler)
	}
}

// F-100: Ohne Zustandsordner sagt --repeat das, statt zu einem --region zu raten; das Merken warnt übersetzbar.
func TestOhneZustandsordner(t *testing.T) {
	i18n.Setze(nil)
	l := neuerLauf(t)
	l.an.letzterPfad = ""
	if code, _, fehler := l.fuehre(Auftrag{Modus: ModusWiederholen}); code != exitFehler || !strings.Contains(fehler, "No state directory") {
		t.Errorf("--repeat: Code %d, fehler %q", code, fehler)
	}
	l.overlayErg = ui.OverlayErgebnis{Monitor: 0, Bereich: image.Rect(0, 0, 2, 2)}
	code, _, fehler := l.fuehre(Auftrag{Modus: ModusBereich})
	if code != exitOK || !strings.Contains(fehler, "Could not remember the area for --repeat: no state directory") {
		t.Errorf("merken: Code %d, fehler %q", code, fehler)
	}
}
