# privatesync

Verschlüsselt Dateien aus einem Arbeitsverzeichnis in ein Sync-Verzeichnis (z.B. Dropbox, Nextcloud) und entschlüsselt sie zurück. Dateien werden mit AES-GCM verschlüsselt, der Schlüssel wird per bcrypt-Hash gesichert und in einer lokalen JSON-Datei gespeichert.

## Zweck

Sensible Dateien können so über unsichere Cloud-Sync-Dienste synchronisiert werden, ohne dass der Anbieter Zugriff auf den Klartext hat. Das Tool verschlüsselt alle Dateien aus dem Arbeitsverzeichnis und legt die verschlüsselten Kopien im Sync-Verzeichnis ab. Unterverzeichnisse werden dabei beibehalten.

```
Arbeitsverzeichnis          Sync-Verzeichnis (z.B. Dropbox)
├── dokument.pdf    →       ├── dokument..pdf.crypt
├── notizen.txt     →       ├── notizen..txt.crypt
└── bilder/         →       └── bilder/
    └── foto.jpg    →           └── foto..jpg.crypt
```

## Konfiguration

Beim ersten Start wird eine Datei `syncsettings.json` im aktuellen Verzeichnis angelegt:

```json
{
 "WorkPath": "/home/user/dokumente",
 "SyncPath": "/home/user/Dropbox/encrypted",
 "Secret": "$2a$14$..."
}
```

| Feld       | Beschreibung                                              |
|------------|-----------------------------------------------------------|
| `WorkPath` | Verzeichnis mit den Originaldateien (Klartext)            |
| `SyncPath` | Verzeichnis für die verschlüsselten Dateien (Cloud-Sync)  |
| `Secret`   | bcrypt-Hash des eingegebenen Passworts (wird automatisch gesetzt) |

Alternativ kann eine `.env`-Datei im Arbeitsverzeichnis angelegt werden.

## Bauen

Voraussetzung: Go 1.26+

```bash
# Abhängigkeiten laden
go mod download

# Binaries für Linux und Windows bauen
./build.sh

# Oder direkt ausführen
go run main.go crypt
```

Die Build-Ausgaben sind:
- `privatesync-linux-amd64`
- `privatesync-windows-amd64.exe`

## Verwendung

```bash
# Dateien verschlüsseln (WorkPath → SyncPath)
./privatesync-linux-amd64 crypt

# Dateien entschlüsseln (SyncPath → WorkPath)
./privatesync-linux-amd64 decrypt
```

Bei jedem Aufruf wird das Passwort interaktiv abgefragt und gegen den gespeicherten bcrypt-Hash geprüft.

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

Aktuelle Abhängigkeiten (`go.mod`):

| Paket | Verwendung |
|-------|------------|
| `github.com/joho/godotenv` | `.env`-Dateien laden |
| `golang.org/x/crypto` | bcrypt-Passwort-Hashing |
