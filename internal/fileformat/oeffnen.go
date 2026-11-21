// SPDX-License-Identifier: GPL-3.0-or-later

package fileformat

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"

	"github.com/Andiministrator/schnapp/internal/document"
)

// MaxDatei begrenzt, wie viel Oeffne liest (wie die Arbeitsdatei, größer als jedes erlaubte Bild samt Ebenen).
const MaxDatei = maxDatei

// ErrKeineDatei meldet einen Pfad, der keine gewöhnliche Datei ist.
var ErrKeineDatei = errors.New("fileformat: keine gewöhnliche Datei")

// Oeffne liest eine Datei für den Editor (ADR-0020): eine Arbeitsdatei mit Original und Objekten, ein gewöhnliches
// PNG oder ein JPEG. Was nicht als Arbeitsdatei mit Ebenen gelesen werden kann (fremdes Bild, beschädigte oder neuere
// Arbeitsdatei), kommt als neues Dokument aus dem sichtbaren Bild mit Fremd = true — der Editor darf es dann nie an
// seinen Platz zurückschreiben, sonst gingen Ebenen einer neueren Fassung verloren.
func Oeffne(pfad string) (*Gelesen, error) {
	st, err := os.Stat(pfad)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s", ErrKeineDatei, pfad)
	}
	// Erst die Größe, dann lesen: sonst kostet eine riesige Datei Gigabytes, bevor eine Prüfung greift (Kritik A-13).
	if st.Size() > MaxDatei {
		return nil, fmt.Errorf("%w: Datei größer als %d Bytes", ErrZuGross, MaxDatei)
	}
	f, err := os.Open(pfad)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	daten, err := io.ReadAll(io.LimitReader(f, MaxDatei+1))
	if err != nil {
		return nil, err
	}
	if len(daten) > MaxDatei {
		return nil, fmt.Errorf("%w: Datei größer als %d Bytes", ErrZuGross, MaxDatei)
	}
	if bytes.HasPrefix(daten, pngSignatur) {
		g, err := LeseArbeitsdatei(bytes.NewReader(daten))
		if err == nil {
			return g, nil
		}
		if !IstBeschaedigt(err) && !errors.Is(err, ErrFormatNeuer) {
			return nil, err
		}
		bild, err2 := LeseBild(bytes.NewReader(daten))
		if err2 != nil {
			return nil, err2
		}
		return ausBild(bild)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(daten))
	if err != nil {
		return nil, fmt.Errorf("%w: weder PNG noch JPEG", ErrKeinPNG)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > document.MaxSeite || cfg.Height > document.MaxSeite ||
		cfg.Width*cfg.Height > document.MaxPixel {
		return nil, fmt.Errorf("%w: Bild %dx%d", ErrZuGross, cfg.Width, cfg.Height)
	}
	bild, err := jpeg.Decode(bytes.NewReader(daten))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBeschaedigt, err)
	}
	return ausBild(bild)
}

func ausBild(bild image.Image) (*Gelesen, error) {
	d, err := document.Neu(bild)
	if err != nil {
		return nil, err
	}
	return &Gelesen{Dokument: d, Fremd: true}, nil
}
