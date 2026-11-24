#!/usr/bin/env python3
"""Select a release version from a frozen Git commit and published GitHub releases.

Repository-specific release matrices, artifacts, publication, and verification
remain in their own workflows.
"""

import json
import os
import re
import subprocess
import sys
from datetime import datetime

VERSION = re.compile(r"^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")
SHA = re.compile(r"^[0-9a-f]{40}$")
REPOSITORY = re.compile(r"^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$")
HEADER = re.compile(r"^([A-Za-z][A-Za-z0-9-]*)(?:\(([^()]+)\))?(!)?: .+$")
FOOTER = re.compile(r"^([A-Za-z][A-Za-z0-9-]*|BREAKING CHANGE)(: | #)(.*)$")
FENCE = re.compile(r"^ {0,3}(`{3,}|~{3,})")
HASH = re.compile(r"^[0-9a-f]{64}$")
MAX_COMPONENT = (1 << 64) - 1


class ReleaseError(Exception):
    """Invalid history, published metadata, or release candidate."""


def command(*args):
    result = subprocess.run(args, capture_output=True, check=False)
    if result.returncode:
        raise ReleaseError(f"{args[0]} {args[1]} failed: {result.stderr.decode('utf-8', 'replace').strip()}")
    return result.stdout


def git(*args):
    return command("git", "-c", "log.showSignature=false", *args).decode("utf-8")


def read_releases(repository):
    response = command("gh", "api", f"repos/{repository}/releases?per_page=100", "--paginate", "--slurp")
    try:
        pages = json.loads(response)
    except (ValueError, UnicodeError) as exc:
        raise ReleaseError("invalid GitHub release catalog") from exc
    if not isinstance(pages, list) or not pages or any(not isinstance(page, list) for page in pages):
        raise ReleaseError("missing or invalid GitHub release catalog")
    releases = []
    seen = set()
    for page in pages:
        for release in page:
            if not isinstance(release, dict):
                raise ReleaseError("invalid release record")
            rid = release.get("id")
            tag = release.get("tag_name")
            draft = release.get("draft")
            prerelease = release.get("prerelease")
            published = release.get("published_at")
            if (type(rid) is not int or rid <= 0 or rid in seen or
                    not isinstance(tag, str) or not tag or
                    type(draft) is not bool or type(prerelease) is not bool or
                    (not draft and not isinstance(published, str))):
                raise ReleaseError("incomplete or duplicate GitHub release metadata")
            seen.add(rid)
            if not draft:
                try:
                    datetime.fromisoformat(published.replace("Z", "+00:00"))
                except ValueError as exc:
                    raise ReleaseError("invalid publication timestamp") from exc
            releases.append(release)
    return releases


def resolve_tag(tag):
    return git("rev-parse", "--verify", f"refs/tags/{tag}^{{commit}}").strip()


def existing_retry(source, releases):
    """Match an immutable gh-settings release, leaving asset checks to its publisher."""
    matches = []
    for release in releases:
        tag = release["tag_name"]
        if not VERSION.fullmatch(tag) or release["prerelease"]:
            continue
        body = release.get("body")
        if not isinstance(body, str) or not body.startswith("<!-- gh-settings-release: "):
            continue
        marker = body.split("\n", 1)[0]
        if not marker.endswith(" -->"):
            continue
        try:
            identity = json.loads(marker[len("<!-- gh-settings-release: "):-len(" -->")])
        except ValueError:
            continue
        if not isinstance(identity, dict) or identity.get("version") != tag or identity.get("commit") != source:
            continue
        if not all(isinstance(identity.get(key), str) and HASH.fullmatch(identity[key]) for key in
                   ("macos_sha256", "linux_sha256", "checksums_sha256")):
            continue
        if resolve_tag(tag) != source:
            raise ReleaseError("existing release tag does not match its source")
        matches.append(tag)
    if len(matches) > 1:
        raise ReleaseError("multiple existing releases identify the same source")
    return matches[0] if matches else None


def classify(message):
    if not message.strip():
        raise ReleaseError("empty commit message")
    lines = message.replace("\r\n", "\n").split("\n")
    header = HEADER.fullmatch(lines[0])
    breaking = bool(header and header.group(3))
    minor = bool(header and header.group(1).lower() in ("feat", "perf"))
    # A trailer must start after a blank line or another trailer, not inside
    # prose or fenced examples. This mirrors the reviewed gh-settings parser.
    in_trailers = False
    blank = False
    fence = ""
    for line in lines[1:]:
        match = FENCE.match(line)
        if fence:
            if (match and match.group(1)[0] == fence[0] and
                    len(match.group(1)) >= len(fence) and not line[match.end():].strip()):
                fence = ""
        elif match:
            fence = match.group(1)
        else:
            footer = FOOTER.fullmatch(line)
            if footer and (blank or in_trailers):
                in_trailers = True
                if footer.group(2) == ": ":
                    breaking |= footer.group(1) in ("BREAKING CHANGE", "BREAKING-CHANGE")
                    minor |= footer.group(1) == "DEPRECATED"
                blank = False
                continue
        blank = not line.strip()
    return breaking, minor


def trees_differ(previous, source, *paths):
    result = subprocess.run(["git", "diff", "--quiet", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none",
                             previous, source, "--", *paths], check=False)
    if result.returncode not in (0, 1):
        raise ReleaseError(f"cannot compare the committed trees {previous} and {source}")
    return result.returncode == 1


def version_components(version):
    if not VERSION.fullmatch(version):
        raise ReleaseError("invalid published stable version")
    numbers = tuple(int(number) for number in version[1:].split("."))
    if any(number > MAX_COMPONENT for number in numbers):
        raise ReleaseError("version component exceeds supported range")
    return numbers


