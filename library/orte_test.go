// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"path/filepath"
	"testing"
)

func TestOrteOhneUserDirs(t *testing.T) {
	env, heim := umgebung(t, nil)
	o, err := OrteAus(env)
	if err != nil {
		t.Fatal(err)
	}
	want := Orte{
		Ablage:     filepath.Join(heim, "Pictures", "schnapp"),
		Index:      filepath.Join(heim, ".local", "share", "schnapp", "bibliothek.jsonl"),
		Vorschau:   filepath.Join(heim, ".cache", "schnapp", "vorschau"),
		Papierkorb: filepath.Join(heim, ".local", "share", "Trash"),
	}
	if o != want {
		t.Errorf("Orte = %+v, soll %+v", o, want)
	}
}

func TestOrteXDGVariablen(t *testing.T) {
	heim := t.TempDir()
	env := func(k string) string {
		return map[string]string{
			"HOME":            heim,
			"XDG_CONFIG_HOME": filepath.Join(heim, "k"),
			"XDG_DATA_HOME":   filepath.Join(heim, "d"),
			"XDG_CACHE_HOME":  filepath.Join(heim, "c"),
		}[k]
	}
	schreibe(t, filepath.Join(heim, "k", "user-dirs.dirs"), "XDG_PICTURES_DIR=\"$HOME/Bilder\"\n")
	o, err := OrteAus(env)
	if err != nil {
		t.Fatal(err)
	}
	want := Orte{
		Ablage:     filepath.Join(heim, "Bilder", "schnapp"),
		Index:      filepath.Join(heim, "d", "schnapp", "bibliothek.jsonl"),
		Vorschau:   filepath.Join(heim, "c", "schnapp", "vorschau"),
		Papierkorb: filepath.Join(heim, "d", "Trash"),
	}
	if o != want {
		t.Errorf("Orte = %+v, soll %+v", o, want)
	}
}

func TestBilderordnerAusUserDirs(t *testing.T) {
	faelle := []struct{ name, inhalt, soll string }{
		{"mit Anführungszeichen und $HOME", "XDG_PICTURES_DIR=\"$HOME/Bilder\"\n", "Bilder"},
		{"ohne Anführungszeichen", "XDG_PICTURES_DIR=$HOME/Fotos\n", "Fotos"},
		{"Leerzeichen im Namen", "XDG_PICTURES_DIR=\"$HOME/Meine Bilder\"\n", "Meine Bilder"},
		{"Kommentar und andere Zeilen", "# XDG_PICTURES_DIR=\"$HOME/Falsch\"\nXDG_MUSIC_DIR=\"$HOME/Musik\"\nXDG_PICTURES_DIR=\"$HOME/Bilder\"\n", "Bilder"},
		{"Zeile fehlt", "XDG_MUSIC_DIR=\"$HOME/Musik\"\n", "Pictures"},
		{"relativer Wert", "XDG_PICTURES_DIR=\"Bilder\"\n", "Pictures"},
		{"$HOME-Präfix ohne Trenner", "XDG_PICTURES_DIR=\"$HOMEBilder\"\n", "Pictures"},
		{"nur das Heimverzeichnis", "XDG_PICTURES_DIR=\"$HOME/\"\n", "Pictures"},
		{"leere Datei", "", "Pictures"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			heim := t.TempDir()
			schreibe(t, filepath.Join(heim, ".config", "user-dirs.dirs"), f.inhalt)
			o, err := OrteAus(func(k string) string {
				if k == "HOME" {
					return heim
				}
				return ""
			})
			if err != nil {
				t.Fatal(err)
			}
			if want := filepath.Join(heim, f.soll, "schnapp"); o.Ablage != want {
				t.Errorf("Ablage = %q, soll %q", o.Ablage, want)
			}
		})
	}
}

func TestOrteOhneHeim(t *testing.T) {
	for _, h := range []string{"", "relativ/pfad"} {
		if _, err := OrteAus(func(k string) string {
			if k == "HOME" {
				return h
			}
			return ""
		}); err == nil {
			t.Errorf("HOME=%q: Fehler erwartet", h)
		}
	}
}
