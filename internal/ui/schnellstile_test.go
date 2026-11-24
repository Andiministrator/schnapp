// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// schnellLauf: Editor mit einem Kasten, eigener Konfigurationsordner, Werkzeug Kasten ohne Auswahl; die Leiste zeigt
// die Schnellstile des Kastens (ADR-0037). Das Fenster ist hoch genug, damit die Leiste nicht rollt: Lagen in
// Inhaltskoordinaten sind dann Fensterkoordinaten.
func schnellLauf(t *testing.T) (*menueLauf, int) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	l, k := ebenenLauf(t)
	l.e.s.Auswahl = 0
	l.e.s.SetzeWerkzeug(canvas.WerkzeugKasten)
	l.setle()
	if !l.e.schnell.gezeigt || len(l.e.schnell.liste) != 7 {
		t.Fatalf("Kontrolle: Schnellstile gezeigt %v, %d Kacheln, Soll 7 für den Kasten", l.e.schnell.gezeigt, len(l.e.schnell.liste))
	}
	return l, k
}

// rechtsklick drückt und löst die rechte Taste an p.
func (l *menueLauf) rechtsklick(p image.Point) {
	l.maus(pointer.Move, p, 0)
	l.setle()
	l.maus(pointer.Press, p, pointer.ButtonSecondary)
	l.maus(pointer.Release, p, 0)
	l.setle()
}

func (l *menueLauf) kachel(j int) image.Point { return mitte(l.e.schnell.rahmen[j]) }

func kastenVon(t *testing.T, l *menueLauf, id int) document.Kasten {
	t.Helper()
	o, ok := l.e.s.D.Objekt(id)
	if !ok {
		t.Fatal("Objekt fehlt")
	}
	return o.(document.Kasten)
}

// Ein Klick auf eine Kachel macht den Stil zum Stand des Werkzeugs; die Kachel bekommt den orangen Rand (ADR-0037
// Punkte 2 und 3).
func TestSchnellstileKlickWirdStandDesWerkzeugs(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	if k.plusRahmen.Empty() || k.plusAn { // der Stand „Rot“ ist schon eine Kachel: „+“ ist grau
		t.Fatalf("Kontrolle: Knopf + %v, wirkt %v", k.plusRahmen, k.plusAn)
	}
	stand := l.e.s.D.Stand()
	l.klick(l.kachel(4)) // Rot, dick
	if b := l.e.s.Breite(); b != 8 {
		t.Errorf("Strichstärke nach „Rot, dick“ %v, Soll 8", b)
	}
	if !l.e.s.SchnellstilPasst(k.liste[4]) || l.e.s.SchnellstilPasst(k.liste[0]) {
		t.Error("nach dem Klick passt nicht genau die gewählte Kachel")
	}
	if l.e.s.D.Stand() != stand {
		t.Error("Klick ohne Objekt änderte das Dokument")
	}
}

// Mit gewähltem Objekt ändert der Stil das Objekt in einem Schritt (ADR-0037 Punkt 3).
func TestSchnellstileKlickAendertGewaehltesObjekt(t *testing.T) {
	l, id := schnellLauf(t)
	l.e.s.Auswahl = id
	l.setle()
	vor := kastenVon(t, l, id)
	l.klick(l.kachel(5)) // Weiß auf Schwarz
	nach := kastenVon(t, l, id)
	if nach.Strich.Farbe == vor.Strich.Farbe || nach.Fuellung.A == 0 {
		t.Fatalf("Kasten nach dem Klick: Strich %v, Füllung %v", nach.Strich.Farbe, nach.Fuellung)
	}
	l.e.s.Rueckgaengig()
	if got := kastenVon(t, l, id); got != vor {
		t.Errorf("ein Rückgängig stellt den Kasten nicht her: %v, Soll %v", got, vor)
	}
}

