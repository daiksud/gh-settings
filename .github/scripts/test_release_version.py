"""Offline contract tests for the gh-settings release calculator."""

import importlib.util
import contextlib
import io
import json
import os
import pathlib
import subprocess
import tempfile
import unittest
from unittest import mock

SCRIPT = pathlib.Path(__file__).with_name("release-version.py")
spec = importlib.util.spec_from_file_location("release_version", SCRIPT)
versioning = importlib.util.module_from_spec(spec)
spec.loader.exec_module(versioning)


class HistoryFixture(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = pathlib.Path(self.tmp.name)
        self.git("init", "--quiet")
        self.git("config", "user.name", "Fixture")
        self.git("config", "user.email", "fixture@example.invalid")
        self.git("config", "commit.gpgsign", "false")
        self.baseline("v0.3.0")

    def git(self, *args):
        return subprocess.check_output(["git", *args], cwd=self.root).decode().strip()

    def baseline(self, tag):
        (self.root / "base").write_text("base")
        self.git("add", ".")
        self.git("commit", "--quiet", "-m", "chore: baseline")
        self.git("tag", tag)
        self.tag = tag
        self.base = self.git("rev-parse", "HEAD")

    def commit(self, message, empty=False, path="change"):
        if not empty:
            file = self.root / path
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_text(file.read_text() + "x" if file.exists() else "x")
            self.git("add", ".")
        self.git("commit", "--quiet", "--allow-empty", "-m", message)
        return self.git("rev-parse", "HEAD")

    def catalog(self, other=None):
        published = {"id": 1, "tag_name": self.tag, "draft": False, "prerelease": False,
                     "published_at": "2026-10-01T00:00:00Z"}
        return [published, *(other or [])]

    def calculate(self, other=None, allow_retry=False, graduate_to_v1=False):
        source = self.git("rev-parse", "HEAD")
        old = pathlib.Path.cwd()
        os.chdir(self.root)
        try:
            return versioning.calculate(source, self.catalog(other), allow_retry, graduate_to_v1)
        finally:
            os.chdir(old)

    def test_version_tiers(self):
        for message, expected, compare in [
            ("fix: correct behavior", "v0.3.1", True),
            ("feat(core): add command", "v0.4.0", False),
            ("perf: speed up search", "v0.4.0", False),
            ("refactor!: remove API", "v0.4.0", False),
            ("docs: explain behavior\n\nDEPRECATED: Old usage", "v0.4.0", False),
            ("fix: change API\n\nBREAKING CHANGE: Update usage", "v0.4.0", False),
            ("docs: examples\n\n```text\nBREAKING CHANGE: Not a footer\n```", "v0.3.1", True),
            ("chore: maintenance", "v0.3.1", True),
            ("build(release): prepare next version", "v0.3.1", True),
            ('Revert "feat: add an old command"', "v0.3.1", True),
            ("Legacy subject\n\nDEPRECATED: Old usage", "v0.4.0", False),
            ("Legacy subject\n\nBREAKING CHANGE: Update usage", "v0.4.0", False),
        ]:
            with self.subTest(message=message):
                # Reset the current fixture to its published boundary.
                self.git("reset", "--hard", self.base)
                self.commit(message)
                self.assertEqual(self.calculate(), (expected, self.tag, self.base, compare))

    def test_action_patch_depends_on_changes_in_the_shipped_action(self):
        for path, compare in [
            ("action.yml", False),
            ("action/install-cli.sh", False),
            ("action/runtime/data", False),
            ("action/README.md", True),
            ("action/templates/report.md", False),
            (".github/actions/build/action.yml", True),
        ]:
            with self.subTest(path=path):
                self.git("reset", "--hard", self.base)
                self.commit("fix: update behavior", path=path)
                self.assertEqual(self.calculate(), ("v0.3.1", self.tag, self.base, compare))

    def test_action_runtime_removal_and_revert_use_net_endpoints(self):
        self.base = self.commit("chore: track installer input", path="action/cli-version")
        self.git("tag", "-f", self.tag)
        self.git("rm", "action/cli-version")
        self.git("commit", "--quiet", "-m", "build: remove installer input")
        self.assertEqual(self.calculate(), ("v0.3.1", self.tag, self.base, False))
        self.git("revert", "--no-edit", "HEAD")
        self.assertEqual(self.calculate(), (self.tag, self.tag, self.base, False))
        self.commit("docs: update usage", path="README.md")
        self.assertEqual(self.calculate(), ("v0.3.1", self.tag, self.base, True))

    def test_changed_tree_with_empty_message_fails_resolution(self):
        (self.root / "change").write_text("content")
        self.git("add", ".")
        self.git("commit", "--quiet", "--allow-empty-message", "-m", "")
        sha = self.git("rev-parse", "HEAD")
        with self.assertRaisesRegex(versioning.ReleaseError, sha):
            self.calculate()

    def test_strongest_signal_wins(self):
        self.commit("fix: correction")
        self.commit("feat: feature")
        self.commit("refactor!: breaking change")
        self.assertEqual(self.calculate()[0], "v0.4.0")

    def test_empty_commits_and_unpublished_tags(self):
        self.commit("build(release): preparation", empty=True)
        self.assertEqual(self.calculate(), (self.tag, self.tag, self.base, False))
        self.commit("fix: correction")
        self.git("tag", "v99.0.0")
        self.assertEqual(self.calculate()[0], "v0.3.1")

    def test_previous_published_timestamp_not_highest_tag(self):
        self.git("tag", "v0.2.9")
        self.commit("fix: patch")
        newer = {"id": 2, "tag_name": "v0.2.9", "draft": False,
                 "prerelease": False, "published_at": "2026-10-02T00:00:00Z"}
        self.assertEqual(self.calculate([newer])[:2], ("v0.2.10", "v0.2.9"))

    def test_retry_with_matching_marker(self):
        sha = self.commit("feat: new feature")
        self.git("tag", "v0.4.0")
        manifest = {"version": "v0.4.0", "commit": sha, "macos_sha256": "a" * 64,
                    "linux_sha256": "b" * 64, "checksums_sha256": "c" * 64}
        released = {"id": 2, "tag_name": "v0.4.0", "draft": False,
                    "prerelease": False, "published_at": "2026-10-02T00:00:00Z",
                    "body": "<!-- gh-settings-release: " + json.dumps(manifest) + " -->\nNotes"}
        self.assertEqual(self.calculate([released], True), ("v0.4.0", "", "", False))
        self.assertEqual(self.calculate([released]), ("v0.4.0", "v0.4.0", sha, False))

    def test_invalid_baseline_and_empty_interval(self):
        self.assertEqual(self.calculate(), (self.tag, self.tag, self.base, False))
        with self.assertRaisesRegex(versioning.ReleaseError, "valid immutable identity"):
            self.calculate(allow_retry=True)
        self.commit("fix: a change")
        (self.root / ".git/shallow").write_text(self.base + "\n")
        with self.assertRaisesRegex(versioning.ReleaseError, "complete Git history"):
            self.calculate()

    def test_graduation_uses_validated_substantive_interval(self):
        for message in ("fix: patch", "feat: feature", "perf: speed", "feat!: breaking", "docs: usage"):
            with self.subTest(message=message):
                self.git("reset", "--hard", self.base)
                self.commit(message)
                self.assertEqual(self.calculate(graduate_to_v1=True), ("v1.0.1", "v0.3.0", self.base, False))
        self.git("reset", "--hard", self.base)
        self.commit("chore: empty", empty=True)
        self.assertEqual(self.calculate(graduate_to_v1=True), (self.tag, self.tag, self.base, False))

    def test_default_and_opt_in_use_the_same_frozen_interval(self):
        self.commit("feat!: new contract")
        self.assertEqual(self.calculate(), ("v0.4.0", "v0.3.0", self.base, False))
        self.assertEqual(self.calculate(graduate_to_v1=True), ("v1.0.1", "v0.3.0", self.base, False))

    def test_promotion_never_duplicates_or_downgrades_published_versions(self):
        self.commit("fix: patch")
        for published in ("v1.0.1", "v1.2.0", "v2.0.0"):
            with self.subTest(published=published):
                self.git("tag", published, self.base)
                # A later-published v0 maintenance baseline must not reset v1.
                older = {"id": 2, "tag_name": published, "draft": False,
                         "prerelease": False, "published_at": "2026-09-30T00:00:00Z"}
                with self.assertRaisesRegex(versioning.ReleaseError, "advance beyond"):
                    self.calculate([older], graduate_to_v1=True)
                self.assertEqual(self.calculate([older])[0], "v0.3.1")

    def test_published_v1_0_1_retains_normal_semver_with_either_choice(self):
        self.git("tag", "v1.0.1", self.base)
        self.tag = "v1.0.1"
        for message, expected in (("fix: patch", "v1.0.2"), ("feat: feature", "v1.1.0"),
                                  ("feat!: break", "v2.0.0")):
            for enabled in (False, True):
                with self.subTest(message=message, enabled=enabled):
                    self.git("reset", "--hard", self.base)
                    self.commit(message)
                    self.assertEqual(self.calculate(graduate_to_v1=enabled),
                                     (expected, self.tag, self.base, message.startswith("fix:")))

    def test_promoted_same_source_retry_survives_checkbox_changes(self):
        sha = self.commit("feat: feature")
        self.git("tag", "v1.0.1")
        marker = {"version": "v1.0.1", "commit": sha, "macos_sha256": "a" * 64,
                  "linux_sha256": "b" * 64, "checksums_sha256": "c" * 64}
        for draft in (False, True):
            release = {"id": 2, "tag_name": "v1.0.1", "draft": draft,
                       "prerelease": False, "published_at": "2026-10-02T00:00:00Z",
                       "body": "<!-- gh-settings-release: " + json.dumps(marker) + " -->"}
            for enabled in (False, True):
                with self.subTest(draft=draft, enabled=enabled):
                    self.assertEqual(self.calculate([release], True, enabled), ("v1.0.1", "", "", False))

    def test_graduation_keeps_immutable_retry(self):
        sha = self.commit("feat: feature")
        self.git("tag", "v0.4.0")
        marker = {"version": "v0.4.0", "commit": sha, "macos_sha256": "a" * 64,
                  "linux_sha256": "b" * 64, "checksums_sha256": "c" * 64}
        release = {"id": 2, "tag_name": "v0.4.0", "draft": True, "prerelease": False,
                   "body": "<!-- gh-settings-release: " + json.dumps(marker) + " -->"}
        self.assertEqual(self.calculate([release], True, False), ("v0.4.0", "", "", False))
        with self.assertRaisesRegex(versioning.ReleaseError, "cannot promote an existing v0 release"):
            self.calculate([release], True, True)
        self.assertEqual(self.git("rev-parse", "v0.4.0"), sha)
        self.git("tag", "v1.0.0")
        marker["version"] = "v1.0.0"
        release["tag_name"] = "v1.0.0"
        release["body"] = "<!-- gh-settings-release: " + json.dumps(marker) + " -->"
        self.assertEqual(self.calculate([release], True, True), ("v1.0.0", "", "", False))
        duplicate = dict(release, id=3)
        with self.assertRaisesRegex(versioning.ReleaseError, "multiple existing releases"):
            self.calculate([release, duplicate], True, True)
        self.git("tag", "-f", "v1.0.0", self.base)
        with self.assertRaisesRegex(versioning.ReleaseError, "does not match"):
            self.calculate([release], True, True)

    def test_graduation_does_not_bypass_baseline_checks(self):
        self.commit("fix: change")
        source = self.git("rev-parse", "HEAD")
        old = pathlib.Path.cwd()
        os.chdir(self.root)
        try:
            with self.assertRaisesRegex(versioning.ReleaseError, "initial release"):
                versioning.calculate(source, [], graduate_to_v1=True)
            with self.assertRaisesRegex(versioning.ReleaseError, "source"):
                versioning.calculate("bad", self.catalog(), graduate_to_v1=True)
            (self.root / ".git/shallow").write_text(self.base + "\n")
            with self.assertRaisesRegex(versioning.ReleaseError, "complete Git history"):
                versioning.calculate(source, self.catalog(), graduate_to_v1=True)
            (self.root / ".git/shallow").unlink()
            tree = self.git("rev-parse", f"{self.base}^{{tree}}")
            unrelated = self.git("commit-tree", tree, "-m", "chore: unrelated")
            self.git("tag", "-f", self.tag, unrelated)
            with self.assertRaisesRegex(versioning.ReleaseError, "not an ancestor"):
                versioning.calculate(source, self.catalog(), graduate_to_v1=True)
            self.git("tag", "-d", self.tag)
            with self.assertRaises(versioning.ReleaseError):
                versioning.calculate(source, self.catalog(), graduate_to_v1=True)
        finally:
            os.chdir(old)

    def test_real_cli_option_parsing(self):
        source = self.commit("fix: change")
        old = pathlib.Path.cwd()
        os.chdir(self.root)
        try:
            with mock.patch.dict(os.environ, {"SOURCE_SHA": source, "GITHUB_REPOSITORY": "owner/repo"}), \
                 mock.patch.object(versioning, "read_releases", return_value=self.catalog()):
                for args, version, compare in (
                        ([], "v0.3.1", "true"),
                        (["--graduate-to-v1"], "v1.0.1", "false"),
                        (["--allow-retry", "--graduate-to-v1"], "v1.0.1", "false"),
                        (["--graduate-to-v1", "--allow-retry"], "v1.0.1", "false")):
                    output = self.root / "outputs"
                    output.unlink(missing_ok=True)
                    with mock.patch.object(versioning.sys, "argv", ["release-version.py", *args]), mock.patch.dict(os.environ, {"GITHUB_OUTPUT": str(output)}):
                        versioning.main()
                    self.assertEqual(output.read_text(), f"version={version}\nprevious-tag=v0.3.0\nprevious-sha={self.base}\n"
                                                         f"release-needed=true\ncompare-binaries={compare}\n")
                for args in (["--graduate-to-v1", "--graduate-to-v1"], ["--unknown"], ["--allow-retry=yes"]):
                    with mock.patch.object(versioning.sys, "argv", ["release-version.py", *args]), contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit):
                        versioning.main()
        finally:
            os.chdir(old)

    def test_empty_push_keeps_baseline_version_and_skips_manual_publication(self):
        source = self.commit("chore: rerun main CI", empty=True)
        output = self.root / "outputs"
        old = pathlib.Path.cwd()
        os.chdir(self.root)
        try:
            env = {"SOURCE_SHA": source, "GITHUB_REPOSITORY": "owner/repo", "GITHUB_OUTPUT": str(output)}
            with mock.patch.dict(os.environ, env), mock.patch.object(versioning, "read_releases", return_value=self.catalog()), \
                 mock.patch.object(versioning.sys, "argv", ["release-version.py", "--allow-retry"]):
                versioning.main()
            self.assertEqual(output.read_text(), f"version={self.tag}\nprevious-tag={self.tag}\nprevious-sha={self.base}\nrelease-needed=false\ncompare-binaries=false\n")
        finally:
            os.chdir(old)

    def test_stable_history_retains_normal_tiers(self):
        self.git("tag", "v1.2.3", self.base)
        self.tag = "v1.2.3"
        for message, expected in (("fix: patch", "v1.2.4"), ("feat: feature", "v1.3.0"), ("feat!: break", "v2.0.0")):
            with self.subTest(message=message):
                self.git("reset", "--hard", self.base)
                self.commit(message)
                self.assertEqual(self.calculate(graduate_to_v1=True)[0], expected)


