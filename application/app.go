package application

import (
	"io"

	store "github.com/wlanboy/privatesync/store"
)

/*Sync process containing workplace and syncspace*/
type Sync struct {
	Storage  *store.Storage
	Prompter store.Prompter
	Out      io.Writer
	ErrOut   io.Writer
}
