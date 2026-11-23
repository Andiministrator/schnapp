// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Tests des Dialog-Kindes (ADR-0030 Nachtrag 4 Punkt 5, F-469, F-471): Das Testprogramm selbst spielt das Kind,
// gesteuert über eine Umgebungsvariable (wie bei Go-Tests mit TestMain üblich); die Kind-Seite läuft mit einer
// Attrappe des Dialogs.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// dialogKindVariable wählt das Verhalten des Testprogramms, wenn es als Dialog-Kind läuft.
const dialogKindVariable = "SCHNAPP_TEST_DIALOGKIND"

// dialogKindAttrappe läuft am Anfang von TestMain. Ist dialogKindVariable gesetzt, ist dieser Prozess das Kind eines
// Tests: Er spielt den Dialog und endet, ohne Tests zu fahren.
func dialogKindAttrappe() {
	art := os.Getenv(dialogKindVariable)
	if art == "" {
		return
	}
	switch art {
	case "erlauben":
		os.Exit(exitOK)
	case "ablehnen":
		os.Exit(exitFehler)
	case "haengen":
		time.Sleep(60 * time.Second)
		os.Exit(exitOK)
	case "stdout":
		fmt.Fprintln(os.Stdout, "PROTOKOLLMUELL")
		os.Exit(exitOK)
	case "pruefe":
		var b bestaetigung
		zeile, _ := bufio.NewReader(os.Stdin).ReadBytes('\n')
		if json.Unmarshal(zeile, &b) != nil || b.Ziel != "monitor DP-1" || b.Kunde != "Agent" || len(os.Args) != 2 ||
			os.Args[1] != bestaetigenArgument {
			os.Exit(exitFehler)
		}
		os.Exit(exitOK)
	case "umgebung": // F-491: die geprüfte Verbindung als Deskriptor 3 (WAYLAND_SOCKET), kein zweites Auflösen
		if os.Getenv("WAYLAND_SOCKET") != "3" {
			os.Exit(exitFehler)
		}
		b := make([]byte, 1) // die Attrappe der Verbindung trägt ein „S“: genau diese ist geerbt, kein zufälliger fd 3
		if n, err := os.NewFile(3, "wayland").Read(b); err != nil || n != 1 || b[0] != 'S' {
			os.Exit(exitFehler)
		}
		os.Exit(exitOK)
	}
	os.Exit(99)
}

func mitKind(t *testing.T, art string) {
	t.Helper()
	t.Setenv(dialogKindVariable, art)
	alt := kindProgramm
	kindProgramm = os.Executable
	t.Cleanup(func() { kindProgramm = alt })
	sitzungAttrappe(t, 4711, 4711)
	t.Setenv("XDG_RUNTIME_DIR", "/lauf")
	t.Setenv("WAYLAND_DISPLAY", "wayland-probe")
	t.Setenv("WAYLAND_SOCKET", "")
}

// sitzungAttrappe ersetzt die Abfragen der Sitzungsbindung (F-491): Prozess hinter dem Wayland-Socket (die
// „Verbindung“ ist eine Pipe), Besitzer von org.kde.KWin und dessen Elternprozess (kein Elternteil: 1).
func sitzungAttrappe(t *testing.T, wayland, kwin int32) {
	t.Helper()
	altW, altK, altE := waylandVerbinde, kwinPid, elternVon
	waylandVerbinde = func(string) (*os.File, int32, error) {
		r, w, err := os.Pipe()
		if err != nil {
			return nil, 0, err
		}
		w.Write([]byte("S"))
		w.Close()
		return r, wayland, nil
	}
	kwinPid = func(context.Context) (int32, error) { return kwin, nil }
	elternVon = func(int32) int32 { return 1 }
	t.Cleanup(func() { waylandVerbinde, kwinPid, elternVon = altW, altK, altE })
}

