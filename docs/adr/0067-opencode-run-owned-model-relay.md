<!-- Records why sandboxed local-model jobs share a confined transport owned by their benchmark run. -->
# ADR-0067: OpenCode Run-Owned Model Relay

* **Status**: Accepted
* **Date**: 2026-10-11

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Share one fixed-destination loopback relay per benchmark run, use OS-assigned ports, confine servers and attached clients, and close active transport at run completion. |
| description | OpenCode local-model jobs need access to the desktop's OpenAI-compatible HTTP endpoint without allowing agents to reach arbitrary remote services. On macOS 26.6.2, Seatbelt rejects numeric hosts in `remote tcp` and `remote ip` rules: hosts must be `localhost` or `*`. A wildcard host with a selected port permits every host on that port. Alternative exact-IP predicates also fail compilation. Live probes confirmed both wildcard access and loopback confinement. |
| role | process |
| scope | tooling |
| area | benchmark |
| languages | generic |

## Context

OpenCode local-model jobs need access to the desktop's OpenAI-compatible HTTP
endpoint without allowing agents to reach arbitrary remote services. On macOS
26.6.2, Seatbelt rejects numeric hosts in `remote tcp` and `remote ip` rules:
hosts must be `localhost` or `*`. A wildcard host with a selected port permits
every host on that port. Alternative exact-IP predicates also fail compilation.
Live probes confirmed both wildcard access and loopback confinement.

## Decision

Each benchmark run owns one fixed-destination TCP relay for its configured local
OpenCode endpoint. Start it before dispatching jobs and share its immutable
loopback URL with every relevant runner. Bind an OS-assigned loopback port so
independent runs cannot clash. Stop it after the run's jobs finish, including on
error or cancellation. No external relay executable or shared daemon is needed.

Parse the upstream once and require HTTP, a numeric IP, and an explicit port.
Preserve the API path when creating the loopback URL. Reject unsupported endpoint
forms rather than weakening the sandbox or TLS validation. Generated provider
configuration and sandbox rules consume the same selected transport URL.

## Invariants

* The relay always dials the one validated upstream address, without DNS lookup.
* A run creates one relay, irrespective of its job count or concurrency.
* OpenCode servers, attached clients, and their children run under the fixture's
  profile. Outbound access is confined to the relay and exact session ports.
* Cleanup closes the listener and active forwarding connections and waits for
  forwarding to stop. One run's cleanup cannot stop another run's relay.
* Existing fixture, outside-file, and symlink restrictions remain enforced.
* Endpoint or relay setup failures are explicit errors, not empty model results.

## Consequences

The desktop model becomes reachable without wildcard remote access or fixed-port
collisions. Concurrent jobs share transport setup, while each OpenCode session
retains its own continuation state. Numeric HTTP endpoints are supported; DNS
names and HTTPS require a separately reviewed transport contract.
