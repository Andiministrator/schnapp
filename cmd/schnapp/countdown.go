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

// countdown wartet sek Sekunden vor der Aufnahme (ADR-0042) und zeigt die verbleibende Zeit als Benachrichtigung, die
// jede Sekunde ersetzt und vor der Aufnahme geschlossen wird. Scheitert der Hinweis (kein Benachrichtigungsdienst),
// wird trotzdem gewartet; nur mit SCHNAPP_DEBUG steht das auf der Fehlerausgabe.
func countdown(sek int, fehler io.Writer) {
	ctx := context.Background()
	ende := countdownJetzt().Add(time.Duration(sek) * time.Second)
	var id uint32
	for n := sek; n > 0; n-- {
		neu, err := countdownErsetze(ctx, id, i18n.T("Screenshot"),
			fmt.Sprintf(i18n.N("Capturing in %d second", "Capturing in %d seconds", n), n))
		if err != nil {
			diagnose(fehler, "Countdown-Hinweis: %v", err)
		} else {
			id = neu
		}
		// Die nächste Sekunde, für die letzte der Zeitpunkt, an dem der Hinweis weg sein muss.
		bis := ende.Add(-time.Duration(n-1) * time.Second)
		if n == 1 {
			bis = ende.Add(-hinweisWeg)
		}
		if d := bis.Sub(countdownJetzt()); d > 0 {
			countdownSchlaf(d)
		}
	}
	if id != 0 {
		if err := countdownSchliesse(ctx, id); err != nil {
			diagnose(fehler, "Countdown-Hinweis schließen: %v", err)
		}
	}
	if d := ende.Sub(countdownJetzt()); d > 0 {
		countdownSchlaf(d)
	}
}
