package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Store) SaveReimbursement(ctx context.Context, principal household.Principal, value ledger.Reimbursement, expected uint64, decision ledger.ReimbursementDecision) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != principal || value.ActorID != principal.UserID() || decision.ActorID != principal.UserID() || decision.ID != value.DecisionID || decision.ReimbursementID != value.ID || decision.Before != expected || decision.After != value.Revision || value.Validate() != nil || decision.Validate() != nil || expected >= command.MaxRevision || value.Revision != expected+1 {
		return ledger.ErrInvalidReimbursement
	}
	family := principal.HouseholdID()
	if expected == 0 {
		tag, insertErr := scope.tx.Exec(ctx, `INSERT INTO want_keep.reimbursements(household_id,id,revision) VALUES($1,$2,1) ON CONFLICT DO NOTHING`, family, value.ID)
		if insertErr != nil {
			return insertErr
		}
		if tag.RowsAffected() != 1 {
			return command.ErrVersionConflict
		}
	} else {
		tag, updateErr := scope.tx.Exec(ctx, `UPDATE want_keep.reimbursements SET revision=$3 WHERE household_id=$1 AND id=$2 AND revision=$4`, family, value.ID, value.Revision, expected)
		if updateErr != nil {
			return updateErr
		}
		if tag.RowsAffected() != 1 {
			return command.ErrVersionConflict
		}
	}
	at, ns := splitInstant(value.RecordedAt)
	var previous, expenseID, expenseRevision any
	if expected > 0 {
		previous = expected
	}
	if value.ExpenseID != "" {
		expenseID, expenseRevision = value.ExpenseID, value.ExpenseRevision
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reimbursement_revisions(household_id,reimbursement_id,revision,previous_revision,creditor_member_id,debtor_member_id,asset,principal,outstanding,state,voided,expense_id,expense_revision,reason,attention_reason,actor_id,decision_id,recorded_at,recorded_ns) VALUES($1,$2,$3,$4,$5,$6,$7,$8::numeric,$9::numeric,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`, family, value.ID, value.Revision, previous, value.CreditorMemberID, value.DebtorMemberID, value.Principal.Asset(), value.Principal.Amount(), value.Outstanding.Amount(), value.State, value.Voided, expenseID, expenseRevision, value.Reason, value.AttentionReason, value.ActorID, value.DecisionID, at, ns)
	if err != nil {
		return err
	}
	for field, revision := range value.FieldVersions {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reimbursement_field_versions(household_id,reimbursement_id,revision,field,changed_revision) VALUES($1,$2,$3,$4,$5)`, family, value.ID, value.Revision, field, revision); err != nil {
			return err
		}
	}
	decisionAt, decisionNS := splitInstant(decision.At)
	var undoOf, settlementID any
	if decision.UndoOf != "" {
		undoOf = decision.UndoOf
	}
	if decision.SettlementID != "" {
		settlementID = decision.SettlementID
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reimbursement_decisions(household_id,id,reimbursement_id,kind,reason,undo_of,settlement_id,actor_id,recorded_at,recorded_ns,before_revision,after_revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, family, decision.ID, value.ID, decision.Kind, decision.Reason, undoOf, settlementID, decision.ActorID, decisionAt, decisionNS, decision.Before, decision.After)
	if err != nil {
		return err
	}
	for _, field := range decision.Fields {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reimbursement_decision_fields(household_id,decision_id,field) VALUES($1,$2,$3)`, family, decision.ID, field); err != nil {
			return err
		}
	}
	if err = s.saveReimbursementSettlements(ctx, scope.tx, family, value, expected, decision.ID); err != nil {
		return err
	}
	event := map[string]string{"create": "created", "correction": "corrected", "settlement": "settled", "undo": "undone", "reference_change": "reference_changed"}[decision.Kind]
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reimbursement_audit(household_id,reimbursement_id,revision,actor_id,decision_id,command_id,event,recorded_at,recorded_ns) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,$7,$8,$9)`, family, value.ID, value.Revision, value.ActorID, decision.ID, commands.CurrentCommandID(ctx), event, at, ns)
	return err
}