// „+“ speichert den Stand als eigenen Stil in styles.ini, ist danach grau (der Stand ist schon eine Kachel); Rechtsklick
// entfernt ihn wieder (ADR-0037 Punkt 5).
func TestSchnellstilePlusUndRechtsklick(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	l.e.s.SetzeBreite(6) // keine eingebaute Kachel hat 6
	l.setle()
	if !k.plusAn {
		t.Fatal("„+“ ist bei neuem Stand grau")
	}
	l.klick(mitte(k.plusRahmen))
	if len(k.liste) != 8 || !k.liste[7].Eigen() {
		t.Fatalf("nach „+“ %d Kacheln, Soll 8 mit eigener am Ende", len(k.liste))
	}
	if l.e.meldung.Text != "Style saved" {
		t.Errorf("Meldung %q, Soll „Style saved“", l.e.meldung.Text)
	}
	pfad, _ := settings.StilePfad()
	if daten, err := os.ReadFile(pfad); err != nil || !strings.Contains(string(daten), "[box]") {
		t.Errorf("styles.ini nach „+“: %v, %q", err, daten)
	}
	if k.plusAn {
		t.Error("„+“ ist nach dem Speichern nicht grau")
	}
	l.klick(mitte(k.plusRahmen))
	if len(k.liste) != 8 {
		t.Errorf("grauer Knopf legte eine Kachel an (%d)", len(k.liste))
	}
	l.klick(l.kachel(7)) // wählt den Stil: der Stand bleibt
	// Rechtsklick auf eine eingebaute Kachel tut nichts, auf die eigene entfernt sie.
	l.rechtsklick(l.kachel(1))
	if len(k.liste) != 8 {
		t.Fatalf("Rechtsklick auf eine eingebaute Kachel: %d Kacheln", len(k.liste))
	}
	l.rechtsklick(l.kachel(7))
	if len(k.liste) != 7 || l.e.meldung.Text != "Style removed" {
		t.Errorf("nach Rechtsklick %d Kacheln, Meldung %q", len(k.liste), l.e.meldung.Text)
	}
	if daten, _ := os.ReadFile(pfad); strings.Contains(string(daten), "[box]") {
		t.Errorf("styles.ini enthält den entfernten Stil: %q", daten)
	}
}

// Ein Linksklick entfernt nichts, ein Rechtsklick wendet nichts an.
func TestSchnellstileRechtsklickWendetNichtAn(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	l.rechtsklick(l.kachel(4))
	if l.e.s.Breite() == 8 {
		t.Error("Rechtsklick wendete den Stil an")
	}
	if len(k.liste) != 7 {
		t.Errorf("%d Kacheln", len(k.liste))
	}
}

