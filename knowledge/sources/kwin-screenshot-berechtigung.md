# KWin: Wegfall der ScreenShot2-Berechtigung und ScreenShot3 (memfd)

- URL: https://invent.kde.org/plasma/kwin — Commits `0ea31ae72eb0` und `5be042886b64`, gelesen über die GitLab-API
  (`/api/v4/projects/plasma%2Fkwin/repository/commits/<sha>`, `/diff`, `/refs?type=tag`)
- Abgerufen: 2026-11-19
- Herausgeber / Stand: KDE, KWin master vom 19.11.2026; Commit-Autor jeweils der KWin-Maintainer
- Verlässlichkeit: offiziell (Quelltext des Herstellers)

## Kernaussagen

- `0ea31ae7`, 18.06.2026, „plugins/screenshot: Drop X-KDE-DBUS-Restricted-Interfaces“: entfernt
  `checkPermissions()` aus `screenshotdbusinterface2.cpp`, den Fehler `org.kde.KWin.ScreenShot2.Error.NoAuthorized`
  und die Abhängigkeit `KF6::Service`; die Doku im XML verliert den Satz zur Berechtigung. Begründung (Zitat):
  „The desktop files can be modified or overriden by the user“.
- Der entfernte Code: Umgebungsvariable `KWIN_SCREENSHOT_NO_PERMISSION_CHECKS=1` schaltete die Prüfung ab; sonst
  PID des Aufrufers → `fetchRestrictedDBusInterfacesFromPid` → muss `org.kde.KWin.ScreenShot2` enthalten.
- Der Commit ist in den Tags `v6.7.90`, `v6.7.91`, `v6.8.0` bis `v6.8.3` enthalten → ab Plasma 6.8 keine Prüfung.
- `5be04288`, 17.11.2026, „plugins/screenshot: Share screenshots with memfd“: neues Interface
  `org.kde.KWin.ScreenShot3` (`screenshotdbusinterface3.cpp`, XML), Methoden geben `fd h` (memfd) zurück; ScreenShot2
  bleibt, intern auf einen gemeinsamen Puffer umgestellt. Laut Commit 4–6 ms (Desktop) bzw. 12–13 ms (Laptop)
  Kopierzeit gespart. Zum Abrufzeitpunkt in keinem Tag.

## Offene Punkte

- Mit welcher Plasma-Version ScreenShot3 ausgeliefert wird (vermutlich 6.9) — bei der Umsetzung der Aufnahmeschicht
  prüfen.
- Ob ScreenShot3 eine eigene Berechtigung einführt — im XML steht keine.