func (s *Store) saveReimbursementSettlements(ctx context.Context, tx pgx.Tx, family household.HouseholdID, value ledger.Reimbursement, expected uint64, decisionID string) error {
	for _, settlement := range value.Settlements {
		var current ledger.SettlementState
		err := tx.QueryRow(ctx, `SELECT e.state FROM want_keep.reimbursement_settlements s JOIN LATERAL(SELECT state FROM want_keep.reimbursement_settlement_events WHERE household_id=s.household_id AND settlement_id=s.id ORDER BY reimbursement_revision DESC LIMIT 1)e ON true WHERE s.household_id=$1 AND s.id=$2`, family, settlement.ID).Scan(&current)
		if errors.Is(err, pgx.ErrNoRows) {
			at, ns := splitInstant(settlement.RecordedAt)
			_, err = tx.Exec(ctx, `INSERT INTO want_keep.reimbursement_settlements(household_id,id,reimbursement_id,created_revision,decision_id,transfer_id,transfer_revision,transfer_key,fingerprint,transfer_asset,transfer_amount,settled_asset,settled_amount,actor_id,recorded_at,recorded_ns) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::numeric,$12,$13::numeric,$14,$15,$16)`, family, settlement.ID, value.ID, value.Revision, settlement.DecisionID, settlement.TransferID, settlement.TransferRevision, settlement.TransferKey, settlement.Fingerprint, settlement.TransferAmount.Asset(), settlement.TransferAmount.Amount(), settlement.SettledAmount.Asset(), settlement.SettledAmount.Amount(), settlement.ActorID, at, ns)
			if err != nil {
				return err
			}
			for _, operationID := range settlement.OperationIDs {
				var revision uint64
				if err = tx.QueryRow(ctx, `SELECT revision FROM want_keep.operations WHERE household_id=$1 AND id=$2`, family, operationID).Scan(&revision); err != nil {
					return err
				}
				if _, err = tx.Exec(ctx, `INSERT INTO want_keep.reimbursement_settlement_operations(household_id,settlement_id,operation_id,operation_revision) VALUES($1,$2,$3,$4)`, family, settlement.ID, operationID, revision); err != nil {
					return err
				}
			}
			current = ""
		} else if err != nil {
			return err
		}
		if current != settlement.State {
			if expected == 0 && settlement.State != ledger.SettlementActive || current == "" && settlement.State != ledger.SettlementActive {
				return ledger.ErrInvalidReimbursement
			}
			if _, err = tx.Exec(ctx, `INSERT INTO want_keep.reimbursement_settlement_events(household_id,settlement_id,reimbursement_id,reimbursement_revision,state,decision_id) VALUES($1,$2,$3,$4,$5,$6)`, family, settlement.ID, value.ID, value.Revision, settlement.State, decisionID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) Reimbursement(ctx context.Context, principal household.Principal, id string) (ledger.Reimbursement, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return ledger.Reimbursement{}, err
	}
	var revision uint64
	if err = q.QueryRow(ctx, `SELECT revision FROM want_keep.reimbursements WHERE household_id=$1 AND id=$2`, principal.HouseholdID(), id).Scan(&revision); errors.Is(err, pgx.ErrNoRows) {
		return ledger.Reimbursement{}, ledger.ErrNotFound
	}
	if err != nil {
		return ledger.Reimbursement{}, err
	}
	return s.reimbursementRevision(ctx, q, principal, id, revision)
}

func (s *Store) ReimbursementRevision(ctx context.Context, principal household.Principal, id string, revision uint64) (ledger.Reimbursement, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return ledger.Reimbursement{}, err
	}
	return s.reimbursementRevision(ctx, q, principal, id, revision)
}

func (s *Store) reimbursementRevision(ctx context.Context, q reader, principal household.Principal, id string, revision uint64) (ledger.Reimbursement, error) {
	value := ledger.Reimbursement{ID: id, Revision: revision, FieldVersions: map[ledger.ReimbursementField]uint64{}}
	var principalAmount, outstanding, asset string
	var expenseID *string
	var expenseRevision *uint64
	var recordedAt time.Time
	var recordedNS int16
	err := q.QueryRow(ctx, `SELECT creditor_member_id,debtor_member_id,asset,principal::text,outstanding::text,state,voided,expense_id::text,expense_revision,reason,attention_reason,actor_id,decision_id,recorded_at,recorded_ns FROM want_keep.reimbursement_revisions WHERE household_id=$1 AND reimbursement_id=$2 AND revision=$3`, principal.HouseholdID(), id, revision).Scan(&value.CreditorMemberID, &value.DebtorMemberID, &asset, &principalAmount, &outstanding, &value.State, &value.Voided, &expenseID, &expenseRevision, &value.Reason, &value.AttentionReason, &value.ActorID, &value.DecisionID, &recordedAt, &recordedNS)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, ledger.ErrNotFound
	}
	if err != nil {
		return value, err
	}
	value.Principal, err = money.NewMoney(principalAmount, money.Asset(asset))
	if err != nil {
		return value, err
	}
	value.Outstanding, err = money.NewMoney(outstanding, money.Asset(asset))
	if err != nil {
		return value, err
	}
	value.RecordedAt, err = restoreInstant(recordedAt, recordedNS)
	if err != nil {
		return value, err
	}
	if expenseID != nil {
		value.ExpenseID, value.ExpenseRevision = *expenseID, *expenseRevision
	}
	rows, err := q.Query(ctx, `SELECT field,changed_revision FROM want_keep.reimbursement_field_versions WHERE household_id=$1 AND reimbursement_id=$2 AND revision=$3`, principal.HouseholdID(), id, revision)
	if err != nil {
		return value, err
	}
	for rows.Next() {
		var field ledger.ReimbursementField
		var changed uint64
		if err = rows.Scan(&field, &changed); err != nil {
			rows.Close()
			return value, err
		}
		value.FieldVersions[field] = changed
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return value, err
	}
	rows.Close()
	value.Settlements, err = s.reimbursementSettlements(ctx, q, principal.HouseholdID(), id, revision)
	if err != nil {
		return value, err
	}
	return value, value.Validate()
}

