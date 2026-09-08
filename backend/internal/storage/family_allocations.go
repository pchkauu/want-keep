package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Store) HouseholdMemberships(ctx context.Context, principal household.Principal) ([]household.Membership, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT id,user_id,active FROM want_keep.memberships WHERE household_id=$1 ORDER BY id`, principal.HouseholdID())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []household.Membership{}
	for rows.Next() {
		member := household.Membership{HouseholdID: principal.HouseholdID()}
		if err = rows.Scan(&member.ID, &member.UserID, &member.Active); err != nil {
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
	recordedAt, recordedNS := splitInstant(rule.RecordedAt)
	_, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.allocation_rule_revisions(household_id,rule_id,revision,priority,state,merchant_id,category_id,actor_id,command_id,recorded_at,recorded_ns) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,NULLIF($7,'')::uuid,$8,NULLIF($9,'')::uuid,$10,$11)`, rule.HouseholdID, rule.ID, rule.Revision, rule.Priority, rule.State, rule.Condition.MerchantID, rule.Condition.CategoryID, rule.ActorID, commands.CurrentCommandID(ctx), recordedAt, recordedNS)
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
	rows, err := q.Query(ctx, `SELECT r.id,r.revision,r.priority,r.state,COALESCE(r.merchant_id::text,''),COALESCE(r.category_id::text,''),r.actor_id,v.recorded_at,v.recorded_ns,
 array_agg(s.member_id::text ORDER BY s.position),array_agg(s.share::text ORDER BY s.position)
FROM want_keep.allocation_rules r
JOIN want_keep.allocation_rule_revisions v ON (v.household_id,v.rule_id,v.revision)=(r.household_id,r.id,r.revision)
JOIN want_keep.allocation_rule_shares s ON (s.household_id,s.rule_id,s.revision)=(r.household_id,r.id,r.revision)
WHERE r.household_id=$1 AND ($2='' OR r.id>$2::uuid)
GROUP BY r.household_id,r.id,r.revision,r.priority,r.state,r.merchant_id,r.category_id,r.actor_id,v.recorded_at,v.recorded_ns
ORDER BY r.id LIMIT $3`, principal.HouseholdID(), after, limit+1)
	if err != nil {
		return nil, "", err
	}
	result, err := scanAllocationRules(rows, principal.HouseholdID())
	if err != nil {
		return nil, "", err
	}
	var next string
	if len(result) > limit {
		result = result[:limit]
		next = result[len(result)-1].ID
	}
	return result, next, nil
}

func (s *Store) MatchingAllocationRules(ctx context.Context, principal household.Principal, merchantID, categoryID string) ([]allocation.Rule, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `WITH eligible AS (
 SELECT r.household_id,r.id,r.revision,r.priority,r.state,r.merchant_id,r.category_id,r.actor_id,v.recorded_at,v.recorded_ns
 FROM want_keep.allocation_rules r
 JOIN want_keep.allocation_rule_revisions v ON (v.household_id,v.rule_id,v.revision)=(r.household_id,r.id,r.revision)
 WHERE r.household_id=$1 AND r.state='active' AND (r.merchant_id IS NULL OR r.merchant_id=$2::uuid) AND (r.category_id IS NULL OR r.category_id=$3::uuid)
), best AS (SELECT MIN(priority) AS priority FROM eligible)
SELECT e.id,e.revision,e.priority,e.state,COALESCE(e.merchant_id::text,''),COALESCE(e.category_id::text,''),e.actor_id,e.recorded_at,e.recorded_ns,
 array_agg(s.member_id::text ORDER BY s.position),array_agg(s.share::text ORDER BY s.position)
FROM eligible e
JOIN best b ON b.priority=e.priority
JOIN want_keep.allocation_rule_shares s ON (s.household_id,s.rule_id,s.revision)=(e.household_id,e.id,e.revision)
GROUP BY e.household_id,e.id,e.revision,e.priority,e.state,e.merchant_id,e.category_id,e.actor_id,e.recorded_at,e.recorded_ns
ORDER BY e.priority,e.id`, principal.HouseholdID(), nullableUUID(merchantID), nullableUUID(categoryID))
	if err != nil {
		return nil, err
	}
	return scanAllocationRules(rows, principal.HouseholdID())
}

