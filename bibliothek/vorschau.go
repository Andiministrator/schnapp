// SPDX-License-Identifier: GPL-3.0-or-later

package bibliothek

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
)

// VorschauSeite ist die längste Seite eines Vorschaubilds in Pixeln.
const VorschauSeite = 256

// VorschauPfad ist der Ort des Vorschaubilds zu einem Eintrag-Pfad: ein Hash des relativen Pfads, damit der
// Name weder Pfad noch Zeit verrät und Unterordner entfallen.
func (b *Bibliothek) VorschauPfad(rel string) string {
	h := sha256.Sum256([]byte(rel))
	return filepath.Join(b.Orte.Vorschau, hex.EncodeToString(h[:12])+".png")
}

func (b *Bibliothek) schreibeVorschau(rel string, bild image.Image) error {
	if err := eigenerOrdner(b.Orte.Vorschau); err != nil {
		return err
	}
	klein := verkleinere(bild, VorschauSeite)
	return schreibeAtomar(b.VorschauPfad(rel), func(w io.Writer) error { return png.Encode(w, klein) })
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

// entferneVorschau löscht das Vorschaubild zu rel, wenn die Aufnahme die Bibliothek verlassen hat (Papierkorb,
// Abgleich): sonst bliebe eine verkleinerte Kopie eines womöglich vertraulichen Bildes im Cache (F-51). Gelöscht
// wird nur genau die eigene Cache-Datei (Ordner und Namensform geprüft); fehlt sie, ist nichts zu tun.
func (b *Bibliothek) entferneVorschau(rel string) error {
	pfad := b.VorschauPfad(rel)
	name := filepath.Base(pfad)
	if filepath.Dir(pfad) != filepath.Clean(b.Orte.Vorschau) || len(name) != 24+len(".png") || filepath.Ext(name) != ".png" {
		return fmt.Errorf("bibliothek: unerwarteter Vorschaupfad %q", pfad)
	}
	if err := os.Remove(pfad); err != nil && !errors.Is(err, fs.ErrNotExist) {
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
	return os.Chmod(pfad, rechteOrdner)
}
