// SPDX-License-Identifier: GPL-3.0-or-later

package sprache

import (
	"reflect"
	"testing"
)

func TestWaehle(t *testing.T) {
	verfuegbar := []string{"de", "pt_BR"}
	faelle := []struct {
		name        string
		umgebung    map[string]string
		einstellung string
		want        string
	}{
		{"nichts gesetzt", nil, "", "en"},
		{"LANG deutsch", map[string]string{"LANG": "de_DE.UTF-8"}, "", "de"},
		{"LANG mit Modifikator", map[string]string{"LANG": "de_AT.UTF-8@euro"}, "", "de"},
		{"LANG unbekannt", map[string]string{"LANG": "fr_FR.UTF-8"}, "", "en"},
		{"LANG C", map[string]string{"LANG": "C"}, "", "en"},
		{"LANG C.UTF-8", map[string]string{"LANG": "C.UTF-8"}, "", "en"},
		{"LANG POSIX", map[string]string{"LANG": "POSIX"}, "", "en"},
		{"LC_MESSAGES vor LANG", map[string]string{"LC_MESSAGES": "de_DE.UTF-8", "LANG": "en_US.UTF-8"}, "", "de"},
		{"LC_ALL vor LC_MESSAGES", map[string]string{"LC_ALL": "en_GB.UTF-8", "LC_MESSAGES": "de_DE.UTF-8"}, "", "en"},
		// Nur die erste gesetzte der drei Variablen zählt, wie bei POSIX: unbekanntes LC_ALL fällt nicht auf LANG.
		{"LC_ALL unbekannt sperrt LANG", map[string]string{"LC_ALL": "fr_FR.UTF-8", "LANG": "de_DE.UTF-8"}, "", "en"},
		{"LC_ALL C sperrt LANG", map[string]string{"LC_ALL": "C", "LANG": "de_DE.UTF-8"}, "", "en"},
		{"LANGUAGE vor LANG", map[string]string{"LANGUAGE": "de", "LANG": "en_US.UTF-8"}, "", "de"},
		{"LANGUAGE Liste, erster unbekannt", map[string]string{"LANGUAGE": "fr:de_CH:en", "LANG": "en_US.UTF-8"}, "", "de"},
		{"LANGUAGE Liste, erster Englisch", map[string]string{"LANGUAGE": "en_US:de", "LANG": "de_DE.UTF-8"}, "", "en"},
		{"LANGUAGE nur unbekannt, dann LANG", map[string]string{"LANGUAGE": "fr:it", "LANG": "de_DE.UTF-8"}, "", "de"},
		{"LANGUAGE mit leeren Einträgen", map[string]string{"LANGUAGE": "::de", "LANG": "en_US.UTF-8"}, "", "de"},
		// Wie GNU gettext: LANGUAGE gilt nicht bei C-Locale oder ohne Locale, und "C" in der Liste beendet sie.
		{"LANGUAGE bei LANG=C ignoriert", map[string]string{"LANGUAGE": "de", "LANG": "C"}, "", "en"},
		{"LANGUAGE bei LC_ALL=C.UTF-8 ignoriert", map[string]string{"LANGUAGE": "de", "LC_ALL": "C.UTF-8", "LANG": "de_DE.UTF-8"}, "", "en"},
		{"LANGUAGE ohne Locale ignoriert", map[string]string{"LANGUAGE": "de"}, "", "en"},
		{"C in LANGUAGE beendet die Liste", map[string]string{"LANGUAGE": "C:de", "LANG": "de_DE.UTF-8"}, "", "en"},
		{"LANGUAGE vor unbekannter Locale", map[string]string{"LANGUAGE": "de", "LANG": "fr_FR.UTF-8"}, "", "de"},
		// glibc ignoriert LANGUAGE nur bei "C"/"C.<Kodierung>", nicht bei "POSIX" (F-44); POSIX in der Liste beendet sie.
		{"LANGUAGE bei LANG=POSIX beachtet", map[string]string{"LANGUAGE": "de", "LANG": "POSIX"}, "", "de"},
		{"POSIX in LANGUAGE beendet die Liste", map[string]string{"LANGUAGE": "POSIX:de", "LANG": "de_DE.UTF-8"}, "", "en"},
		{"Region genau vor Sprache", map[string]string{"LANG": "pt_BR.UTF-8"}, "", "pt_BR"},
		{"Region ohne Datei, Sprache auch nicht", map[string]string{"LANG": "pt_PT.UTF-8"}, "", "en"},
		{"Einstellung vor Umgebung", map[string]string{"LANG": "en_US.UTF-8"}, "de", "de"},
		{"Einstellung unbekannt", map[string]string{"LANG": "de_DE.UTF-8"}, "fr", "de"},
		{"Einstellung Englisch", map[string]string{"LANGUAGE": "de"}, "en", "en"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			umgebung := func(name string) string { return f.umgebung[name] }
			if got := Waehle(umgebung, f.einstellung, verfuegbar); got != f.want {
				t.Errorf("Waehle = %q, want %q", got, f.want)
			}
		})
	}
}

func TestKandidaten(t *testing.T) {
	for angabe, want := range map[string][]string{
		"de_DE.UTF-8@euro": {"de_DE", "de"},
		"de":               {"de"},
		" de_CH ":          {"de_CH", "de"},
		"C":                nil,
		"C.UTF-8":          nil,
		"POSIX":            nil,
		"":                 nil,
		".UTF-8":           nil,
	} {
		if got := Kandidaten(angabe); !reflect.DeepEqual(got, want) {
			t.Errorf("Kandidaten(%q) = %v, want %v", angabe, got, want)
		}
	}
}
