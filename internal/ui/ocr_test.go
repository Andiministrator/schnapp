// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"context"
	"errors"
	"image"
	"image/color"
	"strings"
	"sync"
	"testing"
	"time"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/paint"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/ocr"
	"github.com/Andiministrator/schnapp/internal/wlclip"
)

// erkennerAttrappe spielt tesseract: Sie merkt sich Bild und Sprachen und liefert text bzw. err; mit warten blockiert
// sie, bis der Kontext endet (Abbruch).
type erkennerAttrappe struct {
	mu        sync.Mutex
	text      string
	err       error
	warten    bool
	bild      image.Image
	sprachen  string
	oberfl    string
	abgebr    bool
	aufrufe   int
	gestartet chan struct{}
	stufe     int // > 0: meldet diesen Fortschritt, bevor die Erkennung startet
}

func neueAttrappe(text string) *erkennerAttrappe {
	return &erkennerAttrappe{text: text, gestartet: make(chan struct{}, 8)}
}

func (a *erkennerAttrappe) Sprachen(oberflaeche string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.oberfl = oberflaeche
	if oberflaeche == "de" {
		return "deu+eng"
	}
	return "eng"
}

func (a *erkennerAttrappe) Erkenne(ctx context.Context, b image.Image, sprachen string, fortschritt func(int)) (string, error) {
	a.mu.Lock()
	a.bild, a.sprachen, a.aufrufe = b, sprachen, a.aufrufe+1
	warten, stufe := a.warten, a.stufe
	a.mu.Unlock()
	if stufe > 0 {
		fortschritt(stufe)
	}
	a.gestartet <- struct{}{}
	if warten {
		<-ctx.Done()
		a.mu.Lock()
		a.abgebr = true
		a.mu.Unlock()
		return "", ctx.Err()
	}
	return a.text, a.err
}

// ocrEditor ist ein Editor ohne Fenster mit gefundener Attrappe; das Fenster bietet selbst an (kein Halter-Prozess).
func ocrEditor(t *testing.T, a *erkennerAttrappe) *editor {
	t.Helper()
	t.Setenv("WAYLAND_SOCKET", "1")
	e := testEditor(t)
	e.ocr.gesucht, e.ocr.erkenner = true, a
	return e
}

// warteOcr wartet auf die Antwort der Erkennung und nimmt sie wie im Bild an.
func warteOcr(t *testing.T, e *editor) {
	t.Helper()
	ende := time.Now().Add(10 * time.Second)
	for e.ocr.laeuft && time.Now().Before(ende) {
		e.ocrAbholen()
		time.Sleep(time.Millisecond)
	}
	if e.ocr.laeuft {
		t.Fatal("Erkennung endet nicht")
	}
}

// Ohne Suchergebnis, ohne tesseract, ohne Sprachdaten und bei anderem Fehler ist der Eintrag grau und erklärt warum;
// mit Erkenner ist er bedienbar, während einer Erkennung wieder grau.
func TestOcrEintragGrauMitErklaerung(t *testing.T) {
	e := testEditor(t)
	eintrag := func() menueEintrag {
		t.Helper()
		for _, en := range e.menueEintraegeVon(menueBearbeiten) {
			if en.akt == aktTextKopieren {
				return en
			}
		}
		t.Fatal("kein Eintrag „Copy text from image“ im Menü Bearbeiten")
		return menueEintrag{}
	}
	for _, fall := range []struct {
		gesucht bool
		err     error
		er      textErkenner
		an      bool
		text    string
	}{
		{false, nil, nil, false, "Looking for tesseract…"},
		{true, ocr.ErrKeinProgramm, nil, false, "Install tesseract to recognize text."},
		{true, errors.New("ocr: " + ocr.ErrKeineSprache.Error()), nil, false, // nur der Text, nicht der Fehlerwert
			"Text recognition is not available: tesseract has no language data"},
		{true, ocr.ErrKeineSprache, nil, false, "Install language data for tesseract to recognize text."},
		{true, errors.New("ocr: list languages: kaputt"), nil, false, "Text recognition is not available: list languages: kaputt"},
		{true, nil, neueAttrappe("x"), true, ""},
	} {
		e.ocr.gesucht, e.ocr.sucheErr, e.ocr.erkenner = fall.gesucht, fall.err, fall.er
		en := eintrag()
		if en.an != fall.an || en.erklaerung != fall.text || en.text != "Copy text from image" {
			t.Errorf("%v/%v: an %v, Erklärung %q, erwartet %v %q", fall.gesucht, fall.err, en.an, en.erklaerung, fall.an, fall.text)
		}
	}
	e.ocr.laeuft = true
	if en := eintrag(); en.an || en.erklaerung != "Text recognition is already running." {
		t.Errorf("während der Erkennung: an %v, %q", en.an, en.erklaerung)
	}
	if menueKuerzel(eintrag()) != "Ctrl+Shift+T" {
		t.Errorf("Kürzel %q", menueKuerzel(eintrag()))
	}
}

