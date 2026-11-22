// SPDX-License-Identifier: GPL-3.0-or-later

package notify

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
)

func mitAttrappe(t *testing.T, f func(ctx context.Context, args []any) ([]any, error)) {
	t.Helper()
	alt := rufe
	rufe = f
	t.Cleanup(func() { rufe = alt })
}

func TestZeigeSchicktTitelUndText(t *testing.T) {
	var gesehen []any
	mitAttrappe(t, func(_ context.Context, args []any) ([]any, error) {
		gesehen = args
		return []any{uint32(42)}, nil
	})
	if err := Zeige(context.Background(), "Titel", "Text"); err != nil {
		t.Fatal(err)
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
		err := Zeige(context.Background(), "a", "b")
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
	if err := Zeige(context.Background(), "a", "b"); err != nil {
		t.Fatal(err)
	}
}
