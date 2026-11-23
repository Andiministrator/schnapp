// Mutationsprobe für Schnapp (F-371): Mutanten stehen als Daten in *.json, das Programm wendet je Mutant eine
// Ersetzung in einer KOPIE des Wurzelmoduls an und lässt das Paket testen. Der Arbeitsbaum bleibt unberührt.
// Aufruf und Format: README.md in diesem Ordner.
package main

import (
	"bytes"
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
	Timeout     string `json:"timeout"` // optional: -timeout für go test (z. B. "45s"); für Mutanten, die hängen statt rot zu werden (F-455)
	Gruppe      string `json:"-"`
}

// Ausgang eines Mutanten.
const (
	Rot        = "ROT"
	Ueberlebt  = "ÜBERLEBT"
	Fehler     = "FEHLER"
	Aequivalen = "ÄQUIVALENT"
	// Haengt: Das Paket endete am Test-Timeout, mindestens ein Test lief (F-478). Zählt wie ROT: Der Mutant ist bemerkt.
	Haengt = "HÄNGT"
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
		// Unbekannte Schlüssel (Tippfehler wie "timout") fielen sonst still weg (F-478).
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&ms); err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(n), err)
		}
		g := strings.TrimSuffix(filepath.Base(n), ".json")
		for i := range ms {
			ms[i].Gruppe = g
			if ms[i].Name == "" || ms[i].Datei == "" || ms[i].Paket == "" || ms[i].Alt == "" {
				return nil, fmt.Errorf("%s: Eintrag %d unvollständig (name, datei, paket, alt)", g, i+1)
			}
			// Ein ungültiger timeout bräche erst mitten im Lauf, als „kein FAIL-Name“ (F-478): schon beim Laden ablehnen.
			if ms[i].Timeout != "" {
				if d, err := time.ParseDuration(ms[i].Timeout); err != nil || d <= 0 {
					return nil, fmt.Errorf("%s: Eintrag %d: timeout %q ist keine positive Dauer (z. B. \"60s\")", g, i+1, ms[i].Timeout)
				}
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

	text, runErr := goTest(baum, lauf, m.Paket, m.Timeout)
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
		// Ein Test, der am Test-Timeout hängt, ist kein FAIL-Name, aber ein bemerkter Mutant (F-478).
		if laufend := haengendeTests(text); len(laufend) > 0 {
			return Haengt, laufend, ""
		}
		return Fehler, nil, "kein FAIL-Name: " + kurz(text)
	}
	return Rot, rote, ""
}

// haengendeTests liefert die Tests, die beim Abbruch durch das Test-Timeout liefen („running tests:“ in der Panik-Ausgabe
// von go test). Leer, wenn der Lauf nicht am Timeout endete oder kein Test gestartet war (Hänger in init oder TestMain
// sagt nichts über die Tests: FEHLER). F-478.
func haengendeTests(text string) []string {
	_, rest, ok := strings.Cut(text, "panic: test timed out after ")
	if !ok {
		return nil
	}
	_, rest, ok = strings.Cut(rest, "running tests:\n")
	if !ok {
		return nil
	}
	var namen []string
	for _, z := range strings.Split(rest, "\n") {
		if !strings.HasPrefix(z, "\t") {
			break
		}
		if f := strings.Fields(z); len(f) > 0 {
			namen = append(namen, f[0])
		}
	}
	return namen
}

// goTest testet paket in der Kopie baum.
func goTest(baum, lauf, paket, timeout string) (string, error) {
	args := []string{"test", "-count=1"}
	if timeout != "" {
		args = append(args, "-timeout", timeout)
	}
	cmd := exec.Command("go", append(args, paket)...)
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

// ausgangSchluessel kennzeichnet den Test-Aufruf eines Mutanten: Paket und timeout (F-489).
func ausgangSchluessel(m Mutant) string { return m.Paket + "\x00" + m.Timeout }

// jetzt ist die Uhr für den Laufordner; Tests setzen sie fest (F-495).
var jetzt = time.Now

// neuerLaufordner legt <wurzel>/tmp/mutanten/lauf-<Zeit>-<Zufall> neu an; eindeutig auch bei gleicher Sekunde (F-495).
func neuerLaufordner(wurzel string, t time.Time) (string, error) {
	eltern := filepath.Join(wurzel, "tmp", "mutanten")
	if err := os.MkdirAll(eltern, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(eltern, "lauf-"+t.Format("20060102-150405")+"-")
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
		// Eindeutiger Ordner auch bei zwei Läufen in derselben Sekunde (F-495).
		laufDir, err = neuerLaufordner(w, jetzt())
		if err != nil {
			fmt.Fprintln(aus, "Fehler:", err)
			return 2
		}
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

	// Ausgangslage: Jeder Test-Aufruf (Paket und timeout) muss ohne Mutation grün sein, sonst gälte jeder Mutant als
	// getötet (Sitzungs-QA, Q-3). Mit demselben timeout wie der Mutant: ein zu knapper timeout ließe sonst jeden Mutanten
	// als HÄNGT (rot) durchgehen (F-489). Einmal je Kombination.
	ausgangRot := map[string]string{}
	if !*trocken {
		for _, m := range ms {
			k := ausgangSchluessel(m)
			if _, gesehen := ausgangRot[k]; gesehen {
				continue
			}
			ausgangRot[k] = ""
			if text, err := goTest(baum, laufDir, m.Paket, m.Timeout); err != nil {
				ausgangRot[k] = "Ausgangslage rot (ohne Mutation)"
				if m.Timeout != "" {
					ausgangRot[k] += " mit timeout " + m.Timeout
					if strings.Contains(text, "panic: test timed out") {
						ausgangRot[k] += " (TIMEOUT-ZU-KNAPP: timeout in der Gruppendatei erhöhen)"
					}
				}
				ausgangRot[k] += ": " + kurz(text)
				fmt.Fprintf(aus, "Paket %s: %s\n", m.Paket, ausgangRot[k])
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
		} else if grund := ausgangRot[ausgangSchluessel(m)]; grund != "" {
			status, detail = Fehler, grund
		} else {
			status, rote, detail = testeMutant(baum, laufDir, m)
		}
		anzeige := status
		switch {
		case m.Aequivalent && status == Ueberlebt:
			anzeige = Aequivalen + " (überlebt erwartungsgemäß)"
		case m.Aequivalent && (status == Rot || status == Haengt):
			anzeige = status + " (als äquivalent markiert, Markierung prüfen)"
		case status == Haengt:
			anzeige = Haengt + " (Test-Timeout, zählt als rot)"
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
	fmt.Fprintf(aus, "Summe: %d Mutanten, %d ROT, %d HÄNGT (zählt als rot), %d ÜBERLEBT, %d FEHLER, %d ÄQUIVALENT; zu beanstanden: %d\n",
		len(ms), zaehler[Rot], zaehler[Haengt], zaehler[Ueberlebt], zaehler[Fehler], zaehler[Aequivalen], schlecht)
	if schlecht > 0 {
		return 1
	}
	return 0
}
