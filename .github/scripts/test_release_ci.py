"""Offline state, integrity, and fallback tests for release-ci.py."""

import hashlib
import importlib.util
import io
import os
from pathlib import Path
import subprocess
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
        "path": ".github/workflows/ci-push.yml", "event": "push",
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

    def test_no_ci_run_triggers_full_validation(self):
        with mock.patch.object(ci, "api_items", return_value=[]):
            ci.status()
        self.assertEqual(self.output(), "ci-run-id=\n")

    def test_latest_successful_ci_run_is_selected(self):
        with mock.patch.object(ci, "api_items", return_value=[
            run(1, 1), run(4, 4), run(3, 3)
        ]):
            ci.status()
        self.assertEqual(self.output(), "ci-run-id=4\n")

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

    def test_changed_selected_run_blocks_publication(self):
        with mock.patch.object(ci, "api_items", return_value=[run(10)]):
            with self.assertRaises(ci.Fatal):
                ci.status("9")

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

    def fixture(self):
        linux = archive_for("linux-amd64")
        macos = archive_for("darwin-arm64")
        artifacts = [
            artifact("release-linux-amd64", 11, linux),
            artifact("release-darwin-arm64", 12, macos),
        ]
        archives = {11: linux, 12: macos}
        return artifacts, archives

    def exercise_reuse(self, artifacts, archive_map):
        with (
            mock.patch.object(ci, "select_ci_run", return_value=run(7)),
            mock.patch.object(ci, "api_items", return_value=artifacts),
            mock.patch.object(
                ci, "fetch_archive",
                side_effect=lambda _repo, identifier, _token: archive_map[identifier],
            ),
        ):
            ci.reuse("7", VERSION)

    def test_matching_products_are_reused_without_rebuild(self):
        artifacts, archives = self.fixture()
        self.exercise_reuse(artifacts, archives)
        self.assertEqual(self.output(), "reused=true\n")
        for target in ci.TARGETS:
            path = Path(self.temp.name) / "reused-release" / target
            self.assertTrue((path / ("gh-settings-" + target)).is_file())
            self.assertTrue((path / "release-manifest").is_file())

    def test_missing_expired_or_duplicate_artifact_falls_back(self):
        for variation in ("missing", "expired", "duplicate"):
            with self.subTest(variation=variation):
                artifacts, archives = self.fixture()
                if variation == "missing":
                    artifacts.pop()
                elif variation == "expired":
                    artifacts[0]["expired"] = True
                else:
                    artifacts.append(artifacts[0].copy())
                self.exercise_reuse(artifacts, archives)
                self.assertIn("reused=false\n", self.output())
                self.assertIn("Warning:", (Path(self.temp.name) / "summary").read_text())
                self.assertFalse((Path(self.temp.name) / "reused-release").exists())

    def test_digest_mismatch_warns_and_falls_back(self):
        artifacts, archives = self.fixture()
        artifacts[0]["digest"] = "sha256:" + ("0" * 64)
        self.exercise_reuse(artifacts, archives)
        self.assertEqual(self.output(), "reused=false\n")
        self.assertIn("SHA-256 mismatch", (Path(self.temp.name) / "summary").read_text())

    def test_version_mismatch_warns_and_falls_back(self):
        artifacts, archives = self.fixture()
        archives[11] = archive_for("linux-amd64", "v1.2.4")
        artifacts[0]["digest"] = "sha256:" + hashlib.sha256(archives[11]).hexdigest()
        self.exercise_reuse(artifacts, archives)
        self.assertEqual(self.output(), "reused=false\n")
        self.assertIn("version mismatch", (Path(self.temp.name) / "summary").read_text())

    def test_invalid_binary_manifest_warns_and_falls_back(self):
        artifacts, archives = self.fixture()
        archives[11] = archive_for("linux-amd64", corrupt_binary=True)
        artifacts[0]["digest"] = "sha256:" + hashlib.sha256(archives[11]).hexdigest()
        self.exercise_reuse(artifacts, archives)
        self.assertEqual(self.output(), "reused=false\n")
        self.assertIn("binary SHA-256 is invalid", (Path(self.temp.name) / "summary").read_text())

    def test_bad_archive_does_not_extract(self):
        with self.assertRaises(ci.Unusable):
            ci.extract_product(b"not-a-zip", "linux-amd64", Path(self.temp.name) / "broken")
        self.assertFalse((Path(self.temp.name) / "broken").exists())

    def test_no_run_outputs_fallback_without_network(self):
        with mock.patch.object(ci, "api_items", side_effect=AssertionError("network was used")):
            ci.reuse("", VERSION)
        self.assertEqual(self.output(), "reused=false\n")


if __name__ == "__main__":
    unittest.main()
