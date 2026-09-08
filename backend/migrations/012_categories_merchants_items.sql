CREATE TABLE want_keep.categories (
 household_id uuid NOT NULL,
 id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 parent_id uuid,
 key text NOT NULL CHECK(length(key)<=200),
 name_ru text NOT NULL CHECK(length(name_ru)<=200),
 name_en text NOT NULL CHECK(length(name_en)<=200),
 custom_name text NOT NULL CHECK(length(custom_name)<=200),
 normalized_name text NOT NULL CHECK(length(normalized_name) BETWEEN 1 AND 200),
 state text NOT NULL CHECK(state IN ('active','archived')),
 origin text NOT NULL CHECK(origin IN ('starter','custom')),
 PRIMARY KEY(household_id,id),
 FOREIGN KEY(household_id) REFERENCES want_keep.households(id),
 FOREIGN KEY(household_id,parent_id) REFERENCES want_keep.categories(household_id,id),
 CHECK(parent_id IS NULL OR parent_id<>id),
 CHECK((origin='starter' AND key<>'' AND name_ru<>'' AND name_en<>'') OR
       (origin='custom' AND key='' AND name_ru='' AND name_en='' AND custom_name<>''))
);
CREATE UNIQUE INDEX category_starter_key ON want_keep.categories(household_id,key) WHERE key<>'';

CREATE TABLE want_keep.category_name_claims (
 household_id uuid NOT NULL,
 category_id uuid NOT NULL,
 parent_id uuid,
 normalized_name text NOT NULL CHECK(length(normalized_name) BETWEEN 1 AND 200),
 PRIMARY KEY(household_id,category_id,normalized_name),
 FOREIGN KEY(household_id,category_id) REFERENCES want_keep.categories(household_id,id),
 FOREIGN KEY(household_id,parent_id) REFERENCES want_keep.categories(household_id,id)
);
CREATE UNIQUE INDEX category_active_name_claim ON want_keep.category_name_claims(household_id,COALESCE(parent_id,'00000000-0000-0000-0000-000000000000'::uuid),normalized_name);

CREATE FUNCTION want_keep.refresh_category_name_claims() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
BEGIN
 DELETE FROM want_keep.category_name_claims WHERE household_id=NEW.household_id AND category_id=NEW.id;
 IF NEW.state='active' THEN
  INSERT INTO want_keep.category_name_claims(household_id,category_id,parent_id,normalized_name)
  SELECT NEW.household_id,NEW.id,NEW.parent_id,claim
  FROM (
   SELECT NEW.normalized_name AS claim
   UNION
   SELECT lower(NEW.name_en) WHERE NEW.origin='starter' AND NEW.custom_name=''
  ) names;
 END IF;
 RETURN NEW;
END;
$$;
REVOKE ALL ON FUNCTION want_keep.refresh_category_name_claims() FROM PUBLIC;
CREATE TRIGGER refresh_category_name_claims
AFTER INSERT OR UPDATE OF parent_id,custom_name,normalized_name,state ON want_keep.categories
FOR EACH ROW EXECUTE FUNCTION want_keep.refresh_category_name_claims();

