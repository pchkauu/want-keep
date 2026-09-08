package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Store) ActiveMemberships(ctx context.Context, principal household.Principal) ([]household.Membership, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT id,user_id FROM want_keep.memberships WHERE household_id=$1 AND active ORDER BY id`, principal.HouseholdID())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []household.Membership{}
	for rows.Next() {
		member := household.Membership{HouseholdID: principal.HouseholdID(), Active: true}
		if err = rows.Scan(&member.ID, &member.UserID); err != nil {
			return nil, err
		}
		result = append(result, member)
	}
	return result, rows.Err()
}

func (s *Store) CreateAllocationRule(ctx context.Context, rule allocation.Rule) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if rule.HouseholdID != scope.principal.HouseholdID() || rule.ActorID != scope.principal.UserID() {
		return household.ErrForbidden
	}
	if err = rule.Validate(); err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.allocation_rules(household_id,id,revision,priority,state,merchant_id,category_id,actor_id) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,NULLIF($7,'')::uuid,$8)`, rule.HouseholdID, rule.ID, rule.Revision, rule.Priority, rule.State, rule.Condition.MerchantID, rule.Condition.CategoryID, rule.ActorID)
	if err != nil {
		return err
	}
	return s.insertAllocationRuleRevision(ctx, scope, rule)
}

func (s *Store) SaveAllocationRule(ctx context.Context, rule allocation.Rule, expected uint64) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if rule.HouseholdID != scope.principal.HouseholdID() || rule.ActorID != scope.principal.UserID() {
		return household.ErrForbidden
	}
	if rule.Validate() != nil || rule.Revision != expected+1 {
		return allocation.ErrInvalidRule
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.allocation_rules SET revision=$3,priority=$4,state=$5,merchant_id=NULLIF($6,'')::uuid,category_id=NULLIF($7,'')::uuid,actor_id=$8 WHERE household_id=$1 AND id=$2 AND revision=$9`, rule.HouseholdID, rule.ID, rule.Revision, rule.Priority, rule.State, rule.Condition.MerchantID, rule.Condition.CategoryID, rule.ActorID, expected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return commands.Rejection{Code: "version_conflict"}
	}
	return s.insertAllocationRuleRevision(ctx, scope, rule)
}

func (s *Store) insertAllocationRuleRevision(ctx context.Context, scope *transactionScope, rule allocation.Rule) error {
	_, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.allocation_rule_revisions(household_id,rule_id,revision,priority,state,merchant_id,category_id,actor_id,command_id) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,NULLIF($7,'')::uuid,$8,NULLIF($9,'')::uuid)`, rule.HouseholdID, rule.ID, rule.Revision, rule.Priority, rule.State, rule.Condition.MerchantID, rule.Condition.CategoryID, rule.ActorID, commands.CurrentCommandID(ctx))
	if err != nil {
		return err
	}
	for position, share := range rule.Shares {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.allocation_rule_shares(household_id,rule_id,revision,position,member_id,share) VALUES($1,$2,$3,$4,$5,$6::numeric)`, rule.HouseholdID, rule.ID, rule.Revision, position, share.MemberID, share.Value); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) AllocationRule(ctx context.Context, principal household.Principal, id string) (allocation.Rule, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return allocation.Rule{}, err
	}
	rule, err := loadAllocationRule(ctx, q, principal.HouseholdID(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		return rule, allocation.ErrRuleNotFound
	}
	return rule, err
}

func (s *Store) AllocationRules(ctx context.Context, principal household.Principal, after string, limit int) ([]allocation.Rule, string, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return nil, "", err
	}
	rows, err := q.Query(ctx, `SELECT id FROM want_keep.allocation_rules WHERE household_id=$1 AND ($2='' OR id>$2::uuid) ORDER BY id LIMIT $3`, principal.HouseholdID(), after, limit+1)
	if err != nil {
		return nil, "", err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, "", err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, "", err
	}
	rows.Close()
	var next string
	if len(ids) > limit {
		ids = ids[:limit]
		next = ids[len(ids)-1]
	}
	result := make([]allocation.Rule, 0, len(ids))
	for _, id := range ids {
		rule, err := loadAllocationRule(ctx, q, principal.HouseholdID(), id)
		if err != nil {
			return nil, "", err
		}
		result = append(result, rule)
	}
	return result, next, nil
}

func (s *Store) MatchingAllocationRules(ctx context.Context, principal household.Principal, merchantID, categoryID string) ([]allocation.Rule, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT id FROM want_keep.allocation_rules WHERE household_id=$1 AND state='active' AND (merchant_id IS NULL OR merchant_id=$2::uuid) AND (category_id IS NULL OR category_id=$3::uuid) ORDER BY priority,id`, principal.HouseholdID(), nullableUUID(merchantID), nullableUUID(categoryID))
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	result := make([]allocation.Rule, 0, len(ids))
	for _, id := range ids {
		rule, err := loadAllocationRule(ctx, q, principal.HouseholdID(), id)
		if err != nil {
			return nil, err
		}
		result = append(result, rule)
	}
	return result, nil
}

