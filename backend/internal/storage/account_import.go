package storage

import (
	"context"
	"crypto/sha256"
	"errors"

	"github.com/jackc/pgx/v5"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Store) ResolveImportedAccount(ctx context.Context, p household.Principal, input accounts.ImportInput) (account.Account, bool, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return account.Account{}, false, err
	}
	if scope.principal != p || scope.syncJobID != input.JobID || scope.syncConnectionID != input.ConnectionID || input.JobID == "" {
		return account.Account{}, false, ErrTransactionRequired
	}
	if input.Origin != "historical_backfill" && input.Origin != "live_sync" {
		return account.Account{}, false, account.ErrInvalidAccount
	}
	c, err := s.Connection(ctx, p, input.ConnectionID)
	if err != nil {
		return account.Account{}, false, err
	}
	if input.Provider != c.Provider {
		return account.Account{}, false, ledger.ErrInvalidSource
	}
	if _, err = money.ParseAsset(string(input.Asset)); err != nil {
		return account.Account{}, false, err
	}
	if input.ExternalAssetCode != string(input.Asset) {
		return account.Account{}, false, money.ErrUnsupportedAsset
	}
	external, err := s.ResolveExternalAccount(ctx, c.Provider, input.ExternalID, c.Owner)
	if err != nil {
		return account.Account{}, false, err
	}
	if input.Product == "cash" || !account.Product(input.Product).Valid() {
		return account.Account{}, false, account.ErrInvalidAccount
	}
	digest := sha256.Sum256([]byte(input.Network))
	var id string
	err = scope.tx.QueryRow(ctx, `SELECT id FROM want_keep.accounts WHERE household_id=$1 AND external_account_id=$2 AND product=$3 AND asset=$4 AND network_digest=$5`, p.HouseholdID(), external, input.Product, input.Asset, digest[:]).Scan(&id)
	if err == nil {
		a, e := s.Account(ctx, p, id)
		if e != nil {
			return a, false, e
		}
		if a.Network != input.Network || a.ExternalAssetCode != input.ExternalAssetCode {
			return a, false, ledger.ErrSourceAmbiguous
		}
		return a, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return account.Account{}, false, err
	}
	ownership, err := household.NewOwnership(p.HouseholdID(), household.Personal, c.Owner)
	if err != nil {
		return account.Account{}, false, err
	}
	a := account.Account{ID: newID(), Name: input.Name, Product: input.Product, Asset: input.Asset, Ownership: ownership, Revision: 1, OpeningDate: input.OpeningDate, ExternalAccountID: external, Network: input.Network, ExternalAssetCode: input.ExternalAssetCode}
	if err = a.Validate(); err != nil {
		return a, false, err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.accounts(household_id,id,name,asset,scope,owner_id,product,revision,opening_date,external_account_id,network,external_asset_code,network_digest) VALUES($1,$2,$3,$4,$5,$6,$7,1,$8,$9,$10,$11,$12)`, p.HouseholdID(), a.ID, a.Name, a.Asset, a.Ownership.Scope(), string(c.Owner), a.Product, a.OpeningDate.String(), external, a.Network, a.ExternalAssetCode, digest[:])
	return a, err == nil, err
}
func (s *Store) RecordCardAlias(ctx context.Context, p household.Principal, a account.CardAlias) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != p || scope.syncJobID == "" {
		return ErrTransactionRequired
	}
	if err = a.Validate(); err != nil {
		return err
	}
	owner, err := s.Account(ctx, p, a.AccountID)
	if err != nil {
		return err
	}
	connection, err := s.Connection(ctx, p, scope.syncConnectionID)
	if err != nil {
		return err
	}
	var provider string
	if err = scope.tx.QueryRow(ctx, `SELECT provider FROM want_keep.external_accounts WHERE household_id=$1 AND id=$2`, p.HouseholdID(), owner.ExternalAccountID).Scan(&provider); err != nil {
		return err
	}
	if owner.ExternalOwnerID != connection.Owner || provider != connection.Provider {
		return household.ErrForbidden
	}
	var accountID, label, lastFour string
	err = scope.tx.QueryRow(ctx, `SELECT account_id,label,last_four FROM want_keep.card_aliases WHERE household_id=$1 AND id=$2`, p.HouseholdID(), a.ID).Scan(&accountID, &label, &lastFour)
	if err == nil {
		if accountID != a.AccountID || label != a.Label || lastFour != a.LastFour {
			return ledger.ErrSourceAmbiguous
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.card_aliases(household_id,id,account_id,label,last_four) VALUES($1,$2,$3,$4,$5)`, p.HouseholdID(), a.ID, a.AccountID, a.Label, a.LastFour)
	return err
}

func (s *Store) WithinAccountImport(ctx context.Context, fn func(context.Context) error) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.syncJobID == "" {
		return ErrTransactionRequired
	}
	return s.transact(ctx, func(ctx context.Context, _ *transactionScope) error { return fn(ctx) })
}
func (s *Store) RecordAccountImportIssue(ctx context.Context, p household.Principal, input accounts.ImportInput, reason string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != p || scope.syncJobID != input.JobID || scope.syncConnectionID != input.ConnectionID {
		return ErrTransactionRequired
	}
	j, err := s.Job(ctx, p, input.JobID)
	if err != nil {
		return err
	}
	return s.Quarantine(ctx, j, input.EvidenceRef, reason)
}
