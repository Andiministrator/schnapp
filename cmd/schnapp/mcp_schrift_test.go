// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"slices"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fonts"
)

// TestMCPSchriftordnerAusDemHeim (Sitzungs-QA B-3): Der MCP-Server sucht Schriften im Heim der Benutzerdatenbank und in
// den Systemordnern, nie in Ordnern aus XDG_DATA_HOME oder XDG_DATA_DIRS des Agenten-Clients.
func TestMCPSchriftordnerAusDemHeim(t *testing.T) {
	alt := fonts.Ordner()
	t.Cleanup(func() { fonts.SetzeOrdner(alt) })
	t.Setenv("XDG_DATA_HOME", "/fremd/daten")
	t.Setenv("XDG_DATA_DIRS", "/fremd/a:/fremd/b")
	an := &mcpAnschluesse{}
	an.mitHeim("/heim/nutzer", func(string) string { return "" })
	want := []string{"/heim/nutzer/.local/share/fonts", "/heim/nutzer/.fonts", "/usr/local/share/fonts", "/usr/share/fonts"}
	if o := fonts.Ordner(); !slices.Equal(o, want) {
		t.Errorf("Schriftordner %v, erwartet %v", o, want)
	}
	an.mitHeim("", func(string) string { return "" })
	if o := fonts.Ordner(); !slices.Equal(o, want[2:]) {
		t.Errorf("ohne Heim: %v", o)
	}
}
