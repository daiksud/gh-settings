#!/usr/bin/env python3
"""Read a complete candidate identity without trusting manifest paths."""
import json
import re
import sys

with open(sys.argv[1], encoding="utf-8") as stream:
    value = json.load(stream)
if (not isinstance(value, dict)
        or value.get("source_sha") != sys.argv[2]
        or value.get("target") != sys.argv[3]
        or value.get("binary_name") != "gh-settings"
        or not isinstance(value.get("binary_sha256"), str)
        or not re.fullmatch(r"[0-9a-f]{64}", value["binary_sha256"])
        or not isinstance(value.get("product_version"), str)
        or not re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", value["product_version"])):
    sys.exit("invalid candidate manifest identity")
print(value["binary_sha256"], value["product_version"])
