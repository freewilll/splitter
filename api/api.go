package api

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/freewilll/splitter/cache"
	"github.com/freewilll/splitter/database"
	"github.com/freewilll/splitter/jwt"
	"github.com/freewilll/splitter/ledger"
)

const jwtCookieName = "jwt-token"

type handler func(w http.ResponseWriter, r *http.Request)
type authenticatedHandler func(w http.ResponseWriter, r *http.Request, userID int)

type errorResponse struct {
	Error string `json:"error"`
}

type userResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

type usersResponse struct {
	Users []userResponse `json:"users"`
}

type createUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userID struct {
	ID int `json:"id"`
}

type createExpenseRequest struct {
	Description string   `json:"description"`
	Amount      float64  `json:"amount"`
	CreatedAt   string   `json:"created_at"`
	Users       []userID `json:"users"`
}

// API holds the config and functionality for HTTP REST/JSON API for the application
type API struct {
	db    database.Database // The authoritative data store
	cache cache.Cache       // Cache for balances
}

// serverPort is the TCP port the API listens on
var serverPort = flag.Int("server-port", 8080, "web server port")

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// NewAPI Creates a new instance of the HTTP REST/JSON API for the application
func NewAPI(db database.Database, cache cache.Cache) *API {
	return &API{db: db, cache: cache}
}

// writeJSON marshalls data into a response with content-type application/json
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	result, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	io.WriteString(w, string(result))
}

// writeError writes a status code and error message
func writeError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	writeJSON(w, errorResponse{message})
}

// signin handles user authentication with POST requests to the signin endpoint
// If the user authenticates successfully, a JWT token is set in a cookie
func (api *API) signin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}

	dbh := api.db.Connect()
	defer dbh.Close()

	var a authRequest
	err := json.NewDecoder(r.Body).Decode(&a)
	if err != nil {
		log.Print("Unable to decode and parse json")
		writeError(w, http.StatusBadRequest, "unable to decode and parse json")
		return
	}

	id, err := dbh.AuthenticateUser(a.Email, a.Password)
	if err != nil {
		switch err {
		case database.ErrNotFound, database.ErrPasswordMismatch:
			log.Printf("Authentication failed for '%s'", a.Email)
			writeError(w, http.StatusUnauthorized, "authorization failed")
			return
		default:
			panic(err)
		}
	}

	cookie := jwt.CreateCookie(id, jwtCookieName)
	http.SetCookie(w, &cookie)
}

// requireAuth is a handler wrapper to ensures a user is authenticated. The userID
// is passed on to the next handler in the chain.
func (api *API) requireAuth(pass authenticatedHandler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(jwtCookieName)
		if err != nil {
			if err == http.ErrNoCookie {
				log.Printf("Missing jwt cookie")
				writeError(w, http.StatusUnauthorized, "authorization failed")
				return
			}
			panic(err)
			return
		}

		userID, ok := jwt.VerifyToken(c.Value)
		if !ok {
			writeError(w, http.StatusUnauthorized, "authorization failed")
			return
		}

		// Greetings, Professor Falken.
		pass(w, r, userID)
	}
}

// getUsers returns all users in the database
func (api *API) getUsers(w http.ResponseWriter, r *http.Request) {
	dbh := api.db.Connect()
	defer dbh.Close()

	dbUsers := dbh.GetUsers()
	users := usersResponse{Users: make([]userResponse, len(dbUsers))}
	for i, u := range dbUsers {
		users.Users[i] = userResponse{ID: u.ID, Email: u.Email}
	}

	writeJSON(w, users)
}

// isEmailValid checks if the email provided passes the required structure and length.
func isEmailValid(e string) bool {
	if len(e) < 3 && len(e) > 254 {
		return false
	}
	return emailRegex.MatchString(e)
}

