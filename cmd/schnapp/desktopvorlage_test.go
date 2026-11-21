// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// F-113: „Öffnen mit“ — die Vorlage der Desktop-Datei nennt die lesbaren Formate und reicht die Datei per %f an
// `schnapp DATEI` weiter; die Aktionen bekommen kein %f; die Anführungszeichen um @EXEC@ bleiben (F-76).
func TestDesktopVorlageOeffnenMit(t *testing.T) {
	roh, err := os.ReadFile(filepath.Join("..", "..", "data", "schnapp.desktop.in"))
	if err != nil {
		t.Fatal(err)
	}
	var abschnitt string
	var mime, hauptExec bool
	for _, z := range strings.Split(string(roh), "\n") {
		z = strings.TrimRight(z, "\r")
		if strings.HasPrefix(z, "#") {
			continue
		}
		if strings.HasPrefix(z, "[") {
			abschnitt = z
			continue
		}
		if strings.HasPrefix(z, "Exec=") {
			if !strings.HasPrefix(z, `Exec="@EXEC@"`) {
				t.Errorf("%s: Exec ohne Anführungszeichen um @EXEC@: %q", abschnitt, z)
			}
			hat := strings.Contains(z, "%f")
			if abschnitt == "[Desktop Entry]" {
				hauptExec = hat
			} else if hat {
				t.Errorf("%s: Aktion darf keinen Feldcode f tragen: %q", abschnitt, z)
			}
		}
		if abschnitt == "[Desktop Entry]" && z == "MimeType=image/png;image/jpeg;" {
			mime = true
		}
	}
	if !hauptExec {
		t.Error("Exec im Hauptteil ohne Feldcode f (Prozentzeichen davor)")
	}
	if !mime {
		t.Error("MimeType=image/png;image/jpeg; fehlt im Hauptteil")
	}
}
