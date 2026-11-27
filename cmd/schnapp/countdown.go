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
	countdownJetzt     = time.Now
	countdownSchlaf    = time.Sleep
	countdownErsetze   = notify.Ersetze
	countdownSchliesse = notify.Schliesse
)

// hinweisFrist begrenzt jeden Aufruf des Benachrichtigungsdienstes im Countdown: Ein langsamer oder hängender Dienst
// darf die Wartezeit nicht verlängern (Kritik #338 K-1, F-938).
const hinweisFrist = 800 * time.Millisecond

// countdown wartet sek Sekunden vor der Aufnahme (ADR-0042) und zeigt die verbleibende Zeit als Benachrichtigung, die
// ersetzt wird, sobald sich die Zahl ändert. Die Zahl kommt aus der Restzeit, nicht aus einem Zähler (F-938). Die
// Wartezeit hängt nur an der Uhr: Jeder Aufruf hat hinweisFrist, und nach dem ersten Fehler gibt der Countdown die
// Hinweise auf und wartet nur noch. Der letzte Hinweis wird hinweisWeg vor dem Ende geschlossen; kam das Schließen
// später, wartet er hinweisWeg ab dem Schließen, damit der ausblendende Hinweis nicht im Bild steht (F-939). Ohne
// Benachrichtigungsdienst wird trotzdem gewartet; nur mit SCHNAPP_DEBUG steht das auf der Fehlerausgabe.
func countdown(sek int, fehler io.Writer) {
	ende := countdownJetzt().Add(time.Duration(sek) * time.Second)
	schluss := ende.Add(-hinweisWeg)
	var id uint32
	gezeigt, aufgegeben := 0, false
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
		if d := bis.Sub(countdownJetzt()); d > 0 {
			countdownSchlaf(d)
		}
	}
	if id != 0 {
		ctx, abbrechen := context.WithTimeout(context.Background(), hinweisFrist)
		if err := countdownSchliesse(ctx, id); err != nil {
			diagnose(fehler, "Countdown-Hinweis schließen: %v", err)
		}
		abbrechen()
		if weg := countdownJetzt().Add(hinweisWeg); weg.After(ende) {
			ende = weg
		}
	}
	if d := ende.Sub(countdownJetzt()); d > 0 {
		countdownSchlaf(d)
	}
}
