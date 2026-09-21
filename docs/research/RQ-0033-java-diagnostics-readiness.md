# Java Diagnostics Readiness

* **Status**: Open
* **Category**: Runtime & Architecture
* **Date**: 2026-09-21

## Question

What JDT LS notification and versioning guarantees are sufficient to declare selected-file diagnostics ready after a bounded Java verification edit?

## Current evidence

The backend waits for a selected canonical URI and final document version. Unversioned or stale notifications do not satisfy readiness; timeout is explicit. Hermetic tests cover wrong URI, stale version, and timeout.

## Follow-up

Validate this policy against supported JDT LS releases before expanding diagnostics beyond the selected file.
