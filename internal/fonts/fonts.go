// SPDX-License-Identifier: GPL-3.0-or-later

// Package fonts findet und lädt die Schriften für Text-Beschriftungen (ADR-0021 Nachtrag 10): eingebaut Go und Go Mono,
// dazu die Systemschriften aus den XDG-Schriftordnern, gelesen mit golang.org/x/image/font/sfnt — ohne cgo, ohne
// fontconfig, ohne Index auf der Platte. Jede Familie zählt mit den vier Grundschnitten normal, fett, kursiv und fett
// kursiv; andere Schnitte übergeht sie.
package fonts

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Namen der eingebauten Familien; Go ist die Voreinstellung (leerer Name im Dokument).
const (
	Go     = "Go"
	GoMono = "Go Mono"
)

// Grenzen der Suche (ADR-0021 Nachtrag 10 Punkt 3).
const (
	MaxDateien      = 20000
	MaxDateiGroesse = 64 << 20
	MaxNamenLaenge  = 128
	// MaxGeladen und MaxGeladenBytes begrenzen, was an Schriftdateien im Speicher liegt (Sitzungs-QA B-2): Eine
	// präparierte Arbeitsdatei mit vielen Familien lädt sonst den ganzen Schriftbestand. Darüber gilt Ersatz (Go).
	MaxGeladen      = 32
	MaxGeladenBytes = 256 << 20
	// SucheFrist begrenzt die Suche nach Systemschriften: Der Durchlauf bricht danach ab; Laden wartet höchstens
	// SucheFrist + 1 s auf sie (ein hängender Systemaufruf selbst wird nicht unterbrochen).
	SucheFrist = 10 * time.Second
)

// Schnitt ist einer der vier Grundschnitte: Bit 0 fett, Bit 1 kursiv.
type Schnitt uint8

const (
	Normal Schnitt = iota
	Fett
	Kursiv
	FettKursiv
)

// SchnittAus liefert den Schnitt zu fett und kursiv.
func SchnittAus(fett, kursiv bool) Schnitt {
	s := Normal
	if fett {
		s |= Fett
	}
	if kursiv {
		s |= Kursiv
	}
	return s
}

// ersatzFolge ist die Reihenfolge, in der ein fehlender Schnitt ersetzt wird (Punkt 4): erst der verlangte, dann
// ohne kursiv, dann normal.
func ersatzFolge(s Schnitt) []Schnitt {
	switch s {
	case FettKursiv:
		return []Schnitt{FettKursiv, Fett, Kursiv, Normal}
	case Fett:
		return []Schnitt{Fett, Normal}
	case Kursiv:
		return []Schnitt{Kursiv, Normal}
	}
	return []Schnitt{Normal, Fett, Kursiv, FettKursiv}
}

// quelle ist der Ort eines Schnitts: eingebaute Bytes oder eine Datei (mit Index in einer Sammlung).
type quelle struct {
	da    bool
	roh   []byte
	pfad  string
	index int
	samml bool
}

type familie struct {
	name     string
	schnitte [4]quelle
}

// Schrift ist ein geladener Schnitt. Font ist nebenläufig lesbar, solange jeder Aufrufer seinen eigenen sfnt.Buffer
// nimmt (opentype.NewFace legt einen eigenen an).
type Schrift struct {
	Familie string  // Name der gelieferten Familie (bei Ersatz: Go)
	Schnitt Schnitt // gelieferter Schnitt (kann vom verlangten abweichen)
	Font    *sfnt.Font
	Ersatz  bool // die verlangte Familie fehlt, geliefert ist Go
}

var (
	mu        sync.Mutex
	eingebaut = []familie{
		{name: Go, schnitte: [4]quelle{{da: true, roh: goregular.TTF}, {da: true, roh: gobold.TTF},
			{da: true, roh: goitalic.TTF}, {da: true, roh: gobolditalic.TTF}}},
		{name: GoMono, schnitte: [4]quelle{{da: true, roh: gomono.TTF}, {da: true, roh: gomonobold.TTF},
			{da: true, roh: gomonoitalic.TTF}, {da: true, roh: gomonobolditalic.TTF}}},
	}
	system  []familie // nach der Suche, nach Namen sortiert
	gesucht bool      // die Suche ist fertig
	// suche startet die Suche einmal je Prozess in einer Goroutine; fertigKanal wird danach geschlossen. Beide sind
	// Zeiger bzw. Kanäle, die der Test austauscht.
	suche       = new(sync.Once)
	fertigKanal = make(chan struct{})
	geladen     = map[schluessel]*Schrift{}
	fehler      = map[schluessel]bool{} // Dateien, die sich nicht laden ließen
	dateien     = map[string][]byte{}   // geladene Schriftdateien je Pfad (Sammlungen nur einmal gelesen)
	dateiBytes  int64                   // Summe der Bytes in dateien
	ordner      = schriftOrdner         // für Tests und SetzeOrdner austauschbar
	maxDateien  = MaxDateien            // für Tests austauschbar
	sucheFrist  = SucheFrist            // für Tests austauschbar
)

