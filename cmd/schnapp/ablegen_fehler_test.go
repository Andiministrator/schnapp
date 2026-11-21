// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
)

// F-305: Scheitert das Ablegen einer Aufnahme, zeigt stderr den übersetzten Grund, weder Pfad noch Paketpräfix noch
// rohe Ursache; mit SCHNAPP_DEBUG steht die Ursache im Debug-Log.
func TestAblegenFehlerOhnePfadUndRohtext(t *testing.T) {
	i18n.Setze(nil)
	ordner := t.TempDir()
	roh := fmt.Errorf("open %s: %w", ordner, fs.ErrPermission)

	l := neuerLauf(t)
	l.ablegenErr = roh
	t.Setenv("SCHNAPP_DEBUG", "")
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusFenster})
	if code != exitFehler || aus != "" {
		t.Errorf("Code %d, aus %q, Soll Fehler ohne Pfadzeile", code, aus)
	}
	if !strings.Contains(fehler, "Could not save the screenshot: No permission to write the file.") {
		t.Errorf("Meldung fehlt oder trägt falschen Grund: %q", fehler)
	}
	for _, v := range []string{ordner, "open", "permission denied"} {
		if strings.Contains(fehler, v) {
			t.Errorf("stderr enthält %q: %q", v, fehler)
		}
	}

	l = neuerLauf(t)
	l.ablegenErr = roh
	t.Setenv("SCHNAPP_DEBUG", "1")
	_, _, fehler = l.fuehre(Auftrag{Modus: ModusFenster})
	if !strings.Contains(fehler, "[schnapp] Ursache des Ablegens: ") || !strings.Contains(fehler, ordner) {
		t.Errorf("Debug-Log ohne Ursache: %q", fehler)
	}
}

// F-306: Käme ErrTempRest je ohne Pfad zurück, ist es ein Fehler wie jeder andere: Exit 1, kein Pfad, kein
// „bibliothek:“-Präfix, die Ursache nur im Debug-Log (der frühere Sonderzweig war unerreichbar und ist entfernt).
func TestAblegenTempRestOhnePfadIstFehler(t *testing.T) {
	i18n.Setze(nil)
	ordner := t.TempDir()
	l := neuerLauf(t)
	l.ablegenErr = fmt.Errorf("library: %w: remove %s/.schnapp-tmp-1", library.ErrTempRest, ordner)
	t.Setenv("SCHNAPP_DEBUG", "")
	code, aus, fehler := l.fuehre(Auftrag{Modus: ModusFenster})
	if code != exitFehler || aus != "" {
		t.Errorf("Code %d, aus %q", code, aus)
	}
	if !strings.Contains(fehler, "Could not save the screenshot: ") {
		t.Errorf("Meldung fehlt: %q", fehler)
	}
	for _, v := range []string{ordner, "bibliothek:", ".schnapp-tmp"} {
		if strings.Contains(fehler, v) {
			t.Errorf("stderr enthält %q: %q", v, fehler)
		}
	}
}