// Das Kürzel ohne tesseract schaltet keinen Modus ein, sondern sagt in der Statuszeile, warum.
func TestOcrKuerzelOhneTesseract(t *testing.T) {
	e := testEditor(t)
	e.ocr.gesucht, e.ocr.sucheErr = true, ocr.ErrKeinProgramm
	e.fuehreAus(layout.Context{}, aktTextKopieren, false)
	if e.s.TextBereichLaeuft() || e.meldung.Text != "Install tesseract to recognize text." {
		t.Errorf("Modus %v, Meldung %q", e.s.TextBereichLaeuft(), e.meldung.Text)
	}
}

// Die Suche läuft einmal im Hintergrund; ihr Ergebnis nimmt ocrAbholen an.
func TestOcrSucheEinmal(t *testing.T) {
	e := testEditor(t)
	a := neueAttrappe("x")
	alt := sucheErkenner
	t.Cleanup(func() { sucheErkenner = alt })
	var n int
	var mu sync.Mutex
	sucheErkenner = func(context.Context) (textErkenner, error) {
		mu.Lock()
		n++
		mu.Unlock()
		return a, nil
	}
	e.ocrSucheStarten()
	e.ocrSucheStarten()
	ende := time.Now().Add(10 * time.Second)
	for !e.ocr.gesucht && time.Now().Before(ende) {
		e.ocrAbholen()
		time.Sleep(time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if !e.ocr.gesucht || e.ocr.erkenner != a || n != 1 {
		t.Errorf("gesucht %v, Erkenner %v, %d Suchen", e.ocr.gesucht, e.ocr.erkenner != nil, n)
	}
	if e.ocrGrau() != "" {
		t.Errorf("nach der Suche grau: %q", e.ocrGrau())
	}
	// Eine gescheiterte Suche liefert keinen Erkenner (auch kein nil in der Schnittstelle).
	sucheErkenner = alt
	t.Setenv("PATH", t.TempDir())
	if er, err := sucheErkenner(context.Background()); er != nil || !errors.Is(err, ocr.ErrKeinProgramm) {
		t.Errorf("ohne tesseract: %v %v", er, err)
	}
}

// Ganzer Ablauf ohne Fenster: Bereich ziehen, Erkennung auf dem sichtbaren Bild (Beschriftung drin, auf Weiß, Ausschnitt
// des Bereichs), Sprache der Oberfläche, Text in die Zwischenablage über das Fenster, „Copied N characters“; das
// Werkzeug bleibt.
func TestOcrAblaufBereich(t *testing.T) {
	a := neueAttrappe("Grüße\nZeile")
	e := ocrEditor(t, a)
	gruen := color.NRGBA{G: 0xC0, A: 0xFF}
	if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(10, 10), document.Pt(20, 20)),
		Fuellung: gruen}); err != nil {
		t.Fatal(err)
	}
	e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	e.s.Abb = canvas.Abbildung{Zoom: 2, Versatz: document.Pt(4, 6)}
	e.ocrBeginnen()
	if !e.s.TextBereichLaeuft() || !strings.HasPrefix(e.hinweis(), "Drag over the text") {
		t.Fatalf("Modus %v, Hinweis %q", e.s.TextBereichLaeuft(), e.hinweis())
	}
	// Ansicht (8, 8) bis (24, 22): Fenster = Ansicht·2 + Versatz
	e.s.Druecken(document.Pt(20, 22), canvas.TastePrimaer)
	e.s.Ziehen(document.Pt(52, 50))
	e.s.Loslassen(document.Pt(52, 50))
	e.ocrBereichPruefen()
	if !e.ocr.laeuft || e.meldung.Text != "Recognizing text…" || e.hinweis() != "Recognizing text…" {
		t.Fatalf("läuft %v, Meldung %q, Hinweis %q", e.ocr.laeuft, e.meldung.Text, e.hinweis())
	}
	if e.s.Werkzeug != canvas.WerkzeugPfeil || e.s.TextBereichLaeuft() {
		t.Errorf("nach dem Zug: Werkzeug %v, Modus %v", e.s.Werkzeug, e.s.TextBereichLaeuft())
	}
	warteOcr(t, e)
	a.mu.Lock()
	b, sprachen, oberfl := a.bild, a.sprachen, a.oberfl
	a.mu.Unlock()
	if b == nil || b.Bounds().Size() != image.Pt(16, 14) {
		t.Fatalf("Bild %v, erwartet 16×14", b)
	}
	if c := color.NRGBAModel.Convert(b.At(b.Bounds().Min.X+5, b.Bounds().Min.Y+5)).(color.NRGBA); c != gruen {
		t.Errorf("Bildpunkt im Kasten %v, erwartet die Beschriftung %v (sichtbares Bild, nicht das Original)", c, gruen)
	}
	if oberfl != "en" || sprachen != "eng" {
		t.Errorf("Sprache der Oberfläche %q, Sprachen %q", oberfl, sprachen)
	}
	if !e.kopierLaeuft {
		t.Fatal("kein Kopieren gestartet")
	}
	warteKopieren(t, e, layout.Context{})
	if e.meldung.Text != "Copied 11 characters" {
		t.Errorf("Meldung %q", e.meldung.Text)
	}
	if !e.fensterHaeltKopie || !e.kopieIstText {
		t.Errorf("Fenster hält Kopie %v, Text %v", e.fensterHaeltKopie, e.kopieIstText)
	}
	// Die Warnung beim Schließen spricht vom Text.
	if !e.fensterWarnen() || !strings.Contains(e.meldung.Text, "holds the text only") {
		t.Errorf("Warnung %q", e.meldung.Text)
	}
}

