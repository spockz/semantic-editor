<!-- WHY: Defines every required decision field and its stable metadata table keys for adr-tools-created records. -->
# ADR-0083: Structured ADR Metadata and Search

* **Status**: Accepted
* **Date**: 2026-10-21

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Store fixed decision metadata, define its taxonomy, and generate the validated register and paginated search results deterministically with workspace-relative paths. |
| description | The ADR index currently repeats titles, dates, statuses, and hand-written summaries that are not available from the records themselves. Manual synchronization can leave the register stale, while free-form table headers make parsing fragile. adr-tools can create records from the repository template, but its link commands only update a `## Status` section and do not fit this repository's inline status field. |
| role | architecture |
| scope | documentation |
| area | generic |
| languages | generic |

## Context

The ADR index currently repeats titles, dates, statuses, and hand-written summaries that are not available from the records themselves. Manual synchronization can leave the register stale, while free-form table headers make parsing fragile. Agents need to find applicable decisions quickly and then open the full record through their native file tools. The project also needs to distinguish the shared decision-role taxonomy from the individual first principles and principles that guide its work.

## Decision

Every ADR will contain a required title, concise summary, description, status, date, Context, Decision, Invariants, and Consequences. A fixed two-column metadata table stores `summary`, `description`, `role`, `scope`, `area`, and `languages` in that order. An adjacent HTML comment names the schema version and keys and states that the keys and order must not change. The `role` field records the decision kind. Its known-good values are `first-principle` for foundational premises, `principle` for general design guidance, `invariant` for requirements that must remain true, `architecture` for structural choices, `contract` for interface or protocol rules, and `process` for lifecycle or workflow choices. These values are guidance rather than a closed enumeration. Scope and area values are also guidance; language labels remain open-ended. This ADR defines the taxonomy and its labels; it is not a collection of the project's individual principles. Give each independently adoptable or revisable first principle or principle its own ADR. Keep multiple principles in one record only when they form a cohesive set with a shared rationale and consequence. Do not create a separate ADR for each taxonomy label.

The `adr-tools` `.adr-dir` marker points to `docs/adr`, and `docs/adr/templates/template.md` defines the repository's record shape. Use adr-tools for numbering and template-based creation. It pads the filename but writes an unpadded number in the title; the parser normalizes positive one-to-four-digit title numbers to the canonical four-digit identifier and verifies that it matches the filename. Do not use `adr link` or `adr new -s` for supersession: the installed tool updates only a `## Status` section, while this schema stores status in a `* **Status**:` metadata line, so the link commands leave the records unchanged. Edit supersession links through native file tools, update the record status in the same edit, and regenerate the register with semedit. Do not use adr-tools' basic generated TOC as the source for this richer index.

Semedit validates ADR records and generates only the decision-register section of `docs/adr/README.md` from the records. The architectural summary, active requirements, and other authored material remain outside the generated section. `semantic_search_adrs` performs deterministic lexical search across every metadata label and the Context, Decision, Invariants, and Consequences sections, applies optional metadata filters, returns stable IDs and paginated results, and reports paths relative to the semedit working directory. No model-generated summary or absolute local path is needed.

## Invariants

* Each ADR is the source of truth for its own metadata; the decision register is generated from validated ADR records.
* Metadata keys and order are a versioned schema contract and must not change without migrating every record and consumer in the same change.
* ADR search is deterministic, paginated, and returns only working-directory-relative paths.
* Missing or malformed required metadata fails validation and index generation; it is never silently reconstructed from the index.
* ADR-0083 defines decision-role categories; project-specific first principles and principles live in their own ADRs unless a cohesive set shares one rationale and consequence.

## Consequences

ADR authors maintain a small fixed metadata block, and validation rejects malformed records before publication. The register cannot drift from the records, and search results give agents stable identities and usable paths without forcing full ADR text through the MCP response. Existing ADRs must be migrated to the required schema before the generated register and search operation become authoritative.