func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func loadAllocationRule(ctx context.Context, q reader, householdID household.HouseholdID, id string) (allocation.Rule, error) {
	rule := allocation.Rule{ID: id, HouseholdID: householdID}
	err := q.QueryRow(ctx, `SELECT revision,priority,state,COALESCE(merchant_id::text,''),COALESCE(category_id::text,''),actor_id FROM want_keep.allocation_rules WHERE household_id=$1 AND id=$2`, householdID, id).Scan(&rule.Revision, &rule.Priority, &rule.State, &rule.Condition.MerchantID, &rule.Condition.CategoryID, &rule.ActorID)
	if err != nil {
		return rule, err
	}
	rows, err := q.Query(ctx, `SELECT member_id,share::text FROM want_keep.allocation_rule_shares WHERE household_id=$1 AND rule_id=$2 AND revision=$3 ORDER BY position`, householdID, id, rule.Revision)
	if err != nil {
		return rule, err
	}
	defer rows.Close()
	for rows.Next() {
		var share allocation.Share
		if err = rows.Scan(&share.MemberID, &share.Value); err != nil {
			return rule, err
		}
		rule.Shares = append(rule.Shares, share)
	}
	if err = rows.Err(); err != nil {
		return rule, err
	}
	return rule, rule.Validate()
}

