// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Andiministrator/schnapp/internal/errprefix"
	"github.com/Andiministrator/schnapp/internal/saveprogress"
)

// VorschauSeite ist die längste Seite eines Vorschaubilds in Pixeln.
const VorschauSeite = 256

// VorschauPfad ist der Ort des Vorschaubilds zu einem Eintrag-Pfad: ein Hash des relativen Pfads, damit der
// Name weder Pfad noch Zeit verrät und Unterordner entfallen.
func (b *Bibliothek) VorschauPfad(rel string) string {
	h := sha256.Sum256([]byte(rel))
	return filepath.Join(b.Orte.Vorschau, hex.EncodeToString(h[:12])+".png")
}

// kodiereVorschau und entferneDatei sind Nahtstellen für Tests (Schreib- und Löschfehler nachbauen).
var (
	kodiereVorschau = png.Encode
	entferneDatei   = os.Remove
)

func (b *Bibliothek) schreibeVorschau(rel string, bild image.Image) error {
	return b.schreibeVorschauKlein(rel, func() *image.NRGBA { return verkleinere(bild, VorschauSeite) })
}

// schreibeVorschauKlein wie schreibeVorschau, mit dem fertig verkleinerten Bild von holeKlein (Speichern verkleinert
// nebenläufig zum Kodieren, F-310). holeKlein wird erst nach dem Anlegen des Ordners aufgerufen.
func (b *Bibliothek) schreibeVorschauKlein(rel string, holeKlein func() *image.NRGBA) error {
	if err := eigenerOrdner(b.Orte.Vorschau); err != nil {
		return err
	}
	klein := holeKlein()
	return schreibeAtomar(b.VorschauPfad(rel), func(w io.Writer) error { return kodiereVorschau(w, klein) })
}

// erneuereVorschau schreibt das Vorschaubild zu rel neu. Scheitert das, wird das alte entfernt: Sonst zeigte die
// Vorschau weiter einen früheren Stand — nach dem Schwärzen den ungeschwärzten (Kernprinzip 2, F-115). Liefert
// ErrVorschau, wenn nur das Schreiben scheiterte, und zusätzlich ErrVorschauAlt, wenn das alte Bild bleibt.
//
// Vorher räumt es liegen gebliebene eigene temporäre Dateien im Vorschau-Ordner (Z-3): Ein Absturz zwischen Schreiben
// und Umbenennen ließe sonst ein verkleinertes Bild des alten Stands für immer liegen. Es gelten die Regeln von
// raeumeTempReste (nur Präfix, nur reguläre Dateien, nie ein Symlink; Altersgrenze oder gleicher Inode wie das zu
// ersetzende Vorschaubild); was bleibt oder sich nicht entfernen lässt, kommt als ErrTempRest zurück.
func (b *Bibliothek) erneuereVorschau(rel string, bild image.Image) error {
	return b.erneuereVorschauKlein(rel, func() *image.NRGBA { return verkleinere(bild, VorschauSeite) })
}

// erneuereVorschauKlein wie erneuereVorschau, mit dem verkleinerten Bild von holeKlein (siehe schreibeVorschauKlein).
func (b *Bibliothek) erneuereVorschauKlein(rel string, holeKlein func() *image.NRGBA) error {
	var alt os.FileInfo
	if fi, err := os.Lstat(b.VorschauPfad(rel)); err == nil && fi.Mode().IsRegular() {
		alt = fi
	}
	_, restFehler := raeumeTempReste(b.Orte.Vorschau, time.Now(), false, alt)
	err := b.schreibeVorschauKlein(rel, holeKlein)
	if err == nil {
		return restFehler
	}
	if ferr := b.entferneVorschau(rel); ferr != nil {
		return errors.Join(restFehler, fmt.Errorf("%w: %v; %w: %v", ErrVorschau, err, ErrVorschauAlt, ferr))
	}
	return errors.Join(restFehler, fmt.Errorf("%w: %v", ErrVorschau, err))
}

// verkleinere skaliert das Bild so, dass die längste Seite höchstens seite Pixel misst (nie vergrößern), durch
// Mittelwert über das zugehörige Quellrechteck.
func verkleinere(bild image.Image, seite int) *image.NRGBA {
	b := bild.Bounds()
	w, h := b.Dx(), b.Dy()
	nw, nh := w, h
	if max(w, h) > seite {
		if w >= h {
			nw, nh = seite, max(1, h*seite/w)
		} else {
			nw, nh = max(1, w*seite/h), seite
		}
	}
	aus := image.NewNRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		saveprogress.Puls() // Fortschritt beim Speichern (F-298): bei großen Bildern dauert eine Zeile
		y0, y1 := y*h/nh, max(y*h/nh+1, (y+1)*h/nh)
		for x := 0; x < nw; x++ {
			x0, x1 := x*w/nw, max(x*w/nw+1, (x+1)*w/nw)
			var r, g, bl, a, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, ca := bild.At(b.Min.X+sx, b.Min.Y+sy).RGBA() // vormultipliziert, 16 Bit
					r, g, bl, a, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), a+uint64(ca), n+1
				}
			}
			// Mittelwert (vormultipliziert) und zurück nach NRGBA
			aus.Set(x, y, color.RGBA64{uint16(r / n), uint16(g / n), uint16(bl / n), uint16(a / n)})
		}
	}
	return aus
}

