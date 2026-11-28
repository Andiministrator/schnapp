// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/wlclip"
)

// halterAttrappe ersetzt den Halter-Prozess: merkt Art und Daten; err != nil heißt „kein Halter“.
func halterAttrappe(t *testing.T, err error) (art *wlclip.Art, daten *[]byte) {
	t.Helper()
	t.Setenv("WAYLAND_SOCKET", "")
	ohneDataControl.Store(false)
	t.Cleanup(func() { ohneDataControl.Store(false) })
	a, d := wlclip.Art(-1), []byte(nil)
	alt := uebergib
	t.Cleanup(func() { uebergib = alt })
	uebergib = func(_ string, x wlclip.Art, b []byte, _ time.Duration) (int, string, error) {
		a, d = x, b
		return 1, wlclip.ManagerExt, err
	}
	return &a, &d
}

// leserAttrappe ersetzt das Lesen der Zwischenablage: Angebot typen, Daten je Typ; zählt die Aufrufe.
func leserAttrappe(t *testing.T, inhalt map[string][]byte, typen ...string) *int {
	t.Helper()
	n := 0
	alt := liesZwischenablage
	t.Cleanup(func() { liesZwischenablage = alt })
	liesZwischenablage = func(_ context.Context, wahl func([]string) string, max int64) (wlclip.Gelesen, error) {
		n++
		g := wlclip.Gelesen{Typen: typen}
		g.Typ = wahl(typen)
		if g.Typ != "" {
			g.Daten = inhalt[g.Typ]
			if int64(len(g.Daten)) > max {
				return wlclip.Gelesen{}, wlclip.ErrZuGross
			}
		}
		return g, nil
	}
	return &n
}

// warteEinfuegen wartet auf die Antwort des Lesens und wertet sie aus.
func warteEinfuegen(t *testing.T, e *editor) {
	t.Helper()
	ende := time.Now().Add(10 * time.Second)
	for e.einf.laeuft {
		if time.Now().After(ende) {
			t.Fatal("keine Antwort des Lesens")
		}
		e.einfuegenAbholen()
		time.Sleep(time.Millisecond)
	}
}

// editorMitKasten ist ein Editor mit einem gewählten Kasten (Kennung zurück).
func editorMitKasten(t *testing.T) (*editor, int) {
	t.Helper()
	e := testEditor(t)
	e.s.SetzeFenster(image.Pt(400, 300))
	id, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(5, 5), document.Pt(15, 12)),
		Fuellung: color.NRGBA{B: 255, A: 255}})
	if err != nil {
		t.Fatal(err)
	}
	e.s.Auswahl = id
	return e, id
}

// ADR-0045 Punkte 1 und 2: Strg+C mit Auswahl gibt dem Halter die Objekte (SVG ohne Pixel und ihr PNG) in einem
// Rahmen; ohne Auswahl das Bild; Strg+Umschalt+C immer das Bild.
func TestStrgCKopiertObjekte(t *testing.T) {
	art, daten := halterAttrappe(t, nil)
	e, _ := editorMitKasten(t)
	gtx := layout.Context{}
	e.fuehreAus(gtx, aktKopieren, true)
	warteKopieren(t, e, gtx)
	if *art != wlclip.ArtObjekte {
		t.Fatalf("Art %v, erwartet ArtObjekte", *art)
	}
	svg, bild, err := wlclip.Entpacke(*daten)
	if err != nil {
		t.Fatal(err)
	}
	q, err := fileformat.LeseObjekte(bytes.NewReader(svg))
	if err != nil || len(q.Objekte()) != 1 {
		t.Fatalf("Objekte %v, %v", q, err)
	}
	if b, err := png.Decode(bytes.NewReader(bild)); err != nil || b.Bounds().Size() != image.Pt(10, 7) {
		t.Errorf("Bild der Objekte %v, %v", b, err)
	}
	if e.meldung.Text != "Copied 1 object to the clipboard" || e.fensterHaeltKopie {
		t.Errorf("Meldung %q, Fenster hält %v", e.meldung.Text, e.fensterHaeltKopie)
	}
	e.fuehreAus(gtx, aktBildKopieren, true)
	warteKopieren(t, e, gtx)
	if *art != wlclip.ArtBild {
		t.Errorf("Strg+Umschalt+C: Art %v", *art)
	}
	e.s.Abwaehlen()
	e.fuehreAus(gtx, aktKopieren, true)
	warteKopieren(t, e, gtx)
	if *art != wlclip.ArtBild {
		t.Errorf("Strg+C ohne Auswahl: Art %v", *art)
	}
}

