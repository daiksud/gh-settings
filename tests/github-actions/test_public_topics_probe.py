import contextlib
import importlib.util
import io
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch


SCRIPT = Path(__file__).resolve().parents[2] / ".github/scripts/verify-public-topics.py"
spec = importlib.util.spec_from_file_location("probe", SCRIPT)
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)


class PublicTopicsProbe(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.workspace = self.root / "consumer"
        self.workspace.mkdir()
        self.output = self.root / "output"
        self.env = patch.dict(os.environ, {
            "GITHUB_WORKSPACE": str(self.workspace), "RUNNER_TEMP": str(self.root),
            "GITHUB_OUTPUT": str(self.output), "ACTION_TARGET": probe.TARGET,
            "ACTION_VERSION": probe.VERSION, "ACTION_DIGEST": probe.DIGEST,
        })
        self.env.start()
        self.addCleanup(self.env.stop)
        self.repo = {"full_name": probe.TARGET, "private": False, "archived": False,
                     "id": 123, "owner": {"login": "daiksud", "id": 42, "type": "User"}}
        self.topics = ["retained-topic"]
        self.api_calls = []

        def api(path):
            self.api_calls.append(path)
            return self.repo if path == "" else {"names": self.topics}

        self.api = patch.object(probe, "api", side_effect=api)
        self.api.start()
        self.addCleanup(self.api.stop)
        self.subject = probe.Probe()

    def prepare(self):
        with contextlib.redirect_stdout(io.StringIO()):
            self.subject.prepare()

    def test_lifecycle_retains_original_and_only_commits_topics(self):
        self.prepare()
        state = self.subject.load()
        self.assertEqual(state["original"], ["retained-topic"])
        self.assertEqual(probe.command(["git", "ls-files"], self.workspace).strip(), ".github/settings.yml")
        self.assertEqual(probe.command(["git", "remote", "get-url", "origin"], self.workspace).strip(),
                         "https://github.com/daiksud/test-gh-settings-public.git")
        self.subject.arm()
        self.topics = state["desired"]
        self.subject.check("desired")
        self.subject.restore_config()
        self.assertEqual(self.output.read_text(), "ready=true\n")
        self.assertIn('["retained-topic"]', (self.workspace / ".github/settings.yml").read_text())
        # The helper never mutates the API, including restoration.
        self.topics = state["original"]
        self.subject.check("original")
        self.assertEqual(set(self.api_calls), {"", "/topics"})

    def test_unsafe_targets_and_incomplete_topics_stop_before_configuration(self):
        for change in ({"private": True}, {"archived": True}, {"full_name": "daiksud/gh-settings"}):
            with self.subTest(change=change), patch.dict(self.repo, change):
                with self.assertRaises(RuntimeError):
                    self.prepare()
                self.assertFalse((self.workspace / ".github").exists())
        self.topics = None
        with self.assertRaises(RuntimeError):
            self.prepare()

    def test_marker_and_full_inventory_fail_before_arming(self):
        for topics in ([probe.MARKER], [f"topic-{n}" for n in range(20)]):
            self.topics = topics
            with self.assertRaises(RuntimeError):
                self.prepare()
            self.assertFalse(self.subject.path.exists())

    def test_preapply_drift_and_wrong_package_do_not_arm(self):
        self.prepare()
        self.topics = ["other-edit"]
        with self.assertRaises(RuntimeError):
            self.subject.arm()
        self.assertFalse(self.subject.load()["armed"])
        self.topics = ["retained-topic"]
        with patch.dict(os.environ, {"ACTION_DIGEST": "wrong"}):
            with self.assertRaises(RuntimeError):
                self.subject.arm()
        self.assertFalse(self.subject.load()["armed"])

    def test_unarmed_install_failure_does_not_request_restoration(self):
        self.prepare()
        self.subject.restore_config()
        self.assertFalse(self.output.exists())

    def test_uncertain_apply_can_restore_both_known_states(self):
        self.prepare()
        self.subject.arm()
        for key in ("original", "desired"):
            self.topics = self.subject.load()[key]
            self.subject.restore_config()
            self.assertIn("ready=true", self.output.read_text())

    def test_concurrent_change_or_recreated_target_blocks_restoration(self):
        self.prepare()
        self.subject.arm()
        self.topics = ["someone-elses-edit"]
        with self.assertRaises(RuntimeError):
            self.subject.restore_config()
        self.topics = self.subject.load()["desired"]
        self.repo["id"] += 1
        with self.assertRaises(RuntimeError):
            self.subject.restore_config()
        self.assertFalse(self.output.exists())

    def test_subprocess_diagnostics_cannot_disclose_credentials(self):
        result = type("Result", (), {"returncode": 1, "stdout": "secret", "stderr": "private-key"})()
        with patch.object(probe.subprocess, "run", return_value=result), patch.dict(os.environ, {"GH_DEBUG": "api"}):
            with self.assertRaises(RuntimeError) as failure:
                probe.command(["gh", "api"])
            self.assertNotIn("secret", str(failure.exception))
            self.assertNotIn("private-key", str(failure.exception))
            self.assertNotIn("GH_DEBUG", probe.subprocess.run.call_args.kwargs["env"])

    def test_repeat_requires_both_no_difference_and_independent_membership(self):
        self.prepare()
        self.topics = self.subject.load()["desired"]
        for outcome in ("changed", "", "unmanaged"):
            with patch.dict(os.environ, {"PLAN_OUTCOME": outcome}), patch.object(probe.sys, "argv", ["probe", "verify-repeat"]):
                with self.assertRaises(RuntimeError):
                    probe.main()
        with patch.dict(os.environ, {"PLAN_OUTCOME": "no-difference"}), patch.object(probe.sys, "argv", ["probe", "verify-repeat"]):
            probe.main()
            self.topics = ["retained-topic"]
            with self.assertRaises(RuntimeError):
                probe.main()


if __name__ == "__main__":
    unittest.main()
