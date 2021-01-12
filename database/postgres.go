package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/freewilll/splitter/ledger"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Database schema, to be run once
const schema = `
CREATE TABLE users (
	id 			SERIAL PRIMARY KEY,
	email 		TEXT NOT NULL UNIQUE,
	password 	TEXT
);

CREATE TABLE expenses (
	id 			SERIAL PRIMARY KEY,
	user_id 	INT NOT NULL REFERENCES users,
	description TEXT NOT NULL,
	amount 		DOUBLE PRECISION NOT NULL,
	created_at 	TIMESTAMP NOT NULL
);

CREATE INDEX expenses_user_id ON expenses(user_id);

CREATE TABLE expenses_users (
	expense_id INT NOT NULL REFERENCES expenses,
	user_id INT NOT NULL REFERENCES users
);

CREATE INDEX expenses_users_expense_id ON expenses_users(expense_id);
CREATE INDEX expenses_users_user_id ON expenses_users(user_id);
CREATE UNIQUE INDEX expenses_users_unique_id ON expenses_users(expense_id, user_id);

-- Create three test users with password "secret"
INSERT INTO users (email, password) VALUES('test1@getstream.io', '$2a$08$NNqRkMg.vGfhnvtyrsfVN.uTndun9TuctRpxs5k5NTHjcXybPTQAa');
INSERT INTO users (email, password) VALUES('test2@getstream.io', '$2a$08$NNqRkMg.vGfhnvtyrsfVN.uTndun9TuctRpxs5k5NTHjcXybPTQAa');
INSERT INTO users (email, password) VALUES('test3@getstream.io', '$2a$08$NNqRkMg.vGfhnvtyrsfVN.uTndun9TuctRpxs5k5NTHjcXybPTQAa');
`

// ErrDuplicate is returned when create request fails due to a duplicate entry
var ErrDuplicate = errors.New("Duplicate")

// ErrNotFound is returned when an entry could not be found
var ErrNotFound = errors.New("Not found")

// ErrPasswordMismatch is returned when authentication fails due to a bad password
var ErrPasswordMismatch = errors.New("Password mismatch")

// Config holds the configuration for the postgresql database
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

// PgDatabase implements the Database interface for postgresql
type PgDatabase struct {
	config Config
}

// PgHandle implements the DatabaseHandle interface for postgresql
type PgHandle struct {
	db *sql.DB
}

// NewPgDatabase creates an instance of PgDatabase
func NewPgDatabase(config Config) PgDatabase {
	return PgDatabase{config: config}
}

// Connect creates a connection to the postgres database
func (d PgDatabase) Connect() Handle {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		d.config.Host, d.config.Port, d.config.User, d.config.Password, d.config.Name)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	dbh := new(PgHandle)
	dbh.db = db

	return dbh
}

// Close closes the database handle
func (p PgHandle) Close() {
	p.db.Close()
}

// CreateSchema connects runs SQL to create the schema. This is required to bootstrap
// the database.
func (p PgHandle) CreateSchema() {
	log.Print("Creating database schema")
	_, err := p.db.Exec(schema)
	if err != nil {
		panic(err)
	}
}

// CreateUser inserts a new user into the database. ErrDuplicate is returned
// if another user with the same email already exists.
func (p PgHandle) CreateUser(email string, password string) (int, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		panic(err)
	}

	var id int
	err = p.db.QueryRow(`
        INSERT INTO users (email, password)
        VALUES($1, $2)
        RETURNING id
    `, email, hashedPassword).Scan(&id)
	if err != nil {
		pqErr := err.(*pq.Error)
		switch pqErr.Code.Name() {
		case "unique_violation":
			return 0, ErrDuplicate
		default:
			panic(err)
		}
	}

	return id, nil
}

// AuthenticateUser checks if the user with email/password exists in the database
// and the password matches. ErrNotFound if the user doesn't exist. ErrPasswordMismatch
// is returned if the password mismatches.
func (p PgHandle) AuthenticateUser(email string, password string) (int, error) {
	var dbID int
	var dbPassword string
	err := p.db.QueryRow("SELECT id, password FROM users WHERE email=$1", email).Scan(&dbID, &dbPassword)
	if err != nil {
		log.Printf("Unknown user '%s'", email)
		return 0, ErrNotFound
	}

	if err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password)); err != nil {
		return 0, ErrPasswordMismatch
	}

	return dbID, nil
}

// GetUsers returns all users in the database, ordered by email
func (p PgHandle) GetUsers() []User {
	rows, err := p.db.Query("SELECT id, email FROM users ORDER BY email")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var id int
		var email string
		if err := rows.Scan(&id, &email); err != nil {
			panic(err)
		}
		users = append(users, User{id, email})
	}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	return users
}

// CreateExpense creates entries in the expenses and expenses_users tables.
// The expenses_users tables also includes the owner
func (p PgHandle) CreateExpense(e ledger.Expense) {
	// Insert into expenses and expense_users in a transaction to ensure consistency
	txn, err := p.db.Begin()
	if err != nil {
		panic(err)
	}

	// Insert into expenses
	var expenseID int
	err = p.db.QueryRow(`
        INSERT INTO expenses (user_id, description, amount, created_at)
        VALUES($1, $2, $3, $4)
        RETURNING id
    `, e.OwnerID, e.Description, e.Amount, e.CreatedAt).Scan(&expenseID)
	if err != nil {
		panic(err)
	}

	// Insert into expenses_users
	stmt, err := txn.Prepare(`
        INSERT INTO expenses_users (expense_id, user_id)
        VALUES($1, $2)
    `)
	if err != nil {
		log.Fatal(err)
	}

	// Insert self into user list
	_, err = stmt.Exec(expenseID, e.OwnerID)
	if err != nil {
		panic(err)
	}

	// Insert other users to user list
	for _, u := range e.Users {
		_, err = stmt.Exec(expenseID, u)
		if err != nil {
			panic(err)
		}
	}

	err = txn.Commit()
	if err != nil {
		panic(err)
	}
}

// GetExpenses returns all expenses in the database in order of expense_id and
// created_at
func (p PgHandle) GetExpenses(userID int) []ledger.Expense {
	rows, err := p.db.Query(`
	       SELECT e.id, e.user_id, ue.user_id, e.description, e.amount, e.created_at
	       FROM expenses e JOIN expenses_users ue ON (e.id = ue.expense_id)
	       ORDER BY expense_id, created_at
	   `)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	expensesMap := make(map[int]*ledger.Expense)
	for rows.Next() {
		var expenseID int
		var ownerID int
		var userID int
		var amount float64
		var description string
		var rawCreatedAt string
		if err := rows.Scan(&expenseID, &ownerID, &userID, &description, &amount, &rawCreatedAt); err != nil {
			panic(err)
		}

		createdAt, err := time.Parse(time.RFC3339, rawCreatedAt)
		if err != nil {
			panic(err)
		}

		if _, exists := expensesMap[expenseID]; !exists {
			expensesMap[expenseID] = &ledger.Expense{expenseID, ownerID, make([]int, 0), amount, description, createdAt}
		}
		expensesMap[expenseID].Users = append(expensesMap[expenseID].Users, userID)
	}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	expenses := make([]ledger.Expense, 0)
	for _, expense := range expensesMap {
		expenses = append(expenses, *expense)
	}
	return expenses
}
