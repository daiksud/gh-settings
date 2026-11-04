#!/usr/bin/env python3
"""Fail-closed main CI gate and bounded reuse of its exact release artifacts."""

import hashlib
import io
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
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
    route = f"repos/{repo}/actions/workflows/ci-push.yml/runs?{query}"
    candidates = api_items(route, "workflow_runs")
    if not candidates:
        return None
    for run in candidates:
        nested = run.get("repository")
        if (
            run.get("path") != ".github/workflows/ci-push.yml"
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
            f"ci-push run {latest['id']} attempt {latest['run_attempt']} is "
            f"{latest.get('status')}/{latest.get('conclusion')}; fix CI before releasing"
        )
    return latest


def status(expected=""):
    repo, source = identity()
    latest = select_ci_run(repo, source)
    if expected and (latest is None or str(latest["id"]) != expected):
        raise Fatal("CI run changed since release resolution")
    if latest is None:
        if expected:
            raise Fatal("previously selected CI run is unavailable")
        output("ci-run-id", "")
        note("No matching ci-push run; fresh Screening and Inspection are required")
    else:
        output("ci-run-id", str(latest["id"]))
        note(f"Verified successful ci-push run {latest['id']}, attempt {latest['run_attempt']}")


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


def reuse(run_id, version):
    repo, source = identity()
    if not VERSION.fullmatch(version):
        raise Fatal("invalid intended release version")
    if not run_id:
        output("reused", "false")
        note("No CI artifacts to reuse; run full Screening and Inspection")
        return
    if not run_id.isdecimal() or int(run_id) < 1:
        raise Fatal("invalid source run ID")
    token = field("GH_TOKEN")
    # Recheck provenance, not merely a run-id supplied by a different job.
    current = select_ci_run(repo, source)
    if current is None or str(current["id"]) != run_id:
        raise Fatal("CI run changed before artifact reuse")
    route = f"repos/{repo}/actions/runs/{run_id}/artifacts?per_page=100"
    artifacts = api_items(route, "artifacts")
    base = Path(field("RUNNER_TEMP")) / "reused-release"
    if base.exists():
        raise Fatal("artifact staging directory already exists")
    base.mkdir(parents=True)
    try:
        for target in TARGETS:
            name = "release-" + target
            matches = [item for item in artifacts if item.get("name") == name]
            if len(matches) != 1:
                raise Unusable(f"{name} is missing or duplicated in ci-push run {run_id}")
            item = matches[0]
            if item.get("expired") is True:
                raise Unusable(f"{name} expired in ci-push run {run_id}")
            if item.get("expired") is not False:
                raise Fatal(f"{name} expiration metadata is invalid")
            digest = item.get("digest")
            match = DIGEST.fullmatch(digest) if isinstance(digest, str) else None
            if match is None:
                raise Unusable(f"{name} has no valid archive SHA-256 digest")
            if type(item.get("id")) is not int or item["id"] <= 0:
                raise Fatal(f"{name} has invalid artifact ID")
            context = item.get("workflow_run")
            if context is not None and (
                not isinstance(context, dict)
                or context.get("id") != int(run_id)
                or context.get("head_sha") != source
            ):
                raise Fatal(f"{name} has unexpected producer provenance")
            archive = fetch_archive(repo, item["id"], token)
            observed = hashlib.sha256(archive).hexdigest()
            if observed != match.group(1):
                raise Unusable(
                    f"{name} archive SHA-256 mismatch "
                    f"(expected {match.group(1)}, observed {observed})"
                )
            target_dir = base / target
            extract_product(archive, target, target_dir)
            verify_product(target, version, source, target_dir)
        output("reused", "true")
        note(f"Reusing verified Linux and macOS artifacts from ci-push run {run_id}")
    except Unusable as error:
        shutil.rmtree(base, ignore_errors=True)
        output("reused", "false")
        note(f"Cannot reuse ci-push run {run_id}: {error}; full Screening and Inspection", True)


def main():
    try:
        if len(sys.argv) >= 2 and sys.argv[1] == "status" and len(sys.argv) in (2, 3):
            status(sys.argv[2] if len(sys.argv) == 3 else "")
        elif len(sys.argv) == 4 and sys.argv[1] == "reuse":
            reuse(sys.argv[2], sys.argv[3])
        else:
            raise Fatal("usage: release-ci.py status [EXPECTED_RUN_ID] | reuse RUN_ID VERSION")
    except Fatal as error:
        print(f"::error::{error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