// ADR-0045 Punkt 1: Strg+X kopiert und löscht in einem Schritt; ohne Auswahl nichts.
func TestStrgXSchneidetAus(t *testing.T) {
	art, _ := halterAttrappe(t, nil)
	e, _ := editorMitKasten(t)
	gtx := layout.Context{}
	e.fuehreAus(gtx, aktAusschneiden, true)
	warteKopieren(t, e, gtx)
	if *art != wlclip.ArtObjekte || len(e.s.D.Objekte()) != 0 {
		t.Fatalf("Art %v, %d Objekte", *art, len(e.s.D.Objekte()))
	}
	if e.meldung.Text != "Cut 1 object to the clipboard" {
		t.Errorf("Meldung %q", e.meldung.Text)
	}
	if err := e.s.D.Rueckgaengig(); err != nil || len(e.s.D.Objekte()) != 1 {
		t.Errorf("Rückgängig: %v, %d Objekte", err, len(e.s.D.Objekte()))
	}
	*art = -1
	e.s.Abwaehlen()
	e.fuehreAus(gtx, aktAusschneiden, true)
	if *art != -1 || e.kopierLaeuft {
		t.Error("Strg+X ohne Auswahl kopiert")
	}
}

// ADR-0045 Punkt 3: Strg+V liest nur den Objekt-Typ, fügt ein, wählt und meldet; nur ein Bild ergibt den Hinweis
// auf „New from clipboard“, nur Text „nichts zum Einfügen“.
func TestStrgVFuegtEin(t *testing.T) {
	quelle, id := editorMitKasten(t)
	q, err := quelle.s.D.NurObjekte([]int{id})
	if err != nil {
		t.Fatal(err)
	}
	e := testEditor(t)
	e.s.SetzeFenster(image.Pt(400, 300))
	n := leserAttrappe(t, map[string][]byte{wlclip.MIMEObjekte: fileformat.AngabenSVG(q)}, wlclip.MIMEObjekte, wlclip.MIME)
	t.Setenv("WAYLAND_SOCKET", "")
	e.fuehreAus(layout.Context{}, aktEinfuegen, true)
	warteEinfuegen(t, e)
	if *n != 1 || len(e.s.D.Objekte()) != 1 || e.s.AnzahlGewaehlt() != 1 {
		t.Fatalf("gelesen %d, %d Objekte, %d gewählt", *n, len(e.s.D.Objekte()), e.s.AnzahlGewaehlt())
	}
	if e.meldung.Text != "Pasted 1 object" {
		t.Errorf("Meldung %q", e.meldung.Text)
	}

	leserAttrappe(t, map[string][]byte{wlclip.MIME: {1}}, wlclip.MIME, wlclip.MarkeKlipper)
	e.fuehreAus(layout.Context{}, aktEinfuegen, true)
	warteEinfuegen(t, e)
	if want := i18n.T("The clipboard holds an image. File → New from clipboard (Ctrl+Shift+V) opens it as a new capture."); e.meldung.Text != want {
		t.Errorf("nur Bild: Meldung %q", e.meldung.Text)
	}
	leserAttrappe(t, nil, "text/plain")
	e.fuehreAus(layout.Context{}, aktEinfuegen, true)
	warteEinfuegen(t, e)
	if e.meldung.Text != i18n.T("The clipboard holds nothing to paste.") || len(e.s.D.Objekte()) != 1 {
		t.Errorf("nur Text: Meldung %q, %d Objekte", e.meldung.Text, len(e.s.D.Objekte()))
	}
	leserAttrappe(t, map[string][]byte{wlclip.MIMEObjekte: []byte("<svg")}, wlclip.MIMEObjekte)
	e.fuehreAus(layout.Context{}, aktEinfuegen, true)
	warteEinfuegen(t, e)
	if e.meldung.Text != i18n.T("The objects in the clipboard could not be read.") {
		t.Errorf("kaputt: Meldung %q", e.meldung.Text)
	}
}

