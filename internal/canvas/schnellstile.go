// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

// Schnellstile (ADR-0037, Karte #192): gespeicherte Einstellungen eines Werkzeugs, die ein Klick anwendet. Dieses
// Paket hält die Logik ohne Gio: die Tabelle der eingebauten Stile, die eigenen Stile der Sitzung, Anwenden (als
// Stand des Werkzeugs, mit gewähltem Objekt zusätzlich als ein Rückgängig-Schritt am Objekt), „passt zum Stand“ und
// „aus dem Stand“. Lesen und Schreiben der Datei styles.ini liegt in internal/settings (Umwandlung unten).
//
// Größen stehen auf Faktor 1 und wirken mal document.KantenFaktor des Bilds (ADR-0021 Nachträge 7 und 8), damit ein
// Stil auf einer 4K-Aufnahme gleich aussieht wie auf Full HD.

import (
	"image/color"
	"math"
	"slices"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fonts"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/settings"
)

// SchnellArt ist die Objektart, zu der ein Stil gehört.
type SchnellArt int

const (
	SchnellPfeil SchnellArt = iota
	SchnellKasten
	SchnellText
	SchnellSprechblase // Text mit Spitze (Werkzeug Sprechblase, ADR-0021 Nachtrag 13); eingebaute Stile = die des Texts
	SchnellNummer
	SchnellLupe
	SchnellEllipse    // Kasten mit Form Ellipse (ADR-0047 Punkt 5); eingebaute Stile = die des Kastens
	SchnellTextmarker // multiplizierender Pfeil (ADR-0048 Punkt 6)
)

// MaxEigeneSchnellstile ist die Grenze eigener Stile je Objektart (ADR-0037 Punkt 5).
const MaxEigeneSchnellstile = settings.MaxEigeneStile

// artNamen sind die Abschnittsnamen der Datei, in der Reihenfolge von SchnellArt.
var artNamen = [...]string{settings.ArtPfeil, settings.ArtKasten, settings.ArtText, settings.ArtSprechblase,
	settings.ArtNummer, settings.ArtLupe, settings.ArtEllipse, settings.ArtTextmarker}

// Schnellstil ist ein Stil einer Objektart. Nicht jedes Feld gilt für jede Art: Pfeil und Lupe Farbe und Breite, Kasten
// dazu Füllung, Text und Sprechblase Farbe (Schrift), Füllung (Schild), Schrift, Schriftgrad, Stil und Ausrichtung,
// Nummer Farbe und Radius. Ungültige Felder bleiben 0 (bereinigt). Der Typ ist vergleichbar.
type Schnellstil struct {
	Art SchnellArt
	// Kennung ist bei eingebauten Stilen der feste Name (englisch, für die Übersetzung des Namens), bei eigenen leer.
	Kennung         string
	Farbe           color.NRGBA // Vordergrund
	Fuellung        color.NRGBA // Deckkraft 0 = keine
	Breite          float64     // Faktor 1
	Schriftgrad     float64     // Faktor 1
	Radius          float64     // Faktor 1
	Schrift         string      // leer = Go
	Fett, Kursiv    bool
	Unterstrichen   bool
	Durchgestrichen bool
	Ausrichtung     document.Ausrichtung
	// Schatten (ADR-0021 Nachtrag 26 Punkt 8, ADR-0037 Nachtrag 1): An = der Stil trägt einen (Länge und Weich auf
	// Faktor 1); ohne lässt der Stil den Schatten des Objekts stehen und passt unabhängig davon.
	Schatten document.Schatten
}

// Eigen meldet, ob der Stil ein eigener ist (graue Ecke, entfernbar).
func (st Schnellstil) Eigen() bool { return st.Kennung == "" }

// Name ist der übersetzte Name eines eingebauten Stils (Tooltip, Statuszeile); bei einem eigenen Stil leer.
func (st Schnellstil) Name() string {
	switch st.Kennung {
	case "red":
		return i18n.T("Red")
	case "blue":
		return i18n.T("Blue")
	case "green":
		return i18n.T("Green")
	case "black":
		return i18n.T("Black")
	case "white":
		return i18n.T("White")
	case "red-thick":
		return i18n.T("Red, thick")
	case "white-on-black":
		return i18n.T("White on black")
	case "marker":
		return i18n.T("Marker")
	case "yellow":
		return i18n.T("Yellow")
	case "pink":
		return i18n.T("Pink")
	case "white-on-red":
		return i18n.T("White on red")
	case "black-on-yellow":
		return i18n.T("Black on yellow")
	case "white-on-blue":
		return i18n.T("White on blue")
	case "red-plain":
		return i18n.T("Red, no label")
	case "black-plain":
		return i18n.T("Black, no label")
	}
	return ""
}

