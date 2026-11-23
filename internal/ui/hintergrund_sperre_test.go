// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"context"
	"errors"
	"image"
	"image/color"
	"sync/atomic"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// sperrSpeicher ist eine Speicherfunktion, die bis frei wartet; sie zählt Aufrufe und merkt, ob sie zweimal zugleich
// lief. fehler gibt je Aufruf einen Fehler vor (sonst nil).
type sperrSpeicher struct {
	frei    chan struct{}
	beginnt chan *document.Dokument
	laeuft  atomic.Int32
	doppelt atomic.Bool
	aufrufe atomic.Int32
	fehler  []error
}

func neuerSperrSpeicher() *sperrSpeicher {
	return &sperrSpeicher{frei: make(chan struct{}), beginnt: make(chan *document.Dokument, 8)}
}

func (v *sperrSpeicher) speichern(d *document.Dokument) (Gespeichert, error) {
	if v.laeuft.Add(1) > 1 {
		v.doppelt.Store(true)
	}
	defer v.laeuft.Add(-1)
	n := int(v.aufrufe.Add(1))
	select {
	case v.beginnt <- d:
	default:
	}
	<-v.frei
	if n-1 < len(v.fehler) {
		return Gespeichert{}, v.fehler[n-1]
	}
	return Gespeichert{}, nil
}

// warteBis zeichnet Bilder, bis fertig gilt (höchstens 10 s).
func (l *menueLauf) warteBis(t *testing.T, was string, fertig func() bool) {
	t.Helper()
	ende := time.Now().Add(10 * time.Second)
	for !fertig() && time.Now().Before(ende) {
		l.bild()
		time.Sleep(time.Millisecond)
	}
	if !fertig() {
		t.Fatal(was + " endet nicht")
	}
}

func (l *menueLauf) maus(kind pointer.Kind, p image.Point, tasten pointer.Buttons) {
	l.r.Queue(pointer.Event{Kind: kind, Source: pointer.Mouse, Buttons: tasten, Position: f32.Pt(float32(p.X), float32(p.Y))})
}

// bildPt rechnet Bildpixel (Zoom 1) in Fensterpixel um.
func (l *menueLauf) bildPt(p image.Point) image.Point {
	f := l.e.leinwandFlaeche(l.gtx())
	return p.Add(image.Pt(int(l.e.s.Abb.Versatz.X)+f.Min.X, int(l.e.s.Abb.Versatz.Y)+f.Min.Y))
}

func (l *menueLauf) zug(a, b image.Point) {
	a, b = l.bildPt(a), l.bildPt(b)
	l.maus(pointer.Move, a, 0)
	l.setle()
	l.maus(pointer.Press, a, pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Move, b, pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Release, b, 0)
	l.setle()
}

