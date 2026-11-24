#!/usr/bin/env python3
"""Fail-closed main CI gate and bounded reuse of its exact release artifacts."""

import hashlib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import urllib.error
import urllib.parse
import urllib.request
import zipfile

SOURCE = re.compile(r"^[0-9a-f]{40}$")
VERSION = re.compile(r"^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")
DIGEST = re.compile(r"^sha256:([0-9a-f]{64})$")
TARGETS = ("linux-amd64", "darwin-arm64")
MAX_ARCHIVE_BYTES = 150 * 1024 * 1024
MAX_PRODUCT_BYTES = 100 * 1024 * 1024


class Fatal(Exception):
    """CI status, authorization or transport uncertainty prohibits release."""


class Unusable(Exception):
    """Artifact absence or failed integrity permits full fresh validation."""


def field(name):
    value = os.environ.get(name, "")
    if not value:
        raise Fatal(f"missing {name}")
    return value


def identity():
    repo, source = field("GITHUB_REPOSITORY"), field("SOURCE_SHA")
    if repo != "daiksud/gh-settings" or not SOURCE.fullmatch(source):
        raise Fatal("unexpected release repository or source")
    return repo, source


def output(name, value):
    with open(field("GITHUB_OUTPUT"), "a", encoding="utf-8") as stream:
        stream.write(f"{name}={value}\n")


def note(message, warning=False):
    safe = message.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
    level = "warning" if warning else "notice"
    print(f"::{level}::{safe}", flush=True)
    summary = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary:
        with open(summary, "a", encoding="utf-8") as stream:
            stream.write(f"- {'Warning' if warning else 'Notice'}: {message}\n")


def api_items(route, key):
    token = field("GH_TOKEN")
    if not token.strip():
        raise Fatal("missing Actions read token")
    try:
        proc = subprocess.run(
            ["gh", "api", "--paginate", "--slurp", route],
            capture_output=True, text=True, timeout=120, check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as error:
        raise Fatal("GitHub API is unavailable") from error
    if proc.returncode:
        raise Fatal(f"GitHub API failed while reading {key}")
    try:
        pages = json.loads(proc.stdout)
        if isinstance(pages, dict):
            pages = [pages]
        if not isinstance(pages, list) or not pages:
            raise ValueError("empty page list")
        values = []
        for page in pages:
            if not isinstance(page, dict) or not isinstance(page.get(key), list):
                raise ValueError("invalid page schema")
            values.extend(page[key])
        if not all(isinstance(item, dict) for item in values):
            raise ValueError("invalid resource schema")
        return values
    except (ValueError, TypeError) as error:
        raise Fatal(f"invalid GitHub API {key} response") from error


def select_ci_run(repo, source):
    query = urllib.parse.urlencode({
        "head_sha": source, "branch": "main", "event": "push", "per_page": 100
    })
    route = f"repos/{repo}/actions/workflows/push.yml/runs?{query}"
    candidates = api_items(route, "workflow_runs")
    if not candidates:
        return None
    for run in candidates:
        nested = run.get("repository")
        if (
            run.get("path") != ".github/workflows/push.yml"
            or run.get("event") != "push"
            or run.get("head_branch") != "main"
            or run.get("head_sha") != source
            or not isinstance(nested, dict) or nested.get("full_name") != repo
            or type(run.get("id")) is not int or run["id"] <= 0
            or type(run.get("run_number")) is not int
            or type(run.get("run_attempt")) is not int
            or run["run_number"] < 1 or run["run_attempt"] < 1
        ):
            raise Fatal("CI run provenance is incomplete or unexpected")
    latest = max(candidates, key=lambda r: (r["run_number"], r["id"]))
    if latest.get("status") != "completed" or latest.get("conclusion") != "success":
        raise Fatal(
            f"push run {latest['id']} attempt {latest['run_attempt']} is "
            f"{latest.get('status')}/{latest.get('conclusion')}; fix CI before releasing"
        )
    attempt_route = f"repos/{repo}/actions/runs/{latest['id']}/attempts/{latest['run_attempt']}"
    jobs = api_items(attempt_route + "/jobs?per_page=100", "jobs")
    deployments = [job for job in jobs if job.get("name") == "deployment"]
    if len(deployments) != 1 or deployments[0].get("status") != "completed" or deployments[0].get("conclusion") != "success":
        raise Fatal("required deployment did not complete successfully in the selected attempt")
    return latest


def run_identity(run):
    return (str(run["id"]), str(run["run_attempt"])) if run else ("", "")


def status(expected):
    repo, source = identity()
    run = select_ci_run(repo, source)
    if run_identity(run) != expected:
        raise Fatal("CI run or attempt changed since release resolution")
    if run:
        note(f"Verified successful push run {run['id']}, attempt {run['run_attempt']}")
    else:
        note("Confirmed no matching push run")


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, fp, code, msg, headers, newurl):
        return None


