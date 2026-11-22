// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/errprefix"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// F-416: Ein Fehler eines umgestellten Pakets erscheint ohne Paketpräfix.
func TestExportFehlerTextOhnePaketpraefix(t *testing.T) {
	i18n.Setze(nil)
	for _, paket := range paketPraefixe {
		if got := exportFehlerText(errprefix.Wrap(paket, errors.New("x")), ""); got != "x" {
			t.Errorf("%s: %q, Soll %q", paket, got, "x")
		}
	}
	// Verschachtelt: mehrere Präfixe fallen nacheinander weg.
	if got := exportFehlerText(errprefix.Wrap("export", errprefix.Wrap("portal", errors.New("x"))), ""); got != "x" {
		t.Errorf("verschachtelt: %q", got)
	}
}

// F-416: Jedes Paket unter internal/, das errprefix mit seinem Namen aufruft, steht in paketPraefixe.
func TestPaketPraefixeKennenAlleAufrufer(t *testing.T) {
	re := regexp.MustCompile(`errprefix\.(?:Wrap|Defer)\("([a-z0-9]+)"`)
	dateien, err := filepath.Glob(filepath.Join("..", "*", "*.go"))
	if err != nil || len(dateien) == 0 {
		t.Fatalf("keine Quelldateien gefunden: %v", err)
	}
	pakete := map[string]bool{}
	for _, d := range dateien {
		if strings.HasSuffix(d, "_test.go") {
			continue
		}
		b, err := os.ReadFile(d)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			// Das Paket muss unter internal/ liegen (nicht der Platzhalter aus einem Kommentar).
			if fi, err := os.Stat(filepath.Join("..", m[1])); err == nil && fi.IsDir() {
				pakete[m[1]] = true
			}
		}
	}
	if len(pakete) == 0 {
		t.Fatal("keine Aufrufe von errprefix gefunden")
	}
	for p := range pakete {
		if !slices.Contains(paketPraefixe, p) {
			t.Errorf("Paket %q ruft errprefix auf, fehlt aber in paketPraefixe", p)
		}
	}
}
