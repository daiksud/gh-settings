"""Bounded live Topics probe; only the selected root Action performs writes."""

import json
import os
from pathlib import Path
import re
import subprocess
import sys


TARGET = "daiksud/test-gh-settings-public"
MARKER = "gh-settings-854-public-action"


def command(args, cwd=None):
    environment = os.environ.copy()
    environment.pop("GH_DEBUG", None)
    environment.pop("GITHUB_TOKEN", None)
    if args[0] != "gh":
        environment.pop("GH_TOKEN", None)
    result = subprocess.run(args, cwd=cwd, env=environment, capture_output=True, text=True, timeout=60)
    if result.returncode:
        # Never relay credential-bearing subprocess diagnostics.
        raise RuntimeError(f"{args[0]} failed; inspect the bounded probe stage before retrying")
    return result.stdout


def api(path):
    return json.loads(command(["gh", "api", "--hostname", "github.com", f"repos/{TARGET}{path}"]))


def observe():
    repo = api("")
    owner = repo.get("owner", {})
    if (repo.get("full_name") != TARGET or repo.get("private") is not False
            or repo.get("archived") is not False or owner.get("login") != "daiksud"
            or owner.get("type") != "User" or type(repo.get("id")) is not int
            or repo["id"] <= 0 or type(owner.get("id")) is not int or owner["id"] <= 0):
        raise RuntimeError("target is not the expected active personal public repository")
    topics = api("/topics").get("names")
    if (not isinstance(topics, list) or len(topics) > 20
            or any(not isinstance(t, str) or re.fullmatch(r"[a-z0-9-]{1,50}", t) is None for t in topics)
            or len(set(topics)) != len(topics)):
        raise RuntimeError("complete valid Topics observation required")
    return {"repository_id": repo["id"], "owner_id": owner["id"]}, sorted(topics)


class Probe:
    def __init__(self):
        self.workspace = Path(os.environ["GITHUB_WORKSPACE"])
        self.directory = Path(os.environ["RUNNER_TEMP"]) / "public-action-topics"
        self.path = self.directory / "state.json"

    def save(self, state):
        self.directory.mkdir(mode=0o700, parents=True, exist_ok=True)
        self.path.write_text(json.dumps(state) + "\n")
        self.path.chmod(0o600)

    def load(self):
        return json.loads(self.path.read_text())

    def declaration(self, topics):
        directory = self.workspace / ".github"
        directory.mkdir(exist_ok=True)
        (directory / "settings.yml").write_text(
            "version: daiksud/gh-settings@v0\nrepository:\n  about:\n    topics: " + json.dumps(topics) + "\n"
        )

    def prepare(self):
        if self.path.exists() or (self.workspace / ".git").exists() or (self.workspace / ".github").exists():
            raise RuntimeError("fresh consumer workspace and probe state required")
        identity, original = observe()
        if MARKER in original or len(original) >= 20:
            raise RuntimeError("probe topic must be absent and one topic slot must be available")
        state = {"identity": identity, "original": original, "desired": sorted(original + [MARKER]), "armed": False}
        self.save(state)
        self.declaration(state["desired"])
        command(["git", "init", "--quiet"], self.workspace)
        command(["git", "remote", "add", "origin", f"https://github.com/{TARGET}.git"], self.workspace)
        command(["git", "add", ".github/settings.yml"], self.workspace)
        command(["git", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null",
                 "-c", "user.name=Daiki Sudo", "-c", "user.email=daiksud@proton.me",
                 "commit", "--quiet", "-m", "test: prepare isolated Topics consumer"], self.workspace)
        print(f"target={TARGET}; original_topics={json.dumps(original)}; desired_topics={json.dumps(state['desired'])}")
        print("consumer_sha=" + command(["git", "rev-parse", "HEAD"], self.workspace).strip())

    def check(self, expected):
        state = self.load()
        identity, current = observe()
        if identity != state["identity"] or current != state[expected]:
            raise RuntimeError(f"independent {expected} readback mismatch or target identity drift; no automatic overwrite")
        print(f"independent_readback={expected}; topics={json.dumps(current)}")
        return state

    def arm(self):
        version = os.environ.get("EXPECTED_CLI_VERSION", "")
        digest = os.environ.get("EXPECTED_CLI_SHA256", "")
        if (re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", version) is None
                or re.fullmatch(r"[0-9a-f]{64}", digest) is None
                or os.environ.get("ACTION_TARGET") != TARGET
                or os.environ.get("ACTION_VERSION") != version
                or os.environ.get("ACTION_DIGEST") != digest):
            raise RuntimeError("installed Action identity differs from the verified product")
        state = self.check("original")
        state["armed"] = True
        self.save(state)

    def restore_config(self):
        state = self.load()
        if not state["armed"]:
            print("no apply was armed; no restoration requested")
            return
        identity, current = observe()
        if identity != state["identity"] or current not in (state["original"], state["desired"]):
            raise RuntimeError("restoration blocked by concurrent change; inspect original_topics in prepare log")
        self.declaration(state["original"])
        with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
            output.write("ready=true\n")
        print("original Topics declaration restored locally; root Action restoration still required")


def main():
    probe = Probe()
    mode = sys.argv[1]
    if mode == "prepare":
        probe.prepare()
    elif mode == "arm":
        probe.arm()
    elif mode == "verify-desired":
        probe.check("desired")
    elif mode == "verify-repeat":
        if os.environ.get("PLAN_OUTCOME") != "no-difference":
            raise RuntimeError("repeat plan must report no-difference")
        probe.check("desired")
    elif mode == "restore-config":
        probe.restore_config()
    elif mode == "verify-restored":
        probe.check("original")
        print("RESTORED: independently observed original Topics")
    elif mode == "verify-final-plan":
        if os.environ.get("PLAN_OUTCOME") != "no-difference":
            raise RuntimeError("restored plan must report no-difference")
        probe.check("original")
    else:
        raise RuntimeError("unknown probe stage")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, ValueError, OSError, subprocess.TimeoutExpired) as error:
        print(f"Topics probe failed: {error}", file=sys.stderr)
        sys.exit(1)