// SetzeOrdner legt die Schriftordner fest, statt sie aus HOME und XDG_* zu lesen (MCP-Server: aus dem Heim der
// Benutzerdatenbank, Sitzungs-QA B-3; Tests: hermetisch). Wirkt nur vor der ersten Suche.
func SetzeOrdner(o []string) {
	kopie := append([]string(nil), o...)
	mu.Lock()
	ordner = func() []string { return kopie }
	mu.Unlock()
}

// SucheZuruecksetzen verwirft das Ergebnis der Suche und lässt die nächste von vorn beginnen: nur für Tests, die eine
// Suche „vor und nach ihrem Ende“ prüfen (F-575). Es darf keine Suche laufen.
func SucheZuruecksetzen() {
	mu.Lock()
	defer mu.Unlock()
	system, gesucht = nil, false
	suche, fertigKanal = new(sync.Once), make(chan struct{})
}

// Ordner liefert die Schriftordner, in denen gesucht wird (bzw. würde).
func Ordner() []string {
	mu.Lock()
	o := ordner
	mu.Unlock()
	return o()
}

// starte stößt die Suche einmal je Prozess an und liefert den Kanal, der nach ihrem Ende geschlossen ist.
func starte() chan struct{} {
	mu.Lock()
	k, o := fertigKanal, suche
	mu.Unlock()
	o.Do(func() {
		go func() {
			Suche()
			close(k)
		}()
	})
	return k
}

type schluessel struct {
	familie string
	schnitt Schnitt
}

// familieZu sucht eine Familie, eingebaute zuerst; Groß- und Kleinschreibung zählt. Nur mit mu.
func familieZu(name string) (*familie, bool) {
	if name == "" {
		name = Go
	}
	for i := range eingebaut {
		if eingebaut[i].name == name {
			return &eingebaut[i], true
		}
	}
	i, ok := slices.BinarySearchFunc(system, name, func(f familie, n string) int { return strings.Compare(f.name, n) })
	if ok {
		return &system[i], true
	}
	return nil, false
}

