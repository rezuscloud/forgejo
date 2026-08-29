#!/usr/bin/env bash
set -euo pipefail

# Sync-lag guard for the vendored runner tree: compares runners/RUNNER_UPSTREAM
# against upstream tags on code.forgejo.org.
#
# Severity model (shared with check-runner-pristine.sh) — red means something
# automation should have already fixed:
#   RED  = a patch release in the vendored <major>.<minor> line is older than
#          the window ($LAG_DAYS, default 14) and the pin wasn't bumped —
#          patch-class bumps are auto per the bump policy matrix.
#   WARN = policy-gated manual territory: upstream minor/major releases ahead
#          (minor = manual review, major = evaluation). Reported, not blocking.
#
# Usage: hack/check-runner-lag.sh [LAG_DAYS]

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PIN_FILE="$ROOT/runners/RUNNER_UPSTREAM"
API="https://code.forgejo.org/api/v1/repos/forgejo/runner"
LAG_DAYS="${1:-${LAG_DAYS:-14}}"

PIN="$(tail -n1 "$PIN_FILE" | tr -d '[:space:]')"
[[ "$PIN" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "ERROR: pin '$PIN' is not a vX.Y.Z tag" >&2; exit 1; }

export PIN API LAG_DAYS
python3 <<'PY'
import json, os, re, sys, urllib.request
from datetime import datetime, timezone

pin, api, lag_days = os.environ["PIN"], os.environ["API"], int(os.environ["LAG_DAYS"])
tag_re = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")

def get(url):
    with urllib.request.urlopen(url, timeout=30) as r:
        return json.load(r)

# collect all semver tags (paginated)
tags = {}
page = 1
while True:
    batch = get(f"{api}/tags?limit=50&page={page}")
    for t in batch:
        m = tag_re.match(t["name"])
        if m:
            tags[t["name"]] = (tuple(map(int, m.groups())), t["commit"]["sha"])
    if len(batch) < 50:
        break
    page += 1
if not tags:
    print("ERROR: no vX.Y.Z tags found upstream — API change or network issue", file=sys.stderr)
    sys.exit(1)

pmajor, pminor, _ = next(v[0] for name, v in tags.items() if name == pin)
patch_latest = max((v for v in tags.values() if v[0][:2] == (pmajor, pminor)), default=None)
minor_ahead = sorted(v[0] for v in tags.values() if v[0][0] == pmajor and v[0][1] > pminor)
major_ahead = sorted(v[0] for v in tags.values() if v[0][0] > pmajor)

def age_days(sha):
    created = get(f"{api}/git/commits/{sha}")["created"]
    dt = datetime.fromisoformat(created.replace("Z", "+00:00"))
    return (datetime.now(timezone.utc) - dt).days

# warn tier: minor/major ahead — visible, non-blocking (policy matrix);
# aggregated to the latest release per class so upstream cadence doesn't spam
if minor_ahead:
    latest = "v" + ".".join(map(str, minor_ahead[-1]))
    n = len(minor_ahead)
    print(f"WARN: upstream minor releases ahead of pin {pin}: latest {latest} ({n} tag(s)) — policy: minor bump = manual review")
if major_ahead:
    latest = "v" + ".".join(map(str, major_ahead[-1]))
    n = len(major_ahead)
    print(f"WARN: upstream major releases ahead of pin {pin}: latest {latest} ({n} tag(s)) — policy: major bump = evaluation")

# red tier: patch-lane lag beyond the window
patch_ver = "v" + ".".join(map(str, patch_latest[0])) if patch_latest else None
if patch_ver == pin:
    print(f"✓ pin {pin} is the latest patch of the v{pmajor}.{pminor} line")
elif patch_ver is not None:
    age = age_days(patch_latest[1])
    if age > lag_days:
        print(f"RED: v{pmajor}.{pminor} line has patch {patch_ver} ({age}d old, window {lag_days}d) "
              f"and pin is {pin} — patch bumps are auto", file=sys.stderr)
        print(f"  fix: hack/sync-runner.sh {patch_ver}", file=sys.stderr)
        sys.exit(1)
    print(f"✓ patch lag within window: {patch_ver} released {age}d ago (window {lag_days}d), pin {pin}")
else:
    print(f"✓ pin {pin}: no patch releases in the v{pmajor}.{pminor} line")
PY