func (s *Store) MatchingAllocationRulesAt(ctx context.Context, principal household.Principal, merchantID, categoryID string, at calendar.Instant) ([]allocation.Rule, error) {
	q, err := s.reader(ctx, principal)
	if err != nil {
		return nil, err
	}
	if at.String() == "" {
		return nil, allocation.ErrInvalidRule
	}
	recordedAt, recordedNS := splitInstant(at)
	rows, err := q.Query(ctx, `WITH historical AS (
 SELECT DISTINCT ON (rule_id) household_id,rule_id,revision,priority,state,merchant_id,category_id,actor_id,recorded_at,recorded_ns
 FROM want_keep.allocation_rule_revisions
 WHERE household_id=$1 AND (recorded_at,recorded_ns)<=($4,$5)
 ORDER BY rule_id,revision DESC
), eligible AS (
 SELECT * FROM historical
 WHERE state='active' AND (merchant_id IS NULL OR merchant_id=$2::uuid) AND (category_id IS NULL OR category_id=$3::uuid)
), best AS (SELECT MIN(priority) AS priority FROM eligible)
SELECT e.rule_id,e.revision,e.priority,e.state,COALESCE(e.merchant_id::text,''),COALESCE(e.category_id::text,''),e.actor_id,e.recorded_at,e.recorded_ns,
 array_agg(s.member_id::text ORDER BY s.position),array_agg(s.share::text ORDER BY s.position)
FROM eligible e
JOIN best b ON b.priority=e.priority
JOIN want_keep.allocation_rule_shares s ON (s.household_id,s.rule_id,s.revision)=(e.household_id,e.rule_id,e.revision)
GROUP BY e.household_id,e.rule_id,e.revision,e.priority,e.state,e.merchant_id,e.category_id,e.actor_id,e.recorded_at,e.recorded_ns
ORDER BY e.priority,e.rule_id`, principal.HouseholdID(), nullableUUID(merchantID), nullableUUID(categoryID), recordedAt, recordedNS)
	if err != nil {
		return nil, err
	}
	return scanAllocationRules(rows, principal.HouseholdID())
}

func scanAllocationRules(rows pgx.Rows, householdID household.HouseholdID) ([]allocation.Rule, error) {
	defer rows.Close()
	result := []allocation.Rule{}
	for rows.Next() {
		rule := allocation.Rule{HouseholdID: householdID}
		var recordedAt time.Time
		var recordedNS int16
		var memberIDs, shares []string
		if err := rows.Scan(&rule.ID, &rule.Revision, &rule.Priority, &rule.State, &rule.Condition.MerchantID, &rule.Condition.CategoryID, &rule.ActorID, &recordedAt, &recordedNS, &memberIDs, &shares); err != nil {
			return nil, err
		}
		if len(memberIDs) != len(shares) {
			return nil, allocation.ErrInvalidRule
		}
		var err error
		rule.RecordedAt, err = restoreInstant(recordedAt, recordedNS)
		if err != nil {
			return nil, err
		}
		for index, memberID := range memberIDs {
			rule.Shares = append(rule.Shares, allocation.Share{MemberID: household.MembershipID(memberID), Value: shares[index]})
		}
		if err = rule.Validate(); err != nil {
			return nil, err
		}
		result = append(result, rule)
	}
	return result, rows.Err()
}

func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func loadAllocationRule(ctx context.Context, q reader, householdID household.HouseholdID, id string) (allocation.Rule, error) {
	rule := allocation.Rule{ID: id, HouseholdID: householdID}
	var recordedAt time.Time
	var recordedNS int16
	err := q.QueryRow(ctx, `SELECT r.revision,r.priority,r.state,COALESCE(r.merchant_id::text,''),COALESCE(r.category_id::text,''),r.actor_id,v.recorded_at,v.recorded_ns FROM want_keep.allocation_rules r JOIN want_keep.allocation_rule_revisions v ON (v.household_id,v.rule_id,v.revision)=(r.household_id,r.id,r.revision) WHERE r.household_id=$1 AND r.id=$2`, householdID, id).Scan(&rule.Revision, &rule.Priority, &rule.State, &rule.Condition.MerchantID, &rule.Condition.CategoryID, &rule.ActorID, &recordedAt, &recordedNS)
	if err != nil {
		return rule, err
	}
	rule.RecordedAt, err = restoreInstant(recordedAt, recordedNS)
	if err != nil {
		return rule, err
	}
	return loadAllocationRuleShares(ctx, q, rule)
}

