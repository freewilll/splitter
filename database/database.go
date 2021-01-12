package database

import (
	"github.com/freewilll/splitter/ledger"
)

// User represents a user present in the database
type User struct {
	ID    int
	Email string
}

// Database is an interface that does nothing more than return a database handle
// It is used to configure different types of databases
type Database interface {
	Connect() Handle
}

// Handle is an interface containng methods to manage a database handle
// and perform user, ledger and expenses queries on it.
type Handle interface {
	Close()                                                      // Close the database handle
	CreateSchema()                                               // Create the database schema
	CreateUser(email string, password string) (int, error)       // Create a user
	AuthenticateUser(email string, password string) (int, error) // Authenticate a user
	GetUsers() []User                                            // Get a slice of all users
	CreateExpense(e ledger.Expense)                              // Create an expense entry
	GetExpenses(userID int) []ledger.Expense                     // Get a slice of all exepnses
}