// F-280, F-282, F-285: Während des Speicherns wirken Zeiger (Zug, Doppelklick, Rad), Knöpfe, Menü und Tasten nicht —
// nur Strg+C kopiert. Ein Druck während des Speicherns mit Loslassen danach legt nichts an; danach geht alles wieder.
func TestSperreZeigerKnoepfeMenue(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "1") // kein Halter-Prozess; das Fenster böte selbst an
	i18n.Setze(nil)
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	v := neuerSperrSpeicher()
	l.e.speichern = v.speichern
	l.fokusiere(l.e)
	// Bräche die Sperre des Export-Knopfs, ginge der Dialog auf; ohne Kontext und mit einem Bus, den es nicht gibt,
	// scheitert er still, statt den Testlauf mit einer Panik zu beenden (F-282: die Prüfung sagt dann „Export offen“).
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+t.TempDir()+"/kein-bus")
	l.e.exportCtx = context.Background()
	l.e.wecker = canvas.NeuerWecker(func() {}, weckTakt, weckGrenze)
	defer l.e.wecker.Ende()

	// Kontrolle: Ein Kasten-Zug legt ohne Speichern ein Objekt an.
	l.e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	n0 := len(l.e.s.D.Objekte())
	l.zug(image.Pt(22, 18), image.Pt(34, 27))
	if len(l.e.s.D.Objekte()) != n0+1 {
		t.Fatalf("Kontrolle: Zug legt nichts an (%d → %d)", n0, len(l.e.s.D.Objekte()))
	}
	n1 := len(l.e.s.D.Objekte())
	stand := l.e.s.D.Stand()
	zoom := l.e.s.Abb.Zoom

	l.taste("S", key.ModShortcut)
	if !l.e.s.SpeichernLaeuft() {
		t.Fatal("Speichern läuft nicht")
	}
	l.zug(image.Pt(3, 20), image.Pt(15, 28))
	l.klick(l.bildPt(image.Pt(10, 22)))
	l.klick(l.bildPt(image.Pt(10, 22))) // Doppelklick auf den Text
	sp := l.bildPt(image.Pt(10, 22))
	l.r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(float32(sp.X), float32(sp.Y)), Scroll: f32.Pt(0, 120)})
	l.setle()
	for _, c := range []interface{ Click() }{&l.e.knoepfe.farbe, &l.e.knoepfe.fussFarbe, &l.e.knoepfe.fussZoom} {
		c.Click()
		l.setle()
		// Farb- und Zoomfeld der Fußzeile klappen ihre Liste nicht auf, solange gespeichert wird (F-517, U16).
		if l.e.menue.offen {
			t.Errorf("Feld %T klappt während des Speicherns eine Liste auf (Nr. %d)", c, l.e.menue.welches)
			l.e.menue.offen = false
		}
	}
	for _, w := range l.e.werkzeugKnoepfe() {
		w.c.Click()
		l.setle()
	}
	l.titel().Click()
	l.setle()
	if l.e.menue.offen {
		t.Error("Menü öffnet während des Speicherns")
	}
	l.taste("Z", key.ModShortcut)
	l.taste("E", key.ModShortcut|key.ModShift)
	l.taste(key.NameEscape, 0)
	if len(l.e.s.D.Objekte()) != n1 || l.e.s.D.Stand() != stand || l.e.s.Abb.Zoom != zoom ||
		l.e.s.Werkzeug != canvas.WerkzeugKasten || l.e.s.Zieht() || l.e.exportOffen || l.e.exportAntwort != nil {
		t.Errorf("während des Speicherns: Objekte %d (Soll %d), Stand %d/%d, Zoom %v/%v, Werkzeug %v, zieht %v, Export %v",
			len(l.e.s.D.Objekte()), n1, l.e.s.D.Stand(), stand, l.e.s.Abb.Zoom, zoom, l.e.s.Werkzeug, l.e.s.Zieht(), l.e.exportOffen || l.e.exportAntwort != nil)
	}
	// Strg+C wirkt auch während des Speicherns (F-285).
	l.taste("C", key.ModShortcut)
	if m := l.e.meldung.Text; !l.e.kopierLaeuft && m != i18n.T("Copied to clipboard") || l.e.kopierLaeuft && m != i18n.T("Copying…") {
		t.Errorf("Strg+C während des Speicherns: läuft %v, Meldung %q", l.e.kopierLaeuft, l.e.meldung.Text)
	}
	l.warteBis(t, "Kopieren", func() bool { return !l.e.kopierLaeuft })
	if v.aufrufe.Load() != 1 {
		t.Errorf("Speicheraufrufe %d", v.aufrufe.Load())
	}
	// Druck während des Speicherns, Loslassen danach.
	l.maus(pointer.Move, l.bildPt(image.Pt(3, 3)), 0)
	l.setle()
	l.maus(pointer.Press, l.bildPt(image.Pt(3, 3)), pointer.ButtonPrimary)
	l.setle()
	close(v.frei)
	l.warteBis(t, "Speichern", func() bool { return !l.e.s.SpeichernLaeuft() })
	l.maus(pointer.Move, l.bildPt(image.Pt(15, 12)), pointer.ButtonPrimary)
	l.setle()
	l.maus(pointer.Release, l.bildPt(image.Pt(15, 12)), 0)
	l.setle()
	if len(l.e.s.D.Objekte()) != n1 || l.e.s.Zieht() {
		t.Errorf("Zug über das Ende des Speicherns: Objekte %d (Soll %d), zieht %v", len(l.e.s.D.Objekte()), n1, l.e.s.Zieht())
	}
	if l.e.s.Geaendert() {
		t.Error("nach dem Speichern geändert")
	}
	if !l.fokus(l.e) {
		t.Error("Fokus weg")
	}
	l.zug(image.Pt(3, 20), image.Pt(15, 28))
	if len(l.e.s.D.Objekte()) != n1+1 {
		t.Errorf("Zug nach dem Speichern legt nichts an: %d", len(l.e.s.D.Objekte()))
	}
	if v.doppelt.Load() {
		t.Error("Speicherfunktion lief zweimal gleichzeitig")
	}
}

