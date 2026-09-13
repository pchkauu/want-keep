package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func (s *Store) insertStarterCategories(ctx context.Context, scope *transactionScope, family household.HouseholdID) error {
	var table *string
	if err := scope.tx.QueryRow(ctx, `SELECT to_regclass('want_keep.categories')::text`).Scan(&table); err != nil {
		return err
	}
	if table == nil {
		return nil
	}
	for _, item := range category.StarterCategories(family) {
		normalized, err := category.Normalize(item.DisplayName())
		if err != nil {
			return err
		}
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.categories(household_id,id,revision,parent_id,key,name_ru,name_en,custom_name,normalized_name,state,origin) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,$11)`, family, item.ID, item.Revision, item.ParentID, item.Key, item.NameRU, item.NameEN, item.CustomName, normalized, item.State, item.Origin); err != nil {
			return err
		}
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.category_revisions(household_id,id,revision,parent_id,key,name_ru,name_en,custom_name,normalized_name,state,origin) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,$11)`, family, item.ID, item.Revision, item.ParentID, item.Key, item.NameRU, item.NameEN, item.CustomName, normalized, item.State, item.Origin); err != nil {
			return err
		}
	}
	return nil
}

func scanCategory(row pgx.Row) (category.Category, error) {
	var item category.Category
	err := row.Scan(&item.HouseholdID, &item.ID, &item.Revision, &item.ParentID, &item.Key, &item.NameRU, &item.NameEN, &item.CustomName, &item.State, &item.Origin)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, category.ErrNotFound
	}
	if err != nil {
		return item, err
	}
	return item, item.Validate()
}

func (s *Store) Category(ctx context.Context, p household.Principal, id string) (category.Category, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return category.Category{}, err
	}
	return scanCategory(q.QueryRow(ctx, `SELECT household_id,id,revision,COALESCE(parent_id::text,''),key,name_ru,name_en,custom_name,state,origin FROM want_keep.categories WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id))
}

func (s *Store) ClassificationStates(ctx context.Context, p household.Principal, categoryIDs, merchantIDs []string) (map[string]category.State, map[string]category.State, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, nil, err
	}
	categories := make(map[string]category.State, len(categoryIDs))
	rows, err := q.Query(ctx, `SELECT id::text,state FROM want_keep.categories WHERE household_id=$1 AND id=ANY($2::uuid[])`, p.HouseholdID(), categoryIDs)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id string
		var state category.State
		if err = rows.Scan(&id, &state); err != nil {
			rows.Close()
			return nil, nil, err
		}
		categories[id] = state
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()
	merchants := make(map[string]category.State, len(merchantIDs))
	rows, err = q.Query(ctx, `SELECT id::text,state FROM want_keep.merchants WHERE household_id=$1 AND id=ANY($2::uuid[])`, p.HouseholdID(), merchantIDs)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var state category.State
		if err = rows.Scan(&id, &state); err != nil {
			return nil, nil, err
		}
		merchants[id] = state
	}
	return categories, merchants, rows.Err()
}

func (s *Store) Categories(ctx context.Context, p household.Principal, filter category.Filter, after string, limit int) ([]category.Category, string, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, "", err
	}
	rows, err := q.Query(ctx, `SELECT household_id,id,revision,COALESCE(parent_id::text,''),key,name_ru,name_en,custom_name,state,origin
 FROM want_keep.categories WHERE household_id=$1 AND ($2='' OR state=$2) AND ($3='' OR COALESCE(parent_id::text,'')=$3)
	 AND ($4='' OR strpos(normalized_name,$4)>0 OR strpos(lower(name_ru),$4)>0 OR strpos(lower(name_en),$4)>0)
	 AND ($5='' OR id>$5::uuid) ORDER BY id LIMIT $6`, p.HouseholdID(), filter.State, filter.ParentID, filter.Search, after, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]category.Category, 0, limit)
	next := ""
	for rows.Next() {
		item, err := scanCategory(rows)
		if err != nil {
			return nil, "", err
		}
		if len(items) == limit {
			next = items[len(items)-1].ID
			break
		}
		items = append(items, item)
	}
	return items, next, rows.Err()
}

func (s *Store) CategoryNamesExist(ctx context.Context, p household.Principal, parentID string, normalized []string, exclude string) (bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return false, err
	}
	var found bool
	err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.category_name_claims WHERE household_id=$1 AND COALESCE(parent_id::text,'')=$2 AND normalized_name=ANY($3::text[]) AND ($4='' OR category_id<>$4::uuid))`, p.HouseholdID(), parentID, normalized, exclude).Scan(&found)
	return found, err
}