// Mit Halter: Der Text geht im Textmodus an den Halter (ADR-0024 Nachtrag 9), das Fenster bietet nichts an.
func TestOcrHalterImTextmodus(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "")
	ohneDataControl.Store(false)
	var art wlclip.Art = -1
	var daten []byte
	alt := uebergib
	t.Cleanup(func() { uebergib = alt })
	uebergib = func(_ string, a wlclip.Art, d []byte, _ time.Duration) (int, string, error) {
		art, daten = a, d
		return 1, wlclip.ManagerExt, nil
	}
	e := testEditor(t)
	e.ocr.gesucht, e.ocr.erkenner = true, neueAttrappe("Grüße")
	e.ocrBeginnen()
	e.s.TextBereichGanz()
	e.ocrBereichPruefen()
	warteOcr(t, e)
	warteKopieren(t, e, layout.Context{})
	if art != wlclip.ArtText || string(daten) != "Grüße" {
		t.Errorf("Halter mit Art %v und %q, erwartet Textmodus mit „Grüße“", art, daten)
	}
	if e.fensterHaeltKopie || e.meldung.Text != "Copied 5 characters" {
		t.Errorf("Fenster hält Kopie %v, Meldung %q", e.fensterHaeltKopie, e.meldung.Text)
	}
}

