"""Operator-only API research; not an application connector or login handler."""

import argparse
import base64
from datetime import date, datetime
import fcntl
import json
import os
from pathlib import Path
import re
import ssl
import stat
import tempfile
import urllib.error
import urllib.parse
import urllib.request
import uuid
from zoneinfo import ZoneInfo


class ProbeError(Exception):
    pass


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class BankProbe:
    TOKEN_URL = "https://sso.rbo.raiffeisen.ru/token"
    ACCOUNTS_URL = (
        "https://api.openapi.raiffeisen.ru/api/v1/accounts"
        "?fields=Id,Number,Name,OrganizationName,Currency"
    )
    RESPONSE_LIMIT = 2 * 1024 * 1024

    def __init__(self, directory):
        self.directory = Path(directory)
        resolved = self.directory.resolve()
        if any((parent / ".git").exists() for parent in (resolved, *resolved.parents)):
            raise ProbeError("Credentials and bank responses must stay outside Git repositories")
        self.directory.mkdir(mode=0o700, parents=False, exist_ok=True)
        info = self.directory.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise ProbeError("Research directory must be owned by this user with mode 0700")
        self.opener = urllib.request.build_opener(
            urllib.request.ProxyHandler({}),
            urllib.request.HTTPSHandler(context=ssl.create_default_context()),
            NoRedirect(),
        )

    def read(self, name):
        if Path(name).name != name:
            raise ProbeError("Evidence filename must stay in the research directory")
        file = self.directory / name
        info = file.lstat()
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise ProbeError("Research file permissions are unsafe")
        return json.loads(file.read_bytes())

    def save(self, name, data):
        encoded = json.dumps(data, ensure_ascii=False).encode()
        descriptor, temporary = tempfile.mkstemp(prefix=".pending-", dir=self.directory)
        try:
            with os.fdopen(descriptor, "wb") as output:
                output.write(encoded)
                output.flush()
                os.fsync(output.fileno())
            os.replace(temporary, self.directory / name)
            descriptor = os.open(self.directory, os.O_RDONLY)
            try:
                os.fsync(descriptor)
            finally:
                os.close(descriptor)
        finally:
            if os.path.exists(temporary):
                os.unlink(temporary)

    def lock(self):
        descriptor = os.open(self.directory / "lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
        lock = os.fdopen(descriptor, "r+")
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            lock.close()
            raise ProbeError("Another research command is active") from None
        return lock

    @staticmethod
    def credential(file):
        value = Path(file).read_text().strip()
        if not value or len(value) > 65536 or any(ord(c) < 33 or ord(c) > 126 for c in value):
            raise ProbeError("Credential file has an unexpected format")
        return value

    def request(self, method, url, headers, body=None):
        self.validate_request(method, url, body)
        request = urllib.request.Request(url, data=body, headers=headers, method=method)
        try:
            response = self.opener.open(request, timeout=30)
        except urllib.error.HTTPError as error:
            response = error
        except (OSError, urllib.error.URLError):
            raise ProbeError("Request outcome is unknown; do not repeat token exchange") from None
        with response:
            data = response.read(self.RESPONSE_LIMIT + 1)
            if len(data) > self.RESPONSE_LIMIT:
                raise ProbeError("Response exceeded the research limit; inspect pending state")
            return response.code, data

    @classmethod
    def validate_request(cls, method, url, body):
        if (method, url) in {("POST", cls.TOKEN_URL), ("GET", cls.ACCOUNTS_URL)}:
            return
        if method == "GET" and re.fullmatch(
            r"https://api\.raiffeisen\.ru/bank-statements/v1/reports/"
            r"[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}/(?:status|file)", url,
        ):
            return
        if method == "POST" and url in {
            "https://api.raiffeisen.ru/bank-statements/v1/reports/camt-053?dryRun=true",
            "https://api.raiffeisen.ru/bank-statements/v1/reports/camt-053?dryRun=false",
            "https://api.raiffeisen.ru/bank-statements/v1/reports/camt-052?dryRun=true",
            "https://api.raiffeisen.ru/bank-statements/v1/reports/camt-052?dryRun=false",
        }:
            try:
                value = json.loads(body)
                valid = set(value) == {"accountKeys", "from", "to", "zero"}
                valid = valid and value["zero"] is True and isinstance(value["accountKeys"], list)
                valid = valid and len(value["accountKeys"]) == 1
                valid = valid and bool(re.fullmatch(r"[0-9]{20}", value["accountKeys"][0]))
                today = datetime.now(ZoneInfo("Europe/Moscow")).date()
                start, end = date.fromisoformat(value["from"]), date.fromisoformat(value["to"])
                valid = valid and (start == end == today if "/camt-052?" in url else start <= end < today)
            except (ValueError, TypeError, KeyError):
                valid = False
            if valid:
                return
        raise ProbeError("Request is outside the research allowlist")

    def refresh(self, client_id_file, client_secret_file, refresh_token_file):
        with self.lock():
            marker = self.directory / "refresh-state.json"
            if marker.exists() and self.read(marker.name).get("phase") != "complete":
                raise ProbeError("Previous refresh is unresolved; inspect private evidence before recovery")
            client_id = self.credential(client_id_file)
            client_secret = self.credential(client_secret_file)
            if ":" in client_id:
                raise ProbeError("Invalid client identifier")
            if (self.directory / "tokens.json").exists():
                refresh_token = self.read("tokens.json")["refresh_token"]
            else:
                refresh_token = self.credential(refresh_token_file)
            body = urllib.parse.urlencode({
                "grant_type": "refresh_token", "client_id": client_id,
                "refresh_token": refresh_token,
            }).encode()
            authentication = base64.b64encode(f"{client_id}:{client_secret}".encode()).decode()
            attempt = uuid.uuid4().hex
            self.save(marker.name, {"phase": "pending", "attempt": attempt})
            status, data = self.request("POST", self.TOKEN_URL, {
                "Authorization": f"Basic {authentication}",
                "Content-Type": "application/x-www-form-urlencoded",
                "Accept": "application/json",
            }, body)
            self.save(f"refresh-{attempt}.json", {
                "status": status, "body_base64": base64.b64encode(data).decode(),
            })
            if status != 200:
                self.save(marker.name, {"phase": "rejected", "attempt": attempt, "http_status": status})
                raise ProbeError(f"Token endpoint returned HTTP {status}; response saved privately, no retry")
            try:
                tokens = json.loads(data)
                valid = isinstance(tokens, dict) and tokens.get("token_type", "").lower() == "bearer"
                for key in ("access_token", "id_token", "refresh_token"):
                    value = tokens.get(key) if isinstance(tokens, dict) else None
                    valid = valid and isinstance(value, str) and bool(value) and len(value) <= 65536
                    valid = valid and all(33 <= ord(c) <= 126 for c in (value or ""))
            except (ValueError, TypeError, AttributeError):
                valid = False
            if not valid:
                raise ProbeError("Unexpected token response; private response retained, no retry")
            self.save("tokens.json", tokens)
            self.save(marker.name, {"phase": "complete", "attempt": attempt})
            return {"operation": "refresh", "http_status": status, "tokens_saved": True,
                    "refresh_token_changed": tokens["refresh_token"] != refresh_token}

    def accounts(self):
        with self.lock():
            if self.read("refresh-state.json").get("phase") != "complete":
                raise ProbeError("Refresh outcome must be resolved before reading accounts")
            tokens = self.read("tokens.json")
            status, data = self.request("GET", self.ACCOUNTS_URL, {
                "Authorization": f"Bearer {tokens['access_token']}",
                "ID-Token": tokens["id_token"], "Accept": "application/json",
            })
            evidence = f"accounts-{uuid.uuid4().hex}.json"
            self.save(evidence, {"status": status, "body_base64": base64.b64encode(data).decode()})
            if status != 200:
                raise ProbeError(f"Accounts endpoint returned HTTP {status}; private response retained")
            value = json.loads(data)
            schema = {key: type(item).__name__ for key, item in value.items()} if isinstance(value, dict) else None
            return {"operation": "accounts", "http_status": status, "evidence": evidence,
                    "root_type": type(value).__name__, "root_schema": schema,
                    "array_count": len(value) if isinstance(value, list) else None}

    @classmethod
    def main(cls):
        parser = argparse.ArgumentParser(description=__doc__)
        parser.add_argument("operation", choices=("refresh", "accounts"))
        parser.add_argument("--state-dir", required=True)
        parser.add_argument("--client-id-file")
        parser.add_argument("--client-secret-file")
        parser.add_argument("--refresh-token-file")
        arguments = parser.parse_args()
        if arguments.operation == "refresh" and not all((arguments.client_id_file, arguments.client_secret_file, arguments.refresh_token_file)):
            parser.error("Refresh requires all three credential file paths")
        try:
            probe = cls(arguments.state_dir)
            result = probe.refresh(arguments.client_id_file, arguments.client_secret_file, arguments.refresh_token_file) if arguments.operation == "refresh" else probe.accounts()
            print(json.dumps(result))
        except ProbeError as error:
            print(json.dumps({"error": str(error)}))
            raise SystemExit(1) from None
        except Exception:
            print(json.dumps({"error": "Research command failed; inspect private state without printing credentials"}))
            raise SystemExit(1) from None


if __name__ == "__main__":
    BankProbe.main()