// schnellWerkzeug ist das Werkzeug der Art (false: keines, unbekannte Art).
func schnellWerkzeug(a SchnellArt) (Werkzeug, bool) {
	switch a {
	case SchnellSprechblase:
		return WerkzeugSprechblase, true
	case SchnellPfeil:
		return WerkzeugPfeil, true
	case SchnellKasten:
		return WerkzeugKasten, true
	case SchnellText:
		return WerkzeugText, true
	case SchnellNummer:
		return WerkzeugNummer, true
	case SchnellLupe:
		return WerkzeugLupe, true
	case SchnellEllipse:
		return WerkzeugEllipse, true
	case SchnellTextmarker:
		return WerkzeugTextmarker, true
	}
	return WerkzeugAuswahl, false
}

// werkzeugArt ist die Stilart eines Werkzeugs (false: das Werkzeug hat keine Stile).
func werkzeugArt(w Werkzeug) (SchnellArt, bool) {
	switch w {
	case WerkzeugSprechblase:
		return SchnellSprechblase, true
	case WerkzeugPfeil:
		return SchnellPfeil, true
	case WerkzeugKasten:
		return SchnellKasten, true
	case WerkzeugText:
		return SchnellText, true
	case WerkzeugNummer:
		return SchnellNummer, true
	case WerkzeugLupe:
		return SchnellLupe, true
	case WerkzeugEllipse:
		return SchnellEllipse, true
	case WerkzeugTextmarker:
		return SchnellTextmarker, true
	}
	return 0, false
}

// bereinigt nullt die Felder, die für die Art nicht gelten, und die Kennung (zum Vergleichen und Speichern).
func (st Schnellstil) bereinigt() Schnellstil {
	r := Schnellstil{Art: st.Art, Farbe: st.Farbe}
	switch st.Art {
	case SchnellPfeil, SchnellLupe, SchnellTextmarker:
		r.Breite = st.Breite
	case SchnellKasten, SchnellEllipse:
		r.Fuellung, r.Breite = st.Fuellung, st.Breite
	case SchnellText, SchnellSprechblase:
		r.Fuellung, r.Schriftgrad, r.Schrift = st.Fuellung, st.Schriftgrad, st.Schrift
		r.Fett, r.Kursiv, r.Unterstrichen, r.Durchgestrichen = st.Fett, st.Kursiv, st.Unterstrichen, st.Durchgestrichen
		r.Ausrichtung = st.Ausrichtung
	case SchnellNummer:
		r.Radius = st.Radius
	}
	if st.Schatten.An {
		r.Schatten = st.Schatten
	}
	return r
}

// gueltig meldet, ob der (bereinigte) Stil die Größe hat, die seine Art braucht, und eine gültige Schrift und
// Ausrichtung.
func (st Schnellstil) gueltig() bool {
	switch st.Art {
	case SchnellPfeil, SchnellKasten, SchnellLupe, SchnellEllipse, SchnellTextmarker:
		return st.Breite > 0
	case SchnellText, SchnellSprechblase:
		return st.Schriftgrad > 0 && st.Ausrichtung <= document.AusrichtungRechts &&
			(st.Schrift == "" || fonts.NameGueltig(st.Schrift))
	case SchnellNummer:
		return st.Radius > 0
	}
	return false
}

// mal rechnet die Größen eines Stils auf Faktor f um, gedeckelt auf die Grenzen des Dokuments.
func (st Schnellstil) mal(f float64) Schnellstil {
	st.Breite = math.Min(st.Breite*f, document.MaxGroesse)
	st.Schriftgrad = math.Min(st.Schriftgrad*f, document.MaxSchrift)
	st.Radius = math.Min(st.Radius*f, document.MaxSchrift)
	st.Schatten.Laenge = math.Min(st.Schatten.Laenge*f, document.MaxSchattenLaenge)
	st.Schatten.Weich = math.Min(st.Schatten.Weich*f, document.MaxSchattenWeich)
	return st
}