// Enter nimmt das ganze sichtbare Bild; das Bild ist deckend (durchsichtiger Spalt einer Naht auf Weiß). Deutsch als
// Oberfläche wählt deutsche Sprachdaten, die Mehrzahl kommt aus der Übersetzung.
func TestOcrGanzesBildUndSprache(t *testing.T) {
	a := neueAttrappe("A")
	e := ocrEditor(t, a)
	if err := e.s.D.SetzeStreifen([]document.Streifen{{Von: 15, Bis: 25}}); err != nil {
		t.Fatal(err)
	}
	k, err := i18n.NeuerKatalog("de", nil)
	if err != nil {
		t.Fatal(err)
	}
	i18n.Setze(k)
	t.Cleanup(func() { i18n.Setze(nil) })
	e.ocrBeginnen()
	if !e.s.Enter(false) {
		t.Fatal("Enter im Modus wirkt nicht")
	}
	e.ocrBereichPruefen()
	warteOcr(t, e)
	a.mu.Lock()
	b, sprachen := a.bild, a.sprachen
	a.mu.Unlock()
	if b.Bounds().Size() != e.s.D.Groesse() {
		t.Errorf("Bild %v, erwartet die Ansicht %v", b.Bounds().Size(), e.s.D.Groesse())
	}
	for y := b.Bounds().Min.Y; y < b.Bounds().Max.Y; y++ {
		for x := b.Bounds().Min.X; x < b.Bounds().Max.X; x++ {
			if _, _, _, al := b.At(x, y).RGBA(); al != 0xFFFF {
				t.Fatalf("Pixel (%d,%d) nicht deckend", x, y)
			}
		}
	}
	if sprachen != "deu+eng" {
		t.Errorf("Sprachen %q, erwartet deu+eng", sprachen)
	}
	warteKopieren(t, e, layout.Context{})
	if e.meldung.Text != "Copied 1 character" {
		t.Errorf("Meldung %q", e.meldung.Text)
	}
}

// Kein Text: „No text found“, kein Kopieren; Fehler gehen in die Statuszeile.
func TestOcrLeerUndFehler(t *testing.T) {
	for _, fall := range []struct {
		text   string
		err    error
		melde  string
		fehler bool
	}{
		{"", nil, "No text found", false},
		{"", ocr.ErrLeer, "No text found", false},
		{"", ocr.ErrZuGross, "The area is too large for text recognition.", true},
		{"", ocr.ErrFrist, "Text recognition took too long.", true},
		{"", errors.New("ocr: tesseract failed: exit status 1 (Error opening data file)"),
			"Could not recognize text: tesseract failed: exit status 1 (Error opening data file)", true},
	} {
		a := neueAttrappe(fall.text)
		a.err = fall.err
		e := ocrEditor(t, a)
		e.ocrBeginnen()
		e.s.TextBereichGanz()
		e.ocrBereichPruefen()
		warteOcr(t, e)
		if e.kopierLaeuft || e.meldung.Text != fall.melde || e.meldung.Fehler != fall.fehler {
			t.Errorf("%v: Kopieren %v, Meldung %q (Fehler %v)", fall.err, e.kopierLaeuft, e.meldung.Text, e.meldung.Fehler)
		}
	}
}

// Ein neuer Bereichsmodus räumt eine frühere Meldung weg, damit die Anleitung in der Statuszeile steht; eine haltende
// Meldung (was das nächste Schließen tut) bleibt.
func TestOcrModusZeigtAnleitung(t *testing.T) {
	e := ocrEditor(t, neueAttrappe(""))
	links := func() string {
		s, _, _ := canvas.StatusZeile(&e.meldung, e.s.D.Stand(), e.s.Werkzeug, e.hinweis(), e.s.TextLaeuft())
		return s
	}
	e.melde("No text found", false)
	e.ocrBeginnen()
	if got := links(); !strings.HasPrefix(got, "Drag over the text") {
		t.Errorf("Statuszeile %q, erwartet die Anleitung", got)
	}
	e.s.BeendeTextBereich()
	e.meldeHaltend(haltFenster, "The clipboard holds the image only while this window is open.", true)
	e.ocrBeginnen()
	if got := links(); !strings.HasPrefix(got, "The clipboard holds") {
		t.Errorf("haltende Meldung verdrängt: %q", got)
	}
}