// TestSitzungMitWrapper: In Plasma hält kwin_wayland_wrapper den Socket und ist Elternprozess von kwin_wayland
// (Sitzungs-QA A-1): angenommen. Ein anderer Elternprozess bleibt fremd.
func TestSitzungMitWrapper(t *testing.T) {
	sitzungAttrappe(t, 1679, 1774)
	t.Setenv("XDG_RUNTIME_DIR", "/lauf")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("WAYLAND_SOCKET", "")
	elternVon = func(p int32) int32 {
		if p == 1774 {
			return 1679
		}
		return 1
	}
	f, err := pruefeSitzung(context.Background(), os.Getenv)
	if err != nil {
		t.Fatalf("Wrapper als Elternprozess: %v", err)
	}
	f.Close()
	elternVon = func(int32) int32 { return 4242 }
	if _, err := pruefeSitzung(context.Background(), os.Getenv); !errors.Is(err, errFremdeSitzung) {
		t.Fatalf("fremder Elternprozess: %v", err)
	}
}

// TestElternProzess: der Elternprozess des Tests stimmt mit os.Getppid überein; ein Programmname mit Leerzeichen und
// Klammern verwirrt das Lesen nicht (es nimmt das Feld nach der letzten „)“).
func TestElternProzess(t *testing.T) {
	if got := elternProzess(int32(os.Getpid())); int(got) != os.Getppid() {
		t.Fatalf("Eltern %d, erwartet %d", got, os.Getppid())
	}
	if got := elternProzess(-1); got != 0 {
		t.Errorf("unbekannter Prozess: %d", got)
	}
}

// TestKindNurAufDerAufnehmendenSitzung: Bedient ein anderer Prozess den Wayland-Socket als der, dem KWin auf dem
// Sitzungsbus gehört (noch dessen Elternprozess), startet kein Dialog und es gilt nicht als erlaubt (F-491). Das Kind
// bekommt die geprüfte Verbindung als Deskriptor 3; ist WAYLAND_SOCKET schon gesetzt, startet kein Dialog.
func TestKindNurAufDerAufnehmendenSitzung(t *testing.T) {
	mitKind(t, "erlauben")
	sitzungAttrappe(t, 100, 200)
	ok, err := bestaetigeImKind(context.Background(), "monitor DP-1", "Agent")
	if ok || !errors.Is(err, errFremdeSitzung) {
		t.Fatalf("fremder Compositor: %v, %v; erwartet abgelehnt mit errFremdeSitzung", ok, err)
	}
	mitKind(t, "umgebung")
	if ok, err := bestaetigeImKind(context.Background(), "monitor DP-1", "Agent"); !ok || err != nil {
		t.Fatalf("Umgebung des Kindes falsch: %v, %v", ok, err)
	}
	t.Setenv("WAYLAND_SOCKET", "5")
	if ok, err := bestaetigeImKind(context.Background(), "monitor DP-1", "Agent"); ok || err == nil {
		t.Fatalf("mit WAYLAND_SOCKET: %v, %v; erwartet Fehler", ok, err)
	}
}

// TestWaylandSocketPfad: WAYLAND_DISPLAY relativ zu XDG_RUNTIME_DIR, absolut, Voreinstellung wayland-0.
func TestWaylandSocketPfad(t *testing.T) {
	for _, c := range []struct {
		env  map[string]string
		want string
		ok   bool
	}{
		{map[string]string{"XDG_RUNTIME_DIR": "/lauf", "WAYLAND_DISPLAY": "wayland-1"}, "/lauf/wayland-1", true},
		{map[string]string{"XDG_RUNTIME_DIR": "/lauf"}, "/lauf/wayland-0", true},
		{map[string]string{"WAYLAND_DISPLAY": "/anders/w"}, "/anders/w", true},
		{map[string]string{"WAYLAND_DISPLAY": "wayland-1"}, "", false},
		{map[string]string{"XDG_RUNTIME_DIR": "rel", "WAYLAND_DISPLAY": "wayland-1"}, "", false},
		{map[string]string{"XDG_RUNTIME_DIR": "/lauf", "WAYLAND_SOCKET": "3"}, "", false},
	} {
		got, err := waylandSocketPfad(func(k string) string { return c.env[k] })
		if (err == nil) != c.ok || got != c.want {
			t.Errorf("%v: %q, %v", c.env, got, err)
		}
	}
}