// durch ist die Gegenrichtung von mal (Größen eines Stands auf Faktor 1).
func (st Schnellstil) durch(f float64) Schnellstil {
	st.Breite /= f
	st.Schriftgrad /= f
	st.Radius /= f
	st.Schatten.Laenge /= f
	st.Schatten.Weich /= f
	return st
}

func gleichNah(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}

// gleichWie vergleicht zwei bereinigte Stile (Farben genau, Größen mit der Genauigkeit der Rechnung).
func (st Schnellstil) gleichWie(o Schnellstil) bool {
	a, b := st.bereinigt(), o.bereinigt()
	if !gleichNah(a.Breite, b.Breite) || !gleichNah(a.Schriftgrad, b.Schriftgrad) || !gleichNah(a.Radius, b.Radius) {
		return false
	}
	if !gleicherSchatten(a.Schatten, b.Schatten) {
		return false
	}
	a.Breite, a.Schriftgrad, a.Radius, a.Schatten = 0, 0, 0, document.Schatten{}
	b.Breite, b.Schriftgrad, b.Radius, b.Schatten = 0, 0, 0, document.Schatten{}
	return a == b
}

// gleicherSchatten vergleicht zwei Schatten (Farbe genau, Zahlen mit der Genauigkeit der Rechnung).
func gleicherSchatten(a, b document.Schatten) bool {
	return a.An == b.An && a.Farbe == b.Farbe && gleichNah(a.Winkel, b.Winkel) && gleichNah(a.Laenge, b.Laenge) &&
		gleichNah(a.Weich, b.Weich)
}

// --- Tabelle der eingebauten Stile (ADR-0037 Punkt 4), Größen auf Faktor 1 ---

var (
	stilRot   = Beschriftungsfarben[0]
	stilBlau  = Beschriftungsfarben[1]
	stilGruen = Beschriftungsfarben[2]
	stilDunkl = Beschriftungsfarben[3] // „Schwarz“ der Palette (grau-900)
	stilGelb  = color.NRGBA{R: 0xFF, G: 0xD6, B: 0x0A, A: 0xFF}
	stilKein  = color.NRGBA{} // keine Füllung
)

