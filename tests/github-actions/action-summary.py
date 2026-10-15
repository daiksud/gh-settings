#!/usr/bin/env python3
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("summary", Path(__file__).resolve().parents[2] / "action/render-summary.py")
summary = importlib.util.module_from_spec(spec)
spec.loader.exec_module(summary)


class SummaryTest(unittest.TestCase):
    def setUp(self):
        self.env = {f"SETTINGS_{key}": value for key, value in {
            "ACTOR": "owner", "TRIGGERING_ACTOR": "owner", "EVENT": "push",
            "RUN_ID": "123", "RUN_ATTEMPT": "1", "WORKFLOW_REF": "owner/repo/.github/workflows/settings-main.yml@refs/heads/main",
            "WORKFLOW_SHA": "a" * 40, "CONFIG_SHA": "b" * 40, "TARGET": "owner/repo",
            "CLI_VERSION": "v0.5.0", "CLI_SHA256": "c" * 64,
            "ACTION_REPOSITORY": "action-owner/action-repo", "ACTION_REF": "d" * 40,
        }.items()}

    def test_remote_action_identity_is_separate(self):
        rendered = summary.render("settings.repository.has_issues: true → false\n", 0, self.env)
        self.assertIn('Action repository</th><td>action-owner/action-repo', rendered)
        self.assertIn('Action ref</th><td>' + 'd' * 40, rendered)

    def test_run_and_configuration_links_use_their_own_repositories(self):
        self.env['SETTINGS_TARGET'] = 'config-owner/config-repo'
        self.env['SETTINGS_WORKFLOW_REF'] = 'workflow-owner/workflow-repo/.github/workflows/settings.yml@refs/heads/main'
        rendered = summary.render('', 0, self.env)
        self.assertIn('https://github.com/workflow-owner/workflow-repo/actions/runs/123', rendered)
        self.assertIn('https://github.com/workflow-owner/workflow-repo/blob/' + 'a' * 40, rendered)
        self.assertIn('https://github.com/config-owner/config-repo/blob/' + 'b' * 40 + '/.github/settings.yml', rendered)

    def test_local_action_has_no_invented_remote_pin(self):
        for value in ('', None):
            env = dict(self.env)
            for key in ('SETTINGS_ACTION_REPOSITORY', 'SETTINGS_ACTION_REF'):
                if value is None:
                    del env[key]
                else:
                    env[key] = value
            rendered = summary.render('', 0, env)
            self.assertIn('Action repository</th><td>Local checkout', rendered)
            self.assertIn('Action ref</th><td>Unavailable (local action)', rendered)

    def test_incomplete_remote_action_identity_fails(self):
        for key in ('SETTINGS_ACTION_REPOSITORY', 'SETTINGS_ACTION_REF'):
            for value in ('', None):
                env = dict(self.env)
                if value is None:
                    del env[key]
                else:
                    env[key] = value
                with self.subTest(key=key, value=value), self.assertRaises((KeyError, ValueError)):
                    summary.render('', 0, env)

    def test_first_run_and_rerun_exact_links(self):
        first = summary.render("settings.repository.has_issues: true → false\n", 0, self.env)
        self.assertIn("Observed before → desired after", first)
        self.assertIn("/blob/" + "a" * 40 + "/.github/workflows/settings-main.yml", first)
        self.assertIn("/blob/" + "b" * 40 + "/.github/settings.yml", first)
        self.env["SETTINGS_TRIGGERING_ACTOR"] = "maintainer"
        self.env["SETTINGS_RUN_ATTEMPT"] = "2"
        rerun = summary.render("", 1, self.env)
        self.assertIn("Original actor</th><td>owner", rerun)
        self.assertIn("Initiating / rerun actor</th><td>maintainer", rerun)
        self.assertIn("</a> / 2", rerun)
        self.assertIn("prepare failed", rerun)

    def test_no_change_and_unmanaged(self):
        for output, expected in [("No differences for managed repository settings.\n", "No differences for managed settings."), ("No managed repository settings.\n", "No managed settings.")]:
            self.assertIn(expected, summary.render(output, 0, self.env))

    def test_untrusted_text_cannot_close_data_block(self):
        attack = '"quotes"\n</pre>\n```\n## Approved\n<script>evil</script>\n::notice::approved\n&copy;'
        rendered = summary.render(attack, 0, self.env)
        self.assertIn('&lt;/pre&gt;', rendered)
        self.assertIn('&lt;script&gt;', rendered)
        self.assertIn('&quot;quotes&quot;', rendered)
        self.assertIn('&amp;copy;', rendered)
        self.assertEqual(rendered.count("</pre>"), 1)
        self.assertNotIn("<script>", rendered)
        self.assertIn("<pre>", rendered)

    def test_missing_context_fails(self):
        for key in self.env:
            if key.startswith('SETTINGS_ACTION_'):
                continue
            for value in ('', None):
                env = dict(self.env)
                if value is None:
                    del env[key]
                else:
                    env[key] = value
                with self.subTest(key=key, value=value), self.assertRaises((KeyError, ValueError)):
                    summary.render('', 0, env)


if __name__ == "__main__":
    unittest.main()
