"""Offline state, integrity, and fallback tests for release-ci.py."""

import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock
import zipfile

SCRIPT = Path(__file__).with_name("release-ci.py")
SPEC = importlib.util.spec_from_file_location("release_ci", SCRIPT)
ci = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(ci)
SHA = "a" * 40
VERSION = "v1.2.3"
REPO = "daiksud/gh-settings"


def run(run_id, run_number=1, conclusion="success", status="completed"):
    return {
        "id": run_id, "run_number": run_number, "run_attempt": 1,
        "path": ".github/workflows/push.yml", "event": "push",
        "head_branch": "main", "head_sha": SHA,
        "repository": {"full_name": REPO},
        "status": status, "conclusion": conclusion,
    }


def archive_for(target, version=VERSION, corrupt_binary=False):
    asset = "gh-settings-" + target
    contents = ("native-" + target).encode("ascii")
    digest = hashlib.sha256(contents).hexdigest()
    if corrupt_binary:
        contents += b"-corrupted"
    manifest = (
        f"source-sha={SHA}\nversion={version}\ntarget={target}\n"
        f"asset={asset}\nsha256={digest}\n"
    )
    output = io.BytesIO()
    with zipfile.ZipFile(output, "w") as archive:
        archive.writestr(asset, contents)
        archive.writestr("release-manifest", manifest)
    return output.getvalue()


def artifact(name, identifier, data, run_id=7, expired=False):
    return {
        "name": name, "id": identifier, "expired": expired,
        "digest": "sha256:" + hashlib.sha256(data).hexdigest(),
        "workflow_run": {"id": run_id, "head_sha": SHA},
    }


