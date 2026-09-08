package domain_test

import (
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	"reflect"
	"testing"
)

func TestRetainedParticipationPreservesIndependentEffects(t *testing.T) {
	for _, state := range []ledger.State{ledger.Pending, ledger.Posted, ledger.Cancelled, ledger.Reversed} {
		for _, excluded := range []bool{false, true} {
			r := examples{}.expense()
			r.State = state
			if state == ledger.Pending || state == ledger.Cancelled {
				r.PostedAt = ledger.Revision{}.PostedAt
			}
			if excluded {
				r.AccountingState = ledger.ExcludedFromAccounting
			}
			expected, err := r.BalanceEffects()
			if err != nil {
				t.Fatal(err)
			}
			components, err := r.Components()
			if err != nil {
				t.Fatal(err)
			}
			r.Participation = ledger.Participation{GroupID: "clarification", Kind: "payment", State: "retained"}
			actual, err := r.BalanceEffects()
			if err != nil || !reflect.DeepEqual(expected, actual) {
				t.Fatal(state, excluded, actual, err)
			}
			actualComponents, err := r.Components()
			if err != nil || !reflect.DeepEqual(components, actualComponents) {
				t.Fatal(actualComponents, err)
			}
			r.Participation.Parts = []ledger.Contribution{{}}
			if r.Validate() == nil {
				t.Fatal("retained contribution accepted assigned components")
			}
		}
	}
}