func (s *Store) CategoryHasActiveChildren(ctx context.Context, p household.Principal, id string) (bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return false, err
	}
	var found bool
	err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.categories WHERE household_id=$1 AND parent_id=$2 AND state='active')`, p.HouseholdID(), id).Scan(&found)
	return found, err
}

func (s *Store) CategoryHasChildren(ctx context.Context, p household.Principal, id string) (bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return false, err
	}
	var found bool
	err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.categories WHERE household_id=$1 AND parent_id=$2)`, p.HouseholdID(), id).Scan(&found)
	return found, err
}

func (s *Store) CreateCategory(ctx context.Context, item category.Category) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if item.HouseholdID != scope.principal.HouseholdID() || item.Revision != 1 || item.Origin != category.Custom {
		return household.ErrForbidden
	}
	normalized, err := category.Normalize(item.DisplayName())
	if err != nil {
		return err
	}
	if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.categories(household_id,id,revision,parent_id,key,name_ru,name_en,custom_name,normalized_name,state,origin) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,$11)`, item.HouseholdID, item.ID, item.Revision, item.ParentID, item.Key, item.NameRU, item.NameEN, item.CustomName, normalized, item.State, item.Origin); err != nil {
		return catalogConstraintError(err)
	}
	return s.insertCategoryRevision(ctx, scope, item, normalized)
}

func (s *Store) SaveCategory(ctx context.Context, item category.Category, expected uint64) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if item.HouseholdID != scope.principal.HouseholdID() || item.Revision != expected+1 {
		return household.ErrForbidden
	}
	normalized, err := category.Normalize(item.DisplayName())
	if err != nil {
		return err
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.categories SET revision=$3,parent_id=NULLIF($4,'')::uuid,custom_name=$5,normalized_name=$6,state=$7 WHERE household_id=$1 AND id=$2 AND revision=$8`, item.HouseholdID, item.ID, item.Revision, item.ParentID, item.CustomName, normalized, item.State, expected)
	if err != nil {
		return catalogConstraintError(err)
	}
	if tag.RowsAffected() != 1 {
		return category.ErrVersionConflict
	}
	return s.insertCategoryRevision(ctx, scope, item, normalized)
}

func (s *Store) insertCategoryRevision(ctx context.Context, scope *transactionScope, item category.Category, normalized string) error {
	_, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.category_revisions(household_id,id,revision,parent_id,key,name_ru,name_en,custom_name,normalized_name,state,origin,actor_id,command_id) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,$11,$12,NULLIF($13,'')::uuid)`, item.HouseholdID, item.ID, item.Revision, item.ParentID, item.Key, item.NameRU, item.NameEN, item.CustomName, normalized, item.State, item.Origin, scope.principal.UserID(), commands.CurrentCommandID(ctx))
	return err
}

func catalogConstraintError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		switch pgError.ConstraintName {
		case "category_active_name_claim", "category_starter_key":
			return category.ErrCategoryConflict
		case "merchant_active_alias":
			return category.ErrMerchantConflict
		case "merchant_active_name":
			return category.ErrInvalidCategory
		}
	}
	return err
}

func scanMerchant(row pgx.Row) (category.Merchant, error) {
	var item category.Merchant
	err := row.Scan(&item.HouseholdID, &item.ID, &item.Revision, &item.Name, &item.State)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, category.ErrNotFound
	}
	return item, err
}

func (s *Store) Merchant(ctx context.Context, p household.Principal, id string) (category.Merchant, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return category.Merchant{}, err
	}
	item, err := scanMerchant(q.QueryRow(ctx, `SELECT household_id,id,revision,name,state FROM want_keep.merchants WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id))
	if err != nil {
		return item, err
	}
	item.Aliases, err = s.merchantAliases(ctx, q, p, id)
	if err != nil {
		return item, err
	}
	return item, item.Validate()
}