// F-282: Offenes Menü, dann beginnt ein Speichern (Schließen-Anfrage): Einträge wirken nicht, per Klick wie per Enter.
func TestMenueOffenWaehrendSpeichern(t *testing.T) {
	i18n.Setze(nil)
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	v := neuerSperrSpeicher()
	l.e.speichern = v.speichern
	defer close(v.frei)
	l.oeffneMitTaste()
	if !l.e.menue.offen {
		t.Fatal("Menü nicht offen")
	}
	stand := l.e.s.D.Stand()
	z, abzug := l.e.s.SchliessenAnfrage()
	if z != canvas.SchlussWarten || abzug == nil {
		t.Fatal("kein Speichern")
	}
	l.e.speichernStarten(abzug)
	for _, a := range []aktion{aktRueckgaengig, aktLoeschen, aktGanzesBild} {
		l.e.menue.eintraege[l.index(a)].Click()
		l.setle()
	}
	l.fokusiere(&l.e.menue.eintraege[l.index(aktRueckgaengig)])
	l.taste(key.NameReturn, 0)
	if l.e.s.D.Stand() != stand || l.inhalt() != "Hallo" {
		t.Errorf("Menü wirkt während des Speicherns: Stand %d/%d, Inhalt %q", l.e.s.D.Stand(), stand, l.inhalt())
	}
	// „Schließen“ ist eine Schließen-Anfrage wie der Fensterknopf und geht weiter (F-299, ADR-0020 Nachtrag 6).
	var anfragen int
	l.e.perform = func(a system.Action) {
		if a == system.ActionClose {
			anfragen++
		}
	}
	l.e.menue.eintraege[l.index(aktSchliessen)].Click()
	l.setle()
	if anfragen != 1 {
		t.Errorf("Menüeintrag „Schließen“ während des Speicherns: %d Schließen-Anfragen, erwartet 1", anfragen)
	}
}