// Esc während der Erkennung bricht sie ab (Kontext endet, das eigene tesseract mit ihm); die späte Antwort zählt nicht.
// Hat Esc Vorrang für etwas anderes (Bereichsmodus), bleibt die Erkennung.
func TestOcrEscBrichtAb(t *testing.T) {
	a := neueAttrappe("x")
	a.warten = true
	e := ocrEditor(t, a)
	e.ocrBeginnen()
	e.s.TextBereichGanz()
	e.ocrBereichPruefen()
	<-a.gestartet
	e.s.BeginneTextBereich() // die Sitzung erlaubt den Modus (ocrBeginnen verweigert ihn während der Erkennung)
	if e.ocrEsc() {
		t.Error("Esc im Bereichsmodus bricht die Erkennung ab")
	}
	e.s.BeendeTextBereich()
	if !e.ocrEsc() || e.ocr.laeuft || e.meldung.Text != "Text recognition cancelled." {
		t.Fatalf("Esc: läuft %v, Meldung %q", e.ocr.laeuft, e.meldung.Text)
	}
	ende := time.Now().Add(10 * time.Second)
	for {
		a.mu.Lock()
		ab := a.abgebr
		a.mu.Unlock()
		if ab {
			break
		}
		if time.Now().After(ende) {
			t.Fatal("Kontext der Erkennung endet nicht")
		}
		time.Sleep(time.Millisecond)
	}
	e.ocrAbholen()
	if e.kopierLaeuft || e.meldung.Text != "Text recognition cancelled." {
		t.Errorf("nach Abbruch: Kopieren %v, Meldung %q", e.kopierLaeuft, e.meldung.Text)
	}
	if e.ocrEsc() {
		t.Error("Esc ohne Erkennung verbraucht")
	}
}

// Meldet der Erkenner (schnapp-ocr) Fortschritt, zeigt die Statuszeile ihn; nach dem Abbruch nicht mehr (Karte #242).
func TestOcrFortschrittInDerStatuszeile(t *testing.T) {
	a := neueAttrappe("x")
	a.warten = true
	e := ocrEditor(t, a)
	e.ocrBeginnen()
	e.s.TextBereichGanz()
	e.ocrBereichPruefen()
	<-a.gestartet
	if h := e.ocrHinweis(); h != "Recognizing text…" {
		t.Errorf("ohne Fortschritt: %q", h)
	}
	e.ocrAbbrechen()
	a.mu.Lock()
	a.warten, a.stufe = true, 57
	a.mu.Unlock()
	e.ocrBeginnen()
	e.s.TextBereichGanz()
	e.ocrBereichPruefen()
	<-a.gestartet
	if h := e.ocrHinweis(); h != "Recognizing text… 57 %" {
		t.Errorf("mit Fortschritt: %q", h)
	}
	e.ocrAbbrechen()
	if h := e.ocrHinweis(); h != "" {
		t.Errorf("nach Abbruch: %q", h)
	}
}

// Das Schließen beendet eine laufende Erkennung.
func TestOcrZerstoertBrichtAb(t *testing.T) {
	a := neueAttrappe("x")
	a.warten = true
	e := ocrEditor(t, a)
	e.speichern = func(*document.Dokument) (Gespeichert, error) { return Gespeichert{}, nil }
	e.ocrBeginnen()
	e.s.TextBereichGanz()
	e.ocrBereichPruefen()
	<-a.gestartet
	_ = e.zerstoert(nil)
	ende := time.Now().Add(10 * time.Second)
	for {
		a.mu.Lock()
		ab := a.abgebr
		a.mu.Unlock()
		if ab {
			break
		}
		if time.Now().After(ende) {
			t.Fatal("Schließen beendet die Erkennung nicht")
		}
		time.Sleep(time.Millisecond)
	}
}

// Läuft noch ein Kopieren, bleibt die Antwort der Erkennung liegen, bis es fertig ist; dann geht der Text in die
// Zwischenablage.
func TestOcrWartetAufKopieren(t *testing.T) {
	a := neueAttrappe("abc")
	e := ocrEditor(t, a)
	e.kopierLaeuft, e.kopierStart = true, time.Now()
	e.kopierAntwort = make(chan kopierErgebnis, 1)
	e.ocrBeginnen()
	e.s.TextBereichGanz()
	e.ocrBereichPruefen()
	<-a.gestartet
	time.Sleep(20 * time.Millisecond)
	e.ocrAbholen()
	if !e.ocr.laeuft {
		t.Fatal("Antwort angenommen, obwohl ein Kopieren läuft")
	}
	e.kopierAntwort <- kopierErgebnis{}
	e.kopierenAbholen(layout.Context{})
	warteOcr(t, e)
	warteKopieren(t, e, layout.Context{})
	if e.meldung.Text != "Copied 3 characters" {
		t.Errorf("Meldung %q", e.meldung.Text)
	}
}