// postUsers is the user registration endpoint. Some validation is done, then
// the user is added to the database. A 409 (conflict) is returned if the user already
// exists.
func (api *API) postUsers(w http.ResponseWriter, r *http.Request) {
	dbh := api.db.Connect()
	defer dbh.Close()

	// Decode request
	var u createUserRequest
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		log.Print("Unable to decode and parse json")
		writeError(w, http.StatusBadRequest, "unable to decode and parse json")
		return
	}

	// Validate email and password
	if len(u.Password) < 6 {
		log.Printf("Invalid password")
		writeError(w, http.StatusBadRequest, "invalid password: it must be at least 6 characters")
		return

	}

	if !isEmailValid(u.Email) {
		log.Printf("Invalid email '%s'", u.Email)
		writeError(w, http.StatusBadRequest, "invalid email address")
		return
	}

	// Add the user to the database
	log.Printf("Adding user email='%s'", u.Email)

	id, err := dbh.CreateUser(u.Email, u.Password)
	if err != nil {
		switch err {
		case database.ErrDuplicate:
			log.Printf("User uniqueness failed for email '%s'", u.Email)
			writeError(w, http.StatusConflict, "a user with that email already exists")
		default:
			panic(err)
		}
	}

	writeJSON(w, userResponse{ID: id, Email: u.Email})
}

// users handles the users endpoint for the GET and POST methods
func (api *API) users(w http.ResponseWriter, r *http.Request, userID int) {
	if r.Method == "GET" {
		api.getUsers(w, r)
	} else if r.Method == "POST" {
		api.postUsers(w, r)
	} else {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// postExpenses adds an expense
func (api *API) postExpenses(w http.ResponseWriter, r *http.Request, userID int) {
	if r.Method != "POST" {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}

	dbh := api.db.Connect()
	defer dbh.Close()

	// Decode request
	var e createExpenseRequest
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		log.Print("Unable to decode and parse json")
		writeError(w, http.StatusBadRequest, "unable to decode and parse json")
		return
	}

	// Validate description, amount and created_at
	if e.Description == "" {
		log.Printf("Invalid description '%s'", e.Description)
		writeError(w, http.StatusBadRequest, "description must not be empty")
		return
	}

	if e.Amount <= 0 {
		log.Printf("Invalid amount '%0.2f'", e.Amount)
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	createdAt, err := time.Parse(time.RFC3339, e.CreatedAt)
	if err != nil {
		log.Printf("Unable to parse timestamp '%s'", e.CreatedAt)
		writeError(w, http.StatusBadRequest, "unable to parse created_at")
		return
	}

	// Validate users
	if len(e.Users) < 1 {
		log.Printf("Users list too small '%+v'", e.Users)
		writeError(w, http.StatusBadRequest, "at least one other user must be included in an expense")
		return
	}

	// Ensure user_ids don't include self and are unique
	uniqueUsers := make(map[int]bool, 0)
	for _, u := range e.Users {
		if u.ID == userID {
			log.Print("User list includes self")
			writeError(w, http.StatusBadRequest, "user list must not include self")
			return
		}

		if _, exists := uniqueUsers[u.ID]; exists {
			log.Printf("Duplicate user %d", u.ID)
			writeError(w, http.StatusBadRequest, "duplicate user in user list")
			return
		}

		uniqueUsers[u.ID] = true
	}

	users := make([]int, len(e.Users))
	for i, u := range e.Users {
		users[i] = u.ID
	}

	// Create the entries in the database
	log.Printf(
		"Adding expense user_id=%d, description='%s', amount=%0.2f, created_at=%s users=%+v",
		userID, e.Description, e.Amount, createdAt, users)

	expense := ledger.Expense{
		OwnerID:     userID,
		Description: e.Description,
		Amount:      e.Amount,
		CreatedAt:   createdAt,
		Users:       users,
	}

	dbh.CreateExpense(expense)

	// Write through the entries to the cache
	expenses := dbh.GetExpenses(userID)
	balance := ledger.CalculateBalance(expenses, userID)
	api.cache.SetBalance(balance, userID)
	log.Printf("Balance for user %d is %+v", userID, balance)

	// Return the result to the client
	w.WriteHeader(http.StatusCreated)
}

// getBalance returns the balance from the cache
func (api *API) getBalance(w http.ResponseWriter, r *http.Request, userID int) {
	if r.Method != "GET" {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}

	balance := api.cache.GetBalance(api.db, userID)
	log.Printf("Balance for user %d is %+v", userID, balance)
	writeJSON(w, balance)
}

// Serve starts up the API on serverPort
func (api *API) Serve() {
	http.HandleFunc("/signin", api.signin)
	http.HandleFunc("/users", api.requireAuth(api.users))
	http.HandleFunc("/expenses", api.requireAuth(api.postExpenses))
	http.HandleFunc("/balance", api.requireAuth(api.getBalance))
	log.Printf("Listening on port %d", *serverPort)
	panic(http.ListenAndServe(fmt.Sprintf(":%d", *serverPort), nil))
}