// eingebaut ist die Tabelle; die Reihenfolge ist die der Kacheln.
var eingebaut = []Schnellstil{
	// Pfeil. „Weiß auf Schwarz“ fällt hier auf „Weiß“: ein Pfeil hat keine Füllung (ADR-0037, Ergänzt bei der Umsetzung).
	{Art: SchnellPfeil, Kennung: "red", Farbe: stilRot, Breite: 6},
	{Art: SchnellPfeil, Kennung: "blue", Farbe: stilBlau, Breite: 6},
	{Art: SchnellPfeil, Kennung: "green", Farbe: stilGruen, Breite: 6},
	{Art: SchnellPfeil, Kennung: "black", Farbe: stilDunkl, Breite: 6},
	{Art: SchnellPfeil, Kennung: "red-thick", Farbe: stilRot, Breite: 12},
	{Art: SchnellPfeil, Kennung: "white", Farbe: Weiss, Breite: 6},
	// Kasten
	{Art: SchnellKasten, Kennung: "red", Farbe: stilRot, Fuellung: stilKein, Breite: 6},
	{Art: SchnellKasten, Kennung: "blue", Farbe: stilBlau, Fuellung: stilKein, Breite: 6},
	{Art: SchnellKasten, Kennung: "green", Farbe: stilGruen, Fuellung: stilKein, Breite: 6},
	{Art: SchnellKasten, Kennung: "black", Farbe: stilDunkl, Fuellung: stilKein, Breite: 6},
	{Art: SchnellKasten, Kennung: "red-thick", Farbe: stilRot, Fuellung: stilKein, Breite: 12},
	{Art: SchnellKasten, Kennung: "white-on-black", Farbe: Weiss, Fuellung: stilDunkl, Breite: 6},
	{Art: SchnellKasten, Kennung: "marker", Farbe: stilGelb, Fuellung: color.NRGBA{R: stilGelb.R, G: stilGelb.G, B: stilGelb.B, A: 0x66}, Breite: 6},
	// Text
	{Art: SchnellText, Kennung: "white-on-red", Farbe: Weiss, Fuellung: stilRot, Schriftgrad: 20},
	{Art: SchnellText, Kennung: "black-on-yellow", Farbe: stilDunkl, Fuellung: stilGelb, Schriftgrad: 20},
	{Art: SchnellText, Kennung: "white-on-black", Farbe: Weiss, Fuellung: stilDunkl, Schriftgrad: 20},
	{Art: SchnellText, Kennung: "white-on-blue", Farbe: Weiss, Fuellung: stilBlau, Schriftgrad: 20},
	{Art: SchnellText, Kennung: "red-plain", Farbe: stilRot, Fuellung: stilKein, Schriftgrad: 20},
	{Art: SchnellText, Kennung: "black-plain", Farbe: stilDunkl, Fuellung: stilKein, Schriftgrad: 20},
	// Nummer
	{Art: SchnellNummer, Kennung: "red", Farbe: stilRot, Radius: 14},
	{Art: SchnellNummer, Kennung: "blue", Farbe: stilBlau, Radius: 14},
	{Art: SchnellNummer, Kennung: "green", Farbe: stilGruen, Radius: 14},
	{Art: SchnellNummer, Kennung: "black", Farbe: stilDunkl, Radius: 14},
	// Lupe
	{Art: SchnellLupe, Kennung: "red", Farbe: stilRot, Breite: 6},
	{Art: SchnellLupe, Kennung: "black", Farbe: stilDunkl, Breite: 6},
	{Art: SchnellLupe, Kennung: "white", Farbe: Weiss, Breite: 6},
	// Textmarker (ADR-0048 Punkt 6)
	{Art: SchnellTextmarker, Kennung: "yellow", Farbe: TextmarkerGelb, Breite: TextmarkerBreite},
	{Art: SchnellTextmarker, Kennung: "green", Farbe: TextmarkerGruen, Breite: TextmarkerBreite},
	{Art: SchnellTextmarker, Kennung: "pink", Farbe: TextmarkerRosa, Breite: TextmarkerBreite},
}

// EingebauteSchnellstile liefert die eingebauten Stile der Art a in der Reihenfolge der Kacheln (eine Kopie). Die
// Sprechblase hat dieselben wie der Text (ADR-0021 Nachtrag 13 Punkt 5: ein gemeinsames Paar), die Ellipse dieselben
// wie der Kasten (ADR-0047 Punkt 5), jeweils mit ihrer Art.
func EingebauteSchnellstile(a SchnellArt) []Schnellstil {
	quelle := a
	switch a {
	case SchnellSprechblase:
		quelle = SchnellText
	case SchnellEllipse:
		quelle = SchnellKasten
	}
	var r []Schnellstil
	for _, st := range eingebaut {
		if st.Art == quelle {
			st.Art = a
			r = append(r, st)
		}
	}
	return r
}

// --- Umwandlung zu internal/settings ---

// Einstellung ist der Stil in der Form der Datei styles.ini.
func (st Schnellstil) Einstellung() settings.Stil {
	return settings.Stil{Art: artNamen[st.Art], Farbe: st.Farbe, Fuellung: st.Fuellung, Breite: st.Breite,
		Schriftgrad: st.Schriftgrad, Radius: st.Radius, Schrift: st.Schrift, Fett: st.Fett, Kursiv: st.Kursiv,
		Unterstr: st.Unterstrichen, Durchgestr: st.Durchgestrichen, Ausrichtung: st.Ausrichtung, Schatten: st.Schatten}.Bereinigt()
}

// SchnellstilAus wandelt einen Stil der Datei in einen eigenen Stil (false: unbekannte Art).
func SchnellstilAus(e settings.Stil) (Schnellstil, bool) {
	i := slices.Index(artNamen[:], e.Art)
	if i < 0 {
		return Schnellstil{}, false
	}
	return Schnellstil{Art: SchnellArt(i), Farbe: e.Farbe, Fuellung: e.Fuellung, Breite: e.Breite, Schriftgrad: e.Schriftgrad,
		Radius: e.Radius, Schrift: e.Schrift, Fett: e.Fett, Kursiv: e.Kursiv, Unterstrichen: e.Unterstr,
		Durchgestrichen: e.Durchgestr, Ausrichtung: e.Ausrichtung, Schatten: e.Schatten}.bereinigt(), true
}