// entferneVorschau löscht das Vorschaubild zu rel, wenn es nicht erneuert werden konnte (F-115) oder die Aufnahme die Bibliothek verlassen hat (Papierkorb,
// Abgleich): sonst bliebe eine verkleinerte Kopie eines womöglich vertraulichen Bildes im Cache (F-51). Gelöscht
// wird nur genau die eigene Cache-Datei (Ordner und Namensform geprüft); fehlt sie, ist nichts zu tun.
func (b *Bibliothek) entferneVorschau(rel string) error {
	pfad := b.VorschauPfad(rel)
	name := filepath.Base(pfad)
	if filepath.Dir(pfad) != filepath.Clean(b.Orte.Vorschau) || len(name) != 24+len(".png") || filepath.Ext(name) != ".png" {
		return fmt.Errorf("library: unerwarteter Vorschaupfad %q", pfad)
	}
	if err := entferneDatei(pfad); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// eigenerOrdner legt einen Ordner von Schnapp an und bringt auch einen schon vorhandenen auf 0700 (ADR-0016;
// MkdirAll allein lässt vorhandene Rechte stehen, F-55). Nur für Schnapps eigene Ordner, nie für Bilder- oder
// Papierkorb-Ordner des Nutzers.
func eigenerOrdner(pfad string) error {
	if err := os.MkdirAll(pfad, rechteOrdner); err != nil {
		return err
	}
	// Scheitert das Nachziehen der Rechte (Ordner auf einem NAS, vfat, fremder Besitzer), bleibt Ablegen möglich:
	// eine Aufnahme zu verlieren wäre schlimmer als ein zu offener Ordner, den Schnapp nicht ändern darf (F-67).
	_ = aendereRechte(pfad, rechteOrdner)
	return nil
}

// aendereRechte ist os.Chmod; Tests ersetzen es, um einen scheiternden Chmod nachzubauen (F-67).
var aendereRechte = os.Chmod

// VerwaisteVorschauen listet die Vorschaubilder im Cache, zu denen keine Aufnahme in der Ablage mehr gehört (absolute
// Pfade, sortiert). Sie löscht nichts: Aufräumen ist eine eigene, ausdrücklich freigegebene Aktion.
//
// Maßgeblich ist die Ablage, nicht der Index: Zu jeder PNG-Datei dort wird der erwartete Vorschaupfad gebildet
// (VorschauPfad); alles im Vorschau-Ordner, was die Namensform eines Vorschaubilds hat (24 Hexzeichen, .png) und
// nicht dazugehört, ist verwaist. Andere Dateien im Ordner (fremde, temporäre) werden nicht gemeldet. Ist die
// Ablage nicht lesbar oder fehlt sie (etwa ein nicht eingehängtes Laufwerk), kommt ein Fehler statt einer Liste, in
// der fälschlich alles verwaist wäre. Ein fehlender Vorschau-Ordner ergibt eine leere Liste.
func (b *Bibliothek) VerwaisteVorschauen() (_ []string, fehler error) {
	defer errprefix.Defer("library", &fehler)
	b.mu.Lock()
	defer b.mu.Unlock()
	erwartet := map[string]bool{}
	// Wurzel auflösen: WalkDir steigt an einer Symlink-Wurzel nicht ab, dann gälte alles als verwaist (F-270).
	wurzel, err := filepath.EvalSymlinks(b.Orte.Ablage)
	if err != nil {
		return nil, fmt.Errorf("library: Ablage nicht lesbar: %w", err)
	}
	err = filepath.WalkDir(wurzel, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() || !strings.EqualFold(filepath.Ext(pfad), ".png") {
			return nil
		}
		rel, err := filepath.Rel(wurzel, pfad)
		if err != nil {
			return err
		}
		erwartet[b.VorschauPfad(filepath.ToSlash(rel))] = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("library: Ablage nicht lesbar: %w", err)
	}
	eintraege, err := os.ReadDir(b.Orte.Vorschau)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var verwaist []string
	vorhanden := 0
	for _, e := range eintraege {
		if e.Type().IsRegular() && istVorschauName(e.Name()) {
			vorhanden++
		}
	}
	// Eine Ablage ganz ohne Aufnahme bei gefülltem Cache ist fast sicher ein leerer Einhängepunkt oder ein
	// verschobener Ordner, nicht „alles verwaist“ (F-270): Fehler statt Liste, damit ein Aufräumen auf dieser Liste
	// nie den ganzen Cache trifft.
	if len(erwartet) == 0 && vorhanden > 0 {
		return nil, fmt.Errorf("%w: %d Vorschaubild(er) im Cache", ErrAblageLeer, vorhanden)
	}
	for _, e := range eintraege {
		name := e.Name()
		if !e.Type().IsRegular() || !istVorschauName(name) {
			continue
		}
		if pfad := filepath.Join(b.Orte.Vorschau, name); !erwartet[pfad] {
			verwaist = append(verwaist, pfad)
		}
	}
	sort.Strings(verwaist)
	return verwaist, nil
}

// istVorschauName prüft die Namensform eines Vorschaubilds: 24 Kleinbuchstaben-Hexzeichen und „.png“.
func istVorschauName(name string) bool {
	if len(name) != 24+len(".png") || !strings.HasSuffix(name, ".png") {
		return false
	}
	for _, c := range name[:24] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
