<!-- WHY: Establishes the complete-migration rule for this unreleased project so interface changes leave one implementation path and no compatibility residue. -->
# ADR-0082: Complete Migrations Without Legacy Paths

* **Status**: Accepted
* **Date**: 2026-10-21

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Migrate every affected caller, registration, test, document, skill, and persisted artifact together, then remove the old path. Keep ADR history through linked superseding decisions. |
| description | The project is unreleased and has no compatibility obligations to prior public interfaces. Retaining aliases, fallback wrappers, parallel implementations, or transition paths after a design change creates duplicate behavior, inconsistent capability metadata, and ongoing maintenance work. Architectural decision records have a separate purpose: they preserve the history and rationale of accepted decisions. |
| role | first-principle |
| scope | project |
| area | generic |
| languages | generic |

## Context

The project is unreleased and has no compatibility obligations to prior public interfaces. Retaining aliases, fallback wrappers, parallel implementations, or transition paths after a design change creates duplicate behavior, inconsistent capability metadata, and ongoing maintenance work. Architectural decision records have a separate purpose: they preserve the history and rationale of accepted decisions.

## Decision

Every change to an interface, mechanism, schema, or behavior must migrate all affected production code, callers, registrations, tests, documentation, skills, and persisted artifacts in the same change. Remove the previous path as part of that migration. Do not introduce compatibility aliases, fallback wrappers, parallel implementations, versioned interfaces, or transitional shims.

Accepted ADRs remain historical records. When an accepted decision changes, create a linked superseding ADR and update the old record's status and indexes according to the repository's ADR governance.

## Invariants

* Each behavior has one canonical public identity and one active implementation path after a change.
* All callers, tests, generated documentation, skills, and persisted data are migrated before the change is considered complete.
* No old-name lookup, fallback dispatch, compatibility wrapper, or duplicate implementation remains after migration.
* Accepted ADRs retain their original decision history and point to the superseding record when replaced.

## Consequences

Changes may require a broader atomic diff because consumers and documentation move with the implementation. The project avoids compatibility maintenance and prevents stale paths from becoming permanent. ADR history remains auditable without keeping obsolete executable behavior.
