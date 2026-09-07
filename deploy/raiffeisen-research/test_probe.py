import base64
import json
from datetime import datetime
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch
from zoneinfo import ZoneInfo

from probe import BankProbe, NoRedirect, ProbeError
from statements import StatementProbe
from vps_accounts import VpsAccountProbe


class BankProbeTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        self.probe = BankProbe(self.directory / "state")
        self.files = []
        for name, value in (("client", "synthetic-client"), ("secret", "synthetic-secret"), ("refresh", "initial-refresh")):
            file = self.directory / name
            file.write_text(value)
            self.files.append(file)
        self.tokens = {"access_token": "new-access", "id_token": "new-id",
                       "refresh_token": "rotated-refresh", "token_type": "Bearer"}

    def test_refresh_rotates_and_next_refresh_uses_saved_token(self):
        self.probe.request = Mock(return_value=(200, json.dumps(self.tokens).encode()))
        result = self.probe.refresh(*self.files)
        self.assertTrue(result["refresh_token_changed"])
        self.assertEqual(self.probe.read("tokens.json"), self.tokens)
        self.assertEqual((self.probe.directory / "tokens.json").stat().st_mode & 0o777, 0o600)
        self.probe.refresh(*self.files)
        self.assertIn(b"refresh_token=rotated-refresh", self.probe.request.call_args.args[3])
        self.assertNotIn("new-access", json.dumps(result))

    def test_timeout_prevents_another_exchange(self):
        self.probe.request = Mock(side_effect=ProbeError("Unknown outcome"))
        with self.assertRaises(ProbeError):
            self.probe.refresh(*self.files)
        with self.assertRaisesRegex(ProbeError, "unresolved"):
            self.probe.refresh(*self.files)
        self.assertEqual(self.probe.request.call_count, 1)

    def test_malformed_or_rejected_response_retains_evidence_and_blocks_retry(self):
        for status, body in ((400, b'{"error":"invalid_grant"}'), (200, b"not-json"), (200, b'{}')):
            with self.subTest(status=status, body=body):
                probe = BankProbe(self.directory / f"case-{status}-{len(body)}")
                probe.request = Mock(return_value=(status, body))
                with self.assertRaises(ProbeError):
                    probe.refresh(*self.files)
                with self.assertRaises(ProbeError):
                    probe.refresh(*self.files)
                self.assertEqual(probe.request.call_count, 1)
                self.assertEqual(len(list(probe.directory.glob("refresh-*.json"))), 2)

    def test_credentials_cannot_create_another_header(self):
        self.files[1].write_text("secret\nInjected: value")
        self.probe.request = Mock()
        with self.assertRaises(ProbeError):
            self.probe.refresh(*self.files)
        self.probe.request.assert_not_called()

    def test_unknown_route_and_payment_are_rejected(self):
        for method, url in (("GET", "https://example.invalid/"), ("POST", self.probe.ACCOUNTS_URL)):
            with self.assertRaises(ProbeError):
                self.probe.request(method, url, {})

    def test_redirect_is_never_followed(self):
        self.assertIsNone(NoRedirect().redirect_request(None, None, 302, "", {}, "https://example.invalid/"))

    def test_lock_prevents_overlapping_operations(self):
        with self.probe.lock():
            with self.assertRaises(ProbeError):
                self.probe.lock()

    def test_accounts_do_not_print_financial_values(self):
        self.probe.save("tokens.json", self.tokens)
        self.probe.save("refresh-state.json", {"phase": "complete"})
        self.probe.request = Mock(return_value=(200, b'[{"Number":"synthetic-account"}]'))
        result = self.probe.accounts()
        self.assertEqual(result["array_count"], 1)
        self.assertNotIn("synthetic-account", json.dumps(result))

    def test_shared_directory_is_rejected(self):
        directory = self.directory / "shared"
        directory.mkdir(mode=0o755)
        with self.assertRaises(ProbeError):
            BankProbe(directory)

    def test_git_directory_cannot_store_credentials(self):
        (self.directory / ".git").mkdir()
        target = self.directory / "credentials"
        with self.assertRaises(ProbeError):
            BankProbe(target)
        self.assertFalse(target.exists())

    def statement_probe(self):
        self.probe.save("tokens.json", self.tokens)
        self.probe.save("refresh-state.json", {"phase": "complete"})
        self.probe.save("accounts.json", {"status": 200, "body_base64": base64.b64encode(b'[{"number":"00000000000000000000"}]').decode()})
        return StatementProbe(self.probe, "2020-01-01", "2020-01-31")

    def test_report_generation_unknown_outcome_cannot_be_repeated(self):
        statement = self.statement_probe()
        self.probe.request = Mock(side_effect=ProbeError("Unknown outcome"))
        with self.assertRaises(ProbeError):
            statement.create("accounts.json", False)
        with self.assertRaises(ProbeError):
            statement.create("accounts.json", False)
        self.assertEqual(self.probe.request.call_count, 1)

    def test_report_rejection_is_linked_to_private_response(self):
        statement = self.statement_probe()
        self.probe.request = Mock(return_value=(404, b'{"code":"no-statements"}'))
        with self.assertRaises(ProbeError):
            statement.create("accounts.json", False)
        record = self.probe.read(statement.record)
        self.assertEqual(record["phase"], "rejected")
        self.assertEqual(self.probe.read(record["response_evidence"])["status"], 404)
        with self.assertRaises(ProbeError):
            statement.create("accounts.json", False)
        self.assertEqual(self.probe.request.call_count, 1)

    def test_report_download_waits_for_matching_completed_status(self):
        statement = self.statement_probe()
        report_id = "00000000-0000-4000-8000-000000000001"
        self.probe.request = Mock(return_value=(202, json.dumps({"reportId": report_id}).encode()))
        statement.create("accounts.json", False)
        with self.assertRaises(ProbeError):
            statement.retrieve("file")
        self.probe.request.return_value = (200, json.dumps({"reportId": report_id, "status": "completed"}).encode())
        self.assertEqual(statement.retrieve("status")["status"], "COMPLETED")
        self.assertEqual(self.probe.read(statement.record)["source_status"], "completed")
        self.probe.request.return_value = (200, b"synthetic-file")
        self.assertEqual(statement.retrieve("file")["bytes"], len(b"synthetic-file"))

    def test_statement_payload_is_checked_at_transport_boundary(self):
        url = StatementProbe.BASE + "camt-053?dryRun=false"
        for body in (b'{}', b'{"accountKeys":["not-an-account"],"from":"2020-01-01","to":"2020-01-31","zero":true}'):
            with self.assertRaises(ProbeError):
                self.probe.request("POST", url, {}, body)

    def test_unrecognized_report_status_does_not_allow_download(self):
        statement = self.statement_probe()
        report_id = "00000000-0000-4000-8000-000000000001"
        self.probe.save(statement.record, {"phase": "accepted", "report_id": report_id})
        self.probe.request = Mock(return_value=(200, json.dumps({"reportId": report_id, "status": "unknown"}).encode()))
        with self.assertRaises(ProbeError):
            statement.retrieve("status")
        with self.assertRaises(ProbeError):
            statement.retrieve("file")
        self.assertEqual(self.probe.request.call_count, 1)

    def test_evidence_cannot_escape_private_directory(self):
        with self.assertRaises(ProbeError):
            self.probe.read("../secret")

    def test_intraday_requires_current_moscow_date(self):
        today = datetime.now(ZoneInfo("Europe/Moscow")).date().isoformat()
        StatementProbe(self.probe, today, today, "camt-052")
        with self.assertRaises(ProbeError):
            StatementProbe(self.probe, "2020-01-01", "2020-01-01", "camt-052")
        with self.assertRaises(ProbeError):
            StatementProbe(self.probe, today, today, "camt-053")

    def test_vps_probe_uses_private_stdin_and_only_access_and_id_tokens(self):
        self.probe.save("tokens.json", self.tokens)
        self.probe.save("refresh-state.json", {"phase": "complete"})
        response = {"status": 200, "body_base64": base64.b64encode(b'[{"id":"synthetic-id","number":"synthetic-number"}]').decode()}
        self.probe.save("accounts.json", response)
        with patch("vps_accounts.subprocess.run", return_value=Mock(returncode=0, stdout=json.dumps(response))) as run:
            result = VpsAccountProbe(self.probe).accounts("accounts.json")
        self.assertTrue(result["same_account_identity"])
        self.assertEqual(set(json.loads(run.call_args.kwargs["input"])), {"access_token", "id_token"})
        self.assertNotIn("new-access", str(run.call_args.args[0]))
        self.assertNotIn("synthetic-number", json.dumps(result))
        self.assertIn("StrictHostKeyChecking=yes", run.call_args.args[0])


if __name__ == "__main__":
    unittest.main()
