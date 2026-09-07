package domain

import (
	"crypto/sha256"
	"encoding/json"
	"slices"
)

// Correspondence is normalized, structured provider evidence, not a model score
// or a client assertion. Namespace and movement distinguish reused external IDs.
type Correspondence struct {
	Kind, Namespace, Reference, Network, Movement string
	FromAccountID, ToAccountID                    string
}

func (c Correspondence) Validate() error {
	if !slices.Contains([]string{"payment", "transfer", "exchange"}, c.Kind) || len(c.Namespace) < 1 || len(c.Namespace) > 2000 || len(c.Reference) < 1 || len(c.Reference) > 2000 || len(c.Network) > 200 || len(c.Movement) > 2000 {
		return ErrInvalidSource
	}
	if c.Network != "" && c.Movement == "" || c.Kind != "payment" && (c.FromAccountID == "" || c.ToAccountID == "" || c.FromAccountID == c.ToAccountID) {
		return ErrInvalidSource
	}
	if c.Kind == "payment" && (c.FromAccountID != "" || c.ToAccountID != "") {
		return ErrInvalidSource
	}
	return nil
}

func (c Correspondence) Digest() [32]byte {
	data, _ := json.Marshal([]string{c.Kind, c.Namespace, c.Reference, c.Network, c.Movement, c.FromAccountID, c.ToAccountID})
	return sha256.Sum256(data)
}