def read_limited(response):
    content = response.read(MAX_ARCHIVE_BYTES + 1)
    if len(content) > MAX_ARCHIVE_BYTES:
        raise Unusable("artifact archive exceeds allowed size")
    return content


def fetch_archive(repo, artifact_id, token):
    # Deliberately do NOT forward Authorization to the signed artifact URL.
    api_url = f"https://api.github.com/repos/{repo}/actions/artifacts/{artifact_id}/zip"
    request = urllib.request.Request(api_url, headers={
        "Authorization": "Bearer " + token,
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
    })
    try:
        with urllib.request.build_opener(NoRedirect()).open(request, timeout=30) as response:
            return read_limited(response)
    except urllib.error.HTTPError as error:
        if error.code in (404, 410):
            raise Unusable("artifact archive expired or disappeared") from error
        if error.code not in (301, 302, 303, 307, 308):
            raise Fatal(f"artifact API download failed (HTTP {error.code})") from error
        signed_url = error.headers.get("Location", "")
    except (OSError, urllib.error.URLError) as error:
        raise Fatal("artifact API transport failed") from error
    parsed = urllib.parse.urlsplit(signed_url)
    if parsed.scheme != "https" or not parsed.hostname:
        raise Fatal("artifact download redirect is not HTTPS")
    try:
        # Only the GitHub API receives the token; this request has no credential.
        with urllib.request.urlopen(signed_url, timeout=90) as response:
            return read_limited(response)
    except urllib.error.HTTPError as error:
        if error.code in (404, 410):
            raise Unusable("artifact archive expired before download") from error
        raise Fatal(f"artifact storage download failed (HTTP {error.code})") from error
    except (OSError, urllib.error.URLError) as error:
        raise Fatal("artifact storage transport failed") from error


def extract_product(data, target, directory):
    expected = {"release-manifest", "gh-settings-" + target}
    try:
        with zipfile.ZipFile(io.BytesIO(data)) as archive:
            infos = archive.infolist()
            if len(infos) != 2 or {part.filename for part in infos} != expected:
                raise Unusable(f"{target} archive has invalid inventory")
            directory.mkdir(parents=True, exist_ok=False)
            for part in infos:
                if part.is_dir() or (part.external_attr >> 16) & 0o170000 == 0o120000:
                    raise Unusable(f"{target} archive contains unsafe entries")
                limit = 16384 if part.filename == "release-manifest" else MAX_PRODUCT_BYTES
                if part.file_size > limit:
                    raise Unusable(f"{target} archive entry exceeds size limit")
                with archive.open(part) as stream:
                    content = stream.read(limit + 1)
                if len(content) > limit:
                    raise Unusable(f"{target} archive entry exceeds size limit")
                (directory / part.filename).write_bytes(content)
    except (zipfile.BadZipFile, EOFError, RuntimeError, ValueError, OSError) as error:
        raise Unusable(f"{target} archive could not be decoded") from error


def verify_product(target, version, source, directory):
    manifest = directory / "release-manifest"
    try:
        lines = manifest.read_text(encoding="utf-8").splitlines()
    except (OSError, UnicodeError) as error:
        raise Unusable(f"{target} manifest is unreadable") from error
    if len(lines) == 5 and lines[1].startswith("version="):
        observed = lines[1][len("version="):]
        if observed != version:
            raise Unusable(f"{target} version mismatch (expected {version}, observed {observed})")
    check_output = directory / ".verify-output"
    proc = subprocess.run(
        ["bash", ".github/scripts/release-artifact.sh", "verify",
         target, version, source, str(directory), str(check_output)],
        capture_output=True, text=True, timeout=45, check=False,
    )
    if proc.returncode:
        raise Unusable(f"{target} artifact manifest or binary SHA-256 is invalid")
    check_output.unlink(missing_ok=True)


def checked_artifact(items, run, source, description):
    if len(items) != 1:
        raise Unusable(f"{description} is missing or duplicated")
    item = items[0]
    if item.get("expired") is True:
        raise Unusable(f"{description} expired")
    if item.get("expired") is not False or type(item.get("id")) is not int or item["id"] < 1:
        raise Fatal(f"invalid {description} metadata")
    context = item.get("workflow_run")
    if context is not None and (
        not isinstance(context, dict) or context.get("id") != run["id"] or context.get("head_sha") != source
    ):
        raise Fatal(f"unexpected {description} producer")
    if not isinstance(item.get("digest"), str) or not DIGEST.fullmatch(item["digest"]):
        raise Unusable(f"{description} has no valid archive SHA-256 digest")
    return item