func loadAllocationRuleShares(ctx context.Context, q reader, rule allocation.Rule) (allocation.Rule, error) {
	rows, err := q.Query(ctx, `SELECT member_id,share::text FROM want_keep.allocation_rule_shares WHERE household_id=$1 AND rule_id=$2 AND revision=$3 ORDER BY position`, rule.HouseholdID, rule.ID, rule.Revision)
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
	if snapshot.Basis != nil {
		fallback = *snapshot.Basis
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
	if snapshot.Basis != nil {
		for index, input := range snapshot.Basis.Members {
			var amount, asset, share any
			if input.Amount != nil {
				amount, asset = input.Amount.Amount(), input.Amount.Asset()
			}
			if input.Share != "" {
				share = input.Share
			}
			if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_allocation_fallback_inputs(household_id,operation_id,revision,snapshot_position,position,member_id,amount,asset,share) VALUES($1,$2,$3,$4,$5,$6,$7::numeric,$8,$9::numeric)`, family, revision.OperationID, revision.Revision, position, index, input.MemberID, amount, asset, share); err != nil {
				return err
			}
		}
		for _, ref := range snapshot.Basis.RuleRefs {
			if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_allocation_fallback_rule_refs(household_id,operation_id,revision,snapshot_position,rule_id,rule_revision) VALUES($1,$2,$3,$4,$5,$6)`, family, revision.OperationID, revision.Revision, position, ref.ID, ref.Revision); err != nil {
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
	family, operationID, revisionNumber := principal.HouseholdID(), revision.OperationID, revision.Revision
	rows, err := q.Query(ctx, `SELECT position,COALESCE(item_id::text,''),state,purpose,mode,origin,reason,fallback_mode,fallback_purpose,fallback_origin,fallback_reason FROM want_keep.ledger_allocation_snapshots WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY position`, family, operationID, revisionNumber)
	if err != nil {
		return err
	}
	snapshots := storedAllocationSnapshots{}
	for rows.Next() {
		stored := &storedAllocationSnapshot{}
		var position int
		var basis ledger.AllocationInput
		if err = rows.Scan(&position, &stored.itemID, &stored.value.State, &stored.value.Purpose, &stored.value.Mode, &stored.value.Origin, &stored.value.Reason, &basis.Mode, &basis.Purpose, &basis.Origin, &basis.Reason); err != nil {
			rows.Close()
			return err
		}
		if _, exists := snapshots[position]; exists {
			rows.Close()
			return ledger.ErrInvalidAllocation
		}
		if basis.Mode != "" {
			stored.value.Basis = &basis
		}
		snapshots[position] = stored
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(snapshots) == 0 {
		return nil
	}
	rows, err = q.Query(ctx, `SELECT snapshot_position,member_id,amount::text,asset,share::text FROM want_keep.ledger_allocation_inputs WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY snapshot_position,position`, family, operationID, revisionNumber)
	if err != nil {
		return err
	}
	if err = scanAllocationInputs(rows, snapshots, false); err != nil {
		return err
	}
	rows, err = q.Query(ctx, `SELECT snapshot_position,member_id,amount::text,asset,share::text FROM want_keep.ledger_allocation_fallback_inputs WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY snapshot_position,position`, family, operationID, revisionNumber)
	if err != nil {
		return err
	}
	if err = scanAllocationInputs(rows, snapshots, true); err != nil {
		return err
	}
	rows, err = q.Query(ctx, `SELECT snapshot_position,member_id,asset,amount::text FROM want_keep.ledger_allocation_member_amounts WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY snapshot_position,member_id,asset`, family, operationID, revisionNumber)
	if err != nil {
		return err
	}
	if err = scanAllocationMemberAmounts(rows, snapshots); err != nil {
		return err
	}
	rows, err = q.Query(ctx, `SELECT snapshot_position,asset,amount::text FROM want_keep.ledger_allocation_unallocated WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY snapshot_position,asset`, family, operationID, revisionNumber)
	if err != nil {
		return err
	}
	if err = scanAllocationUnallocated(rows, snapshots); err != nil {
		return err
	}
	rows, err = q.Query(ctx, `SELECT snapshot_position,rule_id,rule_revision FROM want_keep.ledger_allocation_rule_refs WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY snapshot_position,rule_id`, family, operationID, revisionNumber)
	if err != nil {
		return err
	}
	if err = scanAllocationRuleRefs(rows, snapshots, false); err != nil {
		return err
	}
	rows, err = q.Query(ctx, `SELECT snapshot_position,rule_id,rule_revision FROM want_keep.ledger_allocation_fallback_rule_refs WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY snapshot_position,rule_id`, family, operationID, revisionNumber)
	if err != nil {
		return err
	}
	if err = scanAllocationRuleRefs(rows, snapshots, true); err != nil {
		return err
	}
	aggregate, err := snapshots.at(0)
	if err != nil {
		return err
	}
	if aggregate.itemID != "" || aggregate.value.Validate() != nil {
		return ledger.ErrInvalidAllocation
	}
	revision.Allocation = aggregate.value
	for position, stored := range snapshots {
		if position == 0 {
			continue
		}
		if position > len(revision.ReceiptItems) || stored.itemID != revision.ReceiptItems[position-1].ID || stored.value.Validate() != nil {
			return ledger.ErrInvalidAllocation
		}
		revision.ReceiptItems[position-1].Allocation = stored.value
	}
	return nil
}

type storedAllocationSnapshot struct {
	value  ledger.AllocationSnapshot
	itemID string
}

type storedAllocationSnapshots map[int]*storedAllocationSnapshot

func (s storedAllocationSnapshots) at(position int) (*storedAllocationSnapshot, error) {
	snapshot, ok := s[position]
	if !ok {
		return nil, ledger.ErrInvalidAllocation
	}
	return snapshot, nil
}

func scanAllocationInputs(rows pgx.Rows, snapshots storedAllocationSnapshots, basis bool) error {
	defer rows.Close()
	for rows.Next() {
		var position int
		var input ledger.AllocationMemberInput
		var amount, asset, share *string
		if err := rows.Scan(&position, &input.MemberID, &amount, &asset, &share); err != nil {
			return err
		}
		snapshot, err := snapshots.at(position)
		if err != nil {
			return err
		}
		if amount != nil && asset != nil {
			value, parseErr := money.NewMoney(*amount, money.Asset(*asset))
			if parseErr != nil {
				return parseErr
			}
			input.Amount = &value
		}
		if share != nil {
			input.Share = *share
		}
		if basis {
			if snapshot.value.Basis == nil {
				return ledger.ErrInvalidAllocation
			}
			snapshot.value.Basis.Members = append(snapshot.value.Basis.Members, input)
		} else {
			snapshot.value.Inputs = append(snapshot.value.Inputs, input)
		}
	}
	return rows.Err()
}

func scanAllocationMemberAmounts(rows pgx.Rows, snapshots storedAllocationSnapshots) error {
	defer rows.Close()
	for rows.Next() {
		var position int
		var member ledger.MemberAmount
		var asset, amount string
		if err := rows.Scan(&position, &member.MemberID, &asset, &amount); err != nil {
			return err
		}
		snapshot, err := snapshots.at(position)
		if err != nil {
			return err
		}
		member.Money, err = money.NewMoney(amount, money.Asset(asset))
		if err != nil {
			return err
		}
		snapshot.value.Members = append(snapshot.value.Members, member)
	}
	return rows.Err()
}

func scanAllocationUnallocated(rows pgx.Rows, snapshots storedAllocationSnapshots) error {
	defer rows.Close()
	for rows.Next() {
		var position int
		var asset, amount string
		if err := rows.Scan(&position, &asset, &amount); err != nil {
			return err
		}
		snapshot, err := snapshots.at(position)
		if err != nil {
			return err
		}
		value, err := money.NewMoney(amount, money.Asset(asset))
		if err != nil {
			return err
		}
		snapshot.value.Unallocated = append(snapshot.value.Unallocated, value)
	}
	return rows.Err()
}

func scanAllocationRuleRefs(rows pgx.Rows, snapshots storedAllocationSnapshots, basis bool) error {
	defer rows.Close()
	for rows.Next() {
		var position int
		var ref ledger.AllocationRuleRef
		if err := rows.Scan(&position, &ref.ID, &ref.Revision); err != nil {
			return err
		}
		snapshot, err := snapshots.at(position)
		if err != nil {
			return err
		}
		if basis {
			if snapshot.value.Basis == nil {
				return ledger.ErrInvalidAllocation
			}
			snapshot.value.Basis.RuleRefs = append(snapshot.value.Basis.RuleRefs, ref)
		} else {
			snapshot.value.RuleRefs = append(snapshot.value.RuleRefs, ref)
		}
	}
	return rows.Err()
}