// ADR-0045 Punkte 2 und 5: Ohne Halter bietet das Fenster das Bild der Objekte an und merkt sie; ohne Lesemöglichkeit
// fügt Strg+V diese ein, solange das Fenster die Kopie hält, danach meldet es den Grund.
func TestEinfuegenAusDemFenster(t *testing.T) {
	halterAttrappe(t, errors.New("kein Halter"))
	n := leserAttrappe(t, nil)
	e, _ := editorMitKasten(t)
	gtx := layout.Context{}
	e.fuehreAus(gtx, aktKopieren, true)
	warteKopieren(t, e, gtx)
	if !e.fensterHaeltKopie || e.einf.kopie == nil || e.kopieIstText {
		t.Fatalf("Fenster hält %v, Objekte gemerkt %v, Text %v", e.fensterHaeltKopie, e.einf.kopie != nil, e.kopieIstText)
	}
	t.Setenv("WAYLAND_SOCKET", "7")
	e.fuehreAus(gtx, aktEinfuegen, true)
	if *n != 0 || len(e.s.D.Objekte()) != 2 {
		t.Errorf("gelesen %d, %d Objekte", *n, len(e.s.D.Objekte()))
	}
	e.fensterHaeltKopie = false // ein anderes Programm hat übernommen
	e.fuehreAus(gtx, aktEinfuegen, true)
	if len(e.s.D.Objekte()) != 2 || e.meldung.Text != i18n.T("This desktop does not let Schnapp read the clipboard.") {
		t.Errorf("%d Objekte, Meldung %q", len(e.s.D.Objekte()), e.meldung.Text)
	}
}

// ADR-0045 Punkt 4: „New from clipboard“ liest ein Bild (PNG vor JPEG), legt es über die Anbindung ab und wechselt
// dorthin; ohne Bild eine Meldung, ohne Bibliothek grau mit Grund.
func TestNeuAusZwischenablage(t *testing.T) {
	b := image.NewNRGBA(image.Rect(0, 0, 9, 4))
	var p bytes.Buffer
	_ = png.Encode(&p, b)
	leserAttrappe(t, map[string][]byte{wlclip.MIME: p.Bytes()}, "image/jpeg", wlclip.MIME)
	t.Setenv("WAYLAND_SOCKET", "")
	e := testEditor(t)
	if e.neuGrau() == "" {
		t.Error("ohne Bibliothek nicht grau")
	}
	var abgelegt *document.Dokument
	var geoeffnet []string
	e.bib = &Anbindung{
		Ablegen: func(d *document.Dokument) (string, error) { abgelegt = d; return "/ablage/neu.png", nil },
		Oeffnen: func(pfad string) (Geoeffnet, error) {
			geoeffnet = append(geoeffnet, pfad)
			return Geoeffnet{}, errors.New("nicht in diesem Test")
		},
	}
	if g := e.neuGrau(); g != "" {
		t.Fatalf("grau: %q", g)
	}
	e.fuehreAus(layout.Context{}, aktNeuAusZwischenablage, true)
	warteEinfuegen(t, e)
	if abgelegt == nil || abgelegt.Groesse() != image.Pt(9, 4) {
		t.Fatalf("abgelegt %v", abgelegt)
	}
	if e.wechsel.ziel != "/ablage/neu.png" {
		t.Errorf("Wechsel zu %q", e.wechsel.ziel)
	}
	for e.wechsel.laedt {
		e.wechselAbholen()
		time.Sleep(time.Millisecond)
	}
	if !slices.Equal(geoeffnet, []string{"/ablage/neu.png"}) {
		t.Errorf("geöffnet %v", geoeffnet)
	}

	e2 := testEditor(t)
	e2.bib = e.bib
	leserAttrappe(t, nil, "text/plain")
	e2.fuehreAus(layout.Context{}, aktNeuAusZwischenablage, true)
	warteEinfuegen(t, e2)
	if e2.meldung.Text != i18n.T("The clipboard holds no image.") {
		t.Errorf("ohne Bild: Meldung %q", e2.meldung.Text)
	}
}

