package application

import (
	store "../store"
)

/*Sync process containing workplace and syncspace*/
type Sync struct {
	Storage  *store.Storage
	SettingsFilePath string
}
