// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/notify"
)

// Uhr und Pause des Countdowns; Tests tauschen sie. Der letzte Hinweis schließt hinweisWeg (mcp.go, F-490) vor der
// Aufnahme, damit er nicht im Bild steht.
var (
	countdownJetzt   = time.Now
	countdownSchlaf  = time.Sleep
	countdownErsetze = func(ctx context.Context, id uint32, titel, text string) (uint32, error) {
		return countdownMitKnopf(ctx, id, titel, text, aktionAbbrechen, i18n.T("Cancel"))
	}
	countdownMitKnopf  = notify.ErsetzeMitAktion
	countdownSchliesse = notify.Schliesse
	// countdownAbo abonniert die Knöpfe der Hinweise (ADR-0042 Nachtrag 2); ein Fehler heißt: ohne „Abbrechen“.
	countdownAbo = notify.Aktionen
	// countdownWarte schläft d, außer ein Klick auf „Abbrechen“ kommt auf ab; liefert dann die Aktion. Ohne Abonnement
	// (ab nil) schläft es über countdownSchlaf, das Tests als Uhr tauschen.
	countdownWarte = func(d time.Duration, ab <-chan notify.Aktion) (notify.Aktion, bool) {
		if ab == nil {
			countdownSchlaf(d)
			return notify.Aktion{}, false
		}
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-t.C:
			return notify.Aktion{}, false
		case a := <-ab:
			return a, true
		}
	}
)

// aktionAbbrechen ist die Kennung des Knopfs „Cancel“ am Countdown-Hinweis (ADR-0042 Nachtrag 2).
const aktionAbbrechen = "cancel"

// hinweisFrist begrenzt jeden Aufruf des Benachrichtigungsdienstes im Countdown: Ein langsamer oder hängender Dienst
// darf die Wartezeit nicht verlängern (Kritik #338 K-1, F-938).
const hinweisFrist = 800 * time.Millisecond

// countdown wartet sek Sekunden vor der Aufnahme (ADR-0042) und zeigt die verbleibende Zeit als Benachrichtigung, die
// ersetzt wird, sobald sich die Zahl ändert. Die Zahl kommt aus der Restzeit, nicht aus einem Zähler (F-938). Die
// Wartezeit hängt nur an der Uhr: Jeder Aufruf hat hinweisFrist, und nach dem ersten Fehler gibt der Countdown die
// Hinweise auf und wartet nur noch. Der letzte Hinweis wird hinweisWeg vor dem Ende geschlossen; kam das Schließen
// später, wartet er hinweisWeg ab dem Schließen, damit der ausblendende Hinweis nicht im Bild steht (F-939). Ohne
// Benachrichtigungsdienst wird trotzdem gewartet; nur mit SCHNAPP_DEBUG steht das auf der Fehlerausgabe.
// Der Hinweis trägt „Cancel“: Ein Klick darauf schließt ihn, und countdown meldet true (abgebrochen, ADR-0042
// Nachtrag 2). Ohne Abonnement der Knöpfe wartet er wie zuvor.
func countdown(sek int, fehler io.Writer) (abgebrochen bool) {
	ende := countdownJetzt().Add(time.Duration(sek) * time.Second)
	schluss := ende.Add(-hinweisWeg)
	// Das Abonnement lebt bis zum Ende des Countdowns; hinweisFrist gilt nur für das Anmelden (F-1052).
	abo, aboEnde := context.WithCancel(context.Background())
	defer aboEnde()
	knoepfe, err := countdownAbo(abo, hinweisFrist)
	if err != nil {
		diagnose(fehler, "Countdown ohne Abbrechen: %v", err)
		knoepfe = nil
	}
	var id uint32
	gezeigt, aufgegeben := 0, false
	// warte schläft d; ein Klick auf „Abbrechen“ an diesem Hinweis beendet den Countdown.
	warte := func(d time.Duration) bool {
		for d > 0 {
			bis := countdownJetzt().Add(d)
			a, ok := countdownWarte(d, knoepfe)
			if !ok {
				return false
			}
			if id != 0 && a.ID == id && a.Schluessel == aktionAbbrechen {
				return true
			}
			d = bis.Sub(countdownJetzt()) // ein fremder Knopf: weiter bis zum selben Ende
		}
		return false
	}
	schliessen := func() {
		ctx, abbrechen := context.WithTimeout(context.Background(), hinweisFrist)
		if err := countdownSchliesse(ctx, id); err != nil {
			diagnose(fehler, "Countdown-Hinweis schließen: %v", err)
		}
		abbrechen()
	}
	for {
		jetzt := countdownJetzt()
		if !jetzt.Before(schluss) {
			break
		}
		n := int((ende.Sub(jetzt) + time.Second - 1) / time.Second) // aufgerundete Restzeit in Sekunden
		if !aufgegeben && n != gezeigt {
			ctx, abbrechen := context.WithTimeout(context.Background(), hinweisFrist)
			neu, err := countdownErsetze(ctx, id, i18n.T("Screenshot"),
				fmt.Sprintf(i18n.N("Capturing in %d second", "Capturing in %d seconds", n), n))
			abbrechen()
			if err != nil {
				diagnose(fehler, "Countdown-Hinweis: %v", err)
				aufgegeben = true
			} else {
				id, gezeigt = neu, n
			}
		}
		// Bis die Zahl wechselt, höchstens bis zum Schließen; liegt immer nach jetzt.
		bis := ende.Add(-time.Duration(n-1) * time.Second)
		if bis.After(schluss) {
			bis = schluss
		}
		if warte(bis.Sub(countdownJetzt())) {
			schliessen()
			return true
		}
	}
	if id != 0 {
		schliessen()
		if weg := countdownJetzt().Add(hinweisWeg); weg.After(ende) {
			ende = weg
		}
	}
	// Nach dem Schließen hat der Hinweis keinen Knopf mehr: nur noch warten.
	if d := ende.Sub(countdownJetzt()); d > 0 {
		countdownSchlaf(d)
	}
	return false
}