// ADR-0045 Punkt 4: Hält ein Speicher-Hinweis den Wechsel zur neuen Aufnahme auf (eine Datei von außen wird als neue
// Aufnahme gespeichert), sagt die Fußzeile, wo das Bild aus der Zwischenablage liegt; geladen wird nichts.
func TestNeuAusZwischenablageNachHinweis(t *testing.T) {
	b := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	var p bytes.Buffer
	_ = png.Encode(&p, b)
	leserAttrappe(t, map[string][]byte{wlclip.MIME: p.Bytes()}, wlclip.MIME)
	t.Setenv("WAYLAND_SOCKET", "")
	e, _ := editorMitKasten(t) // geändert: der Wechsel speichert erst
	geladen := false
	e.bib = &Anbindung{
		Ablegen: func(*document.Dokument) (string, error) { return "/ablage/neu.png", nil },
		Oeffnen: func(string) (Geoeffnet, error) { geladen = true; return Geoeffnet{}, errors.New("nein") },
	}
	e.speichern = func(*document.Dokument) (Gespeichert, error) {
		return Gespeichert{Hinweis: "Als neue Aufnahme gespeichert."}, nil
	}
	e.fuehreAus(layout.Context{}, aktNeuAusZwischenablage, true)
	warteEinfuegen(t, e)
	warteSpeichern(t, e)
	want := i18n.T("The image from the clipboard is saved as a new capture; open it from the library.")
	if geladen || e.wechsel.ziel != "" || !strings.HasSuffix(e.meldung.Text, want) {
		t.Errorf("geladen %v, Ziel %q, Meldung %q", geladen, e.wechsel.ziel, e.meldung.Text)
	}
}

// ADR-0045 Punkt 3: Was das Budget des Öffnens sprengt, wird nicht eingefügt (hier zu viele Zeichen).
func TestEinfuegenPrueftBudget(t *testing.T) {
	e := testEditor(t)
	q, err := document.OhnePixel(image.Pt(40, 30), image.Rect(0, 0, 40, 30), nil, document.KanteAbriss)
	if err != nil {
		t.Fatal(err)
	}
	for range 101 { // 101 × 10 000 Zeichen über dem ZeichenBudget von 1e6 (je Text höchstens 10 000)
		if _, err := q.Hinzufuegen(document.Text{Ort: document.Pt(1, 10), Inhalt: strings.Repeat("x", 10_000),
			Schriftgrad: 8, Farbe: color.NRGBA{A: 255}}); err != nil {
			t.Fatal(err)
		}
	}
	e.einfuegenAnwenden(q)
	if len(e.s.D.Objekte()) != 0 || e.meldung.Text != i18n.T("Could not paste: the objects would take too long to draw.") {
		t.Errorf("%d Objekte, Meldung %q", len(e.s.D.Objekte()), e.meldung.Text)
	}
}

// F-984: Hält das Fenster selbst eine Objekt-Kopie (Halter gescheitert, Lesen aber möglich), nimmt Strg+V die
// gemerkten Objekte, ohne zu lesen.
func TestEinfuegenGemerktTrotzLesen(t *testing.T) {
	halterAttrappe(t, errors.New("keine Meldung des Halters"))
	n := leserAttrappe(t, map[string][]byte{wlclip.MIME: {1}}, wlclip.MIME)
	e, _ := editorMitKasten(t)
	gtx := layout.Context{}
	e.fuehreAus(gtx, aktKopieren, true)
	warteKopieren(t, e, gtx)
	if !e.lesenMoeglich() || !e.fensterHaeltKopie {
		t.Fatalf("Vorbedingung: lesbar %v, Fenster hält %v", e.lesenMoeglich(), e.fensterHaeltKopie)
	}
	e.fuehreAus(gtx, aktEinfuegen, true)
	if *n != 0 || len(e.s.D.Objekte()) != 2 || e.einf.laeuft {
		t.Errorf("gelesen %d, %d Objekte, läuft %v", *n, len(e.s.D.Objekte()), e.einf.laeuft)
	}
}