// --- Stand der Sitzung ---

// faktor ist der Größenfaktor des Bilds (document.KantenFaktor, wie StilMitFaktor). Wer ihn liest, rechnet mit dem
// Stil: hat er sich seit dem letzten Zugriff geändert (Zuschnitt, Stauchen, Rückgängig), werden die Stile vorher
// umgerechnet (stileAngleichen, F-934).
func (s *Sitzung) faktor() float64 {
	s.stileAngleichen()
	return s.dokFaktor()
}

// dokFaktor ist der Faktor des Dokuments ohne Abgleich der Stile.
func (s *Sitzung) dokFaktor() float64 { return float64(max(1, s.D.KantenFaktor())) }

// stilObjektID ist das Objekt, das ein Stil ändert: der Text in der Eingabe, sonst die Auswahl.
func (s *Sitzung) stilObjektID() int {
	if s.zug == zugText {
		return s.textID
	}
	return s.Auswahl
}

// stilObjekt liefert das Objekt, das ein Stil der Art a ändert (false: keins gewählt oder von anderer Art).
func (s *Sitzung) stilObjekt(a SchnellArt) (document.Objekt, bool) {
	o, ok := s.D.Objekt(s.stilObjektID())
	if !ok {
		return nil, false
	}
	if w, ok := objektWerkzeug(o); ok {
		if wa, ok := werkzeugArt(w); ok && wa == a {
			return o, true
		}
	}
	return nil, false
}

// SchnellArt ist die Art, deren Stile die rechte Leiste zeigt: die des gewählten Objekts, sonst die des Werkzeugs
// (false: Schwärzen, Zuschneiden und Stauchen und Auswählen ohne Objekt haben keine Stile).
func (s *Sitzung) SchnellArt() (SchnellArt, bool) {
	if o, ok := s.D.Objekt(s.stilObjektID()); ok {
		if w, ok := objektWerkzeug(o); ok {
			return werkzeugArt(w)
		}
	}
	return werkzeugArt(s.Werkzeug)
}

// Schnellstile sind die Kacheln der Art a: erst die eingebauten, dann die eigenen in der Reihenfolge des Anlegens.
func (s *Sitzung) Schnellstile(a SchnellArt) []Schnellstil {
	r := EingebauteSchnellstile(a)
	for _, st := range s.eigene {
		if st.Art == a {
			r = append(r, st)
		}
	}
	return r
}

// objektStand ist der Stil, den o trägt, in den Größen des Objekts (nicht durch den Faktor geteilt), mit seinem
// Schatten, wenn er einen hat.
func objektStand(o document.Objekt) (Schnellstil, bool) {
	st, ok := objektStandOhneSchatten(o)
	if ok && o.Ebenenwerte().Schatten.An {
		st.Schatten = o.Ebenenwerte().Schatten
	}
	return st, ok
}

func objektStandOhneSchatten(o document.Objekt) (Schnellstil, bool) {
	switch v := o.(type) {
	case document.Pfeil:
		art := SchnellPfeil
		if v.Multipliziert() {
			art = SchnellTextmarker
		}
		return Schnellstil{Art: art, Farbe: v.Strich.Farbe, Breite: v.Strich.Breite}, true
	case document.Kasten:
		art := SchnellKasten
		if v.Form == document.FormEllipse {
			art = SchnellEllipse
		}
		return Schnellstil{Art: art, Farbe: v.Strich.Farbe, Fuellung: v.Fuellung, Breite: v.Strich.Breite}, true
	case document.Text:
		art := SchnellText
		if v.Spitze != nil {
			art = SchnellSprechblase
		}
		return Schnellstil{Art: art, Farbe: v.Farbe, Fuellung: v.Hintergrund, Schriftgrad: v.Schriftgrad,
			Schrift: v.Schrift, Fett: v.Fett, Kursiv: v.Kursiv, Unterstrichen: v.Unterstrichen,
			Durchgestrichen: v.Durchgestrichen, Ausrichtung: v.Ausrichtung}, true
	case document.Nummer:
		return Schnellstil{Art: SchnellNummer, Farbe: v.Farbe, Radius: v.Radius}, true
	case document.Lupe:
		return Schnellstil{Art: SchnellLupe, Farbe: v.Strich.Farbe, Breite: v.Strich.Breite}, true
	}
	return Schnellstil{}, false
}

