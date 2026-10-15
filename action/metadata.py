#!/usr/bin/env python3
"""Resolve nonsecret checkout and already-verified CLI identity for composite steps."""

import hashlib
import os
from pathlib import Path
import subprocess
import sys
from urllib.parse import unquote, urlsplit


def git(*args):
    return subprocess.check_output(
        ["git", *args], cwd=os.environ["GITHUB_WORKSPACE"], text=True
    )


def repository(origin):
    if origin.startswith(("git@github.com:", "github.com:")):
        origin = "ssh://" + origin.replace(":", "/", 1)
    if not origin.startswith(("https://", "ssh://")):
        raise ValueError("origin must use a supported literal protocol prefix")
    url = urlsplit(origin)
    # Match go-gh's literal www. removal before lowercasing, not urlsplit.hostname.
    host = url.netloc.rsplit("@", 1)[-1].split(":", 1)[0]
    if host.removeprefix("www.").lower() != "github.com":
        raise ValueError("a checkout-scoped GitHub origin is required")
    owner, name = unquote(url.path).strip("/").split("/")
    return f"{owner}/{name.removesuffix('.git')}"


def rewritten_origin(origin):
    try:
        rules = git("config", "--null", "--get-regexp", r"^url\..*\.insteadof$")
    except subprocess.CalledProcessError as error:
        if error.returncode != 1:
            raise
        rules = ""
    rewritten, longest = origin, -1
    for entry in rules.split("\0"):
        key, separator, prefix = entry.partition("\n")
        if separator and origin.startswith(prefix) and len(prefix) > longest:
            rewritten = key.removeprefix("url.").removesuffix(".insteadof") + origin[len(prefix):]
            longest = len(prefix)
    return rewritten


def metadata(binary):
    entries = git("config", "--null", "--show-scope", "--get-all", "remote.origin.url").split("\0")
    origin = next(value.strip() for scope, value in zip(entries[::2], entries[1::2]) if scope in ("local", "worktree"))
    try:
        target = repository(origin)
    except ValueError:
        # Preserve a configured GitHub target before transport rewrites, as the CLI does.
        target = repository(rewritten_origin(origin))
    version = subprocess.check_output([binary, "--version"], text=True).strip().removeprefix("gh-settings ")
    return {
        "config-sha": git("rev-parse", "HEAD").strip(),
        "target": target,
        "cli-version": version,
        "cli-sha256": hashlib.sha256(Path(binary).read_bytes()).hexdigest(),
    }


if __name__ == "__main__":
    try:
        values = metadata(sys.argv[1])
        if any("\r" in value or "\n" in value for value in values.values()):
            raise ValueError("metadata outputs must be single-line values")
        for key, value in values.items():
            print(f"{key}={value}")
    except (OSError, ValueError, KeyError, StopIteration, subprocess.CalledProcessError):
        raise SystemExit("Could not resolve checkout or CLI metadata.")
