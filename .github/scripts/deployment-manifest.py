#!/usr/bin/env python3
"""Record the immutable archives that passed the push workflow's inspection."""

import json
import os
import re
import sys


def artifact(platform):
    artifact_id = os.environ[platform + "_ID"]
    digest = os.environ[platform + "_DIGEST"].removeprefix("sha256:")
    if not re.fullmatch(r"[1-9][0-9]*", artifact_id) or not re.fullmatch(r"[0-9a-f]{64}", digest):
        sys.exit(f"missing or invalid {platform} artifact identity")
    return {"id": int(artifact_id), "digest": "sha256:" + digest}


def main():
    linux, macos = artifact("LINUX"), artifact("MACOS")
    if linux["id"] == macos["id"]:
        sys.exit("expected two distinct native artifacts")
    record = {
        "repository": os.environ["GITHUB_REPOSITORY"],
        "workflow": ".github/workflows/push.yml",
        "event": os.environ["GITHUB_EVENT_NAME"],
        "source_sha": os.environ["GITHUB_SHA"],
        "version": os.environ["VERSION"],
        "run_id": int(os.environ["GITHUB_RUN_ID"]),
        "run_attempt": int(os.environ["GITHUB_RUN_ATTEMPT"]),
        "artifacts": {"linux-amd64": linux, "darwin-arm64": macos},
    }
    print(json.dumps(record, indent=2))


if __name__ == "__main__":
    main()
