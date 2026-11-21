// SPDX-License-Identifier: GPL-3.0-or-later

package capture

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const kwinDienst = "org.kde.KWin"

// Zeitgrenzen. Gewöhnliche Aufnahmen dauern 20–130 ms (Messung im Prototyp); 10 s fangen einen hängenden KWin ab.
// Bei Fenster wartet KWin auf einen Klick des Nutzers, deshalb dort fünf Minuten (danach Abbruch, damit der Aufruf
// nicht ewig hängt).
const (
	zeitgrenze            = 10 * time.Second
	zeitgrenzeInteraktiv  = 5 * time.Minute
	fehlerNichtBerechtigt = ".Error.NoAuthorized" // hinter dem Schnittstellennamen
	fehlerAbgebrochen     = ".Error.Cancelled"
)

// schnittstelle beschreibt eine der beiden KWin-Schnittstellen.
type schnittstelle struct {
	pfad  dbus.ObjectPath
	name  string
	memfd bool // ScreenShot3: KWin gibt einen memfd zurück; ScreenShot2: wir übergeben eine Pipe
}

var (
	schnittstelle3 = schnittstelle{"/org/kde/KWin/ScreenShot3", "org.kde.KWin.ScreenShot3", true}
	schnittstelle2 = schnittstelle{"/org/kde/KWin/ScreenShot2", "org.kde.KWin.ScreenShot2", false}
)

// datenquelle liefert die Pixel-Rohdaten, nachdem KWin geantwortet hat.
type datenquelle interface {
	// lies liefert genau n Bytes (mehr vorhandene werden ignoriert) oder einen Fehler, wenn weniger da sind.
	lies(n uint64) ([]byte, error)
	Close() error
}

// transport ist die Naht zu D-Bus; Tests setzen eine Attrappe ein.
type transport interface {
	// version liest die Eigenschaft Version; Fehler, wenn das Objekt nicht existiert.
	version(ctx context.Context, s schnittstelle) (uint32, error)
	// capture ruft die Methode auf. args sind die Parameter vor den Optionen. Zurück kommt die Antwort-Map
	// und die Quelle der Pixel (der Aufrufer schließt sie).
	capture(ctx context.Context, s schnittstelle, methode string, args []any, opt map[string]dbus.Variant) (map[string]dbus.Variant, datenquelle, error)
	Close() error
}

// KWin ist die Aufnahmeschicht für KDE Plasma.
type KWin struct {
	t transport

	mu sync.Mutex
	s  *schnittstelle // gewählte Schnittstelle, nil bis zum ersten Aufruf
}

// verbindeTransport ist die Naht für den Verbindungsaufbau zum Session-Bus; Tests setzen einen Fehler ein (F-350).
var verbindeTransport = func() (transport, error) {
	t, err := neuerDBusTransport()
	if err != nil {
		return nil, err
	}
	return t, nil
}

// NeuKWin verbindet mit dem Session-Bus. Die Wahl ScreenShot3/ScreenShot2 fällt beim ersten Aufruf.
func NeuKWin() (*KWin, error) {
	t, err := verbindeTransport()
	if err != nil {
		return nil, fmt.Errorf("capture: Session-Bus: %w", err)
	}
	return &KWin{t: t}, nil
}

// Close schließt die D-Bus-Verbindung.
func (k *KWin) Close() error { return k.t.Close() }

var _ Quelle = (*KWin)(nil)

// waehle bestimmt einmal die Schnittstelle: ScreenShot3, wenn das Objekt antwortet, sonst ScreenShot2.
func (k *KWin) waehle(ctx context.Context) (schnittstelle, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.s != nil {
		return *k.s, nil
	}
	for _, s := range []schnittstelle{schnittstelle3, schnittstelle2} {
		if _, err := k.t.version(ctx, s); err == nil {
			k.s = &s
			return s, nil
		} else if ctx.Err() != nil {
			return schnittstelle{}, fmt.Errorf("capture: %w", ctx.Err())
		}
	}
	return schnittstelle{}, &nutzerFehler{sentinel: ErrKeineSchnittstelle}
}

// Bildschirm nimmt den benannten Bildschirm in Originalauflösung auf (CaptureScreen).
func (k *KWin) Bildschirm(ctx context.Context, name string, opt Optionen) (*Bild, error) {
	ctx, abbruch := context.WithTimeout(ctx, zeitgrenze)
	defer abbruch()
	return k.aufnehmen(ctx, "CaptureScreen", []any{name}, opt, name)
}

// AktiverBildschirm nimmt den Bildschirm auf, den KWin für aktiv hält (CaptureActiveScreen). In der
// KWin-Voreinstellung ist das der Monitor unter dem Mauszeiger (gemessen, F-72, knowledge/topics/kwin-screenshot.md).
func (k *KWin) AktiverBildschirm(ctx context.Context, opt Optionen) (*Bild, error) {
	ctx, abbruch := context.WithTimeout(ctx, zeitgrenze)
	defer abbruch()
	return k.aufnehmen(ctx, "CaptureActiveScreen", nil, opt, "")
}

