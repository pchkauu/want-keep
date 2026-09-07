"""Exercise the test-only memory probe under the actual processor's cgroup limits."""
import subprocess
import sys

container = sys.argv[1]
probe = ["docker", "exec", container, "/usr/local/bin/privacy-faultprobe"]
before = int(subprocess.check_output([*probe, "events"], timeout=10))
result = subprocess.run([*probe, "exhaust"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=30)
assert result.returncode == 137, "Memory probe was not killed at the container limit"
after = int(subprocess.check_output([*probe, "events"], timeout=10))
assert after > before, "Kernel did not record an OOM kill"
subprocess.run(["docker", "restart", container], check=True, stdout=subprocess.DEVNULL, timeout=20)
print("Processor cgroup OOM enforcement and restart exercised.")