// F-989: Strg+X löscht erst nach gelungenem Kopieren; scheitert das Kopieren oder ändert sich das Bild dazwischen,
// bleiben die Objekte.
func TestAusschneidenErstNachErfolg(t *testing.T) {
	halterAttrappe(t, nil)
	e, _ := editorMitKasten(t)
	gtx := layout.Context{}
	e.fuehreAus(gtx, aktAusschneiden, true)
	if len(e.s.D.Objekte()) != 1 {
		t.Error("vor der Antwort gelöscht")
	}
	if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(2, 2))}); err != nil {
		t.Fatal(err)
	}
	warteKopieren(t, e, gtx)
	want := i18n.T("Copied to the clipboard; the objects were not removed. Press Ctrl+X again to remove them.")
	if len(e.s.D.Objekte()) != 2 || e.meldung.Text != want {
		t.Errorf("%d Objekte, Meldung %q", len(e.s.D.Objekte()), e.meldung.Text)
	}
	// gescheitertes Kopieren: nichts löschen
	e.einf.schneide, e.einf.schneideStand = []int{1}, e.s.D.Stand()
	e.kopierLaeuft = true
	e.kopierAntwort = make(chan kopierErgebnis, 1)
	e.kopierAntwort <- kopierErgebnis{err: errors.New("kaputt")}
	e.kopierenAbholen(gtx)
	if len(e.s.D.Objekte()) != 2 || e.einf.schneide != nil {
		t.Errorf("nach Fehler %d Objekte, Merker %v", len(e.s.D.Objekte()), e.einf.schneide)
	}
}

// F-987, F-988: Esc bricht ein Lesen ab; Schließen bricht Strg+V ab und wartet auf ein laufendes Ablegen.
func TestEinfuegenAbbruchUndSchliessen(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "")
	alt := liesZwischenablage
	t.Cleanup(func() { liesZwischenablage = alt })
	abgebrochen := make(chan struct{})
	liesZwischenablage = func(ctx context.Context, _ func([]string) string, _ int64) (wlclip.Gelesen, error) {
		<-ctx.Done()
		close(abgebrochen)
		return wlclip.Gelesen{}, ctx.Err()
	}
	e := testEditor(t)
	e.fuehreAus(layout.Context{}, aktEinfuegen, true)
	if !e.einfuegenEsc() || e.einf.laeuft {
		t.Fatal("Esc bricht nicht ab")
	}
	select {
	case <-abgebrochen:
	case <-time.After(5 * time.Second):
		t.Fatal("Kontext nicht abgebrochen")
	}
	if e.meldung.Text != i18n.T("Reading the clipboard cancelled.") {
		t.Errorf("Meldung %q", e.meldung.Text)
	}

	// Ablegen läuft beim Schließen: abwarten, Ort nennen.
	var aus bytes.Buffer
	altAus := fehlerAusgabe
	t.Cleanup(func() { fehlerAusgabe = altAus })
	fehlerAusgabe = &aus
	ch := make(chan einfuegeErgebnis, 1)
	e.einf.laeuft, e.einf.neu, e.einf.antwort = true, true, ch
	go func() { time.Sleep(50 * time.Millisecond); ch <- einfuegeErgebnis{neu: true, pfad: "/ablage/neu.png"} }()
	e.einfuegenAbwarten(5 * time.Second)
	if !strings.Contains(aus.String(), "/ablage/neu.png") || e.einf.laeuft {
		t.Errorf("Ausgabe %q, läuft %v", aus.String(), e.einf.laeuft)
	}
}

