ALTER TABLE want_keep.ledger_participations DROP CONSTRAINT ledger_participations_state_check;
ALTER TABLE want_keep.ledger_participations ADD CHECK(state IN ('waiting','retained','linked'));
