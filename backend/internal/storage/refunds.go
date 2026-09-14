package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Store) ActiveRefundTotals(ctx context.Context, p household.Principal, purchaseID, excludedRefundID string, asset money.Asset) (money.Money, map[string]money.Money, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return money.Money{}, nil, err
	}
	total, err := money.NewMoney("0", asset)
	if err != nil {
		return total, nil, err
	}
	rows, err := q.Query(ctx, `SELECT rr.refund_operation_id,rr.amount::text FROM want_keep.refunds r JOIN want_keep.refund_revisions rr ON (rr.household_id,rr.refund_operation_id,rr.revision)=(r.household_id,r.refund_operation_id,r.revision) WHERE r.household_id=$1 AND r.purchase_operation_id=$2 AND r.refund_operation_id<>$3 AND rr.state IN ('applied','clarification') AND rr.asset=$4 ORDER BY rr.refund_operation_id`, p.HouseholdID(), purchaseID, excludedRefundID, asset)
	if err != nil {
		return total, nil, err
	}
	for rows.Next() {
		var id, amount string
		if err = rows.Scan(&id, &amount); err != nil {
			rows.Close()
			return total, nil, err
		}
		value, parseErr := money.NewMoney(amount, asset)
		if parseErr != nil {
			rows.Close()
			return total, nil, parseErr
		}
		total, err = total.Add(value)
		if err != nil {
			rows.Close()
			return total, nil, err
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return total, nil, err
	}
	rows.Close()
	items := map[string]money.Money{}
	rows, err = q.Query(ctx, `SELECT i.item_id,i.amount::text FROM want_keep.refund_item_portions i JOIN want_keep.refunds r ON (r.household_id,r.refund_operation_id,r.revision)=(i.household_id,i.refund_operation_id,i.revision) JOIN want_keep.refund_revisions rr ON (rr.household_id,rr.refund_operation_id,rr.revision)=(r.household_id,r.refund_operation_id,r.revision) WHERE r.household_id=$1 AND r.purchase_operation_id=$2 AND r.refund_operation_id<>$3 AND rr.state IN ('applied','clarification') AND i.asset=$4 ORDER BY i.item_id`, p.HouseholdID(), purchaseID, excludedRefundID, asset)
	if err != nil {
		return total, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, amount string
		if err = rows.Scan(&id, &amount); err != nil {
			return total, nil, err
		}
		value, parseErr := money.NewMoney(amount, asset)
		if parseErr != nil {
			return total, nil, parseErr
		}
		if current, found := items[id]; found {
			items[id], err = current.Add(value)
		} else {
			items[id] = value
		}
		if err != nil {
			return total, nil, err
		}
	}
	return total, items, rows.Err()
}

func (s *Store) PurchaseValuation(ctx context.Context, p household.Principal, operationID string, revision uint64) (*expenses.ValuationBasis, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	var ref, nativeAmount, nativeAsset, reportingAmount, reportingAsset string
	err = q.QueryRow(ctx, `SELECT basis_ref,native_amount::text,native_asset,reporting_amount::text,reporting_asset FROM want_keep.transaction_historical_values WHERE household_id=$1 AND operation_id=$2 AND operation_revision=$3`, p.HouseholdID(), operationID, revision).Scan(&ref, &nativeAmount, &nativeAsset, &reportingAmount, &reportingAsset)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	native, err := money.NewMoney(nativeAmount, money.Asset(nativeAsset))
	if err != nil {
		return nil, err
	}
	reporting, err := money.NewMoney(reportingAmount, money.Asset(reportingAsset))
	if err != nil {
		return nil, err
	}
	return &expenses.ValuationBasis{Purchase: native, Value: reporting, Ref: ref}, nil
}

