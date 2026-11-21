// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/Andiministrator/schnapp/internal/export"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/portal"
)

// Die Meldung nennt Ursachen, nie Pfade (A-6), und bekannte Fehler erscheinen übersetzt.
func TestExportFehlerText(t *testing.T) {
	i18n.Setze(nil)
	pfad := "/geheimer/ordner/bericht.png"
	for _, c := range []struct {
		name string
		err  error
		soll string
	}{
		{"Pfadfehler", &fs.PathError{Op: "open", Path: pfad, Err: syscall.EACCES}, syscall.EACCES.Error()},
		{"Linkfehler", &os.LinkError{Op: "rename", Old: pfad + ".tmp", New: pfad, Err: syscall.EXDEV}, syscall.EXDEV.Error()},
		{"umhüllt", fmt.Errorf("x: %w", &fs.PathError{Op: "open", Path: pfad, Err: syscall.ENOSPC}), syscall.ENOSPC.Error()},
		{"Verbindung", portal.ErrVerbindung, "The connection to the desktop portal was lost."},
		{"Antwort", fmt.Errorf("%w (Antwort 2)", portal.ErrAntwort), "The file dialog ended without a result."},
		{"ohne Datei", portal.ErrOhneDatei, "The file dialog returned no file."},
		{"beschädigt", fmt.Errorf("%w: x", export.ErrZielBeschaedigt), "The file bericht.png is damaged: writing failed and the old content could not be restored."},
		{"vorhanden", export.ErrZielVorhanden, "A file with this name already exists: bericht.png"},
	} {
		got := exportFehlerText(c.err, "bericht.png")
		if got != c.soll {
			t.Errorf("%s: %q, soll %q", c.name, got, c.soll)
		}
		if strings.Contains(got, "geheimer") {
			t.Errorf("%s: Pfad in der Meldung: %q", c.name, got)
		}
	}
	if got := exportFehlerText(errors.New("sonstiges"), ""); got != "sonstiges" {
		t.Errorf("sonstiger Fehler: %q", got)
	}
}
