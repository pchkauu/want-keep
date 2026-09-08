package domain

import calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"

func (r Revision) InTimezone(zone calendar.Timezone) (Revision, error) {
	r = r.Clone()
	r.Timezone = zone
	var err error
	r.CashDate, err = r.OccurredAt.DateIn(zone)
	if err != nil {
		return r, err
	}
	if r.Type != Opening {
		r.ExpenseMonth, err = calendar.ParseMonth(r.CashDate.String()[:7])
	}
	return r, err
}

func (r Revision) SameFacts(other Revision) bool {
	if len(r.Postings) != len(other.Postings) {
		return false
	}
	for i, p := range r.Postings {
		if p.FeeID != other.Postings[i].FeeID {
			return false
		}
	}
	if (r.Correspondence == nil) != (other.Correspondence == nil) || r.Correspondence != nil && *r.Correspondence != *other.Correspondence {
		return false
	}
	if r.OperationID != other.OperationID || r.Type != other.Type || r.State != other.State || r.PostedAt != other.PostedAt || r.Timezone != other.Timezone || r.Origin != other.Origin || r.PnLBasis != other.PnLBasis || r.AttachmentID != other.AttachmentID || r.AllocationReason != other.AllocationReason || len(r.Postings) != len(other.Postings) {
		return false
	}
	for _, f := range []Field{PrincipalField, FeesField, DateField, PayerField, MerchantField, NoteField, AccountingField, CategoryField, MerchantIDField, ReceiptItemsField, MatchingField} {
		if !r.FieldEqual(other, f) {
			return false
		}
	}
	for i, p := range r.Postings {
		q := other.Postings[i]
		if p.AccountID != q.AccountID || p.Money.Asset() != q.Money.Asset() || !p.SameMoney(q) || p.Role != q.Role || p.Funding != q.Funding || p.Treatment != q.Treatment {
			return false
		}
	}
	return true
}
