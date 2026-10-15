#!/usr/bin/env python3
"""Render only explicit non-secret run metadata and CLI output as quoted HTML data."""
import html
import os
from pathlib import Path
import sys


def render(output, status, env):
    def value(name):
        result = env[f"SETTINGS_{name}"]
        if not result:
            raise ValueError(f"missing SETTINGS_{name}")
        return result

    def quoted(text):
        return html.escape(str(text), quote=True)

    def link(url, label):
        return f'<a href="{quoted(url)}">{quoted(label)}</a>'

    target = value("TARGET")
    config_sha = value("CONFIG_SHA")
    workflow_sha = value("WORKFLOW_SHA")
    workflow_ref = value("WORKFLOW_REF")
    source = workflow_ref.split("@", 1)[0]
    owner, repository, workflow_path = source.split("/", 2)
    run_url = f'https://github.com/{owner}/{repository}/actions/runs/{value("RUN_ID")}'
    rows = [
        ("Original actor", quoted(value("ACTOR"))),
        ("Initiating / rerun actor", quoted(value("TRIGGERING_ACTOR"))),
        ("Event", quoted(value("EVENT"))),
        ("Run / attempt", link(run_url, value("RUN_ID")) + " / " + quoted(value("RUN_ATTEMPT"))),
        ("Workflow path / ref", quoted(workflow_ref)),
        ("Workflow revision", link(f"https://github.com/{owner}/{repository}/blob/{workflow_sha}/{workflow_path}", workflow_sha)),
        ("Target repository", link(f"https://github.com/{target}", target)),
        ("Configuration revision", link(f"https://github.com/{target}/blob/{config_sha}/.github/settings.yml", config_sha)),
        ("Action repository", quoted(value("ACTION_REPOSITORY"))),
        ("Action ref", quoted(value("ACTION_REF"))),
        ("CLI version", quoted(value("CLI_VERSION"))),
        ("CLI SHA-256", quoted(value("CLI_SHA256"))),
    ]
    if status:
        result = "Error — prepare failed; apply remains blocked."
    elif output.strip() == "No differences for managed repository settings.":
        result = "No differences for managed settings."
    elif output.strip() == "No managed repository settings.":
        result = "No managed settings."
    else:
        result = "Observed before → desired after."
    table = "\n".join(f"<tr><th>{quoted(name)}</th><td>{data}</td></tr>" for name, data in rows)
    return (
        "## Settings review\n\n<table>\n" + table + "\n</table>\n\n"
        + f"<p>{quoted(result)} Exit status: {status}.</p>\n\n"
        + "<pre>" + quoted(output) + "</pre>\n\n"
        + "Direct apply uses the declaration and can overwrite intervening remote changes to managed fields. "
        + "Approve only after inspecting this complete Summary; successful prepare alone does not prove publication.\n"
    )


if __name__ == "__main__":
    if len(sys.argv) != 3:
        raise SystemExit("usage: render-summary.py OUTPUT_FILE EXIT_STATUS")
    print(render(Path(sys.argv[1]).read_text(), int(sys.argv[2]), os.environ), end="")
