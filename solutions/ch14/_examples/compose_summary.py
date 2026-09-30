"""Prints one line per service of `docker compose config --format json`:
which image stage it builds, how many replicas, and its health check."""

import json
import sys

config = json.load(sys.stdin)
for name, svc in config["services"].items():
    build = svc.get("build", {}).get("target") or svc.get("image", "-")
    replicas = svc.get("deploy", {}).get("replicas", 1)
    test = svc.get("healthcheck", {}).get("test", [])
    check = " ".join(test[1:]) if test else "-"
    deps = ",".join(svc.get("depends_on", {})) or "-"
    print(f"{name:12} from={build:12} replicas={replicas} waits-for={deps}\n{'':13}health: {check}")