func (s *Store) reimbursementSettlements(ctx context.Context, q reader, family household.HouseholdID, id string, revision uint64) ([]ledger.ReimbursementSettlement, error) {
	rows, err := q.Query(ctx, `SELECT s.id,s.decision_id,s.transfer_id,s.transfer_revision,s.transfer_key,s.fingerprint,s.transfer_asset,s.transfer_amount::text,s.settled_asset,s.settled_amount::text,s.actor_id,s.recorded_at,s.recorded_ns,
 ARRAY(SELECT operation_id::text FROM want_keep.reimbursement_settlement_operations o WHERE o.household_id=s.household_id AND o.settlement_id=s.id ORDER BY operation_id),
 (SELECT state FROM want_keep.reimbursement_settlement_events e WHERE e.household_id=s.household_id AND e.settlement_id=s.id AND e.reimbursement_revision<=$3 ORDER BY e.reimbursement_revision DESC LIMIT 1)
 FROM want_keep.reimbursement_settlements s WHERE s.household_id=$1 AND s.reimbursement_id=$2 AND s.created_revision<=$3 ORDER BY s.recorded_at,s.recorded_ns,s.id`, family, id, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ledger.ReimbursementSettlement{}
	for rows.Next() {
		var value ledger.ReimbursementSettlement
		var transferAsset, transferAmount, settledAsset, settledAmount string
		var recordedAt time.Time
		var recordedNS int16
		if err = rows.Scan(&value.ID, &value.DecisionID, &value.TransferID, &value.TransferRevision, &value.TransferKey, &value.Fingerprint, &transferAsset, &transferAmount, &settledAsset, &settledAmount, &value.ActorID, &recordedAt, &recordedNS, &value.OperationIDs, &value.State); err != nil {
			return nil, err
		}
		value.TransferAmount, err = money.NewMoney(transferAmount, money.Asset(transferAsset))
		if err != nil {
			return nil, err
		}
		value.SettledAmount, err = money.NewMoney(settledAmount, money.Asset(settledAsset))
		if err != nil {
			return nil, err
		}
		value.RecordedAt, err = restoreInstant(recordedAt, recordedNS)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (s *Store) Reimbursements(ctx context.Context, principal household.Principal, filter application.ReimbursementFilter, cursor application.ReimbursementCursor, limit int) ([]ledger.Reimbursement, *application.ReimbursementCursor, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return nil, nil, err
	}
	var cursorAt, cursorNS any
	if cursor.ID != "" {
		cursorAt, cursorNS = splitInstant(cursor.At)
	}
	rows, err := q.Query(ctx, `SELECT r.reimbursement_id,r.revision FROM want_keep.reimbursements c JOIN want_keep.reimbursement_revisions r ON (r.household_id,r.reimbursement_id,r.revision)=(c.household_id,c.id,c.revision) WHERE r.household_id=$1 AND ($2='' OR r.creditor_member_id=NULLIF($2,'')::uuid OR r.debtor_member_id=NULLIF($2,'')::uuid) AND ($3='' OR r.asset::text=$3) AND ($4='' OR r.state::text=$4) AND ($5='' OR (r.recorded_at,r.recorded_ns,r.reimbursement_id)<($6::timestamptz,$7::smallint,NULLIF($5,'')::uuid)) ORDER BY r.recorded_at DESC,r.recorded_ns DESC,r.reimbursement_id DESC LIMIT $8`, principal.HouseholdID(), filter.MemberID, filter.Asset, filter.State, cursor.ID, cursorAt, cursorNS, limit+1)
	if err != nil {
		return nil, nil, err
	}
	refs := [][2]any{}
	for rows.Next() {
		var id string
		var revision uint64
		if err = rows.Scan(&id, &revision); err != nil {
			rows.Close()
			return nil, nil, err
		}
		refs = append(refs, [2]any{id, revision})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()
	return s.reimbursementPage(ctx, q, principal, refs, limit)
}

func (s *Store) ReimbursementHistory(ctx context.Context, principal household.Principal, id string, cursor application.ReimbursementCursor, limit int) ([]ledger.Reimbursement, *application.ReimbursementCursor, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return nil, nil, err
	}
	var cursorAt, cursorNS any
	if cursor.Revision != 0 {
		cursorAt, cursorNS = splitInstant(cursor.At)
	}
	rows, err := q.Query(ctx, `SELECT reimbursement_id,revision FROM want_keep.reimbursement_revisions WHERE household_id=$1 AND reimbursement_id=$2 AND ($3::bigint=0 OR (recorded_at,recorded_ns,revision)<($4::timestamptz,$5::smallint,$3::bigint)) ORDER BY recorded_at DESC,recorded_ns DESC,revision DESC LIMIT $6`, principal.HouseholdID(), id, cursor.Revision, cursorAt, cursorNS, limit+1)
	if err != nil {
		return nil, nil, err
	}
	refs := [][2]any{}
	for rows.Next() {
		var reimbursementID string
		var revision uint64
		if err = rows.Scan(&reimbursementID, &revision); err != nil {
			rows.Close()
			return nil, nil, err
		}
		refs = append(refs, [2]any{reimbursementID, revision})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()
	return s.reimbursementPage(ctx, q, principal, refs, limit, true)
}

func (s *Store) reimbursementPage(ctx context.Context, q reader, principal household.Principal, refs [][2]any, limit int, history ...bool) ([]ledger.Reimbursement, *application.ReimbursementCursor, error) {
	more := len(refs) > limit
	if more {
		refs = refs[:limit]
	}
	result := make([]ledger.Reimbursement, 0, len(refs))
	for _, ref := range refs {
		value, err := s.reimbursementRevision(ctx, q, principal, ref[0].(string), ref[1].(uint64))
		if err != nil {
			return nil, nil, err
		}
		result = append(result, value)
	}
	var next *application.ReimbursementCursor
	if more {
		last := result[len(result)-1]
		next = &application.ReimbursementCursor{At: last.RecordedAt, ID: last.ID}
		if len(history) > 0 && history[0] {
			next.ID = ""
			next.Revision = last.Revision
		}
	}
	return result, next, nil
}

func (s *Store) ReimbursementDecision(ctx context.Context, principal household.Principal, id string) (ledger.ReimbursementDecision, bool, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return ledger.ReimbursementDecision{}, false, err
	}
	value := ledger.ReimbursementDecision{ID: id}
	var undoOf, settlementID *string
	var at time.Time
	var ns int16
	err = q.QueryRow(ctx, `SELECT kind,reason,undo_of::text,settlement_id::text,actor_id,recorded_at,recorded_ns,before_revision,after_revision,reimbursement_id FROM want_keep.reimbursement_decisions WHERE household_id=$1 AND id=$2`, principal.HouseholdID(), id).Scan(&value.Kind, &value.Reason, &undoOf, &settlementID, &value.ActorID, &at, &ns, &value.Before, &value.After, &value.ReimbursementID)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, false, nil
	}
	if err != nil {
		return value, false, err
	}
	if undoOf != nil {
		value.UndoOf = *undoOf
	}
	if settlementID != nil {
		value.SettlementID = *settlementID
	}
	value.At, err = restoreInstant(at, ns)
	if err != nil {
		return value, false, err
	}
	rows, err := q.Query(ctx, `SELECT field FROM want_keep.reimbursement_decision_fields WHERE household_id=$1 AND decision_id=$2 ORDER BY field`, principal.HouseholdID(), id)
	if err != nil {
		return value, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var field ledger.ReimbursementField
		if err = rows.Scan(&field); err != nil {
			return value, false, err
		}
		value.Fields = append(value.Fields, field)
	}
	if err = rows.Err(); err != nil {
		return value, false, err
	}
	return value, true, value.Validate()
}

func (s *Store) ReimbursementDecisionUndone(ctx context.Context, principal household.Principal, id string) (bool, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return false, err
	}
	var found bool
	err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.reimbursement_decisions WHERE household_id=$1 AND undo_of=$2)`, principal.HouseholdID(), id).Scan(&found)
	return found, err
}

func (s *Store) ReimbursementIDsForOperation(ctx context.Context, principal household.Principal, operationID string) ([]string, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT DISTINCT id FROM (
 SELECT r.reimbursement_id AS id FROM want_keep.reimbursements c JOIN want_keep.reimbursement_revisions r ON(r.household_id,r.reimbursement_id,r.revision)=(c.household_id,c.id,c.revision) WHERE r.household_id=$1 AND r.expense_id=$2
 UNION ALL
 SELECT s.reimbursement_id FROM want_keep.reimbursement_settlement_operations o JOIN want_keep.reimbursement_settlements s ON(s.household_id,s.id)=(o.household_id,o.settlement_id) JOIN LATERAL(SELECT state FROM want_keep.reimbursement_settlement_events e WHERE e.household_id=s.household_id AND e.settlement_id=s.id ORDER BY reimbursement_revision DESC LIMIT 1)e ON e.state='active' WHERE o.household_id=$1 AND o.operation_id=$2
 ) affected ORDER BY id`, principal.HouseholdID(), operationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func (s *Store) ActiveTransferUsage(ctx context.Context, principal household.Principal, key string, asset money.Asset) (money.Money, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return money.Money{}, err
	}
	var amount string
	err = q.QueryRow(ctx, `SELECT COALESCE(SUM(s.transfer_amount),0)::text FROM want_keep.reimbursement_settlements s JOIN LATERAL(SELECT state FROM want_keep.reimbursement_settlement_events e WHERE e.household_id=s.household_id AND e.settlement_id=s.id ORDER BY reimbursement_revision DESC LIMIT 1)e ON e.state='active' WHERE s.household_id=$1 AND s.transfer_key=$2 AND s.transfer_asset=$3`, principal.HouseholdID(), key, asset).Scan(&amount)
	if err != nil {
		return money.Money{}, err
	}
	return money.NewMoney(amount, asset)
}