// stand ist der aktuelle Stand der Art a in den Größen des Bilds: die Werte des gewählten Objekts dieser Art, sonst
// die des Werkzeugs (sein Farbpaar und seine Maße).
func (s *Sitzung) stand(a SchnellArt) (Schnellstil, bool) {
	if o, ok := s.stilObjekt(a); ok {
		return objektStand(o)
	}
	w, ok := schnellWerkzeug(a)
	if !ok {
		return Schnellstil{}, false
	}
	z := s.stilVon(w) // der Stil dieses Werkzeugs, nicht des aktiven (ADR-0041, F-928)
	st := Schnellstil{Art: a, Farbe: z.Farbe, Fuellung: z.Fuellung, Breite: z.Breite, Schriftgrad: z.Schriftgrad,
		Radius: z.Radius, Schrift: z.Schrift, Fett: z.Fett, Kursiv: z.Kursiv,
		Unterstrichen: z.Unterstrichen, Durchgestrichen: z.Durchgestrichen, Ausrichtung: z.Ausrichtung}
	if z.Schatten.An {
		st.Schatten = z.Schatten
	}
	return st, true
}

// SchnellstilPasst meldet, ob st zum aktuellen Stand passt (gleiche Werte, ADR-0037 Punkt 2): die Kachel bekommt dann
// den orangen Rand.
//
// Ein Stil ohne Schatten passt unabhängig vom Schatten des Stands (ADR-0021 Nachtrag 26 Punkt 8) — außer ein Stil mit
// Schatten derselben Art passt genau: Dann ist nur er orange, nicht zwei gleich aussehende Kacheln (Kritik K-3 #363b).
func (s *Sitzung) SchnellstilPasst(st Schnellstil) bool {
	if !s.passtGenau(st) {
		return false
	}
	if st.Schatten.An {
		return true
	}
	for _, v := range s.eigene {
		if v.Art == st.Art && v.Schatten.An && s.passtGenau(v) {
			return false
		}
	}
	return true
}

// passtGenau vergleicht st mit dem Stand; ein Stil ohne Schatten sieht den Schatten des Stands nicht an.
func (s *Sitzung) passtGenau(st Schnellstil) bool {
	ist, ok := s.stand(st.Art)
	if ok && !st.Schatten.An {
		ist.Schatten = document.Schatten{}
	}
	return ok && ist.gleichWie(st.mal(s.faktor()))
}

// PassenderSchnellstil ist der Index des ersten Stils in Schnellstile(a), der zum Stand passt, sonst -1.
func (s *Sitzung) PassenderSchnellstil(a SchnellArt) int {
	for i, st := range s.Schnellstile(a) {
		if s.SchnellstilPasst(st) {
			return i
		}
	}
	return -1
}

// SchnellstilAusStand erzeugt einen eigenen Stil aus dem aktuellen Stand der Art a (Größen auf Faktor 1).
func (s *Sitzung) SchnellstilAusStand(a SchnellArt) (Schnellstil, bool) {
	ist, ok := s.stand(a)
	if !ok {
		return Schnellstil{}, false
	}
	return ist.durch(s.faktor()).bereinigt(), true
}

// mitSchnellstil liefert o im Stil st (Größen schon auf den Faktor des Bilds gerechnet); o behält Ort und Inhalt.
func mitSchnellstil(o document.Objekt, st Schnellstil) document.Objekt {
	switch v := o.(type) {
	case document.Pfeil:
		v.Strich.Farbe, v.Strich.Breite = st.Farbe, st.Breite // die Strichart bleibt (ADR-0021 Nachtrag 25 Punkt 8)
		return v
	case document.Kasten:
		v.Strich.Farbe, v.Strich.Breite = st.Farbe, st.Breite // die Strichart bleibt (ADR-0021 Nachtrag 25 Punkt 8)
		v.Fuellung = st.Fuellung
		return v
	case document.Text:
		v.Farbe, v.Hintergrund, v.Schriftgrad = st.Farbe, st.Fuellung, st.Schriftgrad
		v.Schrift, v.Fett, v.Kursiv = st.Schrift, st.Fett, st.Kursiv
		v.Unterstrichen, v.Durchgestrichen, v.Ausrichtung = st.Unterstrichen, st.Durchgestrichen, st.Ausrichtung
		return v
	case document.Nummer:
		v.Farbe, v.Radius = st.Farbe, st.Radius
		return v
	case document.Lupe:
		v.Strich.Farbe, v.Strich.Breite = st.Farbe, st.Breite // die Strichart bleibt (ADR-0021 Nachtrag 25 Punkt 8)
		return v
	}
	return o
}