func (s *Store) SaveRefund(ctx context.Context, p household.Principal, refund expenses.Refund, expected uint64) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if err = refund.Validate(); err != nil {
		return err
	}
	if refund.ActorID != p.UserID() || expected >= command.MaxRevision || refund.Revision != expected+1 {
		return expenses.ErrInvalidRefund
	}
	family := p.HouseholdID()
	if expected == 0 {
		tag, execErr := scope.tx.Exec(ctx, `INSERT INTO want_keep.refunds(household_id,refund_operation_id,purchase_operation_id,revision) VALUES($1,$2,$3,1) ON CONFLICT DO NOTHING`, family, refund.OperationID, refund.PurchaseID)
		if execErr != nil {
			return execErr
		}
		if tag.RowsAffected() != 1 {
			return command.ErrVersionConflict
		}
	} else {
		tag, execErr := scope.tx.Exec(ctx, `UPDATE want_keep.refunds SET revision=$4 WHERE household_id=$1 AND refund_operation_id=$2 AND purchase_operation_id=$3 AND revision=$5`, family, refund.OperationID, refund.PurchaseID, refund.Revision, expected)
		if execErr != nil {
			return execErr
		}
		if tag.RowsAffected() != 1 {
			return command.ErrVersionConflict
		}
	}
	at, ns := splitInstant(refund.RecordedAt)
	var basisNativeAmount, basisNativeAsset, basisValueAmount, basisValueAsset, basisRef, valueAmount, valueAsset any
	if refund.Valuation != nil {
		basis, basisErr := s.PurchaseValuation(ctx, p, refund.PurchaseID, refund.PurchaseRevision)
		if basisErr != nil {
			return basisErr
		}
		if basis == nil || basis.Ref != refund.Valuation.Ref {
			return expenses.ErrInvalidRefund
		}
		basisNativeAmount, basisNativeAsset = basis.Purchase.Amount(), basis.Purchase.Asset()
		basisValueAmount, basisValueAsset, basisRef = basis.Value.Amount(), basis.Value.Asset(), basis.Ref
		valueAmount, valueAsset = refund.Valuation.Value.Amount(), refund.Valuation.Value.Asset()
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.refund_revisions(household_id,refund_operation_id,revision,purchase_operation_id,purchase_revision,refund_revision,actor_id,reason,state,expense_month,cash_date,amount,remaining,asset,valuation_basis_native_amount,valuation_basis_native_asset,valuation_basis_reporting_amount,valuation_basis_reporting_asset,valuation_basis_ref,valuation_amount,valuation_asset,recorded_at,recorded_ns) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::numeric,$13::numeric,$14,$15::numeric,$16,$17::numeric,$18,$19,$20::numeric,$21,$22,$23)`, family, refund.OperationID, refund.Revision, refund.PurchaseID, refund.PurchaseRevision, refund.RefundRevision, refund.ActorID, refund.Reason, refund.State, refund.ExpenseMonth.String()+"-01", refund.CashDate.String(), refund.Amount.Amount(), refund.Remaining.Amount(), refund.Amount.Asset(), basisNativeAmount, basisNativeAsset, basisValueAmount, basisValueAsset, basisRef, valueAmount, valueAsset, at, ns)
	if err != nil {
		return err
	}
	for position, item := range refund.Items {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.refund_item_portions(household_id,refund_operation_id,revision,purchase_operation_id,purchase_revision,position,item_id,amount,asset) VALUES($1,$2,$3,$4,$5,$6,$7,$8::numeric,$9)`, family, refund.OperationID, refund.Revision, refund.PurchaseID, refund.PurchaseRevision, position, item.ItemID, item.Amount.Amount(), item.Amount.Asset()); err != nil {
			return err
		}
	}
	if err = s.saveRefundEffects(ctx, scope, refund.OperationID, refund.Revision, "native", refund.Members, refund.Categories, refund.Unallocated); err != nil {
		return err
	}
	if refund.Valuation != nil {
		if err = s.saveRefundEffects(ctx, scope, refund.OperationID, refund.Revision, "valuation", refund.Valuation.Members, refund.Valuation.Categories, refund.Valuation.Unallocated); err != nil {
			return err
		}
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.refund_review_requests(household_id,refund_operation_id,revision,requested_at,requested_ns) VALUES($1,$2,$3,$4,$5)`, family, refund.OperationID, refund.Revision, at, ns)
	return err
}

func (s *Store) saveRefundEffects(ctx context.Context, scope *transactionScope, id string, revision uint64, basis string, members []expenses.MemberAmount, categories []expenses.CategoryAmount, unallocated []money.Money) error {
	position := 0
	insert := func(dimension, key string, amount money.Money) error {
		var memberID, categoryID any
		if dimension == "member" {
			memberID = key
		} else if dimension == "category" && key != "" {
			categoryID = key
		}
		_, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.refund_effects(household_id,refund_operation_id,revision,basis,dimension,position,member_id,category_id,amount,asset) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::numeric,$10)`, scope.principal.HouseholdID(), id, revision, basis, dimension, position, memberID, categoryID, amount.Amount(), amount.Asset())
		position++
		return err
	}
	for _, effect := range members {
		if err := insert("member", string(effect.MemberID), effect.Amount); err != nil {
			return err
		}
	}
	for _, effect := range categories {
		if err := insert("category", effect.CategoryID, effect.Amount); err != nil {
			return err
		}
	}
	for _, effect := range unallocated {
		if err := insert("unallocated", "", effect); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Refund(ctx context.Context, p household.Principal, operationID string) (expenses.Refund, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return expenses.Refund{}, false, err
	}
	refund, err := s.refund(ctx, q, p, operationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return expenses.Refund{}, false, nil
	}
	return refund, err == nil, err
}

