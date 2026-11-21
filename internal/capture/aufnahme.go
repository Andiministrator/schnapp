// SPDX-License-Identifier: GPL-3.0-or-later

// Package capture ist die austauschbare Aufnahmeschicht (ADR-0007): Der Rest des Programms kennt nur die
// Schnittstelle Quelle. Die erste Umsetzung (KWin) spricht über D-Bus mit org.kde.KWin.ScreenShot3 (memfd,
// bevorzugt) bzw. ScreenShot2 (Pipe); ein Portal-Weg für andere Desktops kommt später als weitere Quelle.
package capture

import (
	"context"
	"errors"
	"image"
	"io"

	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Grenzen für Angaben aus der Antwort der Gegenstelle. KWin ist vertrauenswürdig, aber ein Fehler dort oder ein
// untergeschobener Dienst darf Schnapp nicht zum Absturz oder zu einer Riesenzuteilung bringen.
const (
	// MaxSeite begrenzt Breite und Höhe (gleich document.MaxSeite; hier eigene Konstante, damit die Schicht das
	// Dokumentmodell nicht kennen muss).
	MaxSeite = 1 << 15
	// MaxBytes begrenzt die Rohdaten einer Aufnahme (1 GiB; ein 8K-Bild hat rund 140 MB).
	MaxBytes = 1 << 30
)

// Optionen steuert eine Aufnahme.
type Optionen struct {
	// Mauszeiger nimmt den Mauszeiger mit auf.
	Mauszeiger bool
}

// Bild ist das Ergebnis einer Aufnahme.
//
// Pixel ist *image.NRGBA (nicht vormultipliziert), weil das Dokumentmodell damit arbeitet und Bildschirme in der
// Praxis deckend sind (Alpha 255): Die Umwandlung ist dann verlustfrei. Nur die vormultiplizierten Rohformate
// (ARGB32 Premultiplied, RGBA8888 Premultiplied) müssen zurückgerechnet werden und runden dabei.
type Bild struct {
	Pixel      *image.NRGBA
	Skala      float64 // devicePixelRatio laut KWin; 1, wenn KWin keine Angabe macht
	Bildschirm string  // Name des Bildschirms; leer bei Fensteraufnahmen ohne Angabe
}

// Quelle ist die Aufnahmeschicht. Alle Aufrufe brauchen eine Zeitgrenze oder einen abbrechbaren Context.
type Quelle interface {
	// Bildschirm nimmt den Bildschirm mit dem Namen name (z. B. "DP-1") in Originalauflösung auf.
	Bildschirm(ctx context.Context, name string, opt Optionen) (*Bild, error)
	// AktiverBildschirm nimmt den Bildschirm auf, den der Compositor für aktiv hält (bei KWin in der Voreinstellung
	// der unter dem Mauszeiger).
	AktiverBildschirm(ctx context.Context, opt Optionen) (*Bild, error)
	// Fenster lässt den Nutzer ein Fenster anklicken und nimmt es auf (KWin übernimmt die Auswahl). Bricht der
	// Nutzer ab, kommt ErrAbgebrochen.
	Fenster(ctx context.Context, opt Optionen) (*Bild, error)
	// Close gibt die Verbindung frei.
	io.Closer
}

// Erkennbare Fehler (mit errors.Is prüfen). Ihre Meldungen sind für Nutzer gedacht.
var (
	// ErrNichtBerechtigt: KWin (Plasma bis 6.7) lehnt den Aufruf ab, weil das Programm nicht als Aufnahmeprogramm
	// eingetragen ist (…ScreenShot2.Error.NoAuthorized bzw. …ScreenShot3.Error.NoAuthorized).
	ErrNichtBerechtigt = errors.New("not authorized")
	// ErrAbgebrochen: Der Nutzer hat die Auswahl abgebrochen (…ScreenShot2.Error.Cancelled bzw. …ScreenShot3.Error.Cancelled).
	ErrAbgebrochen = errors.New("cancelled")
	// ErrKeineSchnittstelle: Weder ScreenShot3 noch ScreenShot2 sind erreichbar (kein KWin, kein Wayland-KDE).
	ErrKeineSchnittstelle = errors.New("no screenshot interface")
)

// nutzerFehler trägt eine übersetzte Meldung und gilt für errors.Is als der Sentinel (und als ursache).
type nutzerFehler struct {
	sentinel error
	ursache  error
}

func (f *nutzerFehler) Error() string {
	switch f.sentinel {
	case ErrNichtBerechtigt:
		return i18n.T("Schnapp is not registered as a screenshot program. Install its desktop file and try again.")
	case ErrAbgebrochen:
		return i18n.T("The capture was cancelled.")
	default:
		return i18n.T("KWin offers no screenshot interface. Schnapp needs KDE Plasma on Wayland.")
	}
}

func (f *nutzerFehler) Is(ziel error) bool { return ziel == f.sentinel }

// Unwrap hält die technische Ursache für Protokolle erreichbar.
func (f *nutzerFehler) Unwrap() error { return f.ursache }