func (s *Store) saveLedgerAllocations(ctx context.Context, revision ledger.Revision) error {
	if err := s.saveLedgerAllocation(ctx, revision, 0, "", revision.Allocation); err != nil {
		return err
	}
	for index, item := range revision.ReceiptItems {
		if item.Allocation.State == "" {
			continue
		}
		if err := s.saveLedgerAllocation(ctx, revision, index+1, item.ID, item.Allocation); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) saveLedgerAllocation(ctx context.Context, revision ledger.Revision, position int, itemID string, snapshot ledger.AllocationSnapshot) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if err = snapshot.Validate(); err != nil {
		return err
	}
	family := scope.principal.HouseholdID()
	var fallback ledger.AllocationInput
	if snapshot.Fallback != nil {
		fallback = *snapshot.Fallback
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_allocation_snapshots(household_id,operation_id,revision,position,item_id,state,purpose,mode,origin,reason,fallback_mode,fallback_purpose,fallback_origin,fallback_reason) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, family, revision.OperationID, revision.Revision, position, itemID, snapshot.State, snapshot.Purpose, snapshot.Mode, snapshot.Origin, snapshot.Reason, fallback.Mode, fallback.Purpose, fallback.Origin, fallback.Reason)
	if err != nil {
		return err
	}
	for index, input := range snapshot.Inputs {
		var amount, asset, share any
		if input.Amount != nil {
			amount, asset = input.Amount.Amount(), input.Amount.Asset()
		}
		if input.Share != "" {
			share = input.Share
		}
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_allocation_inputs(household_id,operation_id,revision,snapshot_position,position,member_id,amount,asset,share) VALUES($1,$2,$3,$4,$5,$6,$7::numeric,$8,$9::numeric)`, family, revision.OperationID, revision.Revision, position, index, input.MemberID, amount, asset, share); err != nil {
			return err
		}
	}
	if snapshot.Fallback != nil {
		for index, input := range snapshot.Fallback.Members {
			var amount, asset, share any
			if input.Amount != nil {
				amount, asset = input.Amount.Amount(), input.Amount.Asset()
			}
			if input.Share != "" {
				share = input.Share
			}
			if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_allocation_fallback_inputs(household_id,operation_id,revision,snapshot_position,position,member_id,amount,asset,share) VALUES($1,$2,$3,0,$4,$5,$6::numeric,$7,$8::numeric)`, family, revision.OperationID, revision.Revision, index, input.MemberID, amount, asset, share); err != nil {
				return err
			}
		}
		for _, ref := range snapshot.Fallback.RuleRefs {
			if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_allocation_fallback_rule_refs(household_id,operation_id,revision,rule_id,rule_revision) VALUES($1,$2,$3,$4,$5)`, family, revision.OperationID, revision.Revision, ref.ID, ref.Revision); err != nil {
				return err
			}
		}
	}
	for _, member := range snapshot.Members {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_allocation_member_amounts(household_id,operation_id,revision,snapshot_position,member_id,asset,amount) VALUES($1,$2,$3,$4,$5,$6,$7::numeric)`, family, revision.OperationID, revision.Revision, position, member.MemberID, member.Money.Asset(), member.Money.Amount()); err != nil {
			return err
		}
	}
	for _, amount := range snapshot.Unallocated {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_allocation_unallocated(household_id,operation_id,revision,snapshot_position,asset,amount) VALUES($1,$2,$3,$4,$5,$6::numeric)`, family, revision.OperationID, revision.Revision, position, amount.Asset(), amount.Amount()); err != nil {
			return err
		}
	}
	for _, ref := range snapshot.RuleRefs {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_allocation_rule_refs(household_id,operation_id,revision,snapshot_position,rule_id,rule_revision) VALUES($1,$2,$3,$4,$5,$6)`, family, revision.OperationID, revision.Revision, position, ref.ID, ref.Revision); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadLedgerAllocations(ctx context.Context, q reader, principal household.Principal, revision *ledger.Revision) error {
	snapshot, err := loadLedgerAllocation(ctx, q, principal, revision.OperationID, revision.Revision, 0)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	revision.Allocation = snapshot
	for index := range revision.ReceiptItems {
		itemSnapshot, err := loadLedgerAllocation(ctx, q, principal, revision.OperationID, revision.Revision, index+1)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		revision.ReceiptItems[index].Allocation = itemSnapshot
	}
	return nil
}