// TestWaylandVerbindung: SO_PEERCRED liefert den eigenen Prozess, wenn der Test selbst lauscht, und die Verbindung
// bleibt als Datei offen.
func TestWaylandVerbindung(t *testing.T) {
	t.Chdir(t.TempDir()) // relativ: Ein langer TMPDIR (Mutantenlauf) überschritte die 108 Zeichen eines Socket-Pfads
	socket := "w"
	l, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		if c, err := l.Accept(); err == nil {
			c.Close()
		}
	}()
	f, pid, err := waylandVerbindung(socket)
	if err != nil || int(pid) != os.Getpid() {
		t.Fatalf("Pid %d, %v; erwartet %d", pid, err, os.Getpid())
	}
	defer f.Close()
	if _, err := f.Stat(); err != nil {
		t.Errorf("Verbindung nicht offen: %v", err)
	}
}

// TestKindNurExit0Erlaubt: Erlaubt ist nur Exit 0; ein anderer Exit lehnt ab (Mutant c11).
func TestKindNurExit0Erlaubt(t *testing.T) {
	for art, want := range map[string]bool{"erlauben": true, "ablehnen": false, "pruefe": true} {
		mitKind(t, art)
		ok, err := bestaetigeImKind(context.Background(), "monitor DP-1", "Agent")
		if err != nil || ok != want {
			t.Errorf("%s: %v, %v; erwartet %v", art, ok, err, want)
		}
	}
}

// TestKindFristLehntAb: Antwortet das Kind nicht vor der Frist, beendet der Server es und lehnt ab (Mutant c10).
func TestKindFristLehntAb(t *testing.T) {
	mitKind(t, "haengen")
	alt := bestaetigenFrist
	bestaetigenFrist = 300 * time.Millisecond
	t.Cleanup(func() { bestaetigenFrist = alt })
	start := time.Now()
	ok, err := bestaetigeImKind(context.Background(), "monitor DP-1", "Agent")
	if ok || err != nil {
		t.Errorf("Frist: %v, %v; erwartet abgelehnt", ok, err)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("Frist nicht eingehalten: %v", d)
	}
	// Abbruch des Servers wirkt ebenso.
	ctx, abbrechen := context.WithCancel(context.Background())
	abbrechen()
	if ok, _ := bestaetigeImKind(ctx, "monitor DP-1", "Agent"); ok {
		t.Error("abgebrochener Server: erlaubt")
	}
}

// TestKindOhneProgramm: Lässt sich das Kind nicht starten, ist das ein Fehler, keine Erlaubnis.
func TestKindOhneProgramm(t *testing.T) {
	alt := kindProgramm
	kindProgramm = func() (string, error) { return "", errors.New("kein Programm") }
	t.Cleanup(func() { kindProgramm = alt })
	if ok, err := bestaetigeImKind(context.Background(), "z", "k"); ok || err == nil {
		t.Errorf("%v, %v", ok, err)
	}
}

// TestKindStdoutNieImProtokoll: Was das Kind auf stdout schreibt, landet auf stderr des Servers, nie auf seiner
// stdout (dort steht das Protokoll; Mutant c9).
func TestKindStdoutNieImProtokoll(t *testing.T) {
	mitKind(t, "stdout")
	ausL, ausS, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	fehlerL, fehlerS, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	altAus, altFehler := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = ausS, fehlerS
	ok, kerr := bestaetigeImKind(context.Background(), "monitor DP-1", "Agent")
	os.Stdout, os.Stderr = altAus, altFehler
	ausS.Close()
	fehlerS.Close()
	aus, _ := io.ReadAll(ausL)
	fehler, _ := io.ReadAll(fehlerL)
	ausL.Close()
	fehlerL.Close()
	if !ok || kerr != nil {
		t.Fatalf("Kind: %v, %v", ok, kerr)
	}
	if len(aus) != 0 {
		t.Errorf("Kind schrieb auf die stdout des Servers: %q", aus)
	}
	if !strings.Contains(string(fehler), "PROTOKOLLMUELL") {
		t.Errorf("Ausgabe des Kindes nicht auf stderr: %q", fehler)
	}
}