// Alle nimmt alle Bildschirme zusammengesetzt auf (CaptureWorkspace). Mit native-resolution liefert KWin bei
// gemischter Skalierung ein Bild in der größten Skala; Monitore mit kleinerer Skala werden hochgerechnet
// (knowledge/topics/kwin-screenshot.md). Das gilt wie bei den Einzelaufnahmen: Originalauflösung des feinsten Monitors.
func (k *KWin) Alle(ctx context.Context, opt Optionen) (*Bild, error) {
	ctx, abbruch := context.WithTimeout(ctx, zeitgrenze)
	defer abbruch()
	s, err := k.waehle(ctx)
	if err != nil {
		return nil, err
	}
	// CaptureWorkspace gibt es bei ScreenShot2 erst ab Version 3 (F-345); ältere KWin-Versionen melden sonst nur
	// einen rohen D-Bus-Fehler (UnknownMethod).
	if s == schnittstelle2 {
		if v, err := k.t.version(ctx, s); err == nil && v < 3 {
			return nil, &nutzerFehler{sentinel: ErrKeineGesamtaufnahme}
		}
	}
	return k.aufnehmen(ctx, "CaptureWorkspace", nil, opt, "")
}

// Fenster lässt KWin ein Fenster per Klick wählen (CaptureInteractive, kind 0).
func (k *KWin) Fenster(ctx context.Context, opt Optionen) (*Bild, error) {
	ctx, abbruch := context.WithTimeout(ctx, zeitgrenzeInteraktiv)
	defer abbruch()
	return k.aufnehmen(ctx, "CaptureInteractive", []any{uint32(0)}, opt, "")
}

func (k *KWin) aufnehmen(ctx context.Context, methode string, args []any, opt Optionen, name string) (*Bild, error) {
	s, err := k.waehle(ctx)
	if err != nil {
		return nil, err
	}
	optionen := map[string]dbus.Variant{
		"native-resolution":   dbus.MakeVariant(true),
		"include-cursor":      dbus.MakeVariant(opt.Mauszeiger),
		"hide-caller-windows": dbus.MakeVariant(true),
	}
	antwort, daten, err := k.t.capture(ctx, s, methode, args, optionen)
	if err != nil {
		return nil, fehlerUebersetzen(methode, err)
	}
	defer daten.Close()
	return auswerten(antwort, daten, name)
}

// fehlerUebersetzen macht aus den bekannten KWin-Fehlernamen die erkennbaren Fehlerwerte.
func fehlerUebersetzen(methode string, err error) error {
	var de dbus.Error
	if errors.As(err, &de) {
		// ScreenShot2 und ScreenShot3 melden dieselben Fehler unter eigenem Namensraum
		// (org.kde.KWin.ScreenShot2.Error.Cancelled bzw. org.kde.KWin.ScreenShot3.Error.Cancelled, F-49).
		for _, ns := range []string{schnittstelle2.name, schnittstelle3.name} {
			switch de.Name {
			case ns + fehlerNichtBerechtigt:
				return &nutzerFehler{sentinel: ErrNichtBerechtigt, ursache: err}
			case ns + fehlerAbgebrochen:
				return &nutzerFehler{sentinel: ErrAbgebrochen, ursache: err}
			}
		}
	}
	return fmt.Errorf("capture: %s: %w", methode, err)
}

// auswerten prüft die Antwort-Map, liest die Pixel und wandelt sie um. name ist der angefragte Bildschirmname und
// gilt, wenn KWin keinen "screen" meldet.
func auswerten(antwort map[string]dbus.Variant, daten datenquelle, name string) (*Bild, error) {
	if typ, ok := antwort["type"].Value().(string); !ok || typ != "raw" {
		return nil, fmt.Errorf("capture: unerwarteter Ergebnistyp %v", antwort["type"].Value())
	}
	var w [4]uint64
	for i, schluessel := range []string{"width", "height", "stride", "format"} {
		v, ok := antwort[schluessel]
		if !ok {
			return nil, fmt.Errorf("capture: Antwort ohne %q", schluessel)
		}
		n, ok := zahl(v.Value())
		if !ok {
			return nil, fmt.Errorf("capture: %q hat ungültigen Wert %v (Typ %T)", schluessel, v.Value(), v.Value())
		}
		w[i] = n
	}
	if w[3] > 1<<16 { // QImage::Format ist klein; verhindert die Kürzung auf uint32
		return nil, fmt.Errorf("capture: QImage-Format %d wird nicht unterstützt", w[3])
	}
	noetig, err := groesseRoh(w[0], w[1], w[2]) // Obergrenzen vor dem Lesen: bestimmt, wie viel wir anfordern
	if err != nil {
		return nil, err
	}
	roh, err := daten.lies(noetig)
	if err != nil {
		return nil, fmt.Errorf("capture: Pixeldaten lesen: %w", err)
	}
	pixel, err := rohZuNRGBA(roh, w[0], w[1], w[2], uint32(w[3]))
	if err != nil {
		return nil, err
	}
	b := &Bild{Pixel: pixel, Skala: 1, Bildschirm: name}
	if s, ok := antwort["scale"]; ok {
		b.Skala = skalaAus(s.Value())
	}
	if s, ok := antwort["screen"].Value().(string); ok && s != "" {
		b.Bildschirm = s
	}
	return b, nil
}
