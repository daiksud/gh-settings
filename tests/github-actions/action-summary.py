#!/usr/bin/env python3
import importlib.util
from html import unescape
import os
from pathlib import Path
import subprocess
import sys
import tempfile
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

    def test_workflow_filename_and_ref_can_contain_at(self):
        for extension in ('yml', 'yaml'):
            path = f'.github/workflows/review@prod.{extension}'
            workflow_ref = f'owner/repo/{path}@refs/heads/release@candidate'
            rendered = summary.render('', 0, dict(self.env, SETTINGS_WORKFLOW_REF=workflow_ref))
            with self.subTest(extension=extension):
                self.assertIn('https://github.com/owner/repo/blob/' + 'a' * 40 + '/' + path, rendered)
                self.assertIn(workflow_ref, rendered)

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
        self.assertIn("plan failed", rerun)

    def test_failed_plan_is_an_error_without_claiming_apply_gating(self):
        rendered = summary.render('API request failed\n', 37, self.env)
        self.assertIn('Error — plan failed.', rendered)
        self.assertIn('Exit status: 37.', rendered)
        self.assertIn('can overwrite intervening remote changes', rendered)
        self.assertIn('does not prove publication', rendered)
        self.assertNotIn('apply remains blocked', rendered)
        self.assertNotIn('successful prepare', rendered)

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

    def test_metadata_is_escaped_without_dumping_other_environment(self):
        attack = '\"</td><script>bad</script>[link](https://example.invalid)\n## Forged & text'
        expected = '&quot;&lt;/td&gt;&lt;script&gt;bad&lt;/script&gt;[link](https://example.invalid)'
        for key in self.env:
            env = dict(self.env)
            env.update(GH_TOKEN='unrelated-secret-fixture', GITHUB_TOKEN='unrelated-secret-fixture',
                       SETTINGS_TOKEN='unrelated-secret-fixture', UNRELATED_ENV='unrelated-secret-fixture')
            env[key] = attack
            if key == 'SETTINGS_WORKFLOW_REF':
                env[key] = 'owner/repo/.github/workflows/' + attack + '.yml@refs/heads/main'
            rendered = summary.render('plain CLI text\n', 0, env)
            with self.subTest(key=key):
                self.assertIn(expected, rendered)
                self.assertIn('## Forged &amp; text', rendered)
                self.assertIn(attack, unescape(rendered))
                self.assertNotIn('<script>', rendered)
                self.assertNotIn('unrelated-secret-fixture', rendered)

    def test_blank_metadata_lines_cannot_start_markdown(self):
        for newline in ('\n', '\r', '\r\n'):
            attack = f'ordinary{newline}{newline}## Forged approval{newline}{newline}[Approve](https://example.invalid){newline}{newline}'
            env = dict(self.env, SETTINGS_ACTOR=attack)
            rendered = summary.render('', 0, env)
            normalized = rendered.replace('\r\n', '\n').replace('\r', '\n')
            with self.subTest(newline=repr(newline)):
                self.assertNotIn('\n\n## Forged approval', normalized)
                self.assertIn(attack, unescape(rendered))

    def test_cli_appends_summary_and_preserves_plan_status(self):
        for status, cli_output, expected in (
            (0, 'No differences for managed repository settings.\n', 'No differences for managed settings.'),
            (37, 'API failure\n', 'Error — plan failed.'),
        ):
            with tempfile.TemporaryDirectory() as directory:
                fixture = Path(directory).resolve()
                cli_text = fixture / 'CLI text.txt'
                cli_text.write_text(cli_output)
                summary_file = fixture / 'step Summary.md'
                summary_file.write_text('existing Summary\n')
                env = dict(os.environ, **self.env, GITHUB_STEP_SUMMARY=str(summary_file), GH_TOKEN='unrelated-secret-fixture')
                result = subprocess.run([sys.executable, str(Path(summary.__file__)), str(cli_text), str(status)],
                                        env=env, cwd=fixture, capture_output=True, text=True)
                written = summary_file.read_text()
                self.assertEqual(result.returncode, status, result.stderr)
                self.assertEqual(result.stdout, '')
                self.assertTrue(written.startswith('existing Summary\n'))
                self.assertIn(expected, written)
                self.assertIn(cli_output, written)
                self.assertIn(f'Exit status: {status}.', written)
                self.assertNotIn('unrelated-secret-fixture', written + result.stdout + result.stderr)

    def test_cli_render_failure_preserves_existing_summary_and_outputs(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = Path(directory).resolve()
            cli_text = fixture / 'CLI text.txt'
            cli_text.write_text('No differences for managed repository settings.\n')
            summary_file = fixture / 'step Summary.md'
            summary_file.write_text('existing Summary\n')
            outputs = fixture / 'success outputs.txt'
            outputs.write_text('existing outputs\n')
            env = dict(os.environ, **self.env, GITHUB_STEP_SUMMARY=str(summary_file), GITHUB_OUTPUT=str(outputs),
                       GH_TOKEN='unrelated-secret-fixture')
            del env['SETTINGS_CONFIG_SHA']
            result = subprocess.run([sys.executable, str(Path(summary.__file__)), str(cli_text), '0'],
                                    env=env, cwd=fixture, capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, '')
            self.assertEqual(summary_file.read_text(), 'existing Summary\n')
            self.assertEqual(outputs.read_text(), 'existing outputs\n')
            self.assertNotIn('unrelated-secret-fixture', result.stdout + result.stderr)

    def test_cli_summary_write_failure_emits_no_success_outputs(self):
        with tempfile.TemporaryDirectory() as directory:
            fixture = Path(directory).resolve()
            cli_text = fixture / 'CLI text.txt'
            cli_text.write_text('No differences for managed repository settings.\n')
            summary_file = fixture / 'step Summary.md'
            summary_file.mkdir()
            outputs = fixture / 'success outputs.txt'
            outputs.write_text('existing outputs\n')
            env = dict(os.environ, **self.env, GITHUB_STEP_SUMMARY=str(summary_file), GITHUB_OUTPUT=str(outputs),
                       GH_TOKEN='unrelated-secret-fixture')
            result = subprocess.run([sys.executable, str(Path(summary.__file__)), str(cli_text), '0'],
                                    env=env, cwd=fixture, capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(result.stdout, '')
            self.assertEqual(outputs.read_text(), 'existing outputs\n')
            self.assertNotIn('unrelated-secret-fixture', result.stdout + result.stderr)

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
