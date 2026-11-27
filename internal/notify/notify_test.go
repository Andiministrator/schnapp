// SPDX-License-Identifier: GPL-3.0-or-later

package notify

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
)

// mitAttrappe setzt f für Notify ein; andere Methoden scheitern im Test.
func mitAttrappe(t *testing.T, f func(ctx context.Context, args []any) ([]any, error)) {
	t.Helper()
	alt := rufe
	rufe = func(ctx context.Context, m string, args []any) ([]any, error) {
		if m != methode {
			t.Errorf("unerwartete Methode %s", m)
			return nil, errors.New("unerwartet")
		}
		return f(ctx, args)
	}
	t.Cleanup(func() { rufe = alt })
}

func TestZeigeSchicktTitelUndText(t *testing.T) {
	var gesehen []any
	mitAttrappe(t, func(_ context.Context, args []any) ([]any, error) {
		gesehen = args
		return []any{uint32(42)}, nil
	})
	if id, err := Zeige(context.Background(), "Titel", "Text"); err != nil || id != 42 {
		t.Fatalf("Kennung %d, Fehler %v", id, err)
	}
	if len(gesehen) != 8 || gesehen[0] != "Schnapp" || gesehen[3] != "Titel" || gesehen[4] != "Text" {
		t.Errorf("Argumente: %v", gesehen)
	}
	if _, ok := gesehen[1].(uint32); !ok {
		t.Errorf("replaces_id muss uint32 sein: %T", gesehen[1])
	}
	if _, ok := gesehen[7].(int32); !ok {
		t.Errorf("expire_timeout muss int32 sein: %T", gesehen[7])
	}
	// Dringlichkeit critical (Byte 2): auch bei „Nicht stören“ sichtbar (ADR-0030 Nachtrag 4 Punkt 8, F-466).
	hints, ok := gesehen[6].(map[string]dbus.Variant)
	if !ok {
		t.Fatalf("hints: %T", gesehen[6])
	}
	if u, ok := hints["urgency"].Value().(byte); !ok || u != 2 {
		t.Errorf("urgency %v, erwartet byte 2 (critical)", hints["urgency"])
	}
}

// TestZeigeMeldetJedenFehler: Fehler des Busses, keine Antwort, falscher Typ und Kennung 0 gelten als gescheitert
// (die Aufnahme darf dann nicht weitergehen).
func TestZeigeMeldetJedenFehler(t *testing.T) {
	for name, antwort := range map[string]func() ([]any, error){
		"Busfehler":    func() ([]any, error) { return nil, errors.New("kein Dienst") },
		"leer":         func() ([]any, error) { return nil, nil },
		"falscher Typ": func() ([]any, error) { return []any{"x"}, nil },
		"Kennung 0":    func() ([]any, error) { return []any{uint32(0)}, nil },
	} {
		mitAttrappe(t, func(context.Context, []any) ([]any, error) { return antwort() })
		_, err := Zeige(context.Background(), "a", "b")
		if err == nil {
			t.Errorf("%s: kein Fehler", name)
			continue
		}
		if !strings.HasPrefix(err.Error(), "notify: ") {
			t.Errorf("%s: ohne Präfix: %v", name, err)
		}
	}
}

// TestZeigeHatFrist: Der Aufruf bekommt einen Kontext mit Frist.
func TestZeigeHatFrist(t *testing.T) {
	mitAttrappe(t, func(ctx context.Context, _ []any) ([]any, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("Kontext ohne Frist")
		}
		return []any{uint32(1)}, nil
	})
	if _, err := Zeige(context.Background(), "a", "b"); err != nil {
		t.Fatal(err)
	}
}

// TestSchliesseRuftCloseNotification (F-490): Schliesse schickt CloseNotification mit der Kennung, mit Frist; ein
// Fehler des Busses kommt mit Präfix zurück.
func TestSchliesseRuftCloseNotification(t *testing.T) {
	alt := rufe
	t.Cleanup(func() { rufe = alt })
	var m string
	var args []any
	rufe = func(ctx context.Context, methode string, a []any) ([]any, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("Kontext ohne Frist")
		}
		m, args = methode, a
		return nil, nil
	}
	if err := Schliesse(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	if m != "org.freedesktop.Notifications.CloseNotification" || len(args) != 1 || args[0] != uint32(42) {
		t.Errorf("Aufruf %s %v", m, args)
	}
	rufe = func(context.Context, string, []any) ([]any, error) { return nil, errors.New("kein Dienst") }
	if err := Schliesse(context.Background(), 42); err == nil || !strings.HasPrefix(err.Error(), "notify: ") {
		t.Errorf("Busfehler: %v", err)
	}
}

// ADR-0042: Ersetze schickt die Kennung des ersetzten Hinweises, Dringlichkeit normal und eine kurze Anzeigedauer.
func TestErsetzeSchicktKennungUndNormal(t *testing.T) {
	var gesehen []any
	mitAttrappe(t, func(_ context.Context, args []any) ([]any, error) {
		gesehen = args
		return []any{uint32(9)}, nil
	})
	if id, err := Ersetze(context.Background(), 5, "Titel", "noch 3"); err != nil || id != 9 {
		t.Fatalf("Kennung %d, Fehler %v", id, err)
	}
	if gesehen[1] != uint32(5) || gesehen[4] != "noch 3" || gesehen[7] != kurz {
		t.Errorf("Argumente %v", gesehen)
	}
	if h, ok := gesehen[6].(map[string]dbus.Variant); !ok || h["urgency"].Value() != byte(1) {
		t.Errorf("Hinweise %v", gesehen[6])
	}
}
