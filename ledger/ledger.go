package ledger

import (
	"time"
)

// Expense is a single expense, paid for by a user. The expense is shared by
// at least one more users. The Users slice contains the other users, not including
// the OwnerID of the expense.
type Expense struct {
	ExpenseID   int       // Id of the expense
	OwnerID     int       // User id who paid for the expense
	Users       []int     // Slice of other users that share the expense
	Amount      float64   // Amount the owner paid for
	Description string    // Description, set by the owner
	CreatedAt   time.Time // The time the expense was incurred
}

// Debt represents money owed by one user to another. The amount is negative in case
// of a credit.
type Debt struct {
	UserID int     `json:"user_id"` // The owner of the debt
	Amount float64 `json:"amount"`  // The amount of the debt
}

// Balance is a user's balance
type Balance struct {
	Balance float64 `json:"balance"` // Amount of the balance
	Debit   []Debt  `json:"debit"`   // Money this user owes to other users
	Credit  []Debt  `json:"credit"`  // Money other users owe this user
}

// CalculateBalance takes a []Expense and calculates who owes what and what their
// balance is for a given userID. This is the heart of the application.
func CalculateBalance(expenses []Expense, userID int) Balance {
	var balance float64                    // Total balance
	debts := make(map[int]map[int]float64) // Double map of money owed to other users

	// Loop over all expenses and amend balance and debts
	for _, expense := range expenses {
		// Is userID involved in this expense? If not, skip it
		userTookPart := false
		for _, expenseUserID := range expense.Users {
			if expenseUserID == userID {
				userTookPart = true
			}
		}
		if !userTookPart {
			continue
		}

		owned := expense.OwnerID == userID // Did userID pay for this expense?

		l := len(expense.Users)
		perPersonAmount := float64(expense.Amount) / float64(l)
		var delta float64
		if owned {
			// l-1 users owe userID money
			delta = float64(l-1) * perPersonAmount
		} else {
			// userID owes the expense owner money
			delta = -perPersonAmount
		}

		balance = balance + delta // Change our own balance

		// Amend the debts the debts map
		for _, expenseUserID := range expense.Users {
			// userID never owes themselves anything
			if expenseUserID == expense.OwnerID {
				continue
			}

			// Allocate maps where needed
			if debts[expenseUserID] == nil {
				debts[expenseUserID] = make(map[int]float64)
			}
			if debts[expense.OwnerID] == nil {
				debts[expense.OwnerID] = make(map[int]float64)
			}

			// Amend debit & credits
			debts[expenseUserID][expense.OwnerID] += perPersonAmount
			debts[expense.OwnerID][expenseUserID] -= perPersonAmount
		}
	}

	// Make final debit and credit slices
	debit := make([]Debt, 0)
	credit := make([]Debt, 0)
	userDebts := debts[userID]
	for userID, amount := range userDebts {
		if amount > 0 {
			debit = append(debit, Debt{UserID: userID, Amount: amount})
		} else {
			credit = append(credit, Debt{UserID: userID, Amount: -amount})
		}
	}

	return Balance{Balance: balance, Debit: debit, Credit: credit}
}
