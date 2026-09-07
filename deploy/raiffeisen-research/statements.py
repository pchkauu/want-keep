"""Bounded historical statement research using an existing private token set."""

import argparse
import base64
from datetime import date, datetime
import json
import re
import uuid
from zoneinfo import ZoneInfo

from probe import BankProbe, ProbeError


class StatementProbe:
    BASE = "https://api.raiffeisen.ru/bank-statements/v1/reports/"
    STATUSES = {"CREATED", "STARTED", "FAILED", "STOPPED", "RESTARTED", "COMPLETED", "CANCELLED"}

    def __init__(self, bank, start, end, kind="camt-053"):
        self.bank = bank
        self.start = date.fromisoformat(start).isoformat()
        self.end = date.fromisoformat(end).isoformat()
        self.kind = kind
        today = datetime.now(ZoneInfo("Europe/Moscow")).date().isoformat()
        if kind == "camt-052":
            valid = self.start == self.end == today
        else:
            valid = kind == "camt-053" and self.start <= self.end < today
        if not valid:
            raise ProbeError("Interval does not match the historical/intraday report contract")
        self.record = f"statement-{self.start}-{self.end}.json"

    def headers(self):
        if self.bank.read("refresh-state.json").get("phase") != "complete":
            raise ProbeError("Refresh outcome is unresolved")
        tokens = self.bank.read("tokens.json")
        return {"Authorization": f"Bearer {tokens['access_token']}",
                "Id-Token": tokens["id_token"], "Content-Type": "application/json"}

    def create(self, accounts_evidence, dry_run):
        with self.bank.lock():
            if not dry_run and (self.bank.directory / self.record).exists():
                raise ProbeError("A generation already exists for this interval; inspect its status, do not repeat")
            evidence = self.bank.read(accounts_evidence)
            if evidence["status"] != 200:
                raise ProbeError("Account evidence is not successful")
            accounts = json.loads(base64.b64decode(evidence["body_base64"]))
            if not isinstance(accounts, list) or len(accounts) != 1:
                raise ProbeError("This research command requires one explicitly established account")
            body = json.dumps({"accountKeys": [accounts[0]["number"]], "from": self.start,
                               "to": self.end, "zero": True}).encode()
            url = self.BASE + self.kind + "?dryRun=" + ("true" if dry_run else "false")
            self.bank.validate_request("POST", url, body)
            headers = self.headers()
            output = f"statement-request-{uuid.uuid4().hex}.json"
            record = {"phase": "pending", "accounts_evidence": accounts_evidence,
                      "response_evidence": output, "kind": self.kind}
            if not dry_run:
                self.bank.save(self.record, record)
            status, data = self.bank.request("POST", url, headers, body)
            self.bank.save(output, {"status": status, "body_base64": base64.b64encode(data).decode()})
            if status != (200 if dry_run else 202):
                if not dry_run:
                    record.update(phase="rejected", http_status=status)
                    self.bank.save(self.record, record)
                raise ProbeError(f"Statement request returned HTTP {status}; inspect private response, do not repeat generation")
            if not dry_run:
                report_id = json.loads(data).get("reportId")
                if not isinstance(report_id, str) or not re.fullmatch(r"[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}", report_id):
                    raise ProbeError("Unexpected report identity; response retained, no repeat")
                record.update(phase="accepted", report_id=report_id)
                self.bank.save(self.record, record)
            return {"operation": "statement_check" if dry_run else "statement_create",
                    "http_status": status, "evidence": output}

    def retrieve(self, kind):
        if kind not in {"status", "file"}:
            raise ProbeError("Unsupported report read")
        with self.bank.lock():
            record = self.bank.read(self.record)
            if record.get("phase") != "accepted":
                raise ProbeError("Generation outcome is unresolved")
            if kind == "file" and record.get("status") != "COMPLETED":
                raise ProbeError("The report must be completed before download")
            status, data = self.bank.request("GET", self.BASE + record["report_id"] + "/" + kind, self.headers())
            output = f"statement-{kind}-{uuid.uuid4().hex}.json"
            self.bank.save(output, {"status": status, "body_base64": base64.b64encode(data).decode()})
            if status != 200:
                raise ProbeError(f"Report read returned HTTP {status}; response retained privately")
            result = {"operation": "statement_" + kind, "http_status": status,
                      "evidence": output, "bytes": len(data)}
            if kind == "status":
                value = json.loads(data)
                source_status = value.get("status")
                status = "COMPLETED" if source_status == "completed" else source_status
                if value.get("reportId") != record["report_id"] or status not in self.STATUSES:
                    raise ProbeError("Report identity or status does not match the contract")
                record["status"] = status
                record["source_status"] = source_status
                result["status"] = status
                result["source_status"] = source_status
                self.bank.save(self.record, record)
            return result

    @classmethod
    def main(cls):
        parser = argparse.ArgumentParser(description=__doc__)
        parser.add_argument("operation", choices=("check", "create", "status", "file"))
        parser.add_argument("--state-dir", required=True)
        parser.add_argument("--from-date", required=True)
        parser.add_argument("--to-date", required=True)
        parser.add_argument("--accounts-evidence")
        parser.add_argument("--kind", choices=("camt-053", "camt-052"), default="camt-053")
        arguments = parser.parse_args()
        if arguments.operation in {"check", "create"} and not arguments.accounts_evidence:
            parser.error("Account evidence filename is required")
        try:
            probe = cls(BankProbe(arguments.state_dir), arguments.from_date, arguments.to_date, arguments.kind)
            result = probe.create(arguments.accounts_evidence, arguments.operation == "check") if arguments.operation in {"check", "create"} else probe.retrieve(arguments.operation)
            print(json.dumps(result))
        except ProbeError as error:
            print(json.dumps({"error": str(error)}))
            raise SystemExit(1) from None
        except Exception:
            print(json.dumps({"error": "Statement research failed; inspect private evidence without printing credentials"}))
            raise SystemExit(1) from None


if __name__ == "__main__":
    StatementProbe.main()