// ADR-0022 Punkt 3 über die Oberfläche: Schwärzen, Strg+S, Strg+Z während des Speicherns, danach Strg+Z, Knopf
// Rückgängig und Strg+Y — die Schwärzung bleibt, der Verlauf ist leer.
func TestSchwaerzenSpeichernRueckgaengigUI(t *testing.T) {
	i18n.Setze(nil)
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	v := neuerSperrSpeicher()
	l.e.speichern = v.speichern
	l.fokusiere(l.e)
	if err := l.e.s.D.Schwaerzen(image.Rect(0, 0, 10, 10), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	l.taste("S", key.ModShortcut)
	l.taste("Z", key.ModShortcut) // während des Speicherns
	abzug := <-v.beginnt
	if abzug.OriginalBild().NRGBAAt(3, 3) != (color.NRGBA{A: 255}) {
		t.Error("Abzug nicht geschwärzt")
	}
	close(v.frei)
	l.warteBis(t, "Speichern", func() bool { return !l.e.s.SpeichernLaeuft() })
	l.taste("Z", key.ModShortcut)
	l.zuMenue(aktRueckgaengig) // Menüeintrag Rückgängig (ADR-0031: kein Leistenknopf mehr)
	l.titel().Click()
	l.setle()
	l.e.menue.eintraege[l.index(aktRueckgaengig)].Click()
	l.setle()
	l.taste("Y", key.ModShortcut)
	if l.e.s.D.OriginalBild().NRGBAAt(3, 3) != (color.NRGBA{A: 255}) || l.e.s.D.KannRueckgaengig() || l.e.s.Geaendert() {
		t.Errorf("Schwärzung zurückgeholt oder Verlauf nicht leer: %v, rück %v", l.e.s.D.OriginalBild().NRGBAAt(3, 3), l.e.s.D.KannRueckgaengig())
	}
}

// F-280: Knopf Speichern und Knopf Farbe im selben Bild (zwei Klicks, die sich während eines hängenden Bilds gesammelt
// haben). Die Farbe färbt den ausgewählten Text nicht mehr um, und nach dem Speichern holt Rückgängig das Geschwärzte
// nicht zurück.
func TestSpeichernUndFarbeImSelbenBild(t *testing.T) {
	i18n.Setze(nil)
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	v := neuerSperrSpeicher()
	l.e.speichern = v.speichern
	l.fokusiere(l.e)
	if err := l.e.s.D.Schwaerzen(image.Rect(0, 0, 10, 10), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	stand := l.e.s.D.Stand()
	l.zuMenue(aktSpeichern) // Menüeintrag Speichern und Farbfeld im selben Bild (ADR-0031)
	l.titel().Click()
	l.setle()
	l.e.menue.eintraege[l.index(aktSpeichern)].Click()
	l.e.knoepfe.farbe.Click()
	l.bild()
	if !l.e.s.SpeichernLaeuft() {
		t.Fatal("Speichern läuft nicht")
	}
	if l.e.s.D.Stand() != stand {
		t.Error("Farbe-Knopf im selben Bild ändert das Dokument während des Speicherns")
	}
	close(v.frei)
	l.warteBis(t, "Speichern", func() bool { return !l.e.s.SpeichernLaeuft() })
	l.taste("Z", key.ModShortcut)
	l.taste("Z", key.ModShortcut)
	if l.e.s.D.OriginalBild().NRGBAAt(3, 3) != (color.NRGBA{A: 255}) || l.e.s.D.KannRueckgaengig() {
		t.Errorf("nach gelungenem Speichern mit Schwärzung holt Rückgängig das Geschwärzte zurück (Pixel %v)",
			l.e.s.D.OriginalBild().NRGBAAt(3, 3))
	}
}

// speichernAbwarten (Fenster zerstört): wartet auf die Antwort; nach Signal und Fehler ein weiterer Versuch, nie zwei
// zugleich, und keine Antwort bleibt im Kanal liegen.
func TestSpeichernAbwarten(t *testing.T) {
	for _, fall := range []struct {
		name    string
		signal  bool
		fehler  []error
		aufrufe int32
		schluss bool
	}{
		{"gelungen", false, nil, 1, false},
		{"Fehler ohne Signal", false, []error{errors.New("x")}, 1, false},
		{"Signal, erst Fehler", true, []error{errors.New("x")}, 2, false},
		{"Signal, zweimal Fehler", true, []error{errors.New("x"), errors.New("y")}, 2, true},
	} {
		e := testEditor(t)
		if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5))}); err != nil {
			t.Fatal(err)
		}
		v := neuerSperrSpeicher()
		v.fehler = fall.fehler
		e.speichern = v.speichern
		e.speichernJetzt()
		if fall.signal {
			if zu, a := e.s.SignalAnfrage(); zu || a != nil {
				t.Fatal("Signal während Speichern")
			}
		}
		go func() { time.Sleep(20 * time.Millisecond); close(v.frei) }()
		e.speichernAbwarten()
		if e.s.SpeichernLaeuft() || v.aufrufe.Load() != fall.aufrufe || v.doppelt.Load() || (e.schlussErr != nil) != fall.schluss {
			t.Errorf("%s: läuft %v, Aufrufe %d, doppelt %v, schlussErr %v", fall.name, e.s.SpeichernLaeuft(), v.aufrufe.Load(), v.doppelt.Load(), e.schlussErr)
		}
		select {
		case <-e.speicherAntwort:
			t.Errorf("%s: Antwort liegt noch im Kanal", fall.name)
		default:
		}
	}
}