class ParsingTests(unittest.TestCase):
    def test_fences_and_trailer_positions(self):
        for message, breaking, minor in [
            ("docs: mention\nBREAKING CHANGE: text", False, False),
            ("docs: mention\n\n```\nBREAKING CHANGE: fake\n```", False, False),
            ("fix: change\n\nBREAKING-CHANGE: migrate", True, False),
            ("fix: change\n\nRefs #123\nDEPRECATED: leave", False, True),
            ("docs: example\n\n~~~\nDEPRECATED: fake\n~~~", False, False),
        ]:
            with self.subTest(message=message):
                self.assertEqual(versioning.classify(message), (breaking, minor))

    def test_semver_overflow_and_major(self):
        self.assertEqual(versioning.next_version("v1.2.3", 2), "v2.0.0")
        self.assertEqual(versioning.next_version("v0.2.3", 2), "v0.3.0")
        with self.assertRaises(versioning.ReleaseError):
            versioning.next_version("v1.2.18446744073709551615", 0)

    def test_graduation_component_bounds_and_stable_tiers(self):
        maximum = str(versioning.MAX_COMPONENT)
        self.assertEqual(versioning.next_version(f"v0.{maximum}.{maximum}", 0, True), "v1.0.1")
        with self.assertRaises(versioning.ReleaseError):
            versioning.next_version(f"v0.{versioning.MAX_COMPONENT + 1}.0", 2, True)
        for tier, expected in ((0, "v1.2.4"), (1, "v1.3.0"), (2, "v2.0.0")):
            self.assertEqual(versioning.next_version("v1.2.3", tier, True), expected)
            self.assertEqual(versioning.next_version("v1.2.3", tier), expected)

    def test_catalog_rejects_incomplete_data(self):
        for release in [{"id": 1, "tag_name": "v0.1.0"},
                        {"id": 1, "tag_name": "v0.1.0", "draft": False,
                         "prerelease": False, "published_at": None}]:
            with self.subTest(release=release):
                with mock.patch.object(versioning, "command", return_value=json.dumps([[release]]).encode()):
                    with self.assertRaisesRegex(versioning.ReleaseError, "incomplete"):
                        versioning.read_releases("owner/repo")


if __name__ == "__main__":
    unittest.main()