// Über das Fenster: Strg+Umschalt+T schaltet den Modus ein (Fadenkreuz-Hinweis), ein Zug auf der Leinwand startet die
// Erkennung, Esc vorher bräche ab; ein grauer Eintrag zeigt seine Erklärung beim Überfahren und bei Tastaturfokus.
func TestOcrUeberRouter(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "1")
	i18n.Setze(nil)
	a := neueAttrappe("Hallo")
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.ocr.gesucht, l.e.ocr.erkenner = true, a
	l.fokusiere(l.e)
	l.taste("T", key.ModShortcut|key.ModShift)
	if !l.e.s.TextBereichLaeuft() || l.e.s.Werkzeug != canvas.WerkzeugAuswahl {
		t.Fatalf("Strg+Umschalt+T: Modus %v, Werkzeug %v", l.e.s.TextBereichLaeuft(), l.e.s.Werkzeug)
	}
	l.taste(key.NameEscape, 0)
	if l.e.s.TextBereichLaeuft() {
		t.Fatal("Esc beendet den Modus nicht")
	}
	l.taste("T", key.ModShortcut|key.ModShift)
	l.zug(image.Pt(2, 2), image.Pt(30, 20))
	l.warteBis(t, "Erkennung und Kopieren", func() bool { return !l.e.ocr.laeuft && !l.e.kopierLaeuft })
	if a.aufrufe != 1 || l.e.meldung.Text != "Copied 5 characters" {
		t.Errorf("%d Aufrufe, Meldung %q", a.aufrufe, l.e.meldung.Text)
	}
	if len(l.e.s.D.Objekte()) != 1 { // nur der Text des Laufs, kein neues Objekt, keine Ansichtsverschiebung
		t.Errorf("%d Objekte", len(l.e.s.D.Objekte()))
	}
	// Ohne Halter (WAYLAND_SOCKET) bietet das Fenster den Text an, unter dem Text-Typ von Gio (nicht als Bild).
	if mime, inhalt, ok := l.r.WriteClipboard(); !ok || mime != "application/text" || string(inhalt) != "Hallo" {
		t.Errorf("Zwischenablage %v %q %q", ok, mime, inhalt)
	}
	if !l.e.fensterHaeltKopie || !l.e.kopieIstText {
		t.Errorf("Fenster hält Kopie %v, Text %v", l.e.fensterHaeltKopie, l.e.kopieIstText)
	}

	// grauer Eintrag: Überfahren zeigt die Erklärung
	l.e.ocr.erkenner, l.e.ocr.sucheErr = nil, ocr.ErrKeinProgramm
	l.zuMenue(aktTextKopieren)
	l.oeffneMitTaste()
	i := l.index(aktTextKopieren)
	l.taste(key.NameEnd, 0)
	if !l.fokus(&l.e.menue.eintraege[i]) {
		t.Fatalf("Ende: grauer Eintrag mit Erklärung nicht erreichbar (%s)", menueFokusName(l))
	}
	if got := l.e.menueErklaerung(l.gtx()); got != "Install tesseract to recognize text." {
		t.Errorf("Erklärung bei Fokus %q", got)
	}
	l.taste(key.NameReturn, 0)
	if !l.e.menue.offen || l.e.s.TextBereichLaeuft() {
		t.Errorf("Enter auf grauem Eintrag: offen %v, Modus %v", l.e.menue.offen, l.e.s.TextBereichLaeuft())
	}
	l.taste(key.NameHome, 0)
	if got := l.e.menueErklaerung(l.gtx()); got != "" {
		t.Errorf("Erklärung ohne Fokus auf dem Eintrag: %q", got)
	}
	r := l.e.menue.rahmen[i]
	l.maus(pointer.Move, r.Min.Add(image.Pt(r.Dx()/2, r.Dy()/2)), 0)
	l.setle()
	if got := l.e.menueErklaerung(l.gtx()); got != "Install tesseract to recognize text." {
		t.Errorf("Erklärung beim Überfahren %q", got)
	}
}

