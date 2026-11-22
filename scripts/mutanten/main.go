// Mutationsprobe für Schnapp (F-371): Mutanten stehen als Daten in *.json, das Programm wendet je Mutant eine
// Ersetzung in einer KOPIE des Wurzelmoduls an und lässt das Paket testen. Der Arbeitsbaum bleibt unberührt.
// Aufruf und Format: README.md in diesem Ordner.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Mutant ist ein Eintrag einer Gruppendatei.
type Mutant struct {
	Name        string `json:"name"`
	Datei       string `json:"datei"` // relativ zur Wurzel des Moduls
	Paket       string `json:"paket"` // wie für go test, z. B. ./internal/ui
	Alt         string `json:"alt"`
	Neu         string `json:"neu"`
	Aequivalent bool   `json:"aequivalent"`
	Grund       string `json:"grund"`
	Gruppe      string `json:"-"`
}

// Ausgang eines Mutanten.
const (
	Rot        = "ROT"
	Ueberlebt  = "ÜBERLEBT"
	Fehler     = "FEHLER"
	Aequivalen = "ÄQUIVALENT"
)

// zaehle zählt die nicht überlappenden Vorkommen von alt in text; leeres Muster zählt 0.
func zaehle(text, alt string) int {
	if alt == "" {
		return 0
	}
	return strings.Count(text, alt)
}

// wende ersetzt das genau einmal vorkommende Muster; sonst Fehler „Muster N-mal gefunden“.
func wende(text string, m Mutant) (string, error) {
	if n := zaehle(text, m.Alt); n != 1 {
		return "", fmt.Errorf("Muster %d-mal gefunden", n)
	}
	return strings.Replace(text, m.Alt, m.Neu, 1), nil
}

// ausgelassen nennt die Ordner der obersten Ebene, die nicht in die Kopie kommen.
var ausgelassen = map[string]bool{"tmp": true, ".git": true, ".claude": true}

// kopiereBaum kopiert quelle nach ziel (neu anzulegen). Auf oberster Ebene fallen tmp, .git, .claude weg.
func kopiereBaum(quelle, ziel string) error {
	if err := os.MkdirAll(ziel, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(quelle, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(quelle, pfad)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if ausgelassen[strings.Split(rel, string(filepath.Separator))[0]] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		z := filepath.Join(ziel, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(z, 0o755)
		case d.Type()&fs.ModeSymlink != 0:
			l, err := os.Readlink(pfad)
			if err != nil {
				return err
			}
			return os.Symlink(l, z)
		case d.Type().IsRegular():
			return kopiereDatei(pfad, z)
		}
		return nil
	})
}

func kopiereDatei(quelle, ziel string) error {
	in, err := os.Open(quelle)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(ziel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// ladeGruppen liest alle *.json aus dir; Gruppenname = Dateiname ohne Endung.
func ladeGruppen(dir string) ([]Mutant, error) {
	namen, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(namen)
	var alle []Mutant
	for _, n := range namen {
		b, err := os.ReadFile(n)
		if err != nil {
			return nil, err
		}
		var ms []Mutant
		if err := json.Unmarshal(b, &ms); err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(n), err)
		}
		g := strings.TrimSuffix(filepath.Base(n), ".json")
		for i := range ms {
			ms[i].Gruppe = g
			if ms[i].Name == "" || ms[i].Datei == "" || ms[i].Paket == "" || ms[i].Alt == "" {
				return nil, fmt.Errorf("%s: Eintrag %d unvollständig (name, datei, paket, alt)", g, i+1)
			}
			// Die Datei muss in der Kopie liegen: ../ oder ein absoluter Pfad träfe den Arbeitsbaum (Sitzungs-QA, Q-4).
			if !filepath.IsLocal(filepath.FromSlash(ms[i].Datei)) {
				return nil, fmt.Errorf("%s: Eintrag %d: datei %q liegt nicht im Baum", g, i+1, ms[i].Datei)
			}
		}
		alle = append(alle, ms...)
	}
	return alle, nil
}

// waehle filtert nach Gruppe und Name (leer = alle).
func waehle(alle []Mutant, gruppe, nur string) []Mutant {
	var r []Mutant
	for _, m := range alle {
		if (gruppe == "" || m.Gruppe == gruppe) && (nur == "" || m.Name == nur) {
			r = append(r, m)
		}
	}
	return r
}

// testeMutant wendet m auf die Kopie baum an, testet das Paket und stellt die Datei wieder her.
func testeMutant(baum, lauf string, m Mutant) (status string, rote []string, detail string) {
	datei := filepath.Join(baum, filepath.FromSlash(m.Datei))
	orig, err := os.ReadFile(datei)
	if err != nil {
		return Fehler, nil, err.Error()
	}
	neu, err := wende(string(orig), m)
	if err != nil {
		return Fehler, nil, err.Error()
	}
	if err := os.WriteFile(datei, []byte(neu), 0o644); err != nil {
		return Fehler, nil, err.Error()
	}
	defer os.WriteFile(datei, orig, 0o644)

	text, runErr := goTest(baum, lauf, m.Paket)
	for _, z := range strings.Split(text, "\n") {
		if strings.HasPrefix(z, "--- FAIL") {
			if f := strings.Fields(z); len(f) >= 3 {
				rote = append(rote, f[2])
			}
		}
	}
	if runErr == nil {
		return Ueberlebt, nil, ""
	}
	if strings.Contains(text, "[build failed]") || strings.Contains(text, "[setup failed]") {
		return Fehler, nil, "Bau: " + kurz(text)
	}
	if len(rote) == 0 {
		return Fehler, nil, "kein FAIL-Name: " + kurz(text)
	}
	return Rot, rote, ""
}