// F-991: Solange Strg+V liest, wechselt die Aufnahme nicht; bei offener Bibliothek fügt die Antwort nicht ein.
func TestEinfuegenSperrtWechsel(t *testing.T) {
	e, id := editorMitKasten(t)
	e.bib = &Anbindung{Oeffnen: func(string) (Geoeffnet, error) { return Geoeffnet{}, nil }}
	e.einf.laeuft = true
	if e.wechselAnfrage("/ablage/andere.png") {
		t.Error("Wechsel während des Lesens angenommen")
	}
	e.einf.laeuft = false
	q, _ := e.s.D.NurObjekte([]int{id})
	ch := make(chan einfuegeErgebnis, 1)
	ch <- einfuegeErgebnis{q: q, gelesen: true}
	e.einf.laeuft, e.einf.antwort = true, ch
	e.bibl.offen = true
	e.einfuegenAbholen()
	if len(e.s.D.Objekte()) != 1 || e.meldung.Text != i18n.T("Close the library to paste into the image.") {
		t.Errorf("%d Objekte, Meldung %q", len(e.s.D.Objekte()), e.meldung.Text)
	}
}

// F-990: Mit --ignore-limit über dem Budget nimmt Einfügen trotzdem an.
func TestEinfuegenOhneGrenze(t *testing.T) {
	SetzeExportOhneGrenze(true)
	t.Cleanup(func() { SetzeExportOhneGrenze(false) })
	e := testEditor(t)
	q, _ := document.OhnePixel(image.Pt(40, 30), image.Rect(0, 0, 40, 30), nil, document.KanteAbriss)
	for range 101 {
		_, _ = q.Hinzufuegen(document.Text{Ort: document.Pt(1, 10), Inhalt: strings.Repeat("x", 10_000), Schriftgrad: 8,
			Farbe: color.NRGBA{A: 255}})
	}
	e.einfuegenAnwenden(q)
	if len(e.s.D.Objekte()) != 101 {
		t.Errorf("%d Objekte, Meldung %q", len(e.s.D.Objekte()), e.meldung.Text)
	}
}

// F-986: Mit Fokus im Eingabefeld der Leiste (Hex) wirken Strg+X, Strg+V und Strg+C nicht auf die Objekte.
func TestZwischenablageNichtAusEingabefeld(t *testing.T) {
	halterAttrappe(t, nil)
	n := leserAttrappe(t, nil, wlclip.MIMEObjekte)
	l := neuerMenueLauf(t, image.Pt(1200, 800)) // Text „Hallo“ ist gewählt
	vorher := len(l.e.s.D.Objekte())
	l.klick(mittenPunkt(l.e.seite.fw.hex.rahmen))
	if l.e.fokusFeld(l.gtx()) == nil {
		t.Fatal("Vorbedingung: Fokus im Hex-Feld")
	}
	for _, k := range []key.Name{"X", "V", "C"} {
		l.taste(k, key.ModShortcut)
	}
	if len(l.e.s.D.Objekte()) != vorher || l.e.kopierLaeuft || l.e.einf.laeuft || *n != 0 {
		t.Errorf("%d Objekte (vorher %d), Kopieren %v, Lesen %v/%d", len(l.e.s.D.Objekte()), vorher, l.e.kopierLaeuft,
			l.e.einf.laeuft, *n)
	}
}

// F-993: Rundlauf Strg+C → Halter → Strg+V mit Lupe und Gruppe in eine andere Aufnahme; JPEG allein für „New from
// clipboard“.
func TestRundlaufLupeUndGruppe(t *testing.T) {
	_, daten := halterAttrappe(t, nil)
	e := testEditor(t)
	e.s.SetzeFenster(image.Pt(400, 300))
	for _, o := range []document.Objekt{
		document.Lupe{Quelle: document.R(document.Pt(2, 2), document.Pt(8, 8)), Ziel: document.R(document.Pt(20, 2), document.Pt(32, 14)),
			Strich: document.Strich{Breite: 1, Farbe: color.NRGBA{A: 255}}},
		document.Kasten{Ebene: document.Ebene{Gruppe: 3}, Bereich: document.R(document.Pt(2, 18), document.Pt(6, 22))},
		document.Kasten{Ebene: document.Ebene{Gruppe: 3}, Bereich: document.R(document.Pt(8, 18), document.Pt(12, 22))},
	} {
		if _, err := e.s.D.Hinzufuegen(o); err != nil {
			t.Fatal(err)
		}
	}
	e.s.WaehleAlle()
	gtx := layout.Context{}
	e.fuehreAus(gtx, aktKopieren, true)
	warteKopieren(t, e, gtx)
	svg, _, err := wlclip.Entpacke(*daten)
	if err != nil {
		t.Fatal(err)
	}
	ziel := testEditor(t)
	ziel.s.SetzeFenster(image.Pt(400, 300))
	leserAttrappe(t, map[string][]byte{wlclip.MIMEObjekte: svg}, wlclip.MIMEObjekte)
	ziel.fuehreAus(gtx, aktEinfuegen, true)
	warteEinfuegen(t, ziel)
	os := ziel.s.D.Objekte()
	if len(os) != 3 {
		t.Fatalf("%d Objekte, Meldung %q", len(os), ziel.meldung.Text)
	}
	if _, ok := os[0].(document.Lupe); !ok {
		t.Errorf("erstes %T, erwartet Lupe", os[0])
	}
	if g1, g2 := os[1].Ebenenwerte().Gruppe, os[2].Ebenenwerte().Gruppe; g1 == 0 || g1 != g2 {
		t.Errorf("Gruppen %d %d", g1, g2)
	}
}