// Esc über die Tastatur bricht eine laufende Erkennung ab (tasten vor Sitzung.Esc); ohne Erkennung wirkt Esc wie sonst.
func TestOcrEscUeberRouter(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "1")
	i18n.Setze(nil)
	a := neueAttrappe("x")
	a.warten = true
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.e.ocr.gesucht, l.e.ocr.erkenner = true, a
	l.fokusiere(l.e)
	l.taste("T", key.ModShortcut|key.ModShift)
	l.taste(key.NameReturn, 0) // ganzes Bild
	<-a.gestartet
	if !l.e.ocr.laeuft || l.e.s.Auswahl == 0 {
		t.Fatalf("Vorbedingung: Erkennung läuft %v, Auswahl %d", l.e.ocr.laeuft, l.e.s.Auswahl)
	}
	l.taste(key.NameEscape, 0)
	if l.e.ocr.laeuft || l.e.s.Auswahl == 0 {
		t.Errorf("Esc: Erkennung läuft %v, Auswahl %d (Esc darf nur die Erkennung abbrechen)", l.e.ocr.laeuft, l.e.s.Auswahl)
	}
	l.taste(key.NameEscape, 0)
	if l.e.s.Auswahl != 0 {
		t.Error("zweites Esc hebt die Auswahl nicht auf")
	}
}

// Der gezogene Bereich ist sichtbar: getönt innen, heller Rand außen, daneben unberührt (zeichneTextBereich).
func TestOcrZeichnetBereich(t *testing.T) {
	e := testEditor(t)
	e.s.Abb = canvas.Abbildung{Zoom: 1}
	e.s.BeginneTextBereich()
	e.s.Druecken(document.Pt(10, 10), canvas.TastePrimaer)
	e.s.Ziehen(document.Pt(50, 40))
	zeichne := func() *image.RGBA {
		return headlessBild(t, image.Pt(80, 60), func(gtx layout.Context) {
			paint.Fill(gtx.Ops, color.NRGBA{A: 0xFF})
			e.zeichneTextBereich(gtx)
		})
	}
	b := zeichne()
	innen, rand, aussen := b.RGBAAt(30, 25), b.RGBAAt(30, 10), b.RGBAAt(70, 50)
	if innen.R < 0x20 || innen.R > 0x60 || innen.B > innen.R {
		t.Errorf("innen %v, erwartet leicht orange getönt", innen)
	}
	if rand.R < 0xC0 || rand.G < 0xC0 || rand.B < 0xC0 {
		t.Errorf("Rand %v, erwartet hell", rand)
	}
	if aussen.R != 0 || aussen.G != 0 || aussen.B != 0 {
		t.Errorf("außen %v, erwartet unberührt", aussen)
	}
	e.s.Loslassen(document.Pt(10, 10)) // Klick: kein Bereich, nichts mehr zu zeichnen
	if c := zeichne().RGBAAt(30, 10); c.R != 0 {
		t.Errorf("ohne Zug gezeichnet: %v", c)
	}
}

// Die Leinwand zeichnet den Bereich des laufenden Zugs (Verdrahtung in leinwand).
func TestOcrLeinwandZeichnetBereich(t *testing.T) {
	e := testEditor(t)
	leinwand := func() *image.RGBA {
		return headlessBild(t, image.Pt(80, 60), func(gtx layout.Context) { e.leinwand(gtx) })
	}
	vorher := leinwand().RGBAAt(30, 25)
	e.s.BeginneTextBereich()
	e.s.Druecken(document.Pt(10, 10), canvas.TastePrimaer)
	e.s.Ziehen(document.Pt(50, 40))
	if nachher := leinwand().RGBAAt(30, 25); nachher == vorher {
		t.Errorf("Leinwand unverändert (%v): kein Bereich gezeichnet", nachher)
	}
}