// Tastatur (ADR-0037 Punkt 7): Pfeile wandern, Leertaste und Enter wenden an, Entf entfernt einen eigenen Stil und löscht
// nie das gewählte Objekt.
func TestSchnellstileTastatur(t *testing.T) {
	l, id := schnellLauf(t)
	k := &l.e.schnell
	l.fokusiere(&k.kacheln[2])
	l.taste(key.NameRightArrow, 0)
	if !l.fokus(&k.kacheln[3]) {
		t.Fatalf("Pfeil rechts: Fokus %s", menueFokusName(l))
	}
	l.taste(key.NameLeftArrow, 0)
	l.taste(key.NameLeftArrow, 0)
	l.taste(key.NameLeftArrow, 0) // ganz links bleibt er
	if !l.fokus(&k.kacheln[0]) {
		t.Fatalf("Pfeil links bis zum Anschlag: Fokus %s", menueFokusName(l))
	}
	l.taste(key.NameDownArrow, 0) // eine Reihe tiefer: Stelle 6, die letzte Kachel
	if !l.fokus(&k.kacheln[6]) {
		t.Fatalf("Pfeil runter: Fokus %s", menueFokusName(l))
	}
	l.taste(key.NameRightArrow, 0)
	if !l.fokus(&k.plus) {
		t.Fatalf("Pfeil rechts hinter der letzten Kachel: Fokus %s, Soll „+“", menueFokusName(l))
	}
	l.taste(key.NameHome, 0)
	l.taste(key.NameRightArrow, 0)
	l.taste(key.NameReturn, 0)
	if !l.e.s.SchnellstilPasst(k.liste[1]) {
		t.Error("Enter wendete die Kachel mit Fokus nicht an")
	}
	l.taste(key.NameRightArrow, 0)
	l.taste(key.NameSpace, 0)
	if !l.e.s.SchnellstilPasst(k.liste[2]) {
		t.Error("Leertaste wendete die Kachel mit Fokus nicht an")
	}
	// Entf auf einer eingebauten Kachel: nichts geschieht, das Objekt bleibt.
	l.e.s.Auswahl = id
	l.setle()
	l.fokusiere(&k.kacheln[2])
	l.taste(key.NameDeleteForward, 0)
	if _, ok := l.e.s.D.Objekt(id); !ok {
		t.Fatal("Entf an einer Kachel löschte das gewählte Objekt")
	}
	if len(k.liste) != 7 {
		t.Fatalf("Entf entfernte eine eingebaute Kachel (%d)", len(k.liste))
	}
	// Eigenen Stil anlegen und per Entf entfernen; der Fokus geht auf „+“.
	l.e.s.Auswahl = 0
	l.e.s.SetzeBreite(6)
	l.setle()
	l.fokusiere(&k.plus)
	l.taste(key.NameReturn, 0)
	if len(k.liste) != 8 {
		t.Fatalf("Enter auf „+“ legte keinen Stil an (%d)", len(k.liste))
	}
	l.fokusiere(&k.kacheln[7])
	l.taste(key.NameDeleteForward, 0)
	if len(k.liste) != 7 || l.e.meldung.Text != "Style removed" {
		t.Errorf("Entf auf der eigenen Kachel: %d Kacheln, Meldung %q", len(k.liste), l.e.meldung.Text)
	}
	if !l.fokus(&k.plus) {
		t.Errorf("nach dem Entfernen der letzten Kachel Fokus %s, Soll „+“", menueFokusName(l))
	}
}

// Die Statuszeile nennt die Kachel mit Fokus (ADR-0037 Punkt 2).
func TestSchnellstileHinweisInDerStatuszeile(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	l.fokusiere(&k.kacheln[2])
	if h := l.e.hinweis(); h != "Green" {
		t.Errorf("Hinweis bei Fokus auf der dritten Kachel %q, Soll „Green“", h)
	}
	l.e.s.SetzeBreite(6)
	l.fokusiere(&k.plus)
	if h := l.e.hinweis(); !strings.Contains(h, "custom style") {
		t.Errorf("Hinweis am Knopf „+“: %q", h)
	}
	l.fokusiere(l.e)
	if h := l.e.hinweis(); strings.Contains(h, "Green") || strings.Contains(h, "custom style") {
		t.Errorf("Hinweis nach Fokuswechsel noch von der Kachel: %q", h)
	}
}

// Die Überschrift klappt die Kacheln ein und aus; eingeklappt gibt es keine Kachel zum Klicken.
func TestSchnellstileUeberschriftKlapptEin(t *testing.T) {
	l, _ := schnellLauf(t)
	l.fokusiere(&l.e.schnell.kopf)
	l.taste(key.NameReturn, 0)
	if l.e.schnell.gezeigt || !l.e.schnell.zu {
		t.Fatal("die Überschrift klappt nicht ein")
	}
	l.fokusiere(&l.e.schnell.kopf)
	l.taste(key.NameReturn, 0)
	if !l.e.schnell.gezeigt {
		t.Fatal("die Überschrift klappt nicht wieder aus")
	}
}

