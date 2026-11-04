#!/usr/bin/env python3
"""Run actionlint with a narrow bridge for its missing same-repo syntax support."""

import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
WORKFLOWS = {ROOT / '.github/workflows' / f'{name}.yml'
             for name in ('ci-pr', 'ci-push', 'release')}
BRIDGE = '$/.github/actions/lock-workspace-dependencies'
SCREENING_BRIDGE = '$/.github/actions/lock-workspace-dependencies-screening'
IF_LINE = 'if: ${{ false }}'
ACTION_MESSAGE = ('specifying action "{bridge}" in invalid format because ref is missing. '
                  'available formats are "{owner}/{repo}@{ref}" or '
                  '"{owner}/{repo}/{path}@{ref}"')
IF_MESSAGE = 'constant expression "false" in condition. remove the if: section'


def known_bridge_diagnostic(diagnostic):
    """Match the exact diagnostic at the literal never-run bridge, not other errors."""
    if not isinstance(diagnostic, dict):
        return False
    filename, line = diagnostic.get('filepath'), diagnostic.get('line')
    if not isinstance(filename, str) or type(line) is not int:
        return False
    path = Path(filename).resolve()
    if path not in WORKFLOWS:
        return False
    try:
        lines = path.read_text().splitlines()
    except (OSError, UnicodeError):
        return False
    bridge = SCREENING_BRIDGE if path.name == 'ci-pr.yml' else BRIDGE
    kind, message = diagnostic.get('kind'), diagnostic.get('message')
    if kind == 'action' and message == ACTION_MESSAGE.format(bridge=bridge):
        start = line - 2
    elif kind == 'if-cond' and message == IF_MESSAGE:
        start = line - 1
    else:
        return False
    return (0 <= start < len(lines) - 1
            and lines[start].strip() == IF_LINE
            and lines[start + 1].strip() == f'uses: {bridge}')


def main():
    try:
        result = subprocess.run(
            ['actionlint', '-format', '{{json .}}', '-shellcheck', 'shellcheck', *sys.argv[1:]],
            capture_output=True, text=True, check=False)
    except OSError as error:
        print(f'actionlint could not run: {error}', file=sys.stderr)
        return 1
    # Internal/tool errors must not be mistaken for the two known syntax diagnostics.
    if result.returncode not in (0, 1) or result.stderr:
        print(result.stderr or result.stdout or 'actionlint failed', file=sys.stderr)
        return 1
    try:
        diagnostics = json.loads(result.stdout)
        if not isinstance(diagnostics, list) or any(not isinstance(d, dict) for d in diagnostics):
            raise ValueError('expected a diagnostic array')
    except (ValueError, TypeError) as error:
        print(f'invalid actionlint output: {error}', file=sys.stderr)
        return 1
    if result.returncode and not diagnostics:
        print('actionlint failed without diagnostics', file=sys.stderr)
        return 1
    remaining = [d for d in diagnostics if not known_bridge_diagnostic(d)]
    for diagnostic in remaining:
        print(f"{diagnostic.get('filepath')}:{diagnostic.get('line')}:{diagnostic.get('column')}: "
              f"{diagnostic.get('message')} [{diagnostic.get('kind')}]", file=sys.stderr)
    return 1 if remaining else 0


if __name__ == '__main__':
    sys.exit(main())