func (s *Store) RefundCountForPurchase(ctx context.Context, p household.Principal, purchaseID string) (int, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return 0, err
	}
	var count int
	err = q.QueryRow(ctx, `SELECT count(*) FROM want_keep.refunds WHERE household_id=$1 AND purchase_operation_id=$2`, p.HouseholdID(), purchaseID).Scan(&count)
	return count, err
}

func (s *Store) RefundsForOperation(ctx context.Context, p household.Principal, operationID string) ([]expenses.Refund, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT refund_operation_id FROM want_keep.refunds WHERE household_id=$1 AND (refund_operation_id=$2 OR purchase_operation_id=$2) ORDER BY refund_operation_id`, p.HouseholdID(), operationID)
	if err != nil {
		return nil, err
	}
	var ids []string
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
	result := make([]expenses.Refund, 0, len(ids))
	for _, id := range ids {
		value, loadErr := s.refund(ctx, q, p, id)
		if loadErr != nil {
			return nil, loadErr
		}
		result = append(result, value)
	}
	return result, nil
}

func (s *Store) refund(ctx context.Context, q reader, p household.Principal, operationID string) (expenses.Refund, error) {
	var result expenses.Refund
	var month, cashDate, recordedAt time.Time
	var recordedNS int16
	var amount, remaining, asset string
	var valueAmount, valueAsset, valueRef *string
	err := q.QueryRow(ctx, `SELECT r.refund_operation_id,r.purchase_operation_id,r.revision,rr.purchase_revision,rr.refund_revision,rr.actor_id,rr.reason,rr.state,rr.expense_month,rr.cash_date,rr.amount::text,rr.remaining::text,rr.asset,rr.valuation_amount::text,rr.valuation_asset,rr.valuation_basis_ref,rr.recorded_at,rr.recorded_ns FROM want_keep.refunds r JOIN want_keep.refund_revisions rr ON (rr.household_id,rr.refund_operation_id,rr.revision)=(r.household_id,r.refund_operation_id,r.revision) WHERE r.household_id=$1 AND r.refund_operation_id=$2`, p.HouseholdID(), operationID).Scan(&result.OperationID, &result.PurchaseID, &result.Revision, &result.PurchaseRevision, &result.RefundRevision, &result.ActorID, &result.Reason, &result.State, &month, &cashDate, &amount, &remaining, &asset, &valueAmount, &valueAsset, &valueRef, &recordedAt, &recordedNS)
	if err != nil {
		return result, err
	}
	result.ExpenseMonth, err = calendar.ParseMonth(month.Format("2006-01"))
	if err == nil {
		result.CashDate, err = calendar.ParseDate(cashDate.Format(time.DateOnly))
	}
	if err == nil {
		result.RecordedAt, err = restoreInstant(recordedAt, recordedNS)
	}
	if err == nil {
		result.Amount, err = money.NewMoney(amount, money.Asset(asset))
	}
	if err == nil {
		result.Remaining, err = money.NewMoney(remaining, money.Asset(asset))
	}
	if err != nil {
		return result, err
	}
	if valueAmount != nil && valueAsset != nil && valueRef != nil {
		value, valueErr := money.NewMoney(*valueAmount, money.Asset(*valueAsset))
		if valueErr != nil {
			return result, valueErr
		}
		result.Valuation = &expenses.Valuation{Value: value, Ref: *valueRef}
	}
	return result, s.loadRefundChildren(ctx, q, p, &result)
}

func (s *Store) loadRefundChildren(ctx context.Context, q reader, p household.Principal, result *expenses.Refund) error {
	rows, err := q.Query(ctx, `SELECT item_id,amount::text,asset FROM want_keep.refund_item_portions WHERE household_id=$1 AND refund_operation_id=$2 AND revision=$3 ORDER BY position`, p.HouseholdID(), result.OperationID, result.Revision)
	if err != nil {
		return err
	}
	for rows.Next() {
		var item expenses.ItemPortion
		var amount, asset string
		if err = rows.Scan(&item.ItemID, &amount, &asset); err != nil {
			rows.Close()
			return err
		}
		item.Amount, err = money.NewMoney(amount, money.Asset(asset))
		if err != nil {
			rows.Close()
			return err
		}
		result.Items = append(result.Items, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	return s.loadRefundEffectsMore(ctx, q, p, result)
}

func (s *Store) loadRefundEffects(ctx context.Context, q reader, p household.Principal, result *expenses.Refund, basis string) error {
	rows, err := q.Query(ctx, `SELECT dimension,COALESCE(member_id::text,category_id::text,''),amount::text,asset FROM want_keep.refund_effects WHERE household_id=$1 AND refund_operation_id=$2 AND revision=$3 AND basis=$4 ORDER BY position`, p.HouseholdID(), result.OperationID, result.Revision, basis)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var dimension, key, amount, asset string
		if err = rows.Scan(&dimension, &key, &amount, &asset); err != nil {
			return err
		}
		value, parseErr := money.NewMoney(amount, money.Asset(asset))
		if parseErr != nil {
			return parseErr
		}
		switch dimension {
		case "member":
			if basis == "native" {
				result.Members = append(result.Members, expenses.MemberAmount{MemberID: household.MembershipID(key), Amount: value})
			} else {
				result.Valuation.Members = append(result.Valuation.Members, expenses.MemberAmount{MemberID: household.MembershipID(key), Amount: value})
			}
		case "category":
			if basis == "native" {
				result.Categories = append(result.Categories, expenses.CategoryAmount{CategoryID: key, Amount: value})
			} else {
				result.Valuation.Categories = append(result.Valuation.Categories, expenses.CategoryAmount{CategoryID: key, Amount: value})
			}
		case "unallocated":
			if basis == "native" {
				result.Unallocated = append(result.Unallocated, value)
			} else {
				result.Valuation.Unallocated = append(result.Valuation.Unallocated, value)
			}
		default:
			return expenses.ErrInvalidRefund
		}
	}
	return rows.Err()
}

func (s *Store) loadRefundEffectsMore(ctx context.Context, q reader, p household.Principal, result *expenses.Refund) error {
	if err := s.loadRefundEffects(ctx, q, p, result, "native"); err != nil {
		return err
	}
	if result.Valuation != nil {
		if err := s.loadRefundEffects(ctx, q, p, result, "valuation"); err != nil {
			return err
		}
	}
	return result.Validate()
}