// Nur bei Werkzeug oder Objekt mit Stilen steht der Abschnitt da; bei gewählter Sprechblase zeigt er die der Blase.
func TestSchnellstileNurFuerArtenMitStilen(t *testing.T) {
	l, _ := schnellLauf(t)
	l.e.s.SetzeWerkzeug(canvas.WerkzeugSchwaerzen)
	l.setle()
	if l.e.schnell.gezeigt {
		t.Error("Schwärzen zeigt Schnellstile")
	}
	l.e.s.SetzeWerkzeug(canvas.WerkzeugAuswahl)
	l.setle()
	if l.e.schnell.gezeigt {
		t.Error("Auswählen ohne Objekt zeigt Schnellstile")
	}
	id, err := l.e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(30, 40), Inhalt: "Blase", Schriftgrad: 12,
		Farbe: canvas.Schwaerzfarbe, Spitze: &document.Punkt{X: 10, Y: 10}})
	if err != nil {
		t.Fatal(err)
	}
	l.e.s.Auswahl = id
	l.setle()
	if !l.e.schnell.gezeigt || l.e.schnell.art != canvas.SchnellSprechblase {
		t.Errorf("gewählte Sprechblase: gezeigt %v, Art %v", l.e.schnell.gezeigt, l.e.schnell.art)
	}
}