func TestNeuAusZwischenablageJPEG(t *testing.T) {
	var j bytes.Buffer
	_ = jpeg.Encode(&j, image.NewNRGBA(image.Rect(0, 0, 5, 6)), nil)
	leserAttrappe(t, map[string][]byte{"image/jpeg": j.Bytes()}, "text/plain", "image/jpeg")
	t.Setenv("WAYLAND_SOCKET", "")
	e := testEditor(t)
	var abgelegt *document.Dokument
	e.bib = &Anbindung{
		Ablegen: func(d *document.Dokument) (string, error) { abgelegt = d; return "/ablage/j.png", nil },
		Oeffnen: func(string) (Geoeffnet, error) { return Geoeffnet{}, errors.New("nicht hier") },
	}
	e.fuehreAus(layout.Context{}, aktNeuAusZwischenablage, true)
	warteEinfuegen(t, e)
	if abgelegt == nil || abgelegt.Groesse() != image.Pt(5, 6) {
		t.Errorf("abgelegt %v, Meldung %q", abgelegt, e.meldung.Text)
	}
}

// F-987: Die Taste Esc bricht ein laufendes Lesen ab (über die Tastentabelle, nicht nur die Funktion).
func TestEscBrichtLesenAb(t *testing.T) {
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	abgebrochen := false
	ch := make(chan einfuegeErgebnis, 1)
	l.e.einf.laeuft, l.e.einf.antwort, l.e.einf.abbruch = true, ch, func() { abgebrochen = true }
	l.taste(key.NameEscape, 0)
	if !abgebrochen || l.e.einf.laeuft {
		t.Errorf("abgebrochen %v, läuft %v", abgebrochen, l.e.einf.laeuft)
	}
}

// V-1: Das Budget gilt an der Ziellage. Objekte, die in der Quelle außerhalb des kleinen Ziels lagen und deshalb mittig
// eingefügt würden, werden dort geprüft und abgelehnt.
func TestEinfuegenBudgetAnDerZiellage(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "")
	q, _ := document.OhnePixel(image.Pt(4000, 30), image.Rect(0, 0, 4000, 30), nil, document.KanteAbriss)
	for range 101 {
		_, _ = q.Hinzufuegen(document.Text{Ort: document.Pt(3000, 10), Inhalt: strings.Repeat("x", 10_000),
			Schriftgrad: 8, Farbe: color.NRGBA{A: 255}})
	}
	leserAttrappe(t, map[string][]byte{wlclip.MIMEObjekte: fileformat.AngabenSVG(q)}, wlclip.MIMEObjekte)
	e := testEditor(t)
	e.s.SetzeFenster(image.Pt(400, 300))
	e.fuehreAus(layout.Context{}, aktEinfuegen, true)
	warteEinfuegen(t, e)
	if len(e.s.D.Objekte()) != 0 || e.meldung.Text != i18n.T("Could not paste: the objects would take too long to draw.") {
		t.Errorf("%d Objekte, Meldung %q", len(e.s.D.Objekte()), e.meldung.Text)
	}
}

