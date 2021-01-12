package api

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/freewilll/splitter/cache"
	"github.com/freewilll/splitter/database"
	"github.com/freewilll/splitter/ledger"
)

func TestGetUsers(t *testing.T) {
	// Add a user to the database and ensure the API returns it

	db := database.NewInMemoryDatabase()
	cache := cache.NewInMemoryCache()
	api := NewAPI(db, cache)

	// Add the user to the database
	dbh := db.Connect()
	userID1, _ := dbh.CreateUser("test1@getstream.io", "secret")

	// Call the GET users API and ensure the user is in the response
	request, _ := http.NewRequest(http.MethodGet, "/users", nil)
	response := httptest.NewRecorder()
	api.users(response, request, userID1)
	var gotUsers usersResponse
	err := json.NewDecoder(response.Body).Decode(&gotUsers)
	if err != nil {
		t.Fatalf("Unable to parse response from server '%v'", err)
	}
	wantedUsers := usersResponse{Users: []userResponse{
		{ID: userID1, Email: "test1@getstream.io"}},
	}
	if !reflect.DeepEqual(gotUsers, wantedUsers) {
		t.Errorf("wanted %v,got %v", wantedUsers, gotUsers)
	}

	// Add another user to the database and ensure the API returns both users
	userID2, _ := dbh.CreateUser("test2@getstream.io", "secret")

	api.users(response, request, userID1)
	err = json.NewDecoder(response.Body).Decode(&gotUsers)
	if err != nil {
		t.Fatalf("Unable to parse response from server '%v'", err)
	}
	wantedUsers = usersResponse{Users: []userResponse{
		{ID: userID1, Email: "test1@getstream.io"},
		{ID: userID2, Email: "test2@getstream.io"},
	}}
	if !reflect.DeepEqual(gotUsers, wantedUsers) {
		t.Errorf("wanted %v,got %v", wantedUsers, gotUsers)
	}
}

func TestPostUsers(t *testing.T) {
	// Create a user using the POST users API

	db := database.NewInMemoryDatabase()
	cache := cache.NewInMemoryCache()
	api := NewAPI(db, cache)

	// Create first user, that will make the create user request
	dbh := db.Connect()
	userID, _ := dbh.CreateUser("test1@getstream.io", "secret")

	// Create test@getstream.io user
	body, _ := json.Marshal(createUserRequest{Email: "test@getstream.io", Password: "secret"})
	request, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	api.users(response, request, userID)
	var got userResponse
	err := json.NewDecoder(response.Body).Decode(&got)
	if err != nil {
		t.Fatalf("Unable to parse response from server '%v'", err)
	}
	wanted := userResponse{ID: 2, Email: "test@getstream.io"}
	if !reflect.DeepEqual(wanted, got) {
		t.Errorf("wanted %v,got %v", wanted, got)
	}
}

func TestPostExpenses(t *testing.T) {
	// Post an expense to the API and check GET balance API returns the correct balance

	db := database.NewInMemoryDatabase()
	cache := cache.NewInMemoryCache()
	api := NewAPI(db, cache)

	// Create three users
	dbh := db.Connect()
	userID1, _ := dbh.CreateUser("test1@getstream.io", "secret")
	userID2, _ := dbh.CreateUser("test2@getstream.io", "secret")
	userID3, _ := dbh.CreateUser("test3@getstream.io", "secret")

	// User 1 buys a meal for the other two, for €42
	body, _ := json.Marshal(createExpenseRequest{
		Description: "Food",
		Amount:      42,
		CreatedAt:   "2021-01-01T15:04:05Z",
		Users:       []userID{{userID2}, {userID3}},
	})

	request, _ := http.NewRequest(http.MethodPost, "/expenses", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	api.postExpenses(response, request, userID1)
	if response.Code != http.StatusCreated {
		t.Fatalf("Unable create expense")
	}

	// Call the GET balance API and check balance in response is correct
	// No deep inspection is done, since this is already covered by the ledger tests
	request, _ = http.NewRequest(http.MethodGet, "/balance", nil)
	api.getBalance(response, request, userID1)
	var got ledger.Balance
	err := json.NewDecoder(response.Body).Decode(&got)
	if err != nil {
		t.Fatalf("Unable to parse response from server '%v'", err)
	}
	wantedBalance := 28.0
	if math.Abs(got.Balance-wantedBalance) > 1e-9 {
		t.Errorf("wanted %v,got %v", wantedBalance, got.Balance)
	}
}
