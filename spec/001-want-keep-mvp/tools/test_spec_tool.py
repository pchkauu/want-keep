"""Regression checks for invalid specification graphs and delivery mappings."""

import copy
import json
from pathlib import Path
import tempfile
import unittest

from spec_tool import SpecCatalog


class SpecCatalogTest(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    cls.source = Path(__file__).resolve().parents[1]
    cls.data = json.loads((cls.source / "catalog.json").read_text())

  def errors_for(self, mutate):
    data = copy.deepcopy(self.data)
    mutate(data)
    with tempfile.TemporaryDirectory() as directory:
      root = Path(directory)
      (root / "catalog.json").write_text(json.dumps(data))
      return SpecCatalog(root).validate_catalog()

  def test_current_graph_is_valid(self):
    self.assertEqual([], SpecCatalog(self.source).validate_catalog())

  def test_task_status_override_is_rendered(self):
    catalog = SpecCatalog(self.source)
    body = catalog.task_body(catalog.tasks["task-1.1"])
    self.assertIn("Техническая основа реализована", body)
    self.assertIn("The technical foundation is implemented", body)
    self.assertIn("Интерфейс `make` создан task-1.1", body)

  def test_unknown_acceptance_is_rejected_before_render(self):
    errors = self.errors_for(lambda data: data["tasks"][0]["acceptance"].append("AC-999"))
    self.assertTrue(any("Unresolved AC" in error for error in errors))

  def test_indirect_dependency_cycle_is_rejected(self):
    def mutate(data):
      data["tasks"][0]["depends_on"] = [data["tasks"][1]["id"]]
      data["tasks"][1]["depends_on"] = [data["tasks"][0]["id"]]
    self.assertTrue(any("Dependency cycle" in error for error in self.errors_for(mutate)))

  def test_implementation_cannot_bypass_ready(self):
    def mutate(data):
      next(task for task in data["tasks"] if task["id"] == "task-1.1")["depends_on"] = []
    self.assertTrue(any("bypasses readiness" in error for error in self.errors_for(mutate)))

  def test_missing_translation_is_rejected(self):
    def mutate(data):
      del data["screens"][0]["answer"]["en"]
    self.assertTrue(any("Missing translation" in error for error in self.errors_for(mutate)))

  def test_duplicate_screen_does_not_silently_overwrite(self):
    errors = self.errors_for(lambda data: data["screens"].append(data["screens"][0]))
    self.assertIn("Duplicate screens ID", errors)

  def test_screen_cannot_reference_missing_form(self):
    errors = self.errors_for(lambda data: data["screens"][0]["forms"].append("FORM-99"))
    self.assertTrue(any("Unresolved forms" in error for error in errors))

  def test_screen_acceptance_requires_own_task_coverage(self):
    def mutate(data):
      data["screens"][0]["tasks"] = ["task-0.1"]
    self.assertTrue(any("Screen acceptance lacks task coverage" in error for error in self.errors_for(mutate)))

  def test_foreign_repository_mapping_is_rejected(self):
    def mutate(data):
      data["tasks"][0]["github_url"] = "https://github.com/example/other/issues/1"
    self.assertTrue(any("Invalid GitHub URL" in error for error in self.errors_for(mutate)))

  def test_github_urls_do_not_change_issue_bodies(self):
    baseline = SpecCatalog(self.source)
    data = copy.deepcopy(self.data)
    for number, task in enumerate(data["tasks"], 1):
      task["github_url"] = f"https://github.com/pchkauu/want-keep/issues/{number}"
    with tempfile.TemporaryDirectory() as directory:
      root = Path(directory)
      (root / "catalog.json").write_text(json.dumps(data))
      mapped = SpecCatalog(root)
      for identifier in baseline.tasks:
        self.assertEqual(baseline.task_body(baseline.tasks[identifier]), mapped.task_body(mapped.tasks[identifier]))

  def test_task_progress_is_rendered_in_both_languages(self):
    catalog = SpecCatalog(self.source)
    task = copy.deepcopy(catalog.tasks["task-0.3"])
    task["status"] = {"ru": "Частичное чтение; контракт не закрыт.", "en": "Partial reading; contract unresolved."}
    body = catalog.task_body(task)
    for status in task["status"].values():
      self.assertIn(status, body)
    self.assertNotIn("Research — not started", body)
    task.pop("status")
    self.assertIn("Research — not started", catalog.task_body(task))

  def test_task_progress_requires_both_languages(self):
    errors = self.errors_for(lambda data: data["tasks"][0].update(status={"ru": "Начато"}))
    self.assertTrue(any("Invalid task status translation" in error for error in errors))

  def test_task_progress_rejects_nonstructured_status(self):
    errors = self.errors_for(lambda data: data["tasks"][0].update(status="started"))
    self.assertTrue(any("Invalid task status translation" in error for error in errors))


if __name__ == "__main__":
  unittest.main()