// F-285: „Kopieren …“ beim Start; wird bis zur Antwort bearbeitet (hier geschwärzt), sagt die Meldung, dass der Stand
// beim Kopieren in der Zwischenablage liegt. Ohne Änderung bleibt es bei „Copied to clipboard“.
func TestKopierenMeldungNachAenderung(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "1")
	i18n.Setze(nil)
	l := neuerMenueLauf(t, image.Pt(1200, 800))
	l.fokusiere(l.e)
	l.e.kopieren(l.gtx())
	if l.e.meldung.Text != i18n.T("Copying…") {
		t.Errorf("Meldung beim Start: %q", l.e.meldung.Text)
	}
	if err := l.e.s.D.Schwaerzen(image.Rect(0, 0, 10, 10), document.Schwaerzung{Farbe: color.NRGBA{A: 255}}); err != nil {
		t.Fatal(err)
	}
	l.warteBis(t, "Kopieren", func() bool { return !l.e.kopierLaeuft })
	if l.e.meldung.Text != i18n.T("Copied to clipboard as it was when copying began; later changes are not included.") {
		t.Errorf("Meldung nach Änderung: %q", l.e.meldung.Text)
	}
	l.e.kopieren(l.gtx())
	l.warteBis(t, "Kopieren", func() bool { return !l.e.kopierLaeuft })
	if l.e.meldung.Text != i18n.T("Copied to clipboard") {
		t.Errorf("Meldung ohne Änderung: %q", l.e.meldung.Text)
	}
}

// F-281: Eine Schließen-Anfrage während des Kopierens hält das Fenster offen, bis die Antwort angenommen ist; dann
// wird erneut angefragt und geschlossen. Ist die Frist schon um, schließt sie gleich.
func TestSchliessenWartetAufKopieren(t *testing.T) {
	t.Setenv("WAYLAND_SOCKET", "1")
	e := testEditor(t)
	gtx := layout.Context{}
	e.kopieren(gtx)
	if e.schliessen() {
		t.Fatal("Schließen während des Kopierens schließt sofort")
	}
	if e.nachKopieSchliessen() {
		t.Fatal("erneute Anfrage vor der Antwort")
	}
	warteKopieren(t, e, gtx)
	if !e.nachKopieSchliessen() || e.nachKopieSchliessen() {
		t.Fatal("nach der Antwort keine (oder doppelte) erneute Anfrage")
	}
	// Ohne Halter (WAYLAND_SOCKET) hält nur das Fenster das Bild: das erste Schließen warnt (F-295), das zweite schließt.
	if e.schliessen() || e.meldung.Text != i18n.T("The clipboard holds the image only while this window is open. Close again to discard it.") {
		t.Errorf("Rückfall ohne Warnung vor dem Schließen: Meldung %q", e.meldung.Text)
	}
	if !e.schliessen() {
		t.Error("zweite Anfrage nach der Warnung schließt nicht")
	}

	// Frist um: Die Anfrage schließt, die erneute Anfrage entfällt.
	e = testEditor(t)
	e.kopierLaeuft, e.kopierStart, e.kopierAntwort = true, time.Now().Add(-2*kopierFrist), make(chan kopierErgebnis, 1)
	if !e.schliessen() || e.nachKopieSchliessen() {
		t.Error("nach der Frist hält das Kopieren das Fenster offen")
	}
}

// F-281: Beim Zerstören wartet der Editor auf ein laufendes Kopieren (der Halter startet in der Goroutine), höchstens
// bis zur Frist.
func TestZerstoerenWartetAufKopieren(t *testing.T) {
	alt := kopierFrist
	t.Cleanup(func() { kopierFrist = alt })
	e := testEditor(t)
	ch := make(chan kopierErgebnis, 1)
	e.kopierLaeuft, e.kopierStart, e.kopierAntwort = true, time.Now(), ch
	go func() { time.Sleep(30 * time.Millisecond); ch <- kopierErgebnis{} }()
	if err := e.zerstoert(nil); err != nil || e.kopierLaeuft {
		t.Errorf("Antwort nicht abgewartet: %v, läuft %v", err, e.kopierLaeuft)
	}

	kopierFrist = 40 * time.Millisecond
	e = testEditor(t)
	e.kopierLaeuft, e.kopierStart, e.kopierAntwort = true, time.Now(), make(chan kopierErgebnis) // antwortet nie
	start := time.Now()
	if err := e.zerstoert(nil); err != nil {
		t.Error(err)
	}
	if d := time.Since(start); d < 30*time.Millisecond || d > 5*time.Second {
		t.Errorf("Frist nicht eingehalten: %v", d)
	}
}

