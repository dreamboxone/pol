#!/usr/bin/env python3
"""Resolve package targets from the architecture catalog."""
import json
import pathlib
import sys

targets = json.loads((pathlib.Path(__file__).resolve().parent.parent / "cmd/pol/architectures.json").read_text())
if len(sys.argv) == 1:
    print(json.dumps(list(targets)))
else:
    target = targets[sys.argv[1]]
    print(target["goarch"], target["goarm"])
