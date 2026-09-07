"""One account read from the owner VPS; requires explicit token-transfer approval."""

import argparse
import base64
import json
import shlex
import subprocess
import uuid

from probe import BankProbe, ProbeError


class VpsAccountProbe:
    REMOTE = '''import sys,json,base64,urllib.request,urllib.error,ssl
class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self,*args,**kwargs): return None
try:
    tokens=json.load(sys.stdin)
    opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),urllib.request.HTTPSHandler(context=ssl.create_default_context()),NoRedirect())
    request=urllib.request.Request("https://api.openapi.raiffeisen.ru/api/v1/accounts?fields=Id,Number,Name,OrganizationName,Currency",headers={"Authorization":"Bearer "+tokens["access_token"],"ID-Token":tokens["id_token"],"Accept":"application/json"})
    try: response=opener.open(request,timeout=20)
    except urllib.error.HTTPError as error: response=error
    with response:
        data=response.read(2097153)
        if len(data)>2097152: raise ValueError()
        print(json.dumps({"status":response.code,"body_base64":base64.b64encode(data).decode()}))
except Exception:
    print(json.dumps({"error":"remote_request_failed"}))
    sys.exit(1)
'''

    def __init__(self, bank):
        self.bank = bank

    def accounts(self, baseline):
        with self.bank.lock():
            if self.bank.read("refresh-state.json").get("phase") != "complete":
                raise ProbeError("Refresh outcome is unresolved")
            original = self.bank.read(baseline)
            if original["status"] != 200:
                raise ProbeError("Baseline account read failed")
            expected = json.loads(base64.b64decode(original["body_base64"]))
            tokens = self.bank.read("tokens.json")
            payload = json.dumps({key: tokens[key] for key in ("access_token", "id_token")}).encode()
            command = ["ssh", "-o", "BatchMode=no", "-o", "NumberOfPasswordPrompts=1",
                       "-o", "PreferredAuthentications=password", "-o", "StrictHostKeyChecking=yes",
                       "-o", "ConnectTimeout=10", "want-keep.tech", "python3 -c " + shlex.quote(self.REMOTE)]
            try:
                result = subprocess.run(command, input=payload, capture_output=True, timeout=45)
            except subprocess.TimeoutExpired:
                raise ProbeError("VPS read timed out; captured output suppressed") from None
            if result.returncode:
                raise ProbeError("VPS read failed; captured output suppressed")
            response = json.loads(result.stdout)
            evidence = f"vps-accounts-{uuid.uuid4().hex}.json"
            self.bank.save(evidence, response)
            if response.get("status") != 200:
                raise ProbeError(f"VPS account read returned HTTP {response.get('status')}")
            actual = json.loads(base64.b64decode(response["body_base64"]))
            return {"operation": "vps_accounts", "http_status": 200, "account_count": len(actual),
                    "same_account_identity": [(x["id"], x["number"]) for x in actual] == [(x["id"], x["number"]) for x in expected],
                    "evidence": evidence}

    @classmethod
    def main(cls):
        parser = argparse.ArgumentParser(description=__doc__)
        parser.add_argument("--state-dir", required=True)
        parser.add_argument("--accounts-evidence", required=True)
        arguments = parser.parse_args()
        try:
            print(json.dumps(cls(BankProbe(arguments.state_dir)).accounts(arguments.accounts_evidence)))
        except ProbeError as error:
            print(json.dumps({"error": str(error)}))
            raise SystemExit(1) from None
        except Exception:
            print(json.dumps({"error": "VPS probe failed; captured output suppressed"}))
            raise SystemExit(1) from None


if __name__ == "__main__":
    VpsAccountProbe.main()