// WendeSchnellstilAn wendet st an (ADR-0037 Punkt 3): Der Stand des Werkzeugs der Art wird st (Farbpaar und Maße, mal
// Faktor des Bilds). Ist ein Objekt dieser Art gewählt (oder der Text in der Eingabe), ändert st es zusätzlich als
// **einen** Rückgängig-Schritt, in der Texteingabe als Teil der Eingabe. Liefert, ob sich das Dokument geändert hat.
// Während eines Zugs oder des Speicherns, bei einem ungültigen Stil (gueltig) und
// bei einem, dessen Größen mal Faktor die Grenzen des Dokuments sprengten, geschieht nichts.
func (s *Sitzung) WendeSchnellstilAn(st Schnellstil) bool {
	if s.Zieht() || s.gesperrt() {
		return false
	}
	w, ok := schnellWerkzeug(st.Art)
	if !ok {
		return false
	}
	f := s.faktor()
	st = st.bereinigt()
	if !st.gueltig() || st.Breite*f > document.MaxGroesse || st.Schriftgrad*f > document.MaxSchrift ||
		st.Radius*f > document.MaxSchrift {
		return false
	}
	g := st.mal(f)
	s.merkePaarFarbe(w, FeldVordergrund, g.Farbe)
	if fuellWerkzeug(w) {
		s.merkePaarFarbe(w, FeldFuellung, g.Fuellung)
	}
	s.hsvGemerkt = false
	s.aendereStil(w, func(z *Stil) {
		switch st.Art {
		case SchnellPfeil, SchnellKasten, SchnellLupe, SchnellEllipse, SchnellTextmarker:
			z.Breite = g.Breite
		case SchnellText, SchnellSprechblase:
			z.Schriftgrad, z.Schrift = g.Schriftgrad, g.Schrift
			z.Fett, z.Kursiv = g.Fett, g.Kursiv
			z.Unterstrichen, z.Durchgestrichen, z.Ausrichtung = g.Unterstrichen, g.Durchgestrichen, g.Ausrichtung
		case SchnellNummer:
			z.Radius = g.Radius
		}
		if g.Schatten.An { // ein Stil ohne Schatten lässt den des Werkzeugs und des Objekts stehen
			z.Schatten = g.Schatten
		}
	})
	if s.stilMehrere() { // alle gewählten derselben Art (ADR-0039 Nachtrag 4)
		return s.setzeMehrere(func(o document.Objekt) (document.Objekt, bool) {
			// Werkzeug des Stils: Text und Sprechblase teilen sich einen (paarSchluessel, F-949).
			if ow, ok := objektWerkzeug(o); !ok || paarSchluessel(ow) != paarSchluessel(w) {
				return o, false
			}
			neu := mitSchnellstil(o, g)
			if g.Schatten.An {
				neu = mitSchatten(neu, g.Schatten)
			}
			return neu, true
		}, nil)
	}
	o, ok := s.stilObjekt(st.Art)
	if !ok {
		return false
	}
	neu := mitSchnellstil(o, g)
	if g.Schatten.An {
		neu = mitSchatten(neu, g.Schatten)
	}
	if s.zug == zugText {
		_ = s.D.ZugSetze(neu)
		return false
	}
	if document.Gleich(neu, o) {
		return false
	}
	vor := s.D.Stand()
	ok = s.D.Ersetzen(neu) == nil
	s.merkeAenderung(vor)
	return ok
}

// --- Eigene Stile ---

// EigeneSchnellstile sind die eigenen Stile aller Arten in der Reihenfolge der Datei (zum Speichern).
func (s *Sitzung) EigeneSchnellstile() []Schnellstil { return slices.Clone(s.eigene) }