def next_version(version, tier, graduate_to_v1=False):
    numbers = list(version_components(version))
    if graduate_to_v1 and numbers[0] == 0:
        return "v1.0.1"
    if tier == 2 and numbers[0] == 0:
        tier = 1  # Initial v1 promotion requires explicit opt-in.
    index = 2 - tier
    if numbers[index] == MAX_COMPONENT:
        raise ReleaseError("version component would overflow")
    numbers[index] += 1
    numbers[index + 1:] = [0] * (2 - index)
    return "v" + ".".join(map(str, numbers))


def calculate(source, releases, allow_retry=False, graduate_to_v1=False):
    if not SHA.fullmatch(source):
        raise ReleaseError("source must be an immutable full commit SHA")
    if git("rev-parse", "--is-shallow-repository").strip() != "false":
        raise ReleaseError("complete Git history is required")
    if git("rev-parse", "--verify", f"{source}^{{commit}}").strip() != source:
        raise ReleaseError("source must identify a commit directly")
    if allow_retry:
        version = existing_retry(source, releases)
        if version:
            if graduate_to_v1 and version_components(version)[0] == 0:
                raise ReleaseError("cannot promote an existing v0 release at the same source; use a new substantive source commit")
            return version, "", "", False

    candidates = [release for release in releases if
                  not release["draft"] and not release["prerelease"] and
                  VERSION.fullmatch(release["tag_name"])]
    if not candidates:
        raise ReleaseError("initial release requires a separately chosen baseline")
    latest = max(candidates, key=lambda release: (release["published_at"], release["id"]))
    previous_tag = latest["tag_name"]
    previous_major, _, _ = version_components(previous_tag)
    previous_sha = resolve_tag(previous_tag)
    if not SHA.fullmatch(previous_sha):
        raise ReleaseError("previous release does not resolve to a commit")
    if subprocess.run(["git", "merge-base", "--is-ancestor", previous_sha, source], check=False).returncode:
        raise ReleaseError("previous published release is not an ancestor of source")

    logs = command("git", "-c", "log.showSignature=false", "log", "--reverse", "--topo-order", "--no-decorate",
                   "--no-notes", "--no-color", "--encoding=UTF-8", "-z", "--format=%H%x00%B",
                   f"{previous_sha}..{source}", "--")
    fields = logs.removesuffix(b"\x00").split(b"\x00") if logs else []
    if len(fields) % 2:
        raise ReleaseError("invalid Git log record")
    tier = -1
    for i in range(0, len(fields), 2):
        sha = fields[i].decode("ascii")
        if not SHA.fullmatch(sha):
            raise ReleaseError("invalid commit in Git log")
        # Ignore empty commits even when their messages carry release signals.
        parents = git("rev-list", "--parents", "-n", "1", sha).split()
        if len(parents) == 2 and not trees_differ(parents[1], sha):
            continue
        try:
            breaking, minor = classify(fields[i + 1].decode("utf-8"))
        except ReleaseError as error:
            raise ReleaseError(f"cannot classify commit {sha}: {error}") from error
        tier = max(tier, 2 if breaking else 1 if minor else 0)
    if tier < 0:
        if allow_retry and source == previous_sha:
            raise ReleaseError("existing release at this source lacks a valid immutable identity")
        return previous_tag, previous_tag, previous_sha, False
    compare_binaries = False
    promoting = graduate_to_v1 and previous_major == 0
    if tier == 0 and not promoting:
        if not trees_differ(previous_sha, source):
            return previous_tag, previous_tag, previous_sha, False
        # The root Action and its runtime files ship under the same tag as the CLI.
        # Without a net Action change, the two native builds decide PATCH.
        compare_binaries = not trees_differ(previous_sha, source, "action.yml", "action/",
                                            ":(exclude)action/README.md")
    version = next_version(previous_tag, tier, graduate_to_v1)
    if graduate_to_v1 and any(
            version_components(version) <= version_components(release["tag_name"])
            for release in candidates):
        raise ReleaseError("promotion must advance beyond every published stable version")
    return version, previous_tag, previous_sha, compare_binaries


def main():
    try:
        options = sys.argv[1:]
        allowed = {"--allow-retry", "--graduate-to-v1"}
        if len(options) != len(set(options)) or any(option not in allowed for option in options):
            raise ReleaseError("usage: release-version.py [--allow-retry] [--graduate-to-v1]")
        allow_retry = "--allow-retry" in options
        graduate_to_v1 = "--graduate-to-v1" in options
        repository = os.environ.get("GITHUB_REPOSITORY", "")
        source = os.environ.get("SOURCE_SHA", "")
        if not REPOSITORY.fullmatch(repository):
            raise ReleaseError("invalid repository identity")
        version, previous_tag, previous_sha, compare_binaries = calculate(
            source, read_releases(repository), allow_retry, graduate_to_v1)
        version_components(version)
        if previous_tag or previous_sha:
            version_components(previous_tag)
            if not SHA.fullmatch(previous_sha):
                raise ReleaseError("invalid resolved predecessor source")
        release_needed = "false" if version == previous_tag else "true"
        with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
            print(f"version={version}\nprevious-tag={previous_tag}\nprevious-sha={previous_sha}\n"
                  f"release-needed={release_needed}\ncompare-binaries={str(compare_binaries).lower()}", file=output)
    except (ReleaseError, KeyError, OSError, UnicodeError, ValueError, OverflowError) as error:
        print(f"release-version: {error}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
