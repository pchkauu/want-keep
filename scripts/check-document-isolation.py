"""Reject a document processor runtime with broader privileges than task-1.5."""
import json
import subprocess
import sys

state = json.loads(subprocess.check_output(["docker", "inspect", sys.argv[1]]))[0]
host, config = state["HostConfig"], state["Config"]
assert host["NetworkMode"] == "none"
assert host["Init"] is True
assert host["ReadonlyRootfs"] and not host["Privileged"]
assert config["User"] == "10000:10000"
assert host["Memory"] == 512 * 1024 * 1024
assert host["NanoCpus"] == 1_000_000_000 and host["PidsLimit"] == 64
assert host["CapDrop"] == ["ALL"] and not host["CapAdd"]
assert any(v in ("no-new-privileges", "no-new-privileges:true") for v in host["SecurityOpt"])
assert len(state["Mounts"]) == 1 and state["Mounts"][0]["Destination"] == "/run/want-keep"
assert "size=134217728" in host["Tmpfs"]["/tmp"]
assert not any("KEY" in v or "TOKEN" in v or "DATABASE" in v or "PASSWORD" in v for v in config["Env"])
print("Document processor isolation verified.")