class ReleaseCITests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.environ = mock.patch.dict(os.environ, {
            "GITHUB_REPOSITORY": REPO,
            "SOURCE_SHA": SHA,
            "GH_TOKEN": "fixture-token",
            "RUNNER_TEMP": self.temp.name,
            "GITHUB_OUTPUT": str(Path(self.temp.name) / "outputs"),
            "GITHUB_STEP_SUMMARY": str(Path(self.temp.name) / "summary"),
        })
        self.environ.start()
        self.addCleanup(self.environ.stop)

    def output(self):
        return (Path(self.temp.name) / "outputs").read_text()

    def ci_responses(self, runs, deployments=None):
        if deployments is None:
            deployments = [{"name": "deployment", "status": "completed", "conclusion": "success"}]

        def response(route, key):
            if key == "workflow_runs":
                self.assertIn("/actions/workflows/push.yml/runs?", route)
                self.assertIn(f"head_sha={SHA}", route)
                return runs
            self.assertEqual(key, "jobs")
            self.assertRegex(route, r"/attempts/\d+/jobs\?")
            return deployments
        return mock.patch.object(ci, "api_items", side_effect=response)

    def test_no_ci_run_triggers_full_validation(self):
        with self.ci_responses([]):
            ci.status()
        self.assertEqual(self.output(), "ci-run-id=\nci-run-attempt=\n")

    def test_latest_successful_ci_run_is_selected(self):
        with self.ci_responses([run(1, 1), run(4, 4), run(3, 3)]):
            ci.status()
        self.assertEqual(self.output(), "ci-run-id=4\nci-run-attempt=1\n")

    def test_latest_failed_or_incomplete_run_blocks_release(self):
        for conclusion, status in [
            ("failure", "completed"), ("cancelled", "completed"),
            ("timed_out", "completed"), (None, "in_progress"),
            (None, "queued"), ("skipped", "completed"),
        ]:
            with self.subTest(conclusion=conclusion, status=status):
                with mock.patch.object(ci, "api_items", return_value=[
                    run(1, 1), run(2, 2, conclusion, status)
                ]):
                    with self.assertRaises(ci.Fatal):
                        ci.status()

    def test_provenance_mismatch_blocks_release(self):
        for key, value in [
            ("path", ".github/workflows/other.yml"),
            ("event", "pull_request"), ("head_branch", "other"),
            ("head_sha", "b" * 40), ("repository", {"full_name": "other/repo"}),
        ]:
            with self.subTest(key=key):
                broken = run(2)
                broken[key] = value
                with mock.patch.object(ci, "api_items", return_value=[broken]):
                    with self.assertRaises(ci.Fatal):
                        ci.status()

    def test_changed_selection_blocks_publication(self):
        for expected in (("9", "1"), ("10", "2"), ("", "")):
            with self.subTest(expected=expected), self.ci_responses([run(10)]):
                with self.assertRaises(ci.Fatal):
                    ci.status(expected)

    def test_successful_run_requires_completed_deployment(self):
        for jobs in ([], [{"name": "deployment", "status": "completed", "conclusion": "skipped"}]):
            with self.subTest(jobs=jobs), self.ci_responses([run(10)], jobs):
                with self.assertRaises(ci.Fatal):
                    ci.status()

    def test_api_authentication_error_is_not_a_fallback(self):
        completed = subprocess.CompletedProcess(["gh"], 1, "", "403")
        with mock.patch.object(ci.subprocess, "run", return_value=completed):
            with self.assertRaises(ci.Fatal):
                ci.api_items("repos/daiksud/gh-settings/actions/runs", "artifacts")

    def test_invalid_api_response_is_not_a_fallback(self):
        completed = subprocess.CompletedProcess(["gh"], 0, "{}", "")
        with mock.patch.object(ci.subprocess, "run", return_value=completed):
            with self.assertRaises(ci.Fatal):
                ci.api_items("repos/daiksud/gh-settings/actions/runs", "artifacts")

    def fixture(self, version=VERSION, corrupt_binary=False):
        archives = {
            11: archive_for("linux-amd64", version, corrupt_binary),
            12: archive_for("darwin-arm64", version),
        }
        items = [artifact("candidate-linux-amd64-1", 11, archives[11]),
                 artifact("candidate-darwin-arm64-2", 12, archives[12])]
        env = {
            **os.environ, "GITHUB_SHA": SHA, "VERSION": version, "GITHUB_EVENT_NAME": "push",
            "GITHUB_RUN_ID": "7", "GITHUB_RUN_ATTEMPT": "2",
            "LINUX_ID": "11", "MACOS_ID": "12",
            "LINUX_DIGEST": items[0]["digest"].removeprefix("sha256:"),
            "MACOS_DIGEST": items[1]["digest"].removeprefix("sha256:"),
        }
        manifest = subprocess.run(
            [sys.executable, SCRIPT.with_name("deployment-manifest.py")],
            env=env, check=True, capture_output=True, text=True,
        )
        record = json.loads(manifest.stdout)
        self.record_deployment(items, archives, record)
        return items, archives, record

    @staticmethod
    def record_deployment(items, archives, record):
        output = io.BytesIO()
        with zipfile.ZipFile(output, "w") as archive:
            archive.writestr("deployment.json", json.dumps(record))
        archives[13] = output.getvalue()
        items[:] = [item for item in items if item["id"] != 13]
        items.append(artifact("deployment-2", 13, archives[13]))

    def exercise(self, items, archives, *, probe=False, version=VERSION):
        with (
            mock.patch.object(ci, "select_ci_run", return_value={**run(7), "run_attempt": 2}),
            mock.patch.object(ci, "api_items", return_value=items),
            mock.patch.object(ci, "fetch_archive", side_effect=lambda repo, identifier, token: archives[identifier]),
        ):
            if probe:
                ci.probe()
            else:
                ci.reuse("7", "2", version)

    def test_reuses_only_deployed_ids_including_products_from_prior_attempt(self):
        items, archives, _ = self.fixture()
        other = archive_for("linux-amd64", "v2.0.0")
        items.append(artifact("candidate-linux-amd64-2", 15, other))
        self.exercise(items, archives)
        self.assertEqual(self.output(), "linux-id=11\nmacos-id=12\nreused=true\n")
        self.assertEqual({p.name for p in Path(self.temp.name).iterdir()}, {"outputs", "summary"})

    def test_missing_expired_duplicate_or_replaced_deployed_item_falls_back(self):
        for variation in ("missing", "expired", "duplicate", "replaced", "digest", "damaged"):
            with self.subTest(variation=variation):
                items, archives, _ = self.fixture()
                if variation == "missing":
                    items.pop()
                elif variation == "expired":
                    items[0]["expired"] = True
                elif variation == "duplicate":
                    items.append(items[-1].copy())
                elif variation == "replaced":
                    items[0]["id"] = 28
                elif variation == "digest":
                    items[0]["digest"] = "sha256:" + "0" * 64
                elif variation == "damaged":
                    archives[11] = b"corrupted"
                self.exercise(items, archives)
                self.assertIn("reused=false\n", self.output())
                self.assertIn("Warning:", (Path(self.temp.name) / "summary").read_text())

    def test_stale_or_incomplete_record_falls_back(self):
        invalid = {"run_attempt": 1, "run_id": True, "source_sha": "b" * 40,
                   "event": "pull_request", "artifacts": {}}
        for key, value in invalid.items():
            with self.subTest(key=key):
                items, archives, record = self.fixture()
                self.record_deployment(items, archives, {**record, key: value})
                self.exercise(items, archives)
                self.assertIn("reused=false\n", self.output())

    def test_wrong_api_producer_stops_reuse(self):
        items, archives, _ = self.fixture()
        items[0]["workflow_run"]["head_sha"] = "b" * 40
        with self.assertRaises(ci.Fatal):
            self.exercise(items, archives)

    def test_normal_ci_products_cannot_be_reused_for_opted_in_promotion(self):
        items, archives, _ = self.fixture(version="v0.44.0")
        self.exercise(items, archives, version="v1.0.1")
        self.assertEqual(self.output(), "reused=false\n")
        self.assertIn("expected v1.0.1, observed v0.44.0", (Path(self.temp.name) / "summary").read_text())

    def test_invalid_binary_manifest_warns_and_falls_back(self):
        items, archives, _ = self.fixture(corrupt_binary=True)
        self.exercise(items, archives)
        self.assertEqual(self.output(), "reused=false\n")
        self.assertIn("binary SHA-256 is invalid", (Path(self.temp.name) / "summary").read_text())

    def test_bad_archive_does_not_extract(self):
        with self.assertRaises(ci.Unusable):
            ci.extract_product(b"not-a-zip", "linux-amd64", Path(self.temp.name) / "broken")
        self.assertFalse((Path(self.temp.name) / "broken").exists())

    def test_no_run_triggers_fresh_validation_and_blocks_probe(self):
        with self.ci_responses([]):
            ci.reuse("", "", VERSION)
            with self.assertRaises(ci.Fatal):
                ci.probe()
        self.assertEqual(self.output(), "reused=false\n")

    def test_probe_requires_deployed_linux_id_and_frozen_version(self):
        items, archives, _ = self.fixture()
        items.pop(1)  # The live consumer requires only the inspected Linux product.
        self.exercise(items, archives, probe=True)
        self.assertEqual(self.output(), f"ci-run-id=7\nci-run-attempt=2\nlinux-id=11\nversion={VERSION}\n")
        items.pop(0)
        with self.assertRaises(ci.Unusable):
            self.exercise(items, archives, probe=True)


if __name__ == "__main__":
    unittest.main()