// F-283, ADR-0020 Nachtrag 7: Ein Speichern, das nie zurückkehrt, hält das Fenster nicht für immer. Vor der Frist hält
// die Schließen-Anfrage offen und ein weiteres Signal wirkt nicht; nach der Frist sagt die Fußzeile es, und die
// nächste Anfrage bzw. ein weiteres Signal schließt ohne Antwort. Zerstören wartet dann nicht und speichert nicht noch
// einmal daneben; Editor liefert ErrSpeichernHaengt.
func TestHaengendesSpeichernSchliessen(t *testing.T) {
	alt := speichernFrist
	t.Cleanup(func() { speichernFrist = alt })
	speichernFrist = 50 * time.Millisecond

	for _, weg := range []string{"Schließen", "Signal"} {
		e := testEditor(t)
		e.weitereSignale = make(chan struct{}, 1)
		if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5))}); err != nil {
			t.Fatal(err)
		}
		v := neuerSperrSpeicher() // frei wird erst am Ende geschlossen: das Speichern hängt
		t.Cleanup(func() { close(v.frei) })
		e.speichern = v.speichern
		e.speichernJetzt()
		if e.schliessen() {
			t.Fatalf("%s: schließt sofort", weg)
		}
		e.weitereSignale <- struct{}{}
		if e.weiteresSignalSchliesst() || e.speichernHaengtMelden() {
			t.Fatalf("%s: vor der Frist aufgegeben oder gemeldet", weg)
		}
		time.Sleep(60 * time.Millisecond)
		if !e.speichernHaengtMelden() || e.speichernHaengtMelden() || !e.meldung.Fehler {
			t.Errorf("%s: keine (oder doppelte) Meldung: %q", weg, e.meldung.Text)
		}
		switch weg {
		case "Schließen":
			if !e.schliessen() {
				t.Fatalf("%s: zweite Anfrage nach der Frist schließt nicht", weg)
			}
		case "Signal":
			e.weitereSignale <- struct{}{}
			if !e.weiteresSignalSchliesst() {
				t.Fatalf("%s: weiteres Signal nach der Frist schließt nicht", weg)
			}
		}
		fertig := make(chan error, 1)
		go func() { fertig <- e.zerstoert(nil) }()
		select {
		case err := <-fertig:
			if !errors.Is(err, ErrSpeichernHaengt) {
				t.Errorf("%s: Rückgabe %v", weg, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: Zerstören wartet auf das hängende Speichern", weg)
		}
		if v.aufrufe.Load() != 1 || v.doppelt.Load() {
			t.Errorf("%s: %d Speicheraufrufe, doppelt %v", weg, v.aufrufe.Load(), v.doppelt.Load())
		}
	}
}

// ADR-0020 Nachtrag 7: Beim Zerstören ohne Anfrage wartet der Editor höchstens zerstoerenFrist auf ein Speichern.
func TestZerstoerenMitFrist(t *testing.T) {
	alt := zerstoerenFrist
	t.Cleanup(func() { zerstoerenFrist = alt })
	zerstoerenFrist = 40 * time.Millisecond
	e := testEditor(t)
	if _, err := e.s.D.Hinzufuegen(document.Kasten{Bereich: document.R(document.Pt(1, 1), document.Pt(5, 5))}); err != nil {
		t.Fatal(err)
	}
	v := neuerSperrSpeicher()
	t.Cleanup(func() { close(v.frei) })
	e.speichern = v.speichern
	e.speichernJetzt()
	if err := e.zerstoert(nil); !errors.Is(err, ErrSpeichernHaengt) || v.aufrufe.Load() != 1 {
		t.Errorf("Rückgabe %v, Aufrufe %d", err, v.aufrufe.Load())
	}
}