// Laden liefert den Schnitt s der Familie name (leer = Go), mit Ersatz nach Punkt 4; nie nil. Fehlt die Familie oder
// lässt sich keiner ihrer Schnitte laden, kommt Go mit Ersatz = true. Eine Familie außer Go und Go Mono löst die Suche
// aus, wenn sie noch nicht gelaufen ist (blockierend, einmal je Prozess).
func Laden(name string, s Schnitt) *Schrift {
	if name != "" && name != Go && name != GoMono && !SucheFertig() {
		// Ohne Editor (Export, Rettung, MCP) hat niemand die Suche gestartet; läuft sie schon (Editor), wird auf ihr
		// Ende gewartet (Sitzungs-QA B-5). Sonst setzte ein Export eine vorhandene Systemschrift in Go.
		// Höchstens SucheFrist und eine Sekunde warten: Ein hängender Ordner (toter Netzordner) unterbricht die
		// Frist der Suche nicht, das Fenster darf nicht einfrieren (Verify V-4); dann gilt Go.
		select {
		case <-starte():
		case <-time.After(sucheFrist + time.Second):
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if f, ok := familieZu(name); ok {
		for _, w := range ersatzFolge(s) {
			if sch := ladeSchnitt(f, w); sch != nil {
				return sch
			}
		}
	}
	g, _ := familieZu(Go)
	for _, w := range ersatzFolge(s) {
		if sch := ladeSchnitt(g, w); sch != nil {
			k := *sch
			k.Ersatz = true
			return &k
		}
	}
	panic("fonts: Go-Schrift nicht ladbar") // eingebaute Bytes: nur bei kaputtem Build
}

// ladeSchnitt lädt den Schnitt w von f (aus dem Zwischenspeicher); nil, wenn er fehlt, kaputt ist oder die Grenzen
// MaxGeladen/MaxGeladenBytes erreicht sind. Nur mit mu.
func ladeSchnitt(f *familie, w Schnitt) *Schrift {
	q := f.schnitte[w]
	if !q.da {
		return nil
	}
	k := schluessel{f.name, w}
	if s, ok := geladen[k]; ok {
		return s
	}
	if fehler[k] {
		return nil
	}
	fnt, err := ladeQuelle(q)
	if err != nil {
		if !errors.Is(err, errGrenze) { // die Grenze ist kein Mangel der Datei (Verify V-2)
			fehler[k] = true
		}
		return nil
	}
	s := &Schrift{Familie: f.name, Schnitt: w, Font: fnt}
	geladen[k] = s
	return s
}

// errGrenze meldet, dass MaxGeladen oder MaxGeladenBytes erreicht ist; das ist kein Fehler der Datei und wird nicht
// gemerkt (Verify V-2).
var errGrenze = errors.New("fonts: zu viele Schriftdateien geladen")

func ladeQuelle(q quelle) (*sfnt.Font, error) {
	roh := q.roh
	if roh == nil {
		var ok bool
		if roh, ok = dateien[q.pfad]; !ok {
			st, err := os.Stat(q.pfad)
			if err != nil {
				return nil, err
			}
			if !st.Mode().IsRegular() || st.Size() > MaxDateiGroesse {
				return nil, errors.New("fonts: keine reguläre Datei oder zu groß")
			}
			if len(dateien) >= MaxGeladen || dateiBytes+st.Size() > MaxGeladenBytes {
				return nil, errGrenze
			}
			if roh, err = os.ReadFile(q.pfad); err != nil {
				return nil, err
			}
			if int64(len(roh)) > MaxDateiGroesse || dateiBytes+int64(len(roh)) > MaxGeladenBytes {
				return nil, errGrenze
			}
			// Erst nach gelungenem Parsen behalten (Verify V-2): eine kaputte Datei belegt keinen Platz.
			fnt, err := parse(roh, q)
			if err != nil {
				return nil, err
			}
			dateien[q.pfad] = roh
			dateiBytes += int64(len(roh))
			return fnt, nil
		}
	}
	return parse(roh, q)
}

// parse liest den Schnitt q aus roh (Einzelschrift oder Sammlung).
func parse(roh []byte, q quelle) (*sfnt.Font, error) {
	if q.samml {
		c, err := sfnt.ParseCollection(roh)
		if err != nil {
			return nil, err
		}
		return c.Font(q.index)
	}
	return sfnt.Parse(roh)
}

// Bekannt meldet, ob die Familie name vorhanden ist (eingebaut oder gefunden); leer ist Go.
func Bekannt(name string) bool {
	mu.Lock()
	defer mu.Unlock()
	_, ok := familieZu(name)
	return ok
}

// Familien liefert die Namen aller Familien: eingebaute zuerst, dann die gefundenen nach Namen.
func Familien() []string {
	mu.Lock()
	defer mu.Unlock()
	var n []string
	for _, f := range eingebaut {
		n = append(n, f.name)
	}
	for _, f := range system {
		n = append(n, f.name)
	}
	return n
}

// SucheFertig meldet, ob die Suche nach Systemschriften abgeschlossen ist.
func SucheFertig() bool {
	mu.Lock()
	defer mu.Unlock()
	return gesucht
}

// StarteSuche sucht die Systemschriften einmal je Prozess im Hintergrund; fertig (darf nil sein) wird nach dem Ende
// aus einer Goroutine gerufen.
func StarteSuche(fertig func()) {
	k := starte()
	if fertig != nil {
		go func() {
			<-k
			fertig()
		}()
	}
}

// Suche sucht die Systemschriften jetzt (blockierend) und ersetzt das Ergebnis einer früheren Suche.
func Suche() {
	mu.Lock()
	o := ordner
	mu.Unlock()
	gefunden := durchsuche(o())
	mu.Lock()
	system, gesucht = gefunden, true
	mu.Unlock()
}

// schriftOrdner sind die XDG-Schriftordner (Punkt 3), ohne doppelte.
func schriftOrdner() []string {
	var o []string
	heim, _ := os.UserHomeDir()
	daten := os.Getenv("XDG_DATA_HOME")
	if daten == "" && heim != "" {
		daten = filepath.Join(heim, ".local", "share")
	}
	if daten != "" {
		o = append(o, filepath.Join(daten, "fonts"))
	}
	if heim != "" {
		o = append(o, filepath.Join(heim, ".fonts"))
	}
	dirs := os.Getenv("XDG_DATA_DIRS")
	if dirs == "" {
		dirs = "/usr/local/share:/usr/share"
	}
	for _, d := range filepath.SplitList(dirs) {
		if d != "" {
			o = append(o, filepath.Join(d, "fonts"))
		}
	}
	var eindeutig []string
	for _, d := range o {
		if filepath.IsAbs(d) && !slices.Contains(eindeutig, d) {
			eindeutig = append(eindeutig, d)
		}
	}
	return eindeutig
}

// durchsuche liest die Namen aller Schriften unter den Ordnern (Verweise auf Ordner werden nicht verfolgt).
func durchsuche(ordner []string) []familie {
	nachName := map[string]*familie{}
	var buf sfnt.Buffer
	gezaehlt := 0
	ende := time.Now().Add(sucheFrist)
	for _, wurzel := range ordner {
		_ = filepath.WalkDir(wurzel, func(pfad string, d fs.DirEntry, err error) error {
			if err != nil {
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if gezaehlt >= maxDateien || time.Now().After(ende) {
				return fs.SkipAll
			}
			if d.IsDir() {
				return nil
			}
			endung := strings.ToLower(filepath.Ext(pfad))
			if endung != ".ttf" && endung != ".otf" && endung != ".ttc" && endung != ".otc" {
				return nil
			}
			gezaehlt++
			leseDatei(pfad, endung == ".ttc" || endung == ".otc", &buf, nachName)
			return nil
		})
	}
	var f []familie
	for _, v := range nachName {
		f = append(f, *v)
	}
	slices.SortFunc(f, func(a, b familie) int { return strings.Compare(a.name, b.name) })
	return f
}

// leseDatei trägt die Schnitte der Datei pfad in nachName ein (der erste Fund je Familie und Schnitt gilt).
func leseDatei(pfad string, sammlung bool, buf *sfnt.Buffer, nachName map[string]*familie) {
	st, err := os.Stat(pfad) // folgt einem Verweis auf eine Datei
	if err != nil || !st.Mode().IsRegular() || st.Size() > MaxDateiGroesse {
		return
	}
	d, err := os.Open(pfad)
	if err != nil {
		return
	}
	defer d.Close()
	var fonts []*sfnt.Font
	if sammlung {
		c, err := sfnt.ParseCollectionReaderAt(d)
		if err != nil {
			return
		}
		for i := 0; i < c.NumFonts() && i < 256; i++ {
			f, err := c.Font(i)
			if err != nil {
				f = nil
			}
			fonts = append(fonts, f)
		}
	} else {
		f, err := sfnt.ParseReaderAt(d)
		if err != nil {
			return
		}
		fonts = []*sfnt.Font{f}
	}
	for i, f := range fonts {
		if f == nil {
			continue
		}
		name, schnitt, ok := namenVon(f, buf)
		if !ok || name == Go || name == GoMono || !mitUmrissen(f, buf) {
			continue
		}
		fam := nachName[name]
		if fam == nil {
			fam = &familie{name: name}
			nachName[name] = fam
		}
		if !fam.schnitte[schnitt].da {
			fam.schnitte[schnitt] = quelle{da: true, pfad: pfad, index: i, samml: sammlung}
		}
	}
}

// mitUmrissen meldet, ob die Schrift Glyphenumrisse hat: Farb-Bitmap-Schriften (etwa Emoji) zeichnen Renderer und
// Editor nicht und stehen deshalb nicht in der Liste (Sitzungs-QA B-8). Geprobt wird „A“, sonst die erste Glyphe.
func mitUmrissen(f *sfnt.Font, buf *sfnt.Buffer) bool {
	gi, err := f.GlyphIndex(buf, 'A')
	if err != nil || gi == 0 {
		gi = 1
	}
	if gi >= sfnt.GlyphIndex(f.NumGlyphs()) {
		gi = 0
	}
	_, err = f.LoadGlyph(buf, gi, fixed.I(16), nil)
	return !errors.Is(err, sfnt.ErrColoredGlyph)
}

// namenVon liest Familie und Schnitt; ok ist false für Schnitte außer den vier Grundschnitten oder unbrauchbare Namen.
func namenVon(f *sfnt.Font, buf *sfnt.Buffer) (string, Schnitt, bool) {
	name, err := f.Name(buf, sfnt.NameIDTypographicFamily)
	if err != nil || strings.TrimSpace(name) == "" {
		if name, err = f.Name(buf, sfnt.NameIDFamily); err != nil {
			return "", 0, false
		}
	}
	unter, err := f.Name(buf, sfnt.NameIDTypographicSubfamily)
	if err != nil || strings.TrimSpace(unter) == "" {
		unter, _ = f.Name(buf, sfnt.NameIDSubfamily)
	}
	name = strings.TrimSpace(name)
	if !NameGueltig(name) {
		return "", 0, false
	}
	s, ok := SchnittAusName(unter)
	return name, s, ok
}

// SchnittAusName erkennt einen Grundschnitt am Namen des Schnitts (Punkt 3); ok ist false für andere Schnitte.
func SchnittAusName(unter string) (Schnitt, bool) {
	fett, kursiv := false, false
	for _, w := range strings.Fields(strings.ToLower(unter)) {
		switch w {
		case "regular", "book", "normal", "roman", "plain":
		case "bold":
			fett = true
		case "italic", "oblique":
			kursiv = true
		default:
			return 0, false
		}
	}
	return SchnittAus(fett, kursiv), true
}

// NameGueltig meldet, ob name als Familienname taugt: 1 bis MaxNamenLaenge Zeichen, nicht nur Leerraum, keine
// Steuerzeichen (C0, DEL, C1) und nicht U+FFFD, U+FFFE, U+FFFF (er steht in der Arbeitsdatei und in der Oberfläche;
// docs/format.md 8.3).
func NameGueltig(name string) bool {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > MaxNamenLaenge || !utf8.ValidString(name) {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) || r == 0xfffd || r == 0xfffe || r == 0xffff {
			return false
		}
	}
	return true
}

// Mindestwerte der Sichtbarkeitsgrenzen in em (Go: ADR-0021 Nachtrag 10, ADR-0020 Nachtrag 3): Die Tinte reicht nicht
// weiter als 1,5 em über und 1 em unter die Grundlinie und bis 1,25 em seitlich des Glyphenursprungs.
const (
	HuelleOben     = 1.5
	HuelleUnten    = 1.0
	HuelleSeitlich = 1.25
	// HuelleMax deckelt die Angabe einer kaputten Schrift (head-Rahmen von Millionen Einheiten): größere Werte
	// kosten nur Rechenzeit, sie schneiden nichts ab.
	HuelleMax = 8.0
)

// Huelle liefert, wie weit die Tinte der Glyphen dieser Schrift in em über die Grundlinie (oben), darunter (unten)
// und seitlich des Glyphenursprungs (seitlich) reichen kann: der Rahmen aller Glyphen aus dem head der Schrift mit
// einem Rand von 0,05 em, nie weniger als die Mindestwerte (Go bleibt bei 1,5 / 1 / 1,25). Renderer und Editor
// lassen Zeilen und Glyphen außerhalb des Bildes nur aus, wenn auch diese Tinte es nicht mehr schneidet (F-576).
func (s *Schrift) Huelle() (oben, unten, seitlich float64) {
	oben, unten, seitlich = HuelleOben, HuelleUnten, HuelleSeitlich
	em := float64(s.Font.UnitsPerEm())
	if em <= 0 {
		return
	}
	var puffer sfnt.Buffer
	// Fester ppem 1024 statt unitsPerEm: sfnt rechnet Rahmen × ppem in int32, das liefe bei großem unitsPerEm über
	// (Verify V23-4); 32767 × 1024 × 64 passt.
	const ppem = 1024
	b, err := s.Font.Bounds(&puffer, fixed.Int26_6(ppem<<6), 0)
	if err != nil {
		return
	}
	e := func(v fixed.Int26_6) float64 { return float64(v) / 64 / ppem }
	const rand = 0.05
	cap := func(v float64) float64 { return min(v, HuelleMax) }
	oben = max(oben, cap(-e(b.Min.Y)+rand))
	unten = max(unten, cap(e(b.Max.Y)+rand))
	seitlich = max(seitlich, cap(e(b.Max.X)+rand), cap(-e(b.Min.X)+rand))
	return
}
