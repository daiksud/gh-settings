#!/usr/bin/env python3
"""Render only explicit non-secret run metadata and CLI output as quoted HTML data."""
import html
import os
from pathlib import Path
import sys
from urllib.parse import quote


def render(output, status, env):
    def value(name):
        result = env[f"SETTINGS_{name}"]
        if not result:
            raise ValueError(f"missing SETTINGS_{name}")
        return result

    def quoted(text):
        # Raw metadata blank lines can end a GFM HTML table block.
        return html.escape(str(text), quote=True).replace("\r", "&#13;").replace("\n", "&#10;")

    def link(url, label):
        return f'<a href="{quoted(url)}">{quoted(label)}</a>'

    target = value("TARGET")
    config_sha = value("CONFIG_SHA")
    workflow_sha = value("WORKFLOW_SHA")
    workflow_ref = value("WORKFLOW_REF")
    source, _ = workflow_ref.split("@refs/", 1)
    owner, repository, workflow_path = source.split("/", 2)
    run_url = f'https://github.com/{owner}/{repository}/actions/runs/{value("RUN_ID")}'
    action_repository, action_ref = "Local checkout", "Unavailable (local action)"
    if env.get("SETTINGS_ACTION_REPOSITORY") or env.get("SETTINGS_ACTION_REF"):
        action_repository, action_ref = value("ACTION_REPOSITORY"), value("ACTION_REF")
    rows = [
        ("Original actor", quoted(value("ACTOR"))),
        ("Initiating / rerun actor", quoted(value("TRIGGERING_ACTOR"))),
        ("Event", quoted(value("EVENT"))),
        ("Run / attempt", link(run_url, value("RUN_ID")) + " / " + quoted(value("RUN_ATTEMPT"))),
        ("Workflow path / ref", quoted(workflow_ref)),
        ("Workflow revision", link(f"https://github.com/{owner}/{repository}/blob/{workflow_sha}/{quote(workflow_path, safe='/')}", workflow_sha)),
        ("Target repository", link(f"https://github.com/{target}", target)),
        ("Configuration revision", link(f"https://github.com/{target}/blob/{config_sha}/.github/settings.yml", config_sha)),
        ("Action repository", quoted(action_repository)),
        ("Action ref", quoted(action_ref)),
        ("CLI version", quoted(value("CLI_VERSION"))),
        ("CLI SHA-256", quoted(value("CLI_SHA256"))),
    ]
    if status:
        result = "Error — plan failed."
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
        + "<pre>" + html.escape(output, quote=True) + "</pre>\n\n"
        + "Direct apply uses the declaration and can overwrite intervening remote changes to managed fields. "
        + "Review this Summary before authorizing a separate direct apply; successful rendering alone does not prove publication.\n"
    )


if __name__ == "__main__":
    if len(sys.argv) != 3:
        raise SystemExit("usage: render-summary.py OUTPUT_FILE EXIT_STATUS")
    status = int(sys.argv[2])
    try:
        content = render(Path(sys.argv[1]).read_text(encoding="utf-8"), status, os.environ)
        with Path(os.environ["GITHUB_STEP_SUMMARY"]).open("a", encoding="utf-8") as output:
            output.write(content)
    except (OSError, KeyError, ValueError):
        print("Could not render or write plan Summary.", file=sys.stderr)
        raise SystemExit(status or 1)
    raise SystemExit(status)
