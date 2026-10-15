#!/usr/bin/env python3
"""Resolve nonsecret checkout and already-verified CLI identity for composite steps."""

import hashlib
import os
from pathlib import Path
import subprocess
import sys
from urllib.parse import urlsplit


def git(*args):
    return subprocess.check_output(
        ["git", *args], cwd=os.environ["GITHUB_WORKSPACE"], text=True
    )


def metadata(binary):
    entries = git("config", "--null", "--show-scope", "--get-all", "remote.origin.url").split("\0")
    origin = next(value.strip() for scope, value in zip(entries[::2], entries[1::2]) if scope in ("local", "worktree"))
    url = urlsplit(origin)
    owner, repository = url.path.strip("/").removesuffix(".git").split("/")
    if url.scheme != "https" or url.hostname != "github.com":
        raise ValueError("a checkout-scoped GitHub origin is required")
    version = subprocess.check_output([binary, "--version"], text=True).strip().removeprefix("gh-settings ")
    return {
        "config-sha": git("rev-parse", "HEAD").strip(),
        "target": f"{owner}/{repository}",
        "cli-version": version,
        "cli-sha256": hashlib.sha256(Path(binary).read_bytes()).hexdigest(),
    }


if __name__ == "__main__":
    try:
        values = metadata(sys.argv[1])
        for key, value in values.items():
            print(f"{key}={value}")
    except (OSError, ValueError, KeyError, StopIteration, subprocess.CalledProcessError):
        raise SystemExit("Could not resolve checkout or CLI metadata.")
