# privatesync

privatesync ist ein Kommandozeilen-Tool, das Dateien aus einem Arbeitsverzeichnis verschlüsselt in ein Sync-Verzeichnis kopiert (z. B. Dropbox oder Nextcloud) und sie von dort wieder entschlüsselt. Der Cloud-Anbieter sieht nur verschlüsselte Dateien. Die Verzeichnisstruktur bleibt erhalten, nur die Dateinamen bekommen die Endung `.crypt`.

```
Arbeitsverzeichnis          Sync-Verzeichnis (z. B. Dropbox)
├── dokument.pdf    →       ├── dokument.pdf.crypt
├── notizen.txt     →       ├── notizen.txt.crypt
└── bilder/         →       └── bilder/
    └── foto.jpg    →           └── foto.jpg.crypt
```

## Funktionsweise

- Aus dem Passwort und einem zufälligen Salt leitet das Tool per scrypt einen AES-256-Schlüssel ab.
- Jede Datei wird in Blöcken von 64 KiB mit AES-256-GCM verschlüsselt. Ein Header am Dateianfang enthält Formatversion, Änderungszeit und Nonce-Präfix.
- Der relative Pfad der Datei fließt in die Authentifizierung ein. Wer eine `.crypt`-Datei umbenennt, verschiebt oder einzelne Blöcke vertauscht, bekommt beim Entschlüsseln einen Fehler.
- Unveränderte Dateien überspringt das Tool. Es vergleicht dafür Änderungszeit und Dateigröße.
- Zieldateien schreibt das Tool erst in eine temporäre Datei und benennt sie danach um. Ein abgebrochener Lauf hinterlässt keine halb geschriebenen Dateien.
- Arbeits- und Sync-Verzeichnis dürfen nicht identisch oder ineinander verschachtelt sein.

## Konfiguration

Beim ersten Start fragt das Tool beide Verzeichnisse und das Passwort ab und legt die Einstellungen an. Standardort ist das Konfigurationsverzeichnis des Benutzers:

| System  | Pfad                                                         |
|---------|--------------------------------------------------------------|
| Linux   | `~/.config/privatesync/syncsettings.json`                    |
| macOS   | `~/Library/Application Support/privatesync/syncsettings.json` |
| Windows | `%AppData%\privatesync\syncsettings.json`                    |

Mit `--config <datei>` lässt sich ein anderer Pfad angeben.

```json
{
 "WorkPath": "/home/user/dokumente",
 "SyncPath": "/home/user/Dropbox/encrypted",
 "Secret": "$2a$14$...",
 "Salt": "..."
}
```

| Feld       | Beschreibung                                                     |
|------------|------------------------------------------------------------------|
| `WorkPath` | Verzeichnis mit den Originaldateien (Klartext), `~` ist erlaubt  |
| `SyncPath` | Verzeichnis für die verschlüsselten Dateien, `~` ist erlaubt     |
| `Secret`   | bcrypt-Hash des Passworts, prüft die Eingabe beim Start          |
| `Salt`     | Zufälliger Salt (Base64) für die Schlüsselableitung              |

Die Datei enthält nur den Hash, nicht das Passwort. Das Tool legt sie mit den Rechten `0600` an. Geht der Salt verloren, lassen sich die verschlüsselten Dateien nicht mehr entschlüsseln. Die Datei gehört deshalb ins Backup, aber nicht in das Sync-Verzeichnis.

Liegt eine `syncsettings.json` aus einer älteren Version im aktuellen Verzeichnis und am Standardort noch keine, verwendet das Tool die alte Datei und gibt einen Hinweis aus. Diese Datei sollte an den Standardort verschoben werden.

## Verwendung

```bash
# Dateien verschlüsseln (WorkPath → SyncPath)
privatesync crypt

# Zusätzlich .crypt-Dateien löschen, deren Quelldatei nicht mehr existiert
privatesync crypt --prune

# Dateien entschlüsseln (SyncPath → WorkPath)
privatesync decrypt

# Auch lokale Dateien überschreiben, die neuer sind als die verschlüsselte Kopie
privatesync decrypt --force

# Andere Einstellungsdatei verwenden
privatesync crypt --config ~/privatesync-arbeit.json
```

Bei jedem Aufruf fragt das Tool das Passwort ab und prüft es gegen den gespeicherten Hash. Im Terminal erscheint die Eingabe nicht auf dem Bildschirm.

Ohne `--force` überschreibt `decrypt` keine lokale Datei, die neuer ist als die verschlüsselte Kopie. Solche Dateien meldet das Tool als übersprungen. `--prune` läuft nur, wenn beim Verschlüsseln kein Fehler aufgetreten ist.

Am Ende gibt das Tool eine Zusammenfassung aus, z. B. `crypt: 3 written, 120 unchanged`. Der Exit-Code ist `0` bei Erfolg, `1` bei Fehlern (auch wenn einzelne Dateien fehlgeschlagen sind) und `2` bei falschen Argumenten.

## Grenzen

- Im Sync-Verzeichnis sind Dateinamen, Verzeichnisstruktur und ungefähre Dateigrößen im Klartext lesbar. Das Tool verschlüsselt nur den Inhalt.
- Das Tool merkt sich keinen Sync-Zustand. Wurde eine Datei lokal und auf einem anderen Rechner geändert, entscheidet nur die Änderungszeit: Ist die lokale Datei älter, überschreibt `decrypt` sie.
- `crypt --prune` löscht jede `.crypt`-Datei ohne lokale Quelldatei, auch wenn ein anderer Rechner sie gerade erst hochgeladen hat. Auf einem Rechner, der noch nicht entschlüsselt hat, sollte deshalb zuerst `decrypt` laufen.

## Bauen

Voraussetzung: Go 1.27+

```bash
# Abhängigkeiten laden
go mod download

# Binaries für alle Zielplattformen bauen
./build.sh

# Oder direkt ausführen
go run . crypt
```

`build.sh` erzeugt statisch gelinkte Binaries:

- `privatesync-linux-amd64`
- `privatesync-linux-arm64`
- `privatesync-darwin-arm64`
- `privatesync-windows-amd64.exe`

## Tests

```bash
go test ./...
```

## Abhängigkeiten aktualisieren

```bash
# Alle Abhängigkeiten auf neueste Minor/Patch-Versionen aktualisieren
go get -u ./...

# go.sum neu generieren und nicht mehr benötigte Einträge entfernen
go mod tidy

# Änderungen prüfen
go mod diff

# Build und Tests sicherstellen
go build ./...
go test ./...
```

Verwendete Pakete:

| Paket                 | Verwendung                                                      |
|-----------------------|-----------------------------------------------------------------|
| `golang.org/x/crypto` | bcrypt für den Passwort-Hash, scrypt für die Schlüsselableitung |
| `golang.org/x/term`   | Passworteingabe ohne Anzeige im Terminal                        |
