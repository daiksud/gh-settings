#!/usr/bin/env python3
"""Offline release API responses and payload assertions for release.sh."""
import json
import os
from pathlib import Path
import sys

root = Path(os.environ.get("MOCK_ROOT", "."))
command = sys.argv[1]
if command == "release":
    assets = [{"name": p.name} for p in (root / "assets").iterdir() if p.is_file()]
    if os.environ.get("MOCK_DUPLICATE_ASSET") == "1" and assets:
        assets += [assets[0]]
    tag = os.environ["VERSION"]
    print(json.dumps(dict(id=123, tag_name=tag, name=tag, prerelease=False,
        draft="unknown" if os.environ.get("MOCK_INVALID_DRAFT") == "1" else (root / "draft").read_text() == "true\n",
        body=(root / "body").read_text(), assets=assets)))
elif command == "create":
    value = json.loads(Path(sys.argv[2]).read_text())
    assert isinstance(value.get("body"), str)
    assert value == dict(tag_name=os.environ["VERSION"], target_commitish=os.environ["SOURCE_SHA"], name=os.environ["VERSION"], body=value["body"], draft=True, prerelease=False)
    (root / "body").write_text(value["body"] + "\n")
    (root / "draft").write_text("true\n")
elif command == "catalog":
    value = json.loads((root / "catalog.json").read_text())
    value[0] += [json.loads((root / "current.json").read_text())]
    print(json.dumps(value))
elif command == "predecessor":
    value = json.loads(Path(sys.argv[2]).read_text())
    value[0][0]["tag_name"] = "v1.2.1"
    print(json.dumps(value))
else:
    sys.exit("unknown fixture command")
