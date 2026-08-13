#!/usr/bin/env python3

import json
import re
import subprocess
import sys
from pathlib import Path


PROJECT_MODULE = "github.com/yuchen0x1/sub3api-relay"
HYSTERIA_MODULE = "github.com/apernet/hysteria/core/v2"
HYSTERIA_FALSE_POSITIVE = "GO-2026-5288"
HYSTERIA_FIRST_SAFE_VERSION = (2, 8, 2)


def load_messages(path: Path):
    decoder = json.JSONDecoder()
    content = path.read_text(encoding="utf-8-sig")
    position = 0
    while position < len(content):
        while position < len(content) and content[position].isspace():
            position += 1
        if position >= len(content):
            return
        message, position = decoder.raw_decode(content, position)
        yield message


def module_version(module: str) -> str:
    return subprocess.run(
        ["go", "list", "-m", "-f", "{{.Version}}", module],
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()


def semantic_version(value: str):
    match = re.fullmatch(r"v(\d+)\.(\d+)\.(\d+)(?:[-+].*)?", value)
    if not match:
        raise ValueError(f"unsupported semantic version: {value}")
    return tuple(int(part) for part in match.groups())


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: check-govuln-report.py <govulncheck.json>", file=sys.stderr)
        return 2

    findings = []
    saw_config = False
    saw_sbom = False
    for message in load_messages(Path(sys.argv[1])):
        saw_config = saw_config or "config" in message
        saw_sbom = saw_sbom or "SBOM" in message
        if error := message.get("error"):
            print(f"govulncheck report error: {error}", file=sys.stderr)
            return 1
        finding = message.get("finding")
        if finding and any(frame.get("module") == PROJECT_MODULE for frame in finding.get("trace", [])):
            findings.append(finding)

    if not saw_config or not saw_sbom:
        print("govulncheck report is incomplete", file=sys.stderr)
        return 1

    vulnerability_ids = sorted({finding["osv"] for finding in findings})
    unexpected = [item for item in vulnerability_ids if item != HYSTERIA_FALSE_POSITIVE]
    if unexpected:
        print(f"reachable vulnerabilities found: {', '.join(unexpected)}", file=sys.stderr)
        return 1

    if HYSTERIA_FALSE_POSITIVE in vulnerability_ids:
        version = module_version(HYSTERIA_MODULE)
        if semantic_version(version) < HYSTERIA_FIRST_SAFE_VERSION:
            print(
                f"{HYSTERIA_FALSE_POSITIVE} affects {HYSTERIA_MODULE} {version}",
                file=sys.stderr,
            )
            return 1
        server_traces = [
            finding
            for finding in findings
            if finding["osv"] == HYSTERIA_FALSE_POSITIVE
            and any(
                frame.get("package", "").startswith(f"{HYSTERIA_MODULE}/server")
                for frame in finding.get("trace", [])
            )
        ]
        if server_traces:
            print(
                f"{HYSTERIA_FALSE_POSITIVE} reaches Hysteria server code",
                file=sys.stderr,
            )
            return 1
        print(
            f"Ignored {HYSTERIA_FALSE_POSITIVE}: {HYSTERIA_MODULE} {version} is newer than "
            "the GHSA affected range and only client code is imported."
        )
        return 0

    print("No reachable vulnerabilities found.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
