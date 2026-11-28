// SPDX-License-Identifier: GPL-3.0-or-later

package wlclip

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Andiministrator/schnapp/internal/wlwire"
)

// Lesen der Zwischenablage über data-control (ADR-0045 Punkte 3 und 4): eine eigene kurze Verbindung, ohne Fenster
// und ohne Tastaturfokus. Schnapp liest nur auf ausdrückliche Handlung (Strg+V, „New from clipboard“), nie im
// Hintergrund.

const (
	evDeviceSelection = 1 // selection(object offer, nullable)
	evOfferOffer      = 0 // offer(mime)
	opOfferReceive    = 0 // receive(mime, fd)
	opDeviceDestroy   = 1
)

var (
	// ErrLeer: Die Zwischenablage ist leer.
	ErrLeer = errors.New("wlclip: Zwischenablage leer")
	// ErrZuGross: Der Inhalt ist größer als erlaubt.
	ErrZuGross = errors.New("wlclip: Inhalt der Zwischenablage zu groß")
)

// LeseFrist ist die Gesamtfrist eines Lesens (F-987): Ein Lieferant, der nur tröpfelnd liefert, hält Schnapp nicht
// länger fest.
const LeseFrist = 60 * time.Second

// Gelesen ist das Ergebnis von Lies.
type Gelesen struct {
	Typen []string // alle angebotenen Typen in der Reihenfolge des Angebots
	Typ   string   // der gelesene Typ; leer, wenn wahl keinen wollte
	Daten []byte
}

// Lies verbindet sich mit dem Compositor der Sitzung, fragt die Typen der Zwischenablage ab und liest den Typ, den
// wahl aus ihnen wählt (leer: nichts lesen). Mehr als max Byte sind ErrZuGross. Jeder Schritt hat die Frist Stau, das
// Ganze höchstens LeseFrist; ctx bricht ab (Esc, Schließen, F-987). Mit WAYLAND_SOCKET oder ohne data-control liefert
// Lies den Fehler von HalterMoeglich bzw. ErrKeinDataControl.
func Lies(ctx context.Context, wahl func(typen []string) string, max int64) (Gelesen, error) {
	if err := HalterMoeglich(os.Getenv); err != nil {
		return Gelesen{}, err
	}
	v, err := wlwire.Verbinde()
	if err != nil {
		return Gelesen{}, err
	}
	defer v.Close()
	ctx, ende := context.WithTimeout(ctx, LeseFrist)
	defer ende()
	return lies(ctx, v, wahl, max, Stau)
}

func lies(ctx context.Context, v *wlwire.Verbindung, wahl func([]string) string, max int64, stau time.Duration) (Gelesen, error) {
	// Ein Abbruch weckt jedes wartende Lesen der Verbindung sofort (die Fristen danach prüfen ctx).
	stop := context.AfterFunc(ctx, func() { _ = v.SetDeadline(time.Now()) })
	defer stop()
	registry, globale, err := v.Globale(stau)
	if err != nil {
		return Gelesen{}, err
	}
	var manager string
	var mgr wlwire.Global
	for _, kandidat := range []string{ManagerExt, ManagerWlr} {
		if g, ok := globale[kandidat]; ok {
			manager, mgr = kandidat, g
			break
		}
	}
	if manager == "" {
		return Gelesen{}, ErrKeinDataControl
	}
	sitz, ok := globale["wl_seat"]
	if !ok {
		return Gelesen{}, ErrKeinSitz
	}
	sitzID, err := v.Bind(registry, sitz, "wl_seat", 1)
	if err != nil {
		return Gelesen{}, err
	}
	mgrID, err := v.Bind(registry, mgr, manager, 1)
	if err != nil {
		return Gelesen{}, err
	}
	geraet := v.NeueID()
	if err := v.Sende(mgrID, opManagerGetDevice, new(wlwire.Nachricht).U32(geraet).U32(sitzID)); err != nil {
		return Gelesen{}, err
	}
	angebote := map[uint32][]string{}
	var auswahl uint32
	err = v.Rundlauf(stau, func(e wlwire.Ereignis) error {
		l := wlwire.Leser{B: e.Args}
		switch {
		case e.Obj == geraet && e.Op == evDeviceDataOffer:
			id := l.U32()
			if l.Err != nil {
				return l.Err
			}
			angebote[id] = []string{}
		case e.Obj == geraet && e.Op == evDeviceSelection:
			id := l.U32()
			if l.Err != nil {
				return l.Err
			}
			auswahl = id
		case e.Obj == geraet && e.Op == evDeviceFinished:
			return ErrVerbindungZu
		default:
			if typen, ok := angebote[e.Obj]; ok && e.Op == evOfferOffer {
				m := l.Str()
				if l.Err != nil {
					return l.Err
				}
				if len(typen) < 256 { // ein Angebot mit Tausenden Typen ist kein Grund, Speicher zu füllen
					angebote[e.Obj] = append(typen, m)
				}
			}
		}
		return nil
	})
	if err != nil {
		return Gelesen{}, err
	}
	// Angebote, die nicht die Auswahl sind (etwa die primäre Auswahl), gleich wieder freigeben.
	for id := range angebote {
		if id != auswahl {
			_ = v.Sende(id, opOfferDestroy, nil)
		}
	}
	typen, ok := angebote[auswahl]
	if auswahl == 0 || !ok {
		return Gelesen{}, ErrLeer
	}
	g := Gelesen{Typen: typen}
	defer func() { _ = v.Sende(auswahl, opOfferDestroy, nil) }()
	g.Typ = wahl(typen)
	if g.Typ == "" {
		return g, nil
	}
	r, w, err := os.Pipe()
	if err != nil {
		return g, err
	}
	defer r.Close()
	err = v.Sende(auswahl, opOfferReceive, new(wlwire.Nachricht).Str(g.Typ).FD(int(w.Fd())))
	w.Close() // die Nachricht trägt eine Kopie; ohne Schließen sähe r nie das Ende
	if err != nil {
		return g, err
	}
	stopR := context.AfterFunc(ctx, func() { _ = r.SetReadDeadline(time.Now()) })
	defer stopR()
	g.Daten, err = liesBis(ctx, r, max, stau)
	return g, err
}

// liesBis liest r bis zum Ende, höchstens max Byte, mit der Frist stau je Stück; ctx beendet es (Gesamtfrist, Abbruch).
func liesBis(ctx context.Context, r *os.File, max int64, stau time.Duration) ([]byte, error) {
	var daten []byte
	puffer := make([]byte, 64<<10)
	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("wlclip: Lesen der Zwischenablage abgebrochen: %w", err)
		}
		if err := r.SetReadDeadline(time.Now().Add(stau)); err != nil {
			return nil, err
		}
		n, err := r.Read(puffer)
		daten = append(daten, puffer[:n]...)
		if int64(len(daten)) > max {
			return nil, fmt.Errorf("%w: mehr als %d Byte", ErrZuGross, max)
		}
		if err == io.EOF {
			return daten, nil
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("wlclip: Lesen der Zwischenablage abgebrochen: %w", ctx.Err())
			}
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return nil, fmt.Errorf("wlclip: Zwischenablage liefert nicht (keine Daten seit %v)", stau)
			}
			return nil, err
		}
	}
}
