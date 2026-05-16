package application

import (
	store "github.com/wlanboy/privatesync/store"
)

/*Sync process containing workplace and syncspace*/
type Sync struct {
	Storage  *store.Storage
	SettingsFilePath string
}
