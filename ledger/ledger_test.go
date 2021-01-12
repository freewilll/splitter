package ledger

import (
	"math"
	"testing"
)

const float64EqualityThreshold = 1e-9 // Use in float comparison function

// almostEqual compares two floats
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}

// makeOweMap makes a map out of a slice of debts, keyed by the user id
func makeOweMap(debts []Debt) map[int]float64 {
	m := make(map[int]float64)
	for _, o := range debts {
		m[o.UserID] = o.Amount
	}
	return m
}

// debtsEqual checks if two slices of debts are identical
func debtsEqual(a []Debt, b []Debt) bool {
	am := makeOweMap(a)
	bm := makeOweMap(b)
	if len(am) != len(bm) {
		return false
	}

	for k, v := range am {
		if bm[k] != v {
			return false
		}
	}

	return true
}

// debtsInBalanceEqual checks if credits and debits in a Balance are equal
func debtsInBalanceEqual(a Balance, b Balance) bool {
	return debtsEqual(a.Debit, b.Debit) && debtsEqual(a.Credit, b.Credit)
}

func TestCalculateBalance(t *testing.T) {
	// Test a couple of scenarios of expenses and ensure the balance, credit
	// and debit are correct for all users.

	// User 1 pays €42 split between users 1,2,3
	meal := Expense{
		ExpenseID: 1,
		OwnerID:   1,
		Users:     []int{1, 2, 3},
		Amount:    42,
	}

	// User 2 pays €8 split between users 1,2
	coffee := Expense{
		ExpenseID: 1,
		OwnerID:   2,
		Users:     []int{1, 2},
		Amount:    8,
	}

	tests := []struct {
		Expenses []Expense
		Balances map[int]Balance
	}{
		// Only meal: users 2 and 3 owe user 1 €14.
		{
			[]Expense{meal},
			map[int]Balance{
				1: Balance{Balance: 28, Credit: []Debt{{2, 14}, {3, 14}}},
				2: Balance{Balance: -14, Debit: []Debt{{1, 14}}},
				3: Balance{Balance: -14, Debit: []Debt{{1, 14}}},
			},
		},

		// Only coffee: user 1 owes user 2 €4
		{
			[]Expense{coffee},
			map[int]Balance{
				1: Balance{Balance: -4, Debit: []Debt{{2, 4}}},
				2: Balance{Balance: 4, Credit: []Debt{{1, 4}}},
				3: Balance{Balance: 0},
			},
		},

		// Meal and Coffee: a combination of the two above.
		// User 1 owes User2 €4 for the coffee, which is mixed with
		// what user2 owes user1 for the meal. User 3 is unaffected by the coffee.
		{
			[]Expense{meal, coffee},
			map[int]Balance{
				1: Balance{Balance: 24, Credit: []Debt{{2, 10}, {3, 14}}},
				2: Balance{Balance: -10, Debit: []Debt{{1, 10}}},
				3: Balance{Balance: -14, Debit: []Debt{{1, 14}}},
			},
		},
	}

	for _, test := range tests {
		for userID, balance := range test.Balances {
			got := CalculateBalance(test.Expenses, userID)
			if !almostEqual(balance.Balance, got.Balance) {
				t.Errorf("Balance mismatch, expected: %f, got: %f", balance.Balance, got.Balance)
			}

			if !debtsInBalanceEqual(got, balance) {
				t.Errorf("Owes mismatch, expected: %+v, got: %+v", balance, got)
			}
		}
	}
}