func loadLedgerAllocation(ctx context.Context, q reader, principal household.Principal, operationID string, revision uint64, position int) (ledger.AllocationSnapshot, error) {
	snapshot := ledger.AllocationSnapshot{}
	var fallbackMode ledger.AllocationMode
	var fallbackPurpose ledger.AllocationPurpose
	var fallbackOrigin ledger.AllocationOrigin
	var fallbackReason string
	err := q.QueryRow(ctx, `SELECT state,purpose,mode,origin,reason,fallback_mode,fallback_purpose,fallback_origin,fallback_reason FROM want_keep.ledger_allocation_snapshots WHERE household_id=$1 AND operation_id=$2 AND revision=$3 AND position=$4`, principal.HouseholdID(), operationID, revision, position).Scan(&snapshot.State, &snapshot.Purpose, &snapshot.Mode, &snapshot.Origin, &snapshot.Reason, &fallbackMode, &fallbackPurpose, &fallbackOrigin, &fallbackReason)
	if err != nil {
		return snapshot, err
	}
	if fallbackMode != "" {
		snapshot.Fallback = &ledger.AllocationInput{Mode: fallbackMode, Purpose: fallbackPurpose, Origin: fallbackOrigin, Reason: fallbackReason}
	}
	rows, err := q.Query(ctx, `SELECT member_id,amount::text,asset,share::text FROM want_keep.ledger_allocation_inputs WHERE household_id=$1 AND operation_id=$2 AND revision=$3 AND snapshot_position=$4 ORDER BY position`, principal.HouseholdID(), operationID, revision, position)
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var input ledger.AllocationMemberInput
		var amount, asset, share *string
		if err = rows.Scan(&input.MemberID, &amount, &asset, &share); err != nil {
			rows.Close()
			return snapshot, err
		}
		if amount != nil && asset != nil {
			value, parseErr := money.NewMoney(*amount, money.Asset(*asset))
			if parseErr != nil {
				rows.Close()
				return snapshot, parseErr
			}
			input.Amount = &value
		}
		if share != nil {
			input.Share = *share
		}
		snapshot.Inputs = append(snapshot.Inputs, input)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return snapshot, err
	}
	rows.Close()
	if snapshot.Fallback != nil {
		rows, err = q.Query(ctx, `SELECT member_id,amount::text,asset,share::text FROM want_keep.ledger_allocation_fallback_inputs WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY position`, principal.HouseholdID(), operationID, revision)
		if err != nil {
			return snapshot, err
		}
		for rows.Next() {
			var input ledger.AllocationMemberInput
			var amount, asset, share *string
			if err = rows.Scan(&input.MemberID, &amount, &asset, &share); err != nil {
				rows.Close()
				return snapshot, err
			}
			if amount != nil && asset != nil {
				value, parseErr := money.NewMoney(*amount, money.Asset(*asset))
				if parseErr != nil {
					rows.Close()
					return snapshot, parseErr
				}
				input.Amount = &value
			}
			if share != nil {
				input.Share = *share
			}
			snapshot.Fallback.Members = append(snapshot.Fallback.Members, input)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return snapshot, err
		}
		rows.Close()
		rows, err = q.Query(ctx, `SELECT rule_id,rule_revision FROM want_keep.ledger_allocation_fallback_rule_refs WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY rule_id`, principal.HouseholdID(), operationID, revision)
		if err != nil {
			return snapshot, err
		}
		for rows.Next() {
			var ref ledger.AllocationRuleRef
			if err = rows.Scan(&ref.ID, &ref.Revision); err != nil {
				rows.Close()
				return snapshot, err
			}
			snapshot.Fallback.RuleRefs = append(snapshot.Fallback.RuleRefs, ref)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return snapshot, err
		}
		rows.Close()
	}
	rows, err = q.Query(ctx, `SELECT member_id,asset,amount::text FROM want_keep.ledger_allocation_member_amounts WHERE household_id=$1 AND operation_id=$2 AND revision=$3 AND snapshot_position=$4 ORDER BY member_id,asset`, principal.HouseholdID(), operationID, revision, position)
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var member ledger.MemberAmount
		var asset, amount string
		if err = rows.Scan(&member.MemberID, &asset, &amount); err != nil {
			rows.Close()
			return snapshot, err
		}
		member.Money, err = money.NewMoney(amount, money.Asset(asset))
		if err != nil {
			rows.Close()
			return snapshot, err
		}
		snapshot.Members = append(snapshot.Members, member)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return snapshot, err
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT asset,amount::text FROM want_keep.ledger_allocation_unallocated WHERE household_id=$1 AND operation_id=$2 AND revision=$3 AND snapshot_position=$4 ORDER BY asset`, principal.HouseholdID(), operationID, revision, position)
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var asset, amount string
		if err = rows.Scan(&asset, &amount); err != nil {
			rows.Close()
			return snapshot, err
		}
		value, parseErr := money.NewMoney(amount, money.Asset(asset))
		if parseErr != nil {
			rows.Close()
			return snapshot, parseErr
		}
		snapshot.Unallocated = append(snapshot.Unallocated, value)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return snapshot, err
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT rule_id,rule_revision FROM want_keep.ledger_allocation_rule_refs WHERE household_id=$1 AND operation_id=$2 AND revision=$3 AND snapshot_position=$4 ORDER BY rule_id`, principal.HouseholdID(), operationID, revision, position)
	if err != nil {
		return snapshot, err
	}
	defer rows.Close()
	for rows.Next() {
		var ref ledger.AllocationRuleRef
		if err = rows.Scan(&ref.ID, &ref.Revision); err != nil {
			return snapshot, err
		}
		snapshot.RuleRefs = append(snapshot.RuleRefs, ref)
	}
	if err = rows.Err(); err != nil {
		return snapshot, err
	}
	return snapshot, snapshot.Validate()
}
