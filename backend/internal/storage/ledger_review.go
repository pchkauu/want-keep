package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	domain "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Store) ReviewResult(ctx context.Context, p household.Principal, id string, revision uint64) (ledger.ReviewResult, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return ledger.ReviewResult{}, false, err
	}
	r := ledger.ReviewResult{OperationID: id, Revision: revision}
	var at time.Time
	var ns int16
	err = q.QueryRow(ctx, `SELECT actor_id,state,rationale,payload_hash,recorded_at,recorded_ns FROM want_keep.ledger_review_results WHERE household_id=$1 AND operation_id=$2 AND revision=$3`, p.HouseholdID(), id, revision).Scan(&r.ActorID, &r.State, &r.Rationale, &r.PayloadHash, &at, &ns)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, false, nil
	}
	if err != nil {
		return r, false, err
	}
	r.At, err = restoreInstant(at, ns)
	if err != nil {
		return r, false, err
	}
	rows, err := q.Query(ctx, `SELECT kind,evidence_id,evidence_revision FROM want_keep.ledger_review_evidence WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY kind,evidence_id,evidence_revision`, p.HouseholdID(), id, revision)
	if err != nil {
		return r, false, err
	}
	for rows.Next() {
		var e domain.Evidence
		if err = rows.Scan(&e.Kind, &e.ID, &e.Revision); err != nil {
			return r, false, err
		}
		r.Evidence = append(r.Evidence, e)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return r, false, err
	}
	rows.Close()
	var categoryID, merchantID, merchantAlias string
	var items []byte
	err = q.QueryRow(ctx, `SELECT COALESCE(category_id::text,''),COALESCE(merchant_id::text,''),COALESCE(merchant_alias,''),items FROM want_keep.ledger_classification_proposals WHERE household_id=$1 AND operation_id=$2 AND revision=$3`, p.HouseholdID(), id, revision).Scan(&categoryID, &merchantID, &merchantAlias, &items)
	if err == nil {
		proposal, decodeErr := decodeClassificationProposal(categoryID, merchantID, merchantAlias, items)
		if decodeErr != nil {
			return r, false, decodeErr
		}
		r.Proposal = &proposal
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return r, false, err
	}
	return r, true, nil
}
func (s *Store) SaveReviewResult(ctx context.Context, r ledger.ReviewResult) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal.UserID() != r.ActorID {
		return household.ErrForbidden
	}
	at, ns := splitInstant(r.At)
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_review_results(household_id,operation_id,revision,actor_id,state,rationale,payload_hash,recorded_at,recorded_ns) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, scope.principal.HouseholdID(), r.OperationID, r.Revision, r.ActorID, r.State, r.Rationale, r.PayloadHash, at, ns)
	if err != nil {
		return err
	}
	for _, e := range r.Evidence {
		if err = s.requireDecisionEvidence(ctx, scope, e); err != nil {
			return err
		}
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_review_evidence(household_id,operation_id,revision,kind,evidence_id,evidence_revision) VALUES($1,$2,$3,$4,$5,$6)`, scope.principal.HouseholdID(), r.OperationID, r.Revision, e.Kind, e.ID, e.Revision); err != nil {
			return err
		}
	}
	if r.Proposal != nil {
		items, encodeErr := encodeClassificationProposal(r.Proposal.ReceiptItems)
		if encodeErr != nil {
			return encodeErr
		}
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_classification_proposals(household_id,operation_id,revision,proposal_hash,category_id,merchant_id,merchant_alias,items,rationale,created_at,created_ns) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,NULLIF($6,'')::uuid,NULLIF($7,''),$8,$9,$10,$11)`, scope.principal.HouseholdID(), r.OperationID, r.Revision, r.PayloadHash, r.Proposal.CategoryID, r.Proposal.MerchantID, r.Proposal.MerchantAlias, items, r.Rationale, at, ns); err != nil {
			return err
		}
	}
	return nil
}

type storedClassificationItem struct {
	ID, Name, Quantity, CategoryID string
	Gross, Discount, Asset         string
}

func encodeClassificationProposal(items []domain.ReceiptItem) ([]byte, error) {
	stored := make([]storedClassificationItem, 0, len(items))
	for _, item := range items {
		if err := item.Validate(); err != nil {
			return nil, err
		}
		stored = append(stored, storedClassificationItem{ID: item.ID, Name: item.Name, Quantity: item.Quantity, CategoryID: item.CategoryID, Gross: item.Gross.Amount(), Discount: item.Discount.Amount(), Asset: string(item.Gross.Asset())})
	}
	return json.Marshal(stored)
}

func decodeClassificationProposal(categoryID, merchantID, merchantAlias string, data []byte) (domain.ClassificationProposal, error) {
	var stored []storedClassificationItem
	if err := json.Unmarshal(data, &stored); err != nil {
		return domain.ClassificationProposal{}, err
	}
	proposal := domain.ClassificationProposal{CategoryID: categoryID, MerchantID: merchantID, MerchantAlias: merchantAlias}
	for _, item := range stored {
		gross, err := money.NewMoney(item.Gross, money.Asset(item.Asset))
		if err != nil {
			return domain.ClassificationProposal{}, err
		}
		discount, err := money.NewMoney(item.Discount, money.Asset(item.Asset))
		if err != nil {
			return domain.ClassificationProposal{}, err
		}
		proposal.ReceiptItems = append(proposal.ReceiptItems, domain.ReceiptItem{ID: item.ID, Name: item.Name, Quantity: item.Quantity, CategoryID: item.CategoryID, Gross: gross, Discount: discount})
	}
	return proposal, nil
}