// V-2: Esc während des Ablegens bricht nicht ab (die Aufnahme entsteht), vorher schon; Schließen wartet danach weiter.
func TestEscWaehrendAblegen(t *testing.T) {
	b := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	var p bytes.Buffer
	_ = png.Encode(&p, b)
	leserAttrappe(t, map[string][]byte{wlclip.MIME: p.Bytes()}, wlclip.MIME)
	t.Setenv("WAYLAND_SOCKET", "")
	e := testEditor(t)
	im, weiter := make(chan struct{}), make(chan struct{})
	abgelegt := 0
	e.bib = &Anbindung{
		Ablegen: func(*document.Dokument) (string, error) {
			close(im)
			<-weiter
			abgelegt++
			return "/ablage/neu.png", nil
		},
		Oeffnen: func(string) (Geoeffnet, error) { return Geoeffnet{}, errors.New("nicht hier") },
	}
	e.fuehreAus(layout.Context{}, aktNeuAusZwischenablage, true)
	<-im
	if !e.einfuegenEsc() || !e.einf.laeuft {
		t.Fatalf("Esc während des Ablegens: läuft %v", e.einf.laeuft)
	}
	if e.meldung.Text != i18n.T("The image from the clipboard is being saved as a new capture.") {
		t.Errorf("Meldung %q", e.meldung.Text)
	}
	var aus bytes.Buffer
	alt := fehlerAusgabe
	t.Cleanup(func() { fehlerAusgabe = alt })
	fehlerAusgabe = &aus
	go func() { time.Sleep(30 * time.Millisecond); close(weiter) }()
	e.einfuegenAbwarten(5 * time.Second)
	if abgelegt != 1 || !strings.Contains(aus.String(), "/ablage/neu.png") {
		t.Errorf("abgelegt %d, Ausgabe %q", abgelegt, aus.String())
	}

	// vor dem Ablegen: Esc bricht ab, nichts wird abgelegt
	e2 := testEditor(t)
	e2.bib = &Anbindung{
		Ablegen: func(*document.Dokument) (string, error) { t.Error("abgelegt trotz Esc"); return "", nil },
		Oeffnen: func(string) (Geoeffnet, error) { return Geoeffnet{}, nil },
	}
	altL := liesZwischenablage
	t.Cleanup(func() { liesZwischenablage = altL })
	frei := make(chan struct{})
	liesZwischenablage = func(_ context.Context, wahl func([]string) string, _ int64) (wlclip.Gelesen, error) {
		<-frei
		return wlclip.Gelesen{Typen: []string{wlclip.MIME}, Typ: wahl([]string{wlclip.MIME}), Daten: p.Bytes()}, nil
	}
	e2.fuehreAus(layout.Context{}, aktNeuAusZwischenablage, true)
	ch := e2.einf.antwort
	if !e2.einfuegenEsc() || e2.einf.laeuft {
		t.Fatal("Esc vor dem Ablegen bricht nicht ab")
	}
	close(frei)
	if r := <-ch; !errors.Is(r.err, context.Canceled) {
		t.Errorf("Ergebnis nach Esc: %v", r.err)
	}
}

// F-988, V-4: zerstoert wartet auf ein laufendes „New from clipboard“ und nennt den Ort.
func TestZerstoertWartetAufAblegen(t *testing.T) {
	e := testEditor(t)
	var aus bytes.Buffer
	alt := fehlerAusgabe
	t.Cleanup(func() { fehlerAusgabe = alt })
	fehlerAusgabe = &aus
	ch := make(chan einfuegeErgebnis, 1)
	phase := new(atomic.Int32)
	phase.Store(phaseAblegen)
	e.einf.laeuft, e.einf.neu, e.einf.antwort, e.einf.phase = true, true, ch, phase
	go func() { time.Sleep(30 * time.Millisecond); ch <- einfuegeErgebnis{neu: true, pfad: "/ablage/z.png"} }()
	_ = e.zerstoert(nil)
	if !strings.Contains(aus.String(), "/ablage/z.png") {
		t.Errorf("Ausgabe %q", aus.String())
	}
}