// dialogAttrappe ersetzt den Dialog der Kind-Seite.
func dialogAttrappe(t *testing.T, f func(ctx context.Context, ziel, kunde string) (bool, error)) {
	t.Helper()
	alt := zeigeDialog
	zeigeDialog = f
	t.Cleanup(func() { zeigeDialog = alt })
}

func ohneProbe(t *testing.T) {
	t.Helper()
	if probeGuard() != nil {
		t.Skip("Probe-Build außerhalb des virtuellen KWin: kein Dialog (F-229)")
	}
}

// TestKindSeite: Exit 0 nur, wenn der Dialog erlaubt (Mutant c12); Ziel und Name kommen maskiert an (F-463).
func TestKindSeite(t *testing.T) {
	ohneProbe(t)
	var gesehen []string
	for _, c := range []struct {
		ok   bool
		err  error
		want int
	}{{true, nil, exitOK}, {false, nil, exitFehler}, {true, errors.New("kein Wayland"), exitFehler}} {
		dialogAttrappe(t, func(_ context.Context, ziel, kunde string) (bool, error) {
			gesehen = append(gesehen, ziel+"|"+kunde)
			return c.ok, c.err
		})
		ein := `{"target":"monitor a\nb\u001b[2J","client":"x‮y\nz"}` + "\n"
		var fehler strings.Builder
		if code := laufeBestaetigung(strings.NewReader(ein), &fehler); code != c.want {
			t.Errorf("Dialog %v/%v: Exit %d, erwartet %d", c.ok, c.err, code, c.want)
		}
	}
	for _, g := range gesehen {
		if strings.ContainsAny(g, "\n\x1b‮") {
			t.Errorf("Steuerzeichen im Dialog: %q", g)
		}
	}
}

// TestKindEigeneFrist: Das Kind schließt den Dialog nach seiner Frist selbst und lehnt ab, auch wenn der Dialog danach
// „erlaubt“ meldet; kehrt er gar nicht zurück, beendet es sich hart (F-469).
func TestKindEigeneFrist(t *testing.T) {
	ohneProbe(t)
	altF, altZ, altH, altB := bestaetigenFrist, kindZugabe, kindHart, beendeProzess
	bestaetigenFrist, kindZugabe, kindHart = 20*time.Millisecond, 20*time.Millisecond, 500*time.Millisecond
	var hart atomic.Int32
	beendet := make(chan struct{}, 1)
	beendeProzess = func(code int) { hart.Store(int32(code) + 100); beendet <- struct{}{} }
	t.Cleanup(func() { bestaetigenFrist, kindZugabe, kindHart, beendeProzess = altF, altZ, altH, altB })

	dialogAttrappe(t, func(ctx context.Context, _, _ string) (bool, error) {
		select { // ohne Frist des Kindes käme ctx nie: dann nach 3 s „erlaubt“ (Mutant M43 rot statt hängend)
		case <-ctx.Done():
		case <-time.After(3 * time.Second):
		}
		return true, nil // der Dialog meldet zu spät „erlaubt“
	})
	var fehler strings.Builder
	if code := laufeBestaetigung(strings.NewReader(`{"target":"z"}`+"\n"), &fehler); code != exitFehler {
		t.Errorf("nach der eigenen Frist: Exit %d, erwartet %d", code, exitFehler)
	}
	if hart.Load() != 0 {
		t.Error("hart beendet, obwohl der Dialog zurückkam")
	}

	frei := make(chan struct{})
	dialogAttrappe(t, func(context.Context, string, string) (bool, error) {
		<-frei // hängt, bis der harte Abbruch kam
		return true, nil
	})
	fertig := make(chan int, 1)
	go func() { fertig <- laufeBestaetigung(strings.NewReader(`{"target":"z"}`+"\n"), io.Discard) }()
	select {
	case <-beendet:
	case <-time.After(5 * time.Second):
		t.Fatal("kein harter Abbruch")
	}
	if hart.Load() != int32(exitFehler)+100 {
		t.Errorf("harter Abbruch mit %d, erwartet Exit %d", hart.Load()-100, exitFehler)
	}
	close(frei)
	<-fertig
}
