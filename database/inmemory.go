package database

import (
	"github.com/freewilll/splitter/ledger"
)

// userWithPassword is a database entry for a user
type userWithPassword struct {
	ID       int
	Email    string
	Password string
}

// InMemoryDatabase implements the Database interface for an in memory database
type InMemoryDatabase struct {
	users    []userWithPassword
	expenses []ledger.Expense
}

// InMemoryHandle implements the DatabaseHandle interface for an in memory database
type InMemoryHandle struct {
	db *InMemoryDatabase
}

// NewInMemoryDatabase creates an instance of InMemoryDatabase
func NewInMemoryDatabase() Database {
	db := new(InMemoryDatabase)
	db.users = make([]userWithPassword, 0)
	db.expenses = make([]ledger.Expense, 0)
	return db
}

// Connect creates a handle for the in memory database
func (d *InMemoryDatabase) Connect() Handle {
	h := new(InMemoryHandle)
	h.db = d
	return h
}

// Close is a noop
func (h *InMemoryHandle) Close() {}

// CreateSchema is a noop
func (h *InMemoryHandle) CreateSchema() {}

// CreateUser adds a user
func (h *InMemoryHandle) CreateUser(email string, password string) (int, error) {
	for _, u := range h.db.users {
		if u.Email == email {
			return 0, ErrDuplicate
		}
	}

	userID := len(h.db.users) + 1
	h.db.users = append(h.db.users, userWithPassword{Email: email, Password: password})
	return userID, nil
}

// AuthenticateUser isn't fully implemented. It always returns 1, nil.
func (h *InMemoryHandle) AuthenticateUser(email string, password string) (int, error) {
	return 1, nil
}

// GetUsers returns a list of all users
func (h *InMemoryHandle) GetUsers() []User {
	users := make([]User, 0)
	for i, u := range h.db.users {
		users = append(users, User{ID: i + 1, Email: u.Email})
	}
	return users
}

// CreateExpense creates an expense
func (h *InMemoryHandle) CreateExpense(expense ledger.Expense) {
	expense.Users = append(expense.Users, expense.OwnerID)
	expense.ExpenseID = len(h.db.expenses) + 1
	h.db.expenses = append(h.db.expenses, expense)
}

// GetExpenses returns a list of all expenses
func (h *InMemoryHandle) GetExpenses(userID int) []ledger.Expense {
	return h.db.expenses
}