def checked_archive(repo, item):
    archive = fetch_archive(repo, item["id"], field("GH_TOKEN"))
    digest = "sha256:" + hashlib.sha256(archive).hexdigest()
    if digest != item["digest"]:
        raise Unusable(f"{item['name']} archive SHA-256 mismatch")
    return archive


def deployment_record(data):
    try:
        with zipfile.ZipFile(io.BytesIO(data)) as archive:
            if archive.namelist() != ["deployment.json"]:
                raise Unusable("invalid deployment archive inventory")
            info = archive.getinfo("deployment.json")
            if info.file_size > 16384 or (info.external_attr >> 16) & 0o170000 == 0o120000:
                raise Unusable("invalid deployment record entry")
            with archive.open(info) as stream:
                record = json.loads(stream.read(16385))
        if not isinstance(record, dict):
            raise Unusable("invalid deployment record")
        return record
    except (ValueError, UnicodeError, zipfile.BadZipFile, EOFError, RuntimeError, RecursionError) as error:
        raise Unusable("unreadable deployment record") from error


def deployed_products(repo, source, run, version=None, targets=TARGETS):
    items = api_items(f"repos/{repo}/actions/runs/{run['id']}/artifacts?per_page=100", "artifacts")
    name = f"deployment-{run['run_attempt']}"
    index = checked_artifact([item for item in items if item.get("name") == name], run, source, name)
    record = deployment_record(checked_archive(repo, index))
    expected = {
        "repository": repo, "workflow": ".github/workflows/push.yml", "event": "push",
        "source_sha": source, "run_id": run["id"], "run_attempt": run["run_attempt"],
    }
    if any(type(record.get(key)) is not type(value) or record[key] != value for key, value in expected.items()):
        raise Unusable("deployment identity differs from the selected source or run attempt")
    observed = record.get("version")
    if not isinstance(observed, str) or not VERSION.fullmatch(observed):
        raise Unusable("invalid deployed version")
    if version is not None and observed != version:
        raise Unusable(f"deployment version mismatch (expected {version}, observed {observed})")
    entries = record.get("artifacts")
    if not isinstance(entries, dict) or set(entries) != set(TARGETS):
        raise Unusable("incomplete deployment target inventory")
    products = {}
    for target in targets:
        entry = entries[target]
        if not isinstance(entry, dict) or type(entry.get("id")) is not int or entry["id"] < 1:
            raise Unusable(f"invalid deployed {target} ID")
        product = checked_artifact([item for item in items if item.get("id") == entry["id"]], run, source, target)
        producer = re.fullmatch(rf"candidate-{target}-([1-9][0-9]*)", str(product.get("name")))
        if not producer or int(producer[1]) > run["run_attempt"] or product["digest"] != entry.get("digest"):
            raise Unusable(f"{target} artifact differs from inspected deployment")
        products[target] = product
    return observed, products


def reuse(version):
    repo, source = identity()
    if not VERSION.fullmatch(version):
        raise Fatal("invalid intended release version")
    run = select_ci_run(repo, source)
    run_id, attempt = run_identity(run)
    output("ci-run-id", run_id)
    output("ci-run-attempt", attempt)
    if run is None:
        output("reused", "false")
        note("No push run; run full Screening and Inspection", True)
        return
    try:
        _, products = deployed_products(repo, source, run, version)
        with tempfile.TemporaryDirectory(dir=field("RUNNER_TEMP")) as stage:
            for target, item in products.items():
                directory = Path(stage) / target
                extract_product(checked_archive(repo, item), target, directory)
                verify_product(target, version, source, directory)
        output("linux-id", products["linux-amd64"]["id"])
        output("macos-id", products["darwin-arm64"]["id"])
        output("reused", "true")
        note(f"Verified inspected products from push run {run_id}, attempt {attempt}")
    except Unusable as error:
        output("reused", "false")
        note(f"Cannot reuse push run {run_id}, attempt {attempt}: {error}; full Screening and Inspection", True)


def probe():
    repo, source = identity()
    run = select_ci_run(repo, source)
    if not run:
        raise Fatal("the live probe requires a successful exact-SHA push deployment")
    version, products = deployed_products(repo, source, run, targets=("linux-amd64",))
    output("ci-run-id", run["id"])
    output("ci-run-attempt", run["run_attempt"])
    output("linux-id", products["linux-amd64"]["id"])
    output("version", version)


def main():
    args = sys.argv[1:]
    try:
        if len(args) == 3 and args[0] == "status":
            status(tuple(args[1:]))
        elif len(args) == 2 and args[0] == "reuse":
            reuse(args[1])
        elif args == ["probe"]:
            probe()
        else:
            raise Fatal("usage: release-ci.py status RUN_ID ATTEMPT | reuse VERSION | probe")
    except (Fatal, Unusable) as error:
        print(f"::error::{error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