// SetzeEigeneSchnellstile übernimmt die eigenen Stile aus der Datei (Editor beim Öffnen): je Art höchstens
// MaxEigeneSchnellstile, der Rest fällt weg.
func (s *Sitzung) SetzeEigeneSchnellstile(stile []Schnellstil) {
	s.eigene = nil
	var n [len(artNamen)]int
	for _, st := range stile {
		st = st.bereinigt()
		if st.Art < 0 || int(st.Art) >= len(artNamen) || !st.gueltig() || n[st.Art] >= MaxEigeneSchnellstile {
			continue
		}
		n[st.Art]++
		s.eigene = append(s.eigene, st)
	}
}

// Ergebnis von FuegeSchnellstilHinzu.
type HinzuErgebnis int

const (
	HinzuOK          HinzuErgebnis = iota // der Stil steht am Ende der Reihe
	HinzuVoll                             // schon MaxEigeneSchnellstile eigene dieser Art („+“ ist grau)
	HinzuDoppelt                          // ein Stil mit denselben Werten steht schon da (eingebaut oder eigen)
	HinzuNichts                           // die Art hat keinen Stand (kein Werkzeug)
	HinzuUnbrauchbar                      // der Stand ließe sich nicht in styles.ini schreiben (Strich 0, Schriftname mit Leerraum, F-628)
)

// KannSchnellstilHinzufuegen meldet, ob „+“ wirkt: Die Art hat Stile, ist nicht voll und der Stand ist neu.
func (s *Sitzung) KannSchnellstilHinzufuegen(a SchnellArt) bool {
	_, r := s.neuerStil(a)
	return r == HinzuOK
}

// PruefeSchnellstilHinzu sagt, was „+“ für die Art a täte, ohne es zu tun (die Oberfläche erklärt damit den grauen Knopf).
func (s *Sitzung) PruefeSchnellstilHinzu(a SchnellArt) HinzuErgebnis {
	_, r := s.neuerStil(a)
	return r
}

func (s *Sitzung) neuerStil(a SchnellArt) (Schnellstil, HinzuErgebnis) {
	st, ok := s.SchnellstilAusStand(a)
	if !ok {
		return Schnellstil{}, HinzuNichts
	}
	// Dieselbe Prüfung wie beim Schreiben (F-628): Was die Datei nicht aufnimmt, wird keine Kachel.
	if !st.gueltig() || settings.PruefeStil(st.Einstellung()) != nil {
		return Schnellstil{}, HinzuUnbrauchbar
	}
	n := 0
	for _, e := range s.eigene {
		if e.Art == a {
			n++
		}
	}
	if n >= MaxEigeneSchnellstile {
		return Schnellstil{}, HinzuVoll
	}
	for _, v := range s.Schnellstile(a) {
		if v.gleichWie(st) {
			return Schnellstil{}, HinzuDoppelt
		}
	}
	return st, HinzuOK
}

// FuegeSchnellstilHinzu speichert den aktuellen Stand der Art a als eigenen Stil am Ende der Reihe (ADR-0037
// Punkt 5). Der Aufrufer schreibt danach EigeneSchnellstile in die Datei.
func (s *Sitzung) FuegeSchnellstilHinzu(a SchnellArt) (Schnellstil, HinzuErgebnis) {
	st, r := s.neuerStil(a)
	if r == HinzuOK {
		s.eigene = append(s.eigene, st)
	}
	return st, r
}

// EntferneSchnellstil entfernt den eigenen Stil an Index i von Schnellstile(a) (eingebaute lassen sich nicht
// entfernen). Liefert, ob einer entfernt wurde; ohne Rückfrage und ohne Rückgängig, Stile gehören nicht zum Dokument.
func (s *Sitzung) EntferneSchnellstil(a SchnellArt, i int) bool {
	liste := s.Schnellstile(a)
	if i < 0 || i >= len(liste) || !liste[i].Eigen() {
		return false
	}
	k := i - len(EingebauteSchnellstile(a)) // Nummer unter den eigenen dieser Art
	for j, e := range s.eigene {
		if e.Art != a {
			continue
		}
		if k == 0 {
			s.eigene = slices.Delete(s.eigene, j, j+1)
			return true
		}
		k--
	}
	return false
}
