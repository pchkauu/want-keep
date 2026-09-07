import pathlib
import subprocess
import tempfile
import unittest


class OpenAPIStateTest(unittest.TestCase):
    def test_missing_artifact_never_reports_success(self):
        root = pathlib.Path(__file__).resolve().parents[1]
        script = root / "scripts/check-openapi-state.sh"
        artifacts = [
            "api/openapi.yaml", "api/oapi-codegen.yaml",
            "scripts/generate-openapi.sh",
            "backend/internal/delivery/http/generated/openapi.gen.go",
            "web/src/api/generated/openapi.gen.ts",
        ]
        for missing in artifacts + ["all"]:
            with self.subTest(missing=missing), tempfile.TemporaryDirectory() as directory:
                fixture = pathlib.Path(directory)
                for artifact in artifacts:
                    if artifact == missing or missing == "all":
                        continue
                    destination = fixture / artifact
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    destination.write_text("exit 0\n" if artifact.endswith(".sh") else "fixture\n")
                result = subprocess.run(["sh", str(script)], cwd=fixture, capture_output=True, text=True)
                self.assertEqual(result.returncode, 2, result.stdout + result.stderr)
                self.assertIn("incomplete", result.stderr)


if __name__ == "__main__":
    unittest.main()