func (s *Store) merchantAliases(ctx context.Context, q reader, p household.Principal, id string) ([]category.Alias, error) {
	rows, err := q.Query(ctx, `SELECT id,name,normalized_name,state,origin FROM want_keep.merchant_aliases WHERE household_id=$1 AND merchant_id=$2 ORDER BY id`, p.HouseholdID(), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	aliases := []category.Alias{}
	for rows.Next() {
		var alias category.Alias
		if err = rows.Scan(&alias.ID, &alias.Name, &alias.Normalized, &alias.State, &alias.Origin); err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	return aliases, rows.Err()
}

func (s *Store) Merchants(ctx context.Context, p household.Principal, filter category.Filter, after string, limit int) ([]category.Merchant, string, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, "", err
	}
	rows, err := q.Query(ctx, `SELECT household_id,id,revision,name,state FROM want_keep.merchants WHERE household_id=$1 AND ($2='' OR state=$2) AND ($3='' OR strpos(normalized_name,$3)>0) AND ($4='' OR id>$4::uuid) ORDER BY id LIMIT $5`, p.HouseholdID(), filter.State, filter.Search, after, limit+1)
	if err != nil {
		return nil, "", err
	}
	items := make([]category.Merchant, 0, limit)
	next := ""
	for rows.Next() {
		item, err := scanMerchant(rows)
		if err != nil {
			return nil, "", err
		}
		if len(items) == limit {
			next = items[len(items)-1].ID
			break
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, "", err
	}
	rows.Close()
	for i := range items {
		items[i].Aliases, err = s.merchantAliases(ctx, q, p, items[i].ID)
		if err != nil {
			return nil, "", err
		}
		if err = items[i].Validate(); err != nil {
			return nil, "", err
		}
	}
	return items, next, nil
}

func (s *Store) MerchantNameExists(ctx context.Context, p household.Principal, normalized, exclude string) (bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return false, err
	}
	var found bool
	err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.merchants WHERE household_id=$1 AND state='active' AND normalized_name=$2 AND ($3='' OR id<>$3::uuid))`, p.HouseholdID(), normalized, exclude).Scan(&found)
	return found, err
}

func (s *Store) MerchantAliasOwner(ctx context.Context, p household.Principal, normalized, exclude string) (string, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return "", err
	}
	var id string
	err = q.QueryRow(ctx, `SELECT a.merchant_id FROM want_keep.merchant_aliases a JOIN want_keep.merchants m ON (m.household_id,m.id)=(a.household_id,a.merchant_id) WHERE a.household_id=$1 AND a.normalized_name=$2 AND a.state='active' AND a.origin='user_confirmed' AND m.state='active' AND ($3='' OR a.merchant_id<>$3::uuid) LIMIT 1`, p.HouseholdID(), normalized, exclude).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (s *Store) CreateMerchant(ctx context.Context, item category.Merchant) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if item.HouseholdID != scope.principal.HouseholdID() || item.Revision != 1 {
		return household.ErrForbidden
	}
	name, _ := category.Normalize(item.Name)
	if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.merchants(household_id,id,revision,name,normalized_name,state) VALUES($1,$2,$3,$4,$5,$6)`, item.HouseholdID, item.ID, item.Revision, item.Name, name, item.State); err != nil {
		return catalogConstraintError(err)
	}
	return s.insertMerchantRevision(ctx, scope, item, name)
}

func (s *Store) SaveMerchant(ctx context.Context, item category.Merchant, expected uint64) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if item.HouseholdID != scope.principal.HouseholdID() || item.Revision != expected+1 {
		return household.ErrForbidden
	}
	name, _ := category.Normalize(item.Name)
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.merchants SET revision=$3,name=$4,normalized_name=$5,state=$6 WHERE household_id=$1 AND id=$2 AND revision=$7`, item.HouseholdID, item.ID, item.Revision, item.Name, name, item.State, expected)
	if err != nil {
		return catalogConstraintError(err)
	}
	if tag.RowsAffected() != 1 {
		return category.ErrVersionConflict
	}
	if _, err = scope.tx.Exec(ctx, `UPDATE want_keep.merchant_aliases SET merchant_active=$3 WHERE household_id=$1 AND merchant_id=$2`, item.HouseholdID, item.ID, item.State == category.Active); err != nil {
		return catalogConstraintError(err)
	}
	return s.insertMerchantRevision(ctx, scope, item, name)
}

func (s *Store) insertMerchantRevision(ctx context.Context, scope *transactionScope, item category.Merchant, normalized string) error {
	_, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.merchant_revisions(household_id,id,revision,name,normalized_name,state,actor_id,command_id) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid)`, item.HouseholdID, item.ID, item.Revision, item.Name, normalized, item.State, scope.principal.UserID(), commands.CurrentCommandID(ctx))
	if err != nil {
		return err
	}
	for _, alias := range item.Aliases {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.merchant_aliases(household_id,merchant_id,id,name,normalized_name,state,origin,merchant_active) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(household_id,id) DO UPDATE SET state=EXCLUDED.state,merchant_active=EXCLUDED.merchant_active WHERE merchant_aliases.merchant_id=EXCLUDED.merchant_id AND merchant_aliases.name=EXCLUDED.name AND merchant_aliases.normalized_name=EXCLUDED.normalized_name AND merchant_aliases.origin=EXCLUDED.origin`, item.HouseholdID, item.ID, alias.ID, alias.Name, alias.Normalized, alias.State, alias.Origin, item.State == category.Active); err != nil {
			return catalogConstraintError(err)
		}
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.merchant_alias_revisions(household_id,merchant_id,merchant_revision,alias_id,name,normalized_name,state,origin) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, item.HouseholdID, item.ID, item.Revision, alias.ID, alias.Name, alias.Normalized, alias.State, alias.Origin); err != nil {
			return err
		}
	}
	return nil
}
