package domain

import "slices"

func (r Revision) rolePostings(role Role) []Posting {
	out := []Posting{}
	for _, p := range r.Postings {
		if p.Role == role {
			out = append(out, p)
		}
	}
	return out
}
func (r *Revision) replaceRole(role Role, entries []Posting) {
	out := []Posting{}
	inserted := false
	for _, p := range r.Postings {
		if p.Role == role {
			if !inserted {
				out = append(out, entries...)
				inserted = true
			}
			continue
		}
		out = append(out, p)
	}
	if !inserted {
		out = append(out, entries...)
	}
	r.Postings = out
}

func (r Revision) FieldEqual(other Revision, field Field) bool {
	switch field {
	case PrincipalField, FeesField:
		role := Principal
		if field == FeesField {
			role = Fee
			if r.FeeKnowledge != other.FeeKnowledge {
				return false
			}
		}
		a, b := r.rolePostings(role), other.rolePostings(role)
		if len(a) != len(b) {
			return false
		}
		for i, p := range a {
			q := b[i]
			if p.AccountID != q.AccountID || p.Role != q.Role || p.Funding != q.Funding || (p.Treatment != q.Treatment && !(p.MovesMoney() && q.MovesMoney())) || p.Money.Asset() != q.Money.Asset() || !p.SameMoney(q) {
				return false
			}
		}
		return true
	case DateField:
		return r.OccurredAt == other.OccurredAt
	case PayerField:
		return r.PayerState == other.PayerState && r.PayerMemberID == other.PayerMemberID
	case MerchantField:
		return r.Merchant == other.Merchant
	case NoteField:
		return r.Note == other.Note
	case AccountingField:
		return r.Accounting() == other.Accounting()
	case CategoryField:
		return r.CategoryID == other.CategoryID
	case MerchantIDField:
		return r.MerchantID == other.MerchantID
	case ReceiptItemsField:
		return slices.EqualFunc(r.ReceiptItems, other.ReceiptItems, func(a, b ReceiptItem) bool {
			return a.ID == b.ID && a.Name == b.Name && a.Quantity == b.Quantity && a.CategoryID == b.CategoryID && a.Gross.Asset() == b.Gross.Asset() && a.Gross.Amount() == b.Gross.Amount() && a.Discount.Asset() == b.Discount.Asset() && a.Discount.Amount() == b.Discount.Amount()
		})
	}
	return false
}

func (r *Revision) CopyField(from Revision, field Field) error {
	switch field {
	case PrincipalField:
		r.replaceRole(Principal, from.rolePostings(Principal))
	case FeesField:
		r.replaceRole(Fee, from.rolePostings(Fee))
		r.FeeKnowledge = from.FeeKnowledge
	case DateField:
		r.OccurredAt = from.OccurredAt
	case PayerField:
		r.PayerState = from.PayerState
		r.PayerMemberID = from.PayerMemberID
	case MerchantField:
		r.Merchant = from.Merchant
	case NoteField:
		r.Note = from.Note
	case AccountingField:
		r.AccountingState = from.Accounting()
	case CategoryField:
		r.CategoryID = from.CategoryID
	case MerchantIDField:
		r.MerchantID = from.MerchantID
	case ReceiptItemsField:
		r.ReceiptItems = slices.Clone(from.ReceiptItems)
	default:
		return ErrInvalidRevision
	}
	return nil
}
