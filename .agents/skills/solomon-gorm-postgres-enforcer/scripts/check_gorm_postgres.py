#!/usr/bin/env python3
"""
Lightweight guardrail for detecting raw SQL usage in Go code.

Usage:
  python .agents/skills/solomon-gorm-postgres-enforcer/scripts/check_gorm_postgres.py .
"""

from __future__ import annotations

import pathlib
import re
import sys
from dataclasses import dataclass


ALLOW_TAG = "gorm-postgres-enforcer: allow-raw-sql"

SKIP_DIRS = {
    ".git",
    "vendor",
    "node_modules",
}

# Direct SQL API usage that should be removed from module code.
BAD_CALL_PATTERNS = [
    re.compile(r"\.\s*QueryContext\s*\("),
    re.compile(r"\.\s*QueryRowContext\s*\("),
    re.compile(r"\.\s*QueryRow\s*\("),
    re.compile(r"\.\s*Query\s*\("),
    re.compile(r"\.\s*ExecContext\s*\("),
    re.compile(r"\.\s*Exec\s*\("),
    re.compile(r"\.\s*PrepareContext\s*\("),
    re.compile(r"\.\s*Prepare\s*\("),
]

BAD_IMPORT_PATTERN = re.compile(r'^\s*"database/sql"\s*$')


@dataclass
class Violation:
    path: pathlib.Path
    line_no: int
    message: str


def should_scan(path: pathlib.Path) -> bool:
    if path.suffix != ".go":
        return False
    if path.name.endswith("_test.go"):
        return False
    parts = set(path.parts)
    if parts & SKIP_DIRS:
        return False
    # Ignore generated docs and SQL migration files.
    if "docs" in parts and "httpserver" in parts:
        return False
    return True


def scan_file(path: pathlib.Path) -> list[Violation]:
    violations: list[Violation] = []
    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        text = path.read_text(encoding="latin-1")

    lines = text.splitlines()
    for i, line in enumerate(lines, start=1):
        if ALLOW_TAG in line:
            continue

        if BAD_IMPORT_PATTERN.search(line):
            violations.append(Violation(path, i, 'forbidden import "database/sql"'))
            continue

        if "fmt.Sprintf(" in line and ("SELECT " in line or "INSERT " in line or "UPDATE " in line or "DELETE " in line):
            violations.append(Violation(path, i, "possible SQL string construction with fmt.Sprintf"))

        for pat in BAD_CALL_PATTERNS:
            if pat.search(line):
                violations.append(Violation(path, i, f"forbidden raw SQL API usage: {pat.pattern}"))
                break

    return violations


def main() -> int:
    root = pathlib.Path(sys.argv[1]) if len(sys.argv) > 1 else pathlib.Path(".")
    root = root.resolve()

    violations: list[Violation] = []
    for path in root.rglob("*.go"):
        if should_scan(path):
            violations.extend(scan_file(path))

    if not violations:
        print("check_gorm_postgres: no raw SQL violations found")
        return 0

    print("check_gorm_postgres: violations found")
    for item in violations:
        rel = item.path.relative_to(root)
        print(f"- {rel}:{item.line_no}: {item.message}")
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
