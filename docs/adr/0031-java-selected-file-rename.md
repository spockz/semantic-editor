# ADR-0031: Java Selected-File Semantic Rename

Status: Accepted  
Date: 2026-09-17

## Context

Java rename uses an external JDT LS process and must remain bounded by the
request's explicit workspace trust and selected source file.

## Decision

Java supports trusted semantic rename only for the selected canonical regular
`.java` file. The backend asks JDT LS to prepare and calculate the rename,
accepts only a complete single-file `WorkspaceEdit`, validates UTF-16 ranges,
preimages, versions, annotations, resources, and overlap, then applies spans in
reverse order through the atomic writer. Build import, formatting, imports, and
diagnostics are not invoked.

## Invariants

- Trust is checked before JDT LS discovery, JVM validation, or process launch.
- Foreign files, resources, annotations, malformed UTF-16 ranges, version
  mismatches, stale preimages, multi-file edits, and overlap are rejected before
  writing.
- The managed session is closed and reset after a successful commit.

## Consequences

Java now has a deliberately narrow mutation capability while lookup behavior
and all other language backends remain unchanged.