// Beim Öffnen liest der Editor styles.ini; eine Warnung geht auf die Fehlerausgabe, nicht in die Statuszeile.
func TestSchnellstileLadenBeimOeffnen(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := settings.SchreibeStile([]settings.Stil{{Art: settings.ArtKasten, Farbe: color.NRGBA{R: 1, G: 2, B: 3, A: 255},
		Breite: 5}}); err != nil {
		t.Fatal(err)
	}
	e := testEditor(t)
	e.stileLaden()
	if n := len(e.s.EigeneSchnellstile()); n != 1 {
		t.Errorf("%d eigene Stile nach dem Öffnen, Soll 1", n)
	}
	var aus strings.Builder
	alt := fehlerAusgabe
	fehlerAusgabe = &aus
	t.Cleanup(func() { fehlerAusgabe = alt })
	pfad, _ := settings.StilePfad()
	if err := os.WriteFile(pfad, []byte("[box]\ncolor = nonsense\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	e.stileLaden()
	if aus.Len() == 0 || len(e.s.EigeneSchnellstile()) != 0 {
		t.Errorf("ungültige Datei: Warnung %q, %d Stile", aus.String(), len(e.s.EigeneSchnellstile()))
	}
}

// Eine fehlende Schrift macht die Kachel kenntlich, solange die Suche fertig ist; „Go“ und leer fehlen nie.
func TestSchnellstileFehlendeSchrift(t *testing.T) {
	fonts.Suche()
	if schriftFehlt(canvas.Schnellstil{Art: canvas.SchnellText}) || schriftFehlt(canvas.Schnellstil{Schrift: fonts.Go}) {
		t.Error("Go gilt als fehlend")
	}
	st := canvas.Schnellstil{Art: canvas.SchnellText, Schrift: "Gibt Es Auf Keinem Rechner"}
	if !schriftFehlt(st) {
		t.Fatal("unbekannte Familie gilt nicht als fehlend")
	}
	if h := schnellName(st); !strings.Contains(h, "Gibt Es Auf Keinem Rechner") {
		t.Errorf("Hinweis nennt die Familie nicht: %q", h)
	}
	// Während die Suche läuft (hier: noch nicht gelaufen), gilt keine Familie als fehlend.
	fonts.SucheZuruecksetzen()
	t.Cleanup(fonts.Suche)
	if schriftFehlt(st) {
		t.Error("vor dem Ende der Suche gilt eine Familie als fehlend")
	}
	fonts.Suche()
	if !schriftFehlt(st) {
		t.Error("nach der Suche gilt die unbekannte Familie nicht als fehlend")
	}
}

// Der Editor liest styles.ini schon beim Anlegen (neuerEditor), nicht erst auf Anforderung.
func TestSchnellstileNeuerEditorLiestDieDatei(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := settings.SchreibeStile([]settings.Stil{{Art: settings.ArtLupe, Farbe: color.NRGBA{R: 9, G: 8, B: 7, A: 255},
		Breite: 5}}); err != nil {
		t.Fatal(err)
	}
	d, err := document.Neu(image.NewNRGBA(image.Rect(0, 0, 40, 30)))
	if err != nil {
		t.Fatal(err)
	}
	e := neuerEditor("datei.png", d, nil, nil)
	if n := len(e.s.Schnellstile(canvas.SchnellLupe)); n != 4 {
		t.Errorf("%d Lupen-Stile nach neuerEditor, Soll 3 eingebaute und 1 eigener", n)
	}
}

// Der Tastaturfokus geht vor dem Zeiger: Fokus auf einer Kachel, Zeiger auf einer anderen, die Statuszeile nennt die mit
// dem Fokus.
func TestSchnellstileHinweisFokusVorZeiger(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	l.fokusiere(&k.kacheln[1])
	l.maus(pointer.Move, l.kachel(2), 0)
	l.setle()
	if h := l.e.hinweis(); h != "Blue" {
		t.Errorf("Fokus auf Blau, Zeiger auf Grün: Hinweis %q, Soll „Blue“", h)
	}
	l.fokusiere(&k.kacheln[3])
	l.maus(pointer.Move, l.kachel(2), 0)
	l.setle()
	if h := l.e.hinweis(); h != "Black" {
		t.Errorf("Fokus auf Schwarz, Zeiger auf der früheren Kachel Grün: Hinweis %q, Soll „Black“", h)
	}
}

// Gezeichnet (Gio ohne Fenster): Der Stil, der zum Stand passt, trägt den orangen Rand, die anderen nicht; eigene Kacheln
// tragen die graue Ecke; die Mini-Darstellung hat die Farbe des Stils.
func TestSchnellstileZeichnen(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	l.e.s.SetzeBreite(6)
	l.klick(mitte(k.plusRahmen)) // eigener Stil, passt zum Stand
	l.setle()
	flaeche := image.Pt(1200, 1200)
	bild := headlessBild(t, flaeche, func(gtx layout.Context) { l.e.zeichneSeitenleiste(gtx) })
	zaehle := func(r image.Rectangle, c color.NRGBA) int {
		n := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				p := bild.RGBAAt(x, y)
				if p.R == c.R && p.G == c.G && p.B == c.B && p.A == 255 {
					n++
				}
			}
		}
		return n
	}
	if n := zaehle(k.rahmen[7], farbeAkzent); n < 100 {
		t.Errorf("passender Stil: %d orange Pixel, Soll ein Rand", n)
	}
	if n := zaehle(k.rahmen[0], farbeAkzent); n != 0 {
		t.Errorf("anderer Stil: %d orange Pixel, Soll 0", n)
	}
	if n := zaehle(k.rahmen[0], canvas.Beschriftungsfarben[0]); n < 50 {
		t.Errorf("Kachel „Rot“: %d rote Pixel, Soll ein Kasten", n)
	}
	ecke := image.Rect(k.rahmen[7].Max.X-8, k.rahmen[7].Min.Y, k.rahmen[7].Max.X, k.rahmen[7].Min.Y+4)
	if n := zaehle(ecke, farbeNebentext); n == 0 {
		t.Error("eigene Kachel ohne graue Ecke")
	}
	if n := zaehle(k.rahmen[3], farbeNebentext); n != 0 {
		t.Errorf("eingebaute Kachel mit grauer Ecke (%d Pixel)", n)
	}
}

// Pfeil hoch geht eine Reihe zurück, Ende springt auf „+“ (ADR-0037 Punkt 7).
func TestSchnellstileHochUndEnde(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	l.fokusiere(&k.kacheln[6])
	l.taste(key.NameUpArrow, 0)
	if !l.fokus(&k.kacheln[0]) {
		t.Fatalf("Pfeil hoch von der siebten Kachel: Fokus %s, Soll die erste", menueFokusName(l))
	}
	l.taste(key.NameEnd, 0)
	if !l.fokus(&k.plus) {
		t.Fatalf("Ende: Fokus %s, Soll „+“", menueFokusName(l))
	}
}

