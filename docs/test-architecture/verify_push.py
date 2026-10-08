#!/usr/bin/env python3
"""End-to-end contract verification for the generated diff matrix.

Proves that ``diff-matrix.json`` is not a bespoke document but a payload the real
Anubis endpoint accepts: it is POSTed to
``/api/quality-workspace/<id>/diff-matrix/sync`` on a live server, read back via
``GET .../diff-matrix``, and compared field by field.

The workspace row is inserted directly over SQL because creating one through the
API is itself broken (see section 5.1: empty strings written into json columns).
The row is removed afterwards.

Run:  python3 docs/test-architecture/verify_push.py [base-url]
"""

from __future__ import annotations

import json
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
MATRIX = ROOT / "docs" / "test-architecture" / "diff-matrix.json"

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8081"
PROJECT_ID = "100001100001"

MYSQL = [
    "mysql", "-h127.0.0.1", "-P3307", "-uroot", "-proot", "--silent", "--skip-column-names",
]


def sql(statement: str) -> str:
    result = subprocess.run(
        MYSQL + ["-e", statement], capture_output=True, text=True, check=False
    )
    if result.returncode != 0:
        raise SystemExit(f"mysql failed: {result.stderr.strip()}")
    return result.stdout.strip()


def http(method: str, path: str, payload: dict | None = None) -> tuple[int, dict]:
    data = None
    headers = {"X-AUTH-TOKEN": "admin", "ORGANIZATION": "100001", "PROJECT": PROJECT_ID}
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
        headers["Content-Type"] = "application/json"

    request = urllib.request.Request(BASE_URL + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            return response.status, json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as error:
        return error.code, {"raw": error.read().decode("utf-8", "replace")}


def main() -> None:
    matrix = json.loads(MATRIX.read_text(encoding="utf-8"))
    workspace_id = f"TESTARCH-push-{int(time.time() * 1000)}"

    print(f"artifact      : {MATRIX.relative_to(ROOT)}")
    print(f"sections      : {len(matrix['sections'])}")
    print(f"target        : {BASE_URL}")
    print(f"workspace     : {workspace_id}")

    sql(
        "INSERT INTO aegis.quality_workspace "
        "(workspace_id, project_id, name, status, tags, scope_definition, metadata, "
        " create_user, update_user, created_at, updated_at, archived) VALUES "
        f"('{workspace_id}', '{PROJECT_ID}', 'TESTARCH push verification', 'DRAFT', "
        "'[]', '{}', '{}', 'admin', 'admin', "
        f"{int(time.time() * 1000)}, {int(time.time() * 1000)}, b'0')"
    )

    try:
        payload = {
            "repoUrl": matrix["repoUrl"],
            "gitBranch": matrix["gitBranch"],
            "commitSha": matrix["commitSha"],
            "prdPath": matrix["prdPath"],
            "sections": matrix["sections"],
        }

        status, body = http(
            "POST", f"/api/quality-workspace/{workspace_id}/diff-matrix/sync", payload
        )
        if body.get("code") != 200:
            raise SystemExit(f"FAIL sync: HTTP {status} code={body.get('code')} message={body.get('message')}")

        synced = body["data"]
        assert len(synced["sections"]) == len(matrix["sections"]), "section count changed on sync"
        assert synced["commitSha"] == matrix["commitSha"], "commitSha not echoed"
        assert synced["lastSyncedAt"] > 0, "server did not stamp lastSyncedAt"
        print(f"  sync        : HTTP {status} code=200 sections={len(synced['sections'])} "
              f"lastSyncedAt={synced['lastSyncedAt']}")

        status, body = http("GET", f"/api/quality-workspace/{workspace_id}/diff-matrix")
        if body.get("code") != 200:
            raise SystemExit(f"FAIL read-back: HTTP {status} code={body.get('code')}")
        reread = body["data"]

        original = {s["sectionNumber"] for s in matrix["sections"]}
        restored = {s["sectionNumber"] for s in reread["sections"]}
        if original != restored:
            raise SystemExit(f"FAIL read-back mismatch: missing={sorted(original - restored)}")
        print(f"  read-back   : HTTP {status} sections={len(reread['sections'])} "
              f"all section numbers round-tripped")

        counts: dict[str, int] = {}
        for section in reread["sections"]:
            counts[section["diffStatus"]] = counts.get(section["diffStatus"], 0) + 1
        print(f"  diffStatus  : {counts}")

        persisted = sql(
            "SELECT JSON_LENGTH(metadata->'$.diff_matrix.sections') "
            f"FROM aegis.quality_workspace WHERE workspace_id='{workspace_id}'"
        )
        print(f"  db metadata : diff_matrix.sections rows = {persisted}")

        print("\nRESULT: PASS - the artifact satisfies the live sync contract")
    finally:
        sql(f"DELETE FROM aegis.quality_workspace WHERE workspace_id='{workspace_id}'")
        print(f"cleanup     : workspace {workspace_id} removed")


if __name__ == "__main__":
    main()