CREATE TABLE want_keep.category_revisions (
 household_id uuid NOT NULL,
 id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 parent_id uuid,
 key text NOT NULL,
 name_ru text NOT NULL,
 name_en text NOT NULL,
 custom_name text NOT NULL,
 normalized_name text NOT NULL,
 state text NOT NULL CHECK(state IN ('active','archived')),
 origin text NOT NULL CHECK(origin IN ('starter','custom')),
 actor_id uuid,
 command_id uuid,
 PRIMARY KEY(household_id,id,revision),
 FOREIGN KEY(household_id,id) REFERENCES want_keep.categories(household_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.category_revisions FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.merchants (
 household_id uuid NOT NULL,
 id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 name text NOT NULL CHECK(length(name) BETWEEN 1 AND 200),
 normalized_name text NOT NULL CHECK(length(normalized_name) BETWEEN 1 AND 200),
 state text NOT NULL CHECK(state IN ('active','archived')),
 PRIMARY KEY(household_id,id),
 FOREIGN KEY(household_id) REFERENCES want_keep.households(id)
);
CREATE UNIQUE INDEX merchant_active_name ON want_keep.merchants(household_id,normalized_name) WHERE state='active';

CREATE TABLE want_keep.merchant_revisions (
 household_id uuid NOT NULL,
 id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 name text NOT NULL,
 normalized_name text NOT NULL,
 state text NOT NULL CHECK(state IN ('active','archived')),
 actor_id uuid,
 command_id uuid,
 PRIMARY KEY(household_id,id,revision),
 FOREIGN KEY(household_id,id) REFERENCES want_keep.merchants(household_id,id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(household_id,actor_id) REFERENCES want_keep.memberships(household_id,user_id)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.merchant_revisions FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

CREATE TABLE want_keep.merchant_aliases (
 household_id uuid NOT NULL,
 merchant_id uuid NOT NULL,
 id uuid NOT NULL,
 name text NOT NULL CHECK(length(name) BETWEEN 1 AND 200),
 normalized_name text NOT NULL CHECK(length(normalized_name) BETWEEN 1 AND 200),
 state text NOT NULL CHECK(state IN ('active','archived')),
 origin text NOT NULL CHECK(origin IN ('user_confirmed','review_proposed')),
 merchant_active boolean NOT NULL,
 PRIMARY KEY(household_id,id),
 FOREIGN KEY(household_id,merchant_id) REFERENCES want_keep.merchants(household_id,id)
);
CREATE UNIQUE INDEX merchant_active_alias ON want_keep.merchant_aliases(household_id,normalized_name) WHERE state='active' AND origin='user_confirmed' AND merchant_active;
CREATE INDEX merchant_alias_owner ON want_keep.merchant_aliases(household_id,merchant_id,id);

CREATE TABLE want_keep.merchant_alias_revisions (
 household_id uuid NOT NULL,
 merchant_id uuid NOT NULL,
 merchant_revision want_keep.revision NOT NULL,
 alias_id uuid NOT NULL,
 name text NOT NULL,
 normalized_name text NOT NULL,
 state text NOT NULL CHECK(state IN ('active','archived')),
 origin text NOT NULL CHECK(origin IN ('user_confirmed','review_proposed')),
 PRIMARY KEY(household_id,merchant_id,merchant_revision,alias_id),
 FOREIGN KEY(household_id,merchant_id,merchant_revision) REFERENCES want_keep.merchant_revisions(household_id,id,revision)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.merchant_alias_revisions FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

ALTER TABLE want_keep.transaction_details
 ADD COLUMN category_id uuid,
 ADD COLUMN merchant_id uuid,
 ADD CONSTRAINT transaction_details_category_fk FOREIGN KEY(household_id,category_id) REFERENCES want_keep.categories(household_id,id),
 ADD CONSTRAINT transaction_details_merchant_fk FOREIGN KEY(household_id,merchant_id) REFERENCES want_keep.merchants(household_id,id);

CREATE TABLE want_keep.receipt_items (
 household_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 position integer NOT NULL CHECK(position>=0 AND position<1000),
 id uuid NOT NULL,
 name text NOT NULL CHECK(length(name) BETWEEN 1 AND 2000),
 quantity numeric NOT NULL CHECK(quantity NOT IN ('NaN'::numeric,'Infinity'::numeric,'-Infinity'::numeric) AND quantity>0 AND length(quantity::text)<=256),
 gross want_keep.amount NOT NULL CHECK(gross>0),
 discount want_keep.amount NOT NULL CHECK(discount>=0 AND discount<=gross),
 asset text NOT NULL CHECK(asset IN ('RUB','USD','USDT','USDC','BTC','ETH')),
 category_id uuid,
 PRIMARY KEY(household_id,operation_id,revision,position),
 UNIQUE(household_id,operation_id,revision,id),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.operation_revisions(household_id,operation_id,revision),
 FOREIGN KEY(household_id,category_id) REFERENCES want_keep.categories(household_id,id)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.receipt_items FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

ALTER TABLE want_keep.ledger_decision_entries DROP CONSTRAINT ledger_decision_entries_fields_check;
ALTER TABLE want_keep.ledger_decision_entries ADD CONSTRAINT ledger_decision_entries_fields_check
 CHECK(cardinality(fields)>0 AND fields <@ ARRAY['principal','fees','occurred_at','payer','merchant','note','accounting','category','merchant_identity','receipt_items']::text[]);
ALTER TABLE want_keep.ledger_field_origins DROP CONSTRAINT ledger_field_origins_field_check;
ALTER TABLE want_keep.ledger_field_origins ADD CONSTRAINT ledger_field_origins_field_check
 CHECK(field IN ('principal','fees','occurred_at','payer','merchant','note','accounting','category','merchant_identity','receipt_items','legacy_all'));

CREATE TABLE want_keep.ledger_classification_proposals (
 household_id uuid NOT NULL,
 operation_id uuid NOT NULL,
 revision want_keep.revision NOT NULL,
 proposal_hash text NOT NULL CHECK(proposal_hash ~ '^[0-9a-f]{64}$'),
 category_id uuid,
 merchant_id uuid,
 merchant_alias text,
 items jsonb NOT NULL CHECK(jsonb_typeof(items)='array'),
 rationale text NOT NULL CHECK(length(rationale) BETWEEN 1 AND 2000),
 created_at timestamptz NOT NULL,
 created_ns want_keep.submicro NOT NULL,
 PRIMARY KEY(household_id,operation_id,revision),
 FOREIGN KEY(household_id,operation_id,revision) REFERENCES want_keep.ledger_review_results(household_id,operation_id,revision),
 FOREIGN KEY(household_id,category_id) REFERENCES want_keep.categories(household_id,id),
 FOREIGN KEY(household_id,merchant_id) REFERENCES want_keep.merchants(household_id,id),
 CHECK(merchant_alias IS NULL OR merchant_id IS NOT NULL AND char_length(btrim(merchant_alias)) BETWEEN 1 AND 200)
);
CREATE TRIGGER immutable_history BEFORE UPDATE OR DELETE ON want_keep.ledger_classification_proposals FOR EACH ROW EXECUTE FUNCTION want_keep.reject_history_change();

WITH definitions(id,parent_id,key,name_ru,name_en) AS (VALUES
 ('24972db1-ed6c-4b19-a307-e99724d65861'::uuid,NULL::uuid,'housing','Жильё','Housing'),
 ('f95d48cf-ddfd-4e7d-881a-fe34e1354be8'::uuid,'24972db1-ed6c-4b19-a307-e99724d65861'::uuid,'housing.rent','Аренда','Rent'),
 ('4b3b2ccf-3cb5-4761-a89b-d5a5581d070e'::uuid,'24972db1-ed6c-4b19-a307-e99724d65861'::uuid,'housing.utilities','Коммунальные услуги','Utilities'),
 ('3ec7e6af-303f-4de4-897e-c658d2943b16'::uuid,'24972db1-ed6c-4b19-a307-e99724d65861'::uuid,'housing.repairs','Ремонт','Repairs'),
 ('38848712-33d0-4bc8-a387-d1fb73d6286d'::uuid,'24972db1-ed6c-4b19-a307-e99724d65861'::uuid,'housing.goods','Товары для дома','Household goods'),
 ('04a3e865-3149-44b4-a22e-9dd68bc10ac9'::uuid,NULL::uuid,'food','Еда','Food'),
 ('34197cb4-8410-46f8-8447-056d99dad69e'::uuid,'04a3e865-3149-44b4-a22e-9dd68bc10ac9'::uuid,'food.groceries','Домашняя еда и продукты','Groceries and home food'),
 ('67ff995b-92cd-4474-98fc-b255c71be461'::uuid,'04a3e865-3149-44b4-a22e-9dd68bc10ac9'::uuid,'food.restaurants','Рестораны и кафе','Restaurants and cafes'),
 ('37e1363b-35cd-4869-998f-6c301044f664'::uuid,'04a3e865-3149-44b4-a22e-9dd68bc10ac9'::uuid,'food.delivery','Доставка готовой еды','Prepared food delivery'),
 ('a82c6261-76b7-4d45-993d-b6010128e549'::uuid,NULL::uuid,'transport','Транспорт','Transport'),
 ('0e49e90b-1edb-45b7-ab18-e04060d6043e'::uuid,'a82c6261-76b7-4d45-993d-b6010128e549'::uuid,'transport.taxi','Такси','Taxi'),
 ('4fcfd30e-dea4-4d80-a5c5-914027997550'::uuid,'a82c6261-76b7-4d45-993d-b6010128e549'::uuid,'transport.public','Общественный транспорт','Public transport'),
 ('ba5241cf-9ba1-402b-a0bf-09005141c405'::uuid,'a82c6261-76b7-4d45-993d-b6010128e549'::uuid,'transport.car','Автомобиль','Car'),
 ('9f927980-aefe-4d1f-b2b3-2ee5e726a93a'::uuid,NULL::uuid,'health','Здоровье','Health'),
 ('7d0880e3-8991-4205-a3a9-8c12b2b7bbad'::uuid,'9f927980-aefe-4d1f-b2b3-2ee5e726a93a'::uuid,'health.doctors','Врачи','Doctors'),
 ('b07310ab-c7d7-4b55-9128-a320f357b974'::uuid,'9f927980-aefe-4d1f-b2b3-2ee5e726a93a'::uuid,'health.medicine','Лекарства','Medicines'),
 ('b3f6377b-d839-4e69-8178-4192d50bd840'::uuid,'9f927980-aefe-4d1f-b2b3-2ee5e726a93a'::uuid,'health.insurance','Страхование','Insurance'),
 ('6d0d75e5-ac49-48f6-9f32-177c9f8a7181'::uuid,NULL::uuid,'sport','Спорт','Sport'),
 ('c03694f7-c3aa-4e43-a44e-34fdc20b3b35'::uuid,'6d0d75e5-ac49-48f6-9f32-177c9f8a7181'::uuid,'sport.gym','Спортзал','Gym'),
 ('0945947b-0962-4a15-b093-384bc617e51d'::uuid,'6d0d75e5-ac49-48f6-9f32-177c9f8a7181'::uuid,'sport.equipment','Инвентарь','Equipment'),
 ('62ee5ff2-319f-4ad5-a4d9-95f37e3b9879'::uuid,NULL::uuid,'subscriptions','Подписки и связь','Subscriptions and communications'),
 ('b40ef66e-ed1f-4809-b8ff-991e59df7343'::uuid,'62ee5ff2-319f-4ad5-a4d9-95f37e3b9879'::uuid,'subscriptions.telecom','Связь и интернет','Mobile and internet'),
 ('b2588e39-b59b-452c-870c-8c08014c4475'::uuid,'62ee5ff2-319f-4ad5-a4d9-95f37e3b9879'::uuid,'subscriptions.digital','Цифровые подписки','Digital subscriptions'),
 ('3b7feb8d-d9f1-4139-a813-7819cb1b7bb0'::uuid,NULL::uuid,'shopping','Покупки','Shopping'),
 ('8f15a668-6865-4b92-9611-770873bdd748'::uuid,'3b7feb8d-d9f1-4139-a813-7819cb1b7bb0'::uuid,'shopping.clothing','Одежда','Clothing'),
 ('a94118b4-9e2c-44c0-8d0d-9a804d092aa0'::uuid,'3b7feb8d-d9f1-4139-a813-7819cb1b7bb0'::uuid,'shopping.electronics','Электроника','Electronics'),
 ('f9a13a92-2d6d-4ee8-8978-f42cc7f38526'::uuid,'3b7feb8d-d9f1-4139-a813-7819cb1b7bb0'::uuid,'shopping.other','Другие покупки','Other purchases'),
 ('0565a37b-d547-47cd-92d7-fbd6d23f1ea8'::uuid,NULL::uuid,'travel','Путешествия','Travel'),
 ('65c9b9f7-305f-4c86-9bcb-b89c0fcfa99b'::uuid,NULL::uuid,'education','Образование','Education'),
 ('8c546f11-e201-4465-8b23-e482c3e8be3c'::uuid,NULL::uuid,'taxes_fees','Налоги и комиссии','Taxes and fees'),
 ('2200450b-4566-4bff-a26e-f824a00109de'::uuid,NULL::uuid,'gifts_charity','Подарки и благотворительность','Gifts and charity')
)
INSERT INTO want_keep.categories(household_id,id,revision,parent_id,key,name_ru,name_en,custom_name,normalized_name,state,origin)
SELECT h.id,d.id,1,d.parent_id,d.key,d.name_ru,d.name_en,'',lower(d.name_ru),'active','starter'
FROM want_keep.households h CROSS JOIN definitions d;

INSERT INTO want_keep.category_revisions(household_id,id,revision,parent_id,key,name_ru,name_en,custom_name,normalized_name,state,origin)
SELECT household_id,id,revision,parent_id,key,name_ru,name_en,custom_name,normalized_name,state,origin FROM want_keep.categories;

GRANT SELECT,INSERT ON want_keep.categories,want_keep.merchants,want_keep.merchant_aliases TO want_keep_app;
GRANT SELECT ON want_keep.category_name_claims TO want_keep_app;
GRANT UPDATE(revision,parent_id,custom_name,normalized_name,state) ON want_keep.categories TO want_keep_app;
GRANT UPDATE(revision,name,normalized_name,state) ON want_keep.merchants TO want_keep_app;
GRANT UPDATE(state,merchant_active) ON want_keep.merchant_aliases TO want_keep_app;
GRANT SELECT,INSERT ON want_keep.category_revisions,want_keep.merchant_revisions,want_keep.merchant_alias_revisions,want_keep.receipt_items,want_keep.ledger_classification_proposals TO want_keep_app;