// Ein grauer Knopf „+“ tut nichts und meldet nichts; zwei eigene Stile stehen beide in styles.ini.
func TestSchnellstileMehrereEigeneUndGrauesPlus(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	l.e.s.SetzeBreite(6)
	l.setle()
	l.klick(mitte(k.plusRahmen))
	l.e.s.SetzeBreite(7)
	l.setle()
	l.klick(mitte(k.plusRahmen))
	pfad, _ := settings.StilePfad()
	if daten, _ := os.ReadFile(pfad); strings.Count(string(daten), "[box]") != 2 {
		t.Errorf("styles.ini mit zwei eigenen Stilen: %q", daten)
	}
	l.e.melde("", false)
	l.klick(mitte(k.plusRahmen)) // der Stand ist schon eine Kachel
	if l.e.meldung.Text != "" || len(k.liste) != 9 {
		t.Errorf("grauer Knopf: Meldung %q, %d Kacheln", l.e.meldung.Text, len(k.liste))
	}
}

// Scheitert das Schreiben von styles.ini, steht der Fehler in der Fußzeile; der Stil gilt in der Sitzung weiter.
func TestSchnellstileSpeicherfehlerWirdGemeldet(t *testing.T) {
	l, _ := schnellLauf(t)
	alt := aendereStile
	aendereStile = func(func([]settings.Stil) []settings.Stil) ([]settings.Stil, bool, error) {
		return nil, false, errors.New("settings: kein Platz")
	}
	t.Cleanup(func() { aendereStile = alt })
	k := &l.e.schnell
	l.e.s.SetzeBreite(6)
	l.setle()
	l.klick(mitte(k.plusRahmen))
	if !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "kein Platz") || strings.Contains(l.e.meldung.Text, "settings:") {
		t.Errorf("Meldung %q (Fehler %v), Soll der Fehler ohne Paketpräfix", l.e.meldung.Text, l.e.meldung.Fehler)
	}
	if len(k.liste) != 8 {
		t.Errorf("%d Kacheln, der Stil soll in der Sitzung bleiben", len(k.liste))
	}
}

// Hat sich die Liste seit dem Zeichnen geändert, entfernt ein Rechtsklick nicht die Kachel, die jetzt an der Stelle steht.
func TestSchnellstileEntfernenMitVeralteterListe(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	l.e.s.SetzeBreite(6)
	l.setle()
	l.klick(mitte(k.plusRahmen))
	l.e.s.SetzeBreite(7)
	l.setle()
	l.klick(mitte(k.plusRahmen))
	eigene := l.e.s.EigeneSchnellstile()
	if len(eigene) != 2 {
		t.Fatalf("%d eigene Stile, Soll 2", len(eigene))
	}
	l.e.s.SetzeEigeneSchnellstile([]canvas.Schnellstil{eigene[1], eigene[0]}) // vertauscht, ohne neues Bild
	l.e.schnellEntferne(l.gtx(), 7)
	if n := len(l.e.s.EigeneSchnellstile()); n != 2 {
		t.Errorf("%d eigene Stile nach dem Entfernen mit veralteter Liste, Soll 2", n)
	}
}

// Ein Stil der Datei mit unbekannter Art kommt nicht in die Sitzung.
func TestSchnellstileUnbekannteArtFaelltWeg(t *testing.T) {
	alt := ladeStile
	ladeStile = func() ([]settings.Stil, []string) {
		return []settings.Stil{{Art: "bogus", Farbe: color.NRGBA{A: 255}},
			{Art: settings.ArtKasten, Farbe: color.NRGBA{R: 1, A: 255}, Breite: 5}}, nil
	}
	t.Cleanup(func() { ladeStile = alt })
	e := testEditor(t)
	e.stileLaden()
	if n := len(e.s.EigeneSchnellstile()); n != 1 {
		t.Errorf("%d eigene Stile, Soll 1 (der mit unbekannter Art fällt weg)", n)
	}
}
