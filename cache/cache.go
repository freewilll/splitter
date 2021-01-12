package cache

import (
	"github.com/freewilll/splitter/database"
	"github.com/freewilll/splitter/ledger"
)

// Cache is an interface used for caching the ledger's balance
type Cache interface {
	SetBalance(balance ledger.Balance, userID int)
	GetBalance(db database.Database, userID int) ledger.Balance
}