// goTest testet paket in der Kopie baum.
func goTest(baum, lauf, paket string) (string, error) {
	cmd := exec.Command("go", "test", "-count=1", paket)
	cmd.Dir = baum
	g := filepath.Join(lauf, "g")
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOFLAGS=-tags=novulkan,nox11", "TMPDIR="+g, "GOTMPDIR="+g)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func kurz(s string) string {
	if len(s) > 500 {
		s = s[len(s)-500:]
	}
	return strings.ReplaceAll(strings.TrimSpace(s), "\n", " | ")
}

func main() {
	os.Exit(lauf(os.Args[1:], os.Stdout))
}

func lauf(args []string, aus io.Writer) int {
	fs := flag.NewFlagSet("mutanten", flag.ContinueOnError)
	fs.SetOutput(aus)
	wurzel := fs.String("wurzel", "../..", "Wurzel des Moduls, das gemutet wird")
	dir := fs.String("mutanten", ".", "Ordner mit den *.json-Gruppendateien")
	gruppe := fs.String("gruppe", "", "nur diese Gruppe (Dateiname ohne .json)")
	nur := fs.String("nur", "", "nur den Mutanten mit diesem Namen")
	trocken := fs.Bool("trocken", false, "nur die Muster zählen, nichts kopieren oder testen")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	w, err := filepath.Abs(*wurzel)
	if err != nil {
		fmt.Fprintln(aus, "Fehler:", err)
		return 2
	}
	alle, err := ladeGruppen(*dir)
	if err != nil {
		fmt.Fprintln(aus, "Fehler:", err)
		return 2
	}
	ms := waehle(alle, *gruppe, *nur)
	if len(ms) == 0 {
		fmt.Fprintln(aus, "Fehler: kein Mutant gewählt")
		return 2
	}

	baum, laufDir := w, ""
	if !*trocken {
		laufDir = filepath.Join(w, "tmp", "mutanten", "lauf-"+time.Now().Format("20060102-150405"))
		baum = filepath.Join(laufDir, "baum")
		if err := os.MkdirAll(filepath.Join(laufDir, "g"), 0o755); err != nil {
			fmt.Fprintln(aus, "Fehler:", err)
			return 2
		}
		fmt.Fprintf(aus, "Kopie nach %s\n", baum)
		if err := kopiereBaum(w, baum); err != nil {
			fmt.Fprintln(aus, "Fehler beim Kopieren:", err)
			return 2
		}
	}

	// Ausgangslage: Jedes Paket muss ohne Mutation grün sein, sonst gälte jeder Mutant als getötet (Sitzungs-QA, Q-3).
	ausgangRot := map[string]string{}
	if !*trocken {
		for _, m := range ms {
			if _, gesehen := ausgangRot[m.Paket]; gesehen {
				continue
			}
			ausgangRot[m.Paket] = ""
			if text, err := goTest(baum, laufDir, m.Paket); err != nil {
				ausgangRot[m.Paket] = "Ausgangslage rot (ohne Mutation): " + kurz(text)
				fmt.Fprintf(aus, "Paket %s: %s\n", m.Paket, ausgangRot[m.Paket])
			}
		}
	}

	zaehler := map[string]int{}
	schlecht := 0
	for _, m := range ms {
		var status, detail string
		var rote []string
		t := time.Now()
		if *trocken {
			b, err := os.ReadFile(filepath.Join(baum, filepath.FromSlash(m.Datei)))
			if err != nil {
				status, detail = Fehler, err.Error()
			} else if _, err := wende(string(b), m); err != nil {
				status, detail = Fehler, err.Error()
			} else {
				status = "MUSTER OK"
			}
		} else if grund := ausgangRot[m.Paket]; grund != "" {
			status, detail = Fehler, grund
		} else {
			status, rote, detail = testeMutant(baum, laufDir, m)
		}
		anzeige := status
		switch {
		case m.Aequivalent && status == Ueberlebt:
			anzeige = Aequivalen + " (überlebt erwartungsgemäß)"
		case m.Aequivalent && status == Rot:
			anzeige = "ROT (als äquivalent markiert, Markierung prüfen)"
		case status == Ueberlebt || status == Fehler:
			schlecht++
		case status == "MUSTER OK":
		}
		if m.Aequivalent && status == Ueberlebt {
			zaehler[Aequivalen]++
		} else {
			zaehler[status]++
		}
		fmt.Fprintf(aus, "[%s] %s: %s", m.Gruppe, m.Name, anzeige)
		if !*trocken {
			fmt.Fprintf(aus, " (%.0f s)", time.Since(t).Seconds())
		}
		if len(rote) > 0 {
			fmt.Fprintf(aus, " rot: %s", strings.Join(rote, " "))
		}
		if detail != "" {
			fmt.Fprintf(aus, " — %s", detail)
		}
		if m.Aequivalent && m.Grund != "" {
			fmt.Fprintf(aus, " [Grund: %s]", m.Grund)
		}
		fmt.Fprintln(aus)
	}
	fmt.Fprintf(aus, "Summe: %d Mutanten, %d ROT, %d ÜBERLEBT, %d FEHLER, %d ÄQUIVALENT; zu beanstanden: %d\n",
		len(ms), zaehler[Rot], zaehler[Ueberlebt], zaehler[Fehler], zaehler[Aequivalen], schlecht)
	if schlecht > 0 {
		return 1
	}
	return 0
}
