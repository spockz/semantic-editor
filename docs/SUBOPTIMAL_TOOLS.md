<!-- Records semantic-tool failures and workarounds so repeated faults are diagnosable. -->

# Suboptimal Model Context Protocol (MCP) Tools & Dogfooding Feedback Log

## 2026-10-17: Batch function insertion rejected multiple declarations

* Tool: `semantic_batch` containing `semantic_insert_construct` for timeout reflection helpers in `tools/benchmark-harness/driver.go`.
* Failure: The earlier field and method edits were applied, but helper insertion failed with `function insertion requires exactly one function declaration` because the snippet contained two functions.
* Workaround: Insert each helper as a separate declaration; inspect the partial batch result and final diff before continuing.
* Root cause: Each function insertion operation accepts exactly one function declaration, while batch child edits are not rolled back when a later child fails.

## 2026-10-17: Semantic insertion kind mismatch for method

* Tool: `semantic_insert_construct`, target `tools/benchmark-harness/driver.go`.
* Failure: Inserting a declaration with a `*Runner` receiver as `kind: "function"` failed with `source declaration does not match construct kind "function"`; no mutation was reported.
* Workaround: Retry with the semantically matching `kind: "method"` after confirming the receiver declaration.
* Root cause: The insertion tool distinguishes receiver methods from package-level functions and validates the supplied construct kind.

## 2026-10-17: Semantic batch verification stopped at workspace trust gate

* Tool: `semantic_batch` replacing benchmark reflection lifecycle helpers in `tools/benchmark-harness/`.
* Failure: Edits were reported as intermediate; final verification failed because the worktree's `tools/go-cache-root.sh` requires explicit workspace trust.
* Workaround: Retry the scoped verification only after explicitly enabling trust for this user-selected repository root, then inspect every child result and the final diff.
* Root cause: Repository verification executes configured shell hooks, and the tool requires explicit trust before running the newly present shell hook.

## 2026-10-17: Semantic batch linter verification failed after test edits

* Tool: `semantic_batch` adding timeout-resume regression coverage in `tools/benchmark-harness/session_test.go` with explicit workspace trust.
* Failure: Child edits were reported before final verification; the configured `golangci-lint` hook exited with status 3 and no diagnostic text was retained in the tool response.
* Workaround: Read back the working tree and run repository formatting/check commands to capture the linter failure before making further edits.
* Root cause: The semantic batch runs repository hooks before returning success, but this hook supplied only an exit status for the aggregate operation.

## 2026-10-16: Function extraction rejected assignment statement in Normalize

* Tool: `semantic_extract_function`, target `request/request.go`, function `Normalize`.
* Failure: Extraction of normalization statements (`kind = strings.TrimSpace(kind)`, `kind = strings.ToLower(kind)`) was rejected with "cannot extract Normalize: only one leading := declaration with matching named results is supported".
* Workaround: Applied function insertion and body replacement via semantic tools.
* Root cause: Go function extraction requires extracted statements to be pure call expressions or a single leading `:=` short variable declaration; reassignment statements (`=`) are rejected.

## 2026-10-16: construct replacement rejected multiple declarations

* Tool: `semantic_replace_construct`, targeting `TestBuildBenchmarkPlanInjectsSkillOnlyIntoSemeditArms` in `tools/benchmark-harness/planner_test.go`.
* Failure: a replacement snippet containing the test plus several helper declarations was rejected with `multiple declarations found in snippet`.
* Workaround: replace the test function alone, then insert each helper as a separate function declaration.
* Root cause: the construct replacement operation accepts one declaration per request.

## 2026-10-16: range-loop anchor rejected during shared cache setup

* Tool: `semantic_insert_construct`, targeting `TestGoAndFormattingPruneBenchmarkData` in `cmd/semedit/benchmark_exclusions_test.go`.
* Failure: the existing `for relative, contents := range files` anchor was reported not found; no mutation was reported.
* Workaround: targeted fixture patch copies the new shared Make configuration and resolver alongside the fixture Makefile.
* Root cause: the supplied range-loop header did not resolve as a statement anchor; exact selector support was not established.

## 2026-10-16: suggestion worktree archival refused

* Tool: `codex_app.archive_worktree`.
* Target: the attached `ios-4216a269` semantic-editor checkout for chat `01a14595-6049-7143-9408-ba2a0d055060`.
* Observed failure: after checked integration into local main at `b64cf98e`, the tool returned `The task's primary checkout cannot be removed.` No removal occurred.
* Workaround: retain the checkout and leave the chat open; do not bypass the managed-worktree rejection with shell deletion.
* Root cause: the application classifies this attached checkout as primary even though Git reports it as a linked worktree. The reason for that classification is unknown.

## 2026-10-16: session wrap-up, Optimize MCP package tests

* Context: chat `01a145d6-be27-71f1-8e33-074391d6ef1a`, linked worktree `beb3`, original detached HEAD `1ff9655e`; replayed onto local main `22a3d539` on `codex/mcp-test-efficiency`. The main checkout was clean at discovery. This session owns the checkout according to chat metadata and matching task/path.
* Coverage: both available root-history pages and all returned history for workers `mcp_test_audit`, `schema_reuse_safety`, and `implement_validator`. The API supplies arguments, status, and end-to-end duration for 80 semantic calls. It omits detailed result bodies and some orchestration-only errors. Two additional unavailable-operation attempts are preserved in incident entries below. CLI test execution is behavioral verification, not a reconstructed per-operation agent-call inventory. Compacted or missing result details remain explicitly unknown.
* Batch inventory: 127 submitted child descriptors across 11 parents, not 127 executed calls. A child after a stopping failure is skipped. Child references below use parent execution ID plus one-based descriptor number. Batch parent time includes its children and verification; child timing is unavailable because it was not separately instrumented.
* Baselines below are hypothetical workflows using available local tools, not replayed benchmarks. No mutation was replayed merely for wrap-up timing. Original pre-edit inputs and equivalent checker timing were not captured, so local runtimes are unavailable; semantic/built-in ratios are inconclusive. Discovery included tool-inventory checks, argument preparation, import repair, retries, and final `make check`; those costs prevent a claim that fewer editing calls alone saved agent effort.

| Call reference | Tool / target / intent | Outcome and QoL / agent-efficiency comparison | Semantic runtime | Local built-in runtime / comparison limit |
| :--- | :--- | :--- | :--- | :--- |
| `2edc08c8-14ea-4903-bef3-9087c04f50e6` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestMCPIntegerNotationAndRangeValidation | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 5 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `5762bb4b-32e1-4590-911f-df17b3f0cc90` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / schemaWorkspace | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 8 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `b615eb10-220d-4392-b0dd-0dacac7adac9` | `semantic_inspect_symbol`: internal/mcp/server_test.go / TestMCPMavenTestFailureIncludesStructuredResult | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 7 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `81e8c36b-61ef-4c1f-9349-80bdd830bba9` | `semantic_inspect_symbol`: internal/mcp/server_test.go / assertMCPMavenToolFailure | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 3 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `ffb6224d-7874-4f99-ba3b-6ef0ed84be3c` | `semantic_inspect_symbol`: internal/mcp/failure_contract_test.go / TestUnknownAndDisabledReloadToolCallsAreJSONRPCFailures | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 2 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `35180d85-4228-43e0-ad0a-3304d13c0af1` | `semantic_batch`: batch descriptors below | **harmful**: Child 1 applied; final hook lacked trust. Targeted apply_patch changes, import formatting and make check; manual payloads avoid tool selectors but do not reproduce AST/preimage guarantees. | 12776 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `3f7680b6-09e2-42b7-8927-4b7f2a3590fb` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestConstructReplacementSchemasAreExposedAndBatchable | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 9 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `345a7955-0688-487b-a0ac-da2cbac297f5` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestGeneratedSemanticInputSchemasRejectUnknownProperties | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 9 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `195589e9-2c43-4264-a5ec-509c50a9c4cb` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestScaffoldPurposeHeaderSchemaIsOptional | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 6 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `acc4f216-ada8-40d6-860b-3eab45d89416` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestMCPContinuationDescriptionsCoverConstructSources | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 5 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `466907a1-87e1-4c64-ae1d-23ee908c32a8` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestMoveConstructIsAdvertisedInBatchWithDestinationReceipt | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 5 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `fdbec9c1-b6b6-4a8f-b32a-c7c93a538ee1` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestMCPStaleContinuationsAreNotAdvertised | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 5 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `a6caf206-d665-4eb2-a97e-bfc1875032c4` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestMCPToolsAdvertiseStructuredOutputSchemas | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 10 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `39a7799f-af82-4caa-9f18-bfb8e5a8f934` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestMCPFullCatalogDescriptionsAndContinuations | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 6 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `bc1d7311-5195-4055-87db-8351e79f818a` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestSourceFieldsResolveToAdvertisedOutputSchema | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 9 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `776fd37b-3bd9-4266-8fdc-12e1d108b499` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestStageReceiptContinuationGraph | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 9 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `64ce59fa-8764-4818-bce3-7aa77d7ed081` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestDefaultMCPCatalogDocumentsHighRiskToolBehavior | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 8 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `2692a568-d1d0-4385-9b93-8f4ddc9b7d23` | `semantic_batch`: batch descriptors below | **harmful**: Child 1 rejected declaration rename; later descriptors skipped; buffered revision conflict reported. Targeted apply_patch changes, import formatting and make check; manual payloads avoid tool selectors but do not reproduce AST/preimage guarantees. | 6236 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `6197544f-2e68-4772-bef7-7338aa569cca` | `semantic_rename`: internal/mcp/schema_test.go / TestConstructReplacementSchemasAreExposedAndBatchable | **beneficial**: Rename applied with semantic identity/reference handling; no manual repair recorded. rg references, apply_patch for each use, then formatting/type checks; plain text rename alone lacks equivalent guarantees. | 12127 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `afd6edf1-ea8b-4bd1-b9dd-8c5d015a6384` | `semantic_rename`: internal/mcp/schema_test.go / TestGeneratedSemanticInputSchemasRejectUnknownProperties | **beneficial**: Rename applied with semantic identity/reference handling; no manual repair recorded. rg references, apply_patch for each use, then formatting/type checks; plain text rename alone lacks equivalent guarantees. | 7981 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `bf9dc8e9-9c84-4a37-8002-b53ebd849b30` | `semantic_rename`: internal/mcp/schema_test.go / TestScaffoldPurposeHeaderSchemaIsOptional | **beneficial**: Rename applied with semantic identity/reference handling; no manual repair recorded. rg references, apply_patch for each use, then formatting/type checks; plain text rename alone lacks equivalent guarantees. | 5896 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `c9b43a4e-bb74-4c51-9bda-c4a3d9366ca6` | `semantic_rename`: internal/mcp/schema_test.go / TestMCPContinuationDescriptionsCoverConstructSources | **beneficial**: Rename applied with semantic identity/reference handling; no manual repair recorded. rg references, apply_patch for each use, then formatting/type checks; plain text rename alone lacks equivalent guarantees. | 6658 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `52b9707e-f7ce-4f06-8ac3-f0d5b25b2418` | `semantic_rename`: internal/mcp/schema_test.go / TestMoveConstructIsAdvertisedInBatchWithDestinationReceipt | **beneficial**: Rename applied with semantic identity/reference handling; no manual repair recorded. rg references, apply_patch for each use, then formatting/type checks; plain text rename alone lacks equivalent guarantees. | 7117 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `9f350a42-ed2a-410d-8542-ccbefc940760` | `semantic_rename`: internal/mcp/schema_test.go / TestMCPStaleContinuationsAreNotAdvertised | **beneficial**: Rename applied with semantic identity/reference handling; no manual repair recorded. rg references, apply_patch for each use, then formatting/type checks; plain text rename alone lacks equivalent guarantees. | 6174 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `0cd9ecee-fd9d-446b-a2ec-6d5914ee38d0` | `semantic_batch`: batch descriptors below | **harmful**: Children 1–2 applied; child 3 ambiguous statement/expression; later descriptors skipped. Targeted apply_patch changes, import formatting and make check; manual payloads avoid tool selectors but do not reproduce AST/preimage guarantees. | 9603 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `5fceebf3-267c-486d-85a1-fb90409e8a5f` | `semantic_batch`: batch descriptors below | **harmful**: All 17 children applied; final hook plugin mismatch. Targeted apply_patch changes, import formatting and make check; manual payloads avoid tool selectors but do not reproduce AST/preimage guarantees. | 59532 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `9b3f8aa3-c138-46ea-9a4a-4f8529e79312` | `semantic_inspect_symbol`: internal/mcptest/validate.go / Server | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 14 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `d567f3d4-b41f-41fe-812e-dc63d7cf8516` | `semantic_inspect_symbol`: internal/mcptest/validate.go / probeSchemas | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 2 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `93a12286-aa78-4dda-b864-352a7dbc11e4` | `semantic_inspect_symbol`: internal/mcptest/validate.go / responseValidator.ensureFull | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 1 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `3e5ecfa4-8a92-4338-93fe-52251ac55cfa` | `semantic_inspect_symbol`: internal/mcptest/validate.go / responseValidator.resetCatalog | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 2 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `3e54b848-c1ec-4734-bbf0-1a1b2a8cd51e` | `semantic_inspect_symbol`: internal/mcptest/validate_test.go / TestServerDefersCatalogDiscoveryUntilAToolResultNeedsValidation | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 3 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `80d3988f-4d58-4f06-babd-e736275808f9` | `semantic_inspect_symbol`: internal/mcptest/validate.go / schemasFromResult | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 10 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `a873cc71-a28f-493d-a907-5366482cfdcc` | `semantic_inspect_symbol`: internal/operation/operation.go / Def | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 2 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `500c4b67-6bab-4925-9e04-8fabb723b5cb` | `semantic_batch`: batch descriptors below | **harmful**: All 13 children applied; final hook plugin mismatch. Targeted apply_patch changes, import formatting and make check; manual payloads avoid tool selectors but do not reproduce AST/preimage guarantees. | 81139 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `df682d0d-2682-44c9-8081-5eec04fc38f8` | `semantic_batch`: batch descriptors below | **harmful**: All 4 import children applied; final hook plugin mismatch. Targeted apply_patch changes, import formatting and make check; manual payloads avoid tool selectors but do not reproduce AST/preimage guarantees. | 12863 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `858159b0-312b-4697-ad69-c1e293d74e22` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / decodeRemainingTypedOutputResults | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 7 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `e86c86f7-7064-425a-91e4-f5370583adbc` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / mcpResultKey | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 5 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `6b356ec1-de91-4a04-8221-ddc3b9560764` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / mcpStructuredResult | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 8 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `a1a306ca-b505-4f4a-a50c-0f388fe4b37b` | `semantic_inspect_symbol`: internal/mcp/failure_contract_test.go / decodeFailureResponses | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 1 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `f9f656f5-f02b-4cc6-b70b-80580dcb5991` | `semantic_inspect_symbol`: internal/mcp/failure_contract_test.go / failureCallResponse | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 2 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `364d924b-8df4-45b2-b08d-1795c43b61ba` | `semantic_batch`: batch descriptors below | **harmful**: All 20 children applied; final hook plugin mismatch. Targeted apply_patch changes, import formatting and make check; manual payloads avoid tool selectors but do not reproduce AST/preimage guarantees. | 92845 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `b9e03c33-5e93-448d-aca0-d5141d891b92` | `semantic_replace_construct`: internal/mcp/failure_contract_test.go / TestUnknownAndDisabledReloadToolCallsAreJSONRPCFailures | **neutral**: Two-statement replacement rejected without mutation; corrected to one declaration. Expected contract rejection. Bounded read plus apply_patch and checks; no selector discovery but weaker automatic structural validation. | 938 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `a2f201f0-0eb6-469a-a346-42edc9ebc3f1` | `semantic_replace_construct`: internal/mcp/failure_contract_test.go / TestUnknownAndDisabledReloadToolCallsAreJSONRPCFailures | **beneficial**: Targeted construct edit applied; subsequent configured checks verified the resulting tree. Bounded read, apply_patch, formatting and checks; direct patch is plausible but lacks semantic selector/preimage validation. | 1743 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `04f7c17f-4176-462c-915b-b2ff4798894d` | `semantic_replace_construct`: internal/mcptest/validate_test.go / TestResponseValidatorRefreshesLiveCatalogAndInvalidatesFallback | **neutral**: Function-literal text selector failed without mutation despite a reported candidate; structured path/replacement recovery required. Bounded read plus apply_patch and checks; no selector discovery but weaker automatic structural validation. | 1136 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `04dcb690-f67d-4acf-adf7-25fa0df3ac7a` | `semantic_outline`: internal/mcp/server.go | **beneficial**: Declaration inventory retrieved without mutation; narrowed later reads. rg declaration names plus bounded reads; source text does not itself supply an equivalent AST hierarchy. | 33 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `d4d48a36-fd0f-4360-9083-90c81fefe30e` | `semantic_inspect_symbol`: internal/mcp/server.go / Server.catalogTools | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 5 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `21e370d7-6716-40c8-9851-b2f81eb864e5` | `semantic_inspect_symbol`: internal/mcp/server.go / Server.requestView | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 4 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `ea1b3c49-bb83-46f5-9401-91da212c7064` | `semantic_inspect_symbol`: internal/mcp/server.go / requestViewFromConfiguration | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 3 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `5c124395-8eb8-42ff-a803-2f9bda776a6e` | `semantic_inspect_symbol`: internal/mcp/server.go / Server.configurationSnapshot | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 4 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `8b06c8fe-2051-4930-94d2-0b40ae75f003` | `semantic_inspect_symbol`: internal/mcp/server.go / Server.activeLanguagesForRoot | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 4 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `4c699053-c898-4c03-98c1-6191eaaa0219` | `semantic_inspect_symbol`: internal/mcptest/validate.go / responseValidator | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 1 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `b9ce80a0-a608-46a7-bb07-4a7cbbd0bf89` | `semantic_inspect_symbol`: internal/mcptest/validate.go / responseValidator.recordResponse | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 1 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `62f1aa0a-7b6c-4ae8-9482-231f523852c1` | `semantic_inspect_symbol`: internal/mcptest/validate.go / responseValidator.validateToolResult | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 1 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `11fa9998-3e29-4e0c-a44c-5e0945411fc1` | `semantic_inspect_symbol`: internal/mcp/language_snapshot_wire_test.go / languageWireProbeInput | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 1 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `8bcda4fd-87f6-4342-9b45-6fe1002e8e91` | `semantic_inspect_symbol`: internal/mcp/language_snapshot_wire_test.go / languageWireProbeRegistry | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 1 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `5f8607d3-078b-4000-84c5-4b1c8d43a8f3` | `semantic_inspect_symbol`: internal/mcp/language_snapshot_wire_test.go / TestLanguageSnapshotGuardsDirectAndBatchWireCalls | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 1 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `2a98f11f-5763-4634-8f86-8ae291c17dd3` | `semantic_inspect_symbol`: internal/mcp/language_snapshot_wire_test.go / writeWireLanguageSource | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 0 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `258b677c-90c8-4c39-bce8-7aa8529e39f2` | `semantic_batch`: batch descriptors below | **harmful**: Both children applied; final hook plugin mismatch. Targeted apply_patch changes, import formatting and make check; manual payloads avoid tool selectors but do not reproduce AST/preimage guarantees. | 7559 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `95caae02-4f13-4c92-bb12-30a4bc00298b` | `semantic_batch`: batch descriptors below | **harmful**: 11 children applied; child 12 hit stale LSP shutdown/exit deadlines; later descriptors skipped. Disk inspection preceded retry. Targeted apply_patch changes, import formatting and make check; manual payloads avoid tool selectors but do not reproduce AST/preimage guarantees. | 45305 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `abc352ed-d957-4ac4-919c-e20ac99d790f` | `semantic_inspect_symbol`: internal/mcptest/validate.go / responseValidator.captureCatalogs | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 7 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `8aa41c2f-11da-4435-aafe-b45c934fc7e1` | `semantic_inspect_symbol`: internal/mcp/server.go / Server.handleToolCall | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 21 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `e275d94d-9b98-49df-a3af-4afb22447685` | `semantic_batch`: batch descriptors below | **harmful**: All 9 remaining children applied; final hook plugin mismatch. Targeted apply_patch changes, import formatting and make check; manual payloads avoid tool selectors but do not reproduce AST/preimage guarantees. | 40024 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `15182fd7-0e1c-4468-81b8-eea156b4dee4` | `semantic_rename`: internal/mcptest/validate_test.go / TestResponseValidatorValidatesHiddenToolWithLazyFallback | **beneficial**: Rename applied with semantic identity/reference handling; no manual repair recorded. rg references, apply_patch for each use, then formatting/type checks; plain text rename alone lacks equivalent guarantees. | 17593 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `3f43fbfd-36ac-4d98-a7b6-c686f67244b2` | `semantic_verify`: internal/mcptest | **harmful**: Check-only hook failed on incompatible linter/plugin and unrelated diagnostics; no mutation. Pinned make check and race tests were authoritative. make check plus selected race tests; configured Go environment differs from inferred hook. | 4180 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `05d98d11-ff01-4171-ba07-c612064f9d54` | `semantic_inspect_symbol`: internal/mcptest/validate.go / responseValidator.recordResponse | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 93 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `3bb7c0d0-fcff-4fea-9d11-8eccecd49ba2` | `semantic_outline`: internal/mcp | **beneficial**: Declaration inventory retrieved without mutation; narrowed later reads. rg declaration names plus bounded reads; source text does not itself supply an equivalent AST hierarchy. | 355 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `8e33504b-73b3-49bb-9d85-3d525cf4cb01` | `semantic_inspect_symbol`: internal/mcptest/validate.go / responseValidator.recordResponse | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 12 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `27145ee8-916b-4dba-9186-ff966b0884c3` | `semantic_outline`: internal/mcp/output_contract_test.go | **beneficial**: Declaration inventory retrieved without mutation; narrowed later reads. rg declaration names plus bounded reads; source text does not itself supply an equivalent AST hierarchy. | 10 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `49c87446-a767-4abb-8e53-14243d9022b2` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / listToolsAtWithRegistry | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 48 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `b2375582-7b32-4fe4-96aa-6a16add2c03e` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestMCPContinuationDescriptionsNameSourcesWithinBudget | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 13 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `0120d0d5-32d7-430c-949e-1eca3f777ef9` | `semantic_inspect_symbol`: internal/mcptest/validate.go / responseValidator.validateToolResult | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 30 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `a1f42ed7-c1ef-4b7b-aed5-ee66f359e7dd` | `semantic_inspect_symbol`: internal/mcp/failure_contract_test.go / TestMCPRejectedConstructsPreserveSource | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 56 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `45e311c3-dac0-4d22-b2b4-724054876307` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestMCPIntegerNotationAndRangeValidation | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 134 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `d70b5c57-0ec4-4962-8505-f2ccd339d0a5` | `semantic_inspect_symbol`: internal/mcp/schema_test.go / TestMCPIntegerNotationAndRangeValidation | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 121 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `15393c6e-142f-4756-b940-32ea7977377b` | `semantic_rename`: internal/mcp/schema_test.go / TestMCPContinuationDescriptionsNameSourcesWithinBudget | **beneficial**: Rename applied with semantic identity/reference handling; no manual repair recorded. rg references, apply_patch for each use, then formatting/type checks; plain text rename alone lacks equivalent guarantees. | 19921 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `45531f68-9cd5-477e-b628-60b1eeccf33f` | `semantic_replace_construct`: internal/mcp/schema_test.go / TestMCPFullCatalogDescriptionsAndContinuations | **beneficial**: Targeted construct edit applied; subsequent configured checks verified the resulting tree. Bounded read, apply_patch, formatting and checks; direct patch is plausible but lacks semantic selector/preimage validation. | 4002 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `59d4c129-93b6-4a41-b8e0-86ad38e65b0c` | `semantic_batch`: batch descriptors below | **harmful**: Two removal descriptors submitted by audit worker; result marked failed. Individual persistence and failure detail are not recoverable from returned history; later source inspection confirmed consolidated coverage. Targeted apply_patch changes, import formatting and make check; manual payloads avoid tool selectors but do not reproduce AST/preimage guarantees. | 18526 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `3de6d358-c547-46e9-977a-181435a77256` | `semantic_outline`: internal/mcp | **beneficial**: Declaration inventory retrieved without mutation; narrowed later reads. rg declaration names plus bounded reads; source text does not itself supply an equivalent AST hierarchy. | 195 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `1aece8a7-ee42-452e-89d1-db58920de63a` | `semantic_inspect_symbol`: internal/mcptest/validate.go / responseValidator | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 24 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |
| `4724c4ff-1ee1-4e9c-93a1-5dbf234aeb22` | `semantic_inspect_symbol`: internal/mcptest/validate.go / responseValidator | **beneficial**: Complete declaration retrieved without mutation; source, signature and imports were useful. JSON-wrapped inspection required decoding. rg for declaration, then bounded file read; hypothetical two steps versus one inspection, plus semantic discovery. | 2 ms, history end-to-end | unavailable: equivalent local workflow not timed on original input; no ratio |

### Batch child descriptors and execution boundaries

* `35180d85-4228-43e0-ad0a-3304d13c0af1`: Child 1 applied; final hook lacked trust.
  * `#1` `semantic_replace_body`: internal/mcp/schema_test.go / TestMCPIntegerNotationAndRangeValidation. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
* `2692a568-d1d0-4385-9b93-8f4ddc9b7d23`: Child 1 rejected declaration rename; later descriptors skipped; buffered revision conflict reported.
  * `#1` `semantic_replace_construct`: internal/mcp/schema_test.go / TestConstructReplacementSchemasAreExposedAndBatchable. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#2` `semantic_replace_construct`: internal/mcp/schema_test.go / assertConstructReplacementSchemasAreExposedAndBatchable. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#3` `semantic_replace_construct`: internal/mcp/schema_test.go / assertConstructReplacementSchemasAreExposedAndBatchable. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#4` `semantic_replace_construct`: internal/mcp/schema_test.go / TestGeneratedSemanticInputSchemasRejectUnknownProperties. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#5` `semantic_replace_construct`: internal/mcp/schema_test.go / assertGeneratedSemanticInputSchemasRejectUnknownProperties. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#6` `semantic_replace_construct`: internal/mcp/schema_test.go / assertGeneratedSemanticInputSchemasRejectUnknownProperties. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#7` `semantic_replace_construct`: internal/mcp/schema_test.go / TestScaffoldPurposeHeaderSchemaIsOptional. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#8` `semantic_replace_construct`: internal/mcp/schema_test.go / assertScaffoldPurposeHeaderSchemaIsOptional. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#9` `semantic_replace_construct`: internal/mcp/schema_test.go / assertScaffoldPurposeHeaderSchemaIsOptional. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#10` `semantic_replace_construct`: internal/mcp/schema_test.go / TestMCPContinuationDescriptionsCoverConstructSources. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#11` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPContinuationDescriptionsCoverConstructSources. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#12` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPContinuationDescriptionsCoverConstructSources. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#13` `semantic_replace_construct`: internal/mcp/schema_test.go / TestMoveConstructIsAdvertisedInBatchWithDestinationReceipt. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#14` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMoveConstructIsAdvertisedInBatchWithDestinationReceipt. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#15` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMoveConstructIsAdvertisedInBatchWithDestinationReceipt. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#16` `semantic_replace_construct`: internal/mcp/schema_test.go / TestMCPStaleContinuationsAreNotAdvertised. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#17` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPStaleContinuationsAreNotAdvertised. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#18` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPStaleContinuationsAreNotAdvertised. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#19` `semantic_replace_body`: internal/mcp/schema_test.go / TestMCPFullCatalogDescriptionsAndContinuations. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
* `0cd9ecee-fd9d-446b-a2ec-6d5914ee38d0`: Children 1–2 applied; child 3 ambiguous statement/expression; later descriptors skipped.
  * `#1` `semantic_replace_construct`: internal/mcp/schema_test.go / assertConstructReplacementSchemasAreExposedAndBatchable. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#2` `semantic_replace_construct`: internal/mcp/schema_test.go / assertConstructReplacementSchemasAreExposedAndBatchable. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#3` `semantic_replace_construct`: internal/mcp/schema_test.go / assertConstructReplacementSchemasAreExposedAndBatchable. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#4` `semantic_replace_construct`: internal/mcp/schema_test.go / assertGeneratedSemanticInputSchemasRejectUnknownProperties. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#5` `semantic_replace_construct`: internal/mcp/schema_test.go / assertGeneratedSemanticInputSchemasRejectUnknownProperties. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#6` `semantic_replace_construct`: internal/mcp/schema_test.go / assertGeneratedSemanticInputSchemasRejectUnknownProperties. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#7` `semantic_replace_construct`: internal/mcp/schema_test.go / assertScaffoldPurposeHeaderSchemaIsOptional. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#8` `semantic_replace_construct`: internal/mcp/schema_test.go / assertScaffoldPurposeHeaderSchemaIsOptional. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#9` `semantic_replace_construct`: internal/mcp/schema_test.go / assertScaffoldPurposeHeaderSchemaIsOptional. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#10` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPContinuationDescriptionsCoverConstructSources. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#11` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPContinuationDescriptionsCoverConstructSources. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#12` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPContinuationDescriptionsCoverConstructSources. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#13` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMoveConstructIsAdvertisedInBatchWithDestinationReceipt. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#14` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMoveConstructIsAdvertisedInBatchWithDestinationReceipt. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#15` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMoveConstructIsAdvertisedInBatchWithDestinationReceipt. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#16` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPStaleContinuationsAreNotAdvertised. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#17` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPStaleContinuationsAreNotAdvertised. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#18` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPStaleContinuationsAreNotAdvertised. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#19` `semantic_replace_body`: internal/mcp/schema_test.go / TestMCPFullCatalogDescriptionsAndContinuations. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
* `5fceebf3-267c-486d-85a1-fb90409e8a5f`: All 17 children applied; final hook plugin mismatch.
  * `#1` `semantic_replace_construct`: internal/mcp/schema_test.go / assertConstructReplacementSchemasAreExposedAndBatchable. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#2` `semantic_replace_construct`: internal/mcp/schema_test.go / assertGeneratedSemanticInputSchemasRejectUnknownProperties. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#3` `semantic_replace_construct`: internal/mcp/schema_test.go / assertGeneratedSemanticInputSchemasRejectUnknownProperties. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#4` `semantic_replace_construct`: internal/mcp/schema_test.go / assertGeneratedSemanticInputSchemasRejectUnknownProperties. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#5` `semantic_replace_construct`: internal/mcp/schema_test.go / assertScaffoldPurposeHeaderSchemaIsOptional. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#6` `semantic_replace_construct`: internal/mcp/schema_test.go / assertScaffoldPurposeHeaderSchemaIsOptional. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#7` `semantic_replace_construct`: internal/mcp/schema_test.go / assertScaffoldPurposeHeaderSchemaIsOptional. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#8` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPContinuationDescriptionsCoverConstructSources. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#9` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPContinuationDescriptionsCoverConstructSources. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#10` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPContinuationDescriptionsCoverConstructSources. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#11` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMoveConstructIsAdvertisedInBatchWithDestinationReceipt. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#12` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMoveConstructIsAdvertisedInBatchWithDestinationReceipt. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#13` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMoveConstructIsAdvertisedInBatchWithDestinationReceipt. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#14` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPStaleContinuationsAreNotAdvertised. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#15` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPStaleContinuationsAreNotAdvertised. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#16` `semantic_replace_construct`: internal/mcp/schema_test.go / assertMCPStaleContinuationsAreNotAdvertised. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#17` `semantic_replace_body`: internal/mcp/schema_test.go / TestMCPFullCatalogDescriptionsAndContinuations. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
* `500c4b67-6bab-4925-9e04-8fabb723b5cb`: All 13 children applied; final hook plugin mismatch.
  * `#1` `semantic_replace_body`: internal/mcp/server_test.go / TestMCPMavenTestFailureIncludesStructuredResult. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#2` `semantic_replace_construct`: internal/mcp/server_test.go / assertMCPMavenToolFailure. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#3` `semantic_replace_body`: internal/mcp/failure_contract_test.go / TestUnknownAndDisabledReloadToolCallsAreJSONRPCFailures. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#4` `semantic_replace_body`: internal/mcptest/validate.go / responseValidator.ensureFull. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#5` `semantic_replace_body`: internal/mcptest/validate.go / probeSchemas. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#6` `semantic_replace_body`: internal/mcptest/validate.go / compileSchema. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#7` `semantic_replace_body`: internal/mcptest/validate_test.go / TestCompileOutputSchemaReusesExactSchemaContentConcurrently. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#8` `semantic_replace_body`: internal/mcptest/validate_test.go / TestServerDefersCatalogDiscoveryUntilAToolResultNeedsValidation. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#9` `semantic_insert_construct`: internal/mcptest/validate_test.go / TestFullCatalogFallbackPreservesMutationOnlyProfile. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#10` `semantic_replace_construct`: internal/mcptest/validate.go / Server. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#11` `semantic_replace_body`: internal/mcptest/validate.go / Server.Initialize. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#12` `semantic_replace_body`: internal/mcptest/validate.go / NewServer. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#13` `semantic_replace_construct`: internal/mcptest/validate.go / compiledSchemaCache. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
* `df682d0d-2682-44c9-8081-5eec04fc38f8`: All 4 import children applied; final hook plugin mismatch.
  * `#1` `semantic_organize_imports`: internal/mcptest/validate_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#2` `semantic_organize_imports`: internal/mcp/failure_contract_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#3` `semantic_organize_imports`: internal/mcp/server_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#4` `semantic_organize_imports`: internal/mcp/schema_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
* `364d924b-8df4-45b2-b08d-1795c43b61ba`: All 20 children applied; final hook plugin mismatch.
  * `#1` `semantic_insert_construct`: internal/mcp/schema_test.go / decodeMCPResponses. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#2` `semantic_insert_construct`: internal/mcp/schema_test.go / assertMCPContinuationDescriptionBudget. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#3` `semantic_insert_construct`: internal/mcp/schema_test.go / assertMCPFaultContractDescriptions. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#4` `semantic_insert_construct`: internal/mcp/schema_test.go / assertMCPInlineVariableCatalog. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#5` `semantic_replace_body`: internal/mcp/schema_test.go / TestMCPFullCatalogDescriptionsAndContinuations. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#6` `semantic_insert_construct`: internal/mcp/schema_test.go / mcpIntegerCase. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#7` `semantic_replace_body`: internal/mcp/schema_test.go / TestMCPIntegerNotationAndRangeValidation. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#8` `semantic_insert_construct`: internal/mcp/schema_test.go / mcpIntegerRequests. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#9` `semantic_insert_construct`: internal/mcp/schema_test.go / assertMCPIntegerSession. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#10` `semantic_insert_construct`: internal/mcp/schema_test.go / assertMCPIntegerResponse. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#11` `semantic_replace_body`: internal/mcp/server_test.go / TestMCPMavenTestFailureIncludesStructuredResult. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#12` `semantic_replace_body`: internal/mcp/failure_contract_test.go / TestUnknownAndDisabledReloadToolCallsAreJSONRPCFailures. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#13` `semantic_insert_construct`: internal/mcp/failure_contract_test.go / assertMCPMethodNotFound. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#14` `semantic_replace_body`: internal/mcptest/validate_test.go / TestCompileOutputSchemaReusesExactSchemaContentConcurrently. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#15` `semantic_replace_body`: internal/mcptest/validate_test.go / TestServerDefersCatalogDiscoveryUntilAToolResultNeedsValidation. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#16` `semantic_insert_construct`: internal/mcptest/validate_test.go / assertDeferredCatalogDiscovery. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#17` `semantic_organize_imports`: internal/mcp/schema_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#18` `semantic_organize_imports`: internal/mcp/server_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#19` `semantic_organize_imports`: internal/mcp/failure_contract_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#20` `semantic_organize_imports`: internal/mcptest/validate_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
* `258b677c-90c8-4c39-bce8-7aa8529e39f2`: Both children applied; final hook plugin mismatch.
  * `#1` `semantic_insert_construct`: internal/mcp/language_snapshot_wire_test.go / TestSchemaValidationCapturesLanguagesBeforeFirstMutation. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#2` `semantic_organize_imports`: internal/mcp/language_snapshot_wire_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
* `95caae02-4f13-4c92-bb12-30a4bc00298b`: 11 children applied; child 12 hit stale LSP shutdown/exit deadlines; later descriptors skipped. Disk inspection preceded retry.
  * `#1` `semantic_replace_body`: internal/mcptest/validate.go / Server.Serve. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#2` `semantic_replace_body`: internal/mcptest/validate.go / Server.Initialize. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#3` `semantic_insert_construct`: internal/mcptest/validate.go / captureCatalogs. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#4` `semantic_remove_construct`: internal/mcptest/validate.go / responseValidator.ensureAdvertised. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#5` `semantic_remove_construct`: internal/mcptest/validate.go / responseValidator.ensureFull. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#6` `semantic_replace_body`: internal/mcptest/validate.go / responseValidator.setAdvertised. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#7` `semantic_replace_body`: internal/mcptest/validate.go / responseValidator.recordResponse. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#8` `semantic_replace_body`: internal/mcptest/validate.go / responseValidator.validateToolResult. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#9` `semantic_replace_construct`: internal/mcptest/validate.go / responseValidator. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#10` `semantic_replace_body`: internal/mcptest/validate.go / NewServer. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#11` `semantic_remove_construct`: internal/mcptest/validate_test.go / TestResponseValidatorRefreshesLiveCatalogAndInvalidatesFallback. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#12` `semantic_remove_construct`: internal/mcptest/validate_test.go / TestServerDefersCatalogDiscoveryUntilAToolResultNeedsValidation. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#13` `semantic_remove_construct`: internal/mcptest/validate_test.go / assertDeferredCatalogDiscovery. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#14` `semantic_insert_construct`: internal/mcptest/validate_test.go / TestLiveCatalogRefreshPreservesFrozenFallback. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#15` `semantic_replace_body`: internal/mcptest/validate_test.go / TestFullCatalogFallbackPreservesMutationOnlyProfile. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#16` `semantic_insert_construct`: internal/mcptest/validate_test.go / TestCatalogCaptureIsReusedUntilInitialization. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#17` `semantic_replace_body`: internal/mcp/language_snapshot_wire_test.go / TestSchemaValidationCapturesLanguagesBeforeFirstMutation. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#18` `semantic_insert_construct`: internal/mcp/language_snapshot_wire_test.go / assertFirstMutationSchemaSnapshot. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#19` `semantic_organize_imports`: internal/mcptest/validate.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#20` `semantic_organize_imports`: internal/mcptest/validate_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#21` `semantic_organize_imports`: internal/mcp/language_snapshot_wire_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
* `e275d94d-9b98-49df-a3af-4afb22447685`: All 9 remaining children applied; final hook plugin mismatch.
  * `#1` `semantic_remove_construct`: internal/mcptest/validate_test.go / assertDeferredCatalogDiscovery. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#2` `semantic_insert_construct`: internal/mcptest/validate_test.go / TestLiveCatalogRefreshPreservesFrozenFallback. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#3` `semantic_replace_body`: internal/mcptest/validate_test.go / TestFullCatalogFallbackPreservesMutationOnlyProfile. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#4` `semantic_insert_construct`: internal/mcptest/validate_test.go / TestCatalogCaptureIsReusedUntilInitialization. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#5` `semantic_replace_body`: internal/mcp/language_snapshot_wire_test.go / TestSchemaValidationCapturesLanguagesBeforeFirstMutation. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#6` `semantic_insert_construct`: internal/mcp/language_snapshot_wire_test.go / assertFirstMutationSchemaSnapshot. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#7` `semantic_organize_imports`: internal/mcptest/validate.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#8` `semantic_organize_imports`: internal/mcptest/validate_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#9` `semantic_organize_imports`: internal/mcp/language_snapshot_wire_test.go. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
* `59d4c129-93b6-4a41-b8e0-86ad38e65b0c`: Two removal descriptors submitted by audit worker; result marked failed. Individual persistence and failure detail are not recoverable from returned history; later source inspection confirmed consolidated coverage.
  * `#1` `semantic_remove_construct`: internal/mcp/schema_test.go / TestMCPToolDescriptionsCondenseFaultContracts. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).
  * `#2` `semantic_remove_construct`: internal/mcp/schema_test.go / TestInlineVariableToolSchemaAndContinuations. QoL and baseline follow the parent outcome; separate runtime unavailable (parent-scoped measurement only).

### Unexposed-operation attempts and manual fallbacks

* `U1`, worker `implement_validator`: `semantic_workspace_outline` was absent from the callable inventory; no mutation. **Harmful** discovery friction; fallback `rg --files` plus bounded file reads. Semantic runtime unavailable because the orchestration TypeError has no complete call timing; local runtime unavailable because that complete fallback was not timed. See the workspace-outline incident below.
* `U2`, same worker: `semantic_replace_body` was absent for `Server.Initialize`; no mutation through that attempt. **Harmful** friction; the worker used targeted source patches for validator/cache work and tests, with later semantic repairs and configured checks. Semantic runtime unavailable for the same reason; local patch/format/check workflow was not separately timed. See `docs/MANUAL_EDITS.md` and the body-replacement incident below.
* Wrap-up conflict resolution used atomic patches because source files contained Git markers. The source resolution retained main's `assertNavigationToolDescriptions` and this session's shared catalog helper. Documentation changes used ordinary atomic patches; no semantic failure is implied by documentation editing. No new semantic calls were made during wrap-up.

* Recovered call ratings: 66 beneficial, 2 neutral, 12 harmful, 0 unknown, plus U1/U2 harmful unavailable-operation attempts. Batch child ratings are associated with their parent rather than double-counted; skipped descriptors are not calls.
* Net agent efficiency: inspection and outlines reduced navigation preparation, and semantic rename preserved reference-aware guarantees. The complete editing workflow was worse than its successful-call count suggests: repeated selector failures, large verification diffs, trust negotiation, missing imports, and incompatible inferred hooks required retries and additional local checks. A verified direct-patch workflow would likely have required fewer round trips for these test-only changes, but that is a qualitative estimate, not a measured saving. Exact-schema caching and test-session consolidation improved the tests' CPU use; those product runtime measurements are distinct from agent-tool latency and are recorded in RQ-0020.
* Runtime conclusion: per-call semantic latencies above are measured from the history API. No comparable local baseline or complete end-to-end editing-workflow duration was measured, so semantic-tool speedup is unestablished. Navigation took milliseconds; rename and verification-bearing batches took seconds. No wrap-up benchmark campaign was run.
* Related incidents: trust-required numeric consolidation, declaration-rename rejection, statement/expression ambiguity, final-hook plugin mismatch, two-statement rejection, function-literal selector mismatch, stale diagnostic teardown, and check-only disagreement are retained below. The audit worker's failed removal batch is included above with its result-detail limitation rather than assigning an invented root cause.

## 2026-10-16: deferred closure statement selection during rebase integration

* Tool: one `semantic_batch` parent, first child `semantic_replace_construct`, targeting `internal/sourcebuf/buffer.go`, `Buffer.Flush`.
* Intent: replace `clear(b.symbolAliases)` and `clear(b.invalidatedSymbolAliases)` with nil assignments, then insert a buffer-reuse regression.
* Observed: first child rejected `clear(b.symbolAliases)` as not found; candidates included the containing deferred function literal but not its body statements. No source mutation occurred and later children did not run. Parent wall time: 12.6 seconds.
* Root cause: nested statements inside a function literal are outside the enclosing function's construct-selection traversal; caller selected an unsupported nested target.
* Workaround: targeted `apply_patch` for the lifecycle repair and regression during conflict integration; root performs canonical validation. No additional semantic calls. Hypothetical patch workflow needs one patch and formatting/checks; no paired runtime baseline was measured.

## 2026-10-16: Session wrap-up, warm MCP performance and artifact exclusions

* Context: `codex/warm-mcp-operation-costs`, chat `01a13c05-667b-7512-9144-22f9a2d08b53`. The available detailed history covers the final artifact-exclusion work. Earlier performance, ADR-audit, and child-agent calls are represented by the existing incident entries and RQ-0070; their complete per-call inventory is unavailable after compaction. This assessment does not claim exhaustive coverage of those earlier calls.
* Five recoverable attempts: two beneficial, two neutral, one harmful; no batch parents or children in this final phase. Wrap-up itself used no semantic calls.

| Calls | Tool / target / intent | Outcome / mutation state | Built-in alternative and agent efficiency | QoL / friction |
| :--- | :--- | :--- | :--- | :--- |
| A1 | `semantic_outline`, `internal/sourcefiles`, discover shared policy declarations | Complete structured outline; no mutation | `rg` declaration headers plus source reads would locate the small file with less output; structured ranges avoided manual coordinate extraction | Neutral: useful navigation, but source metadata was repeated in text and structured output by the installed server |
| A2 | `semantic_insert_construct`, `IsBenchmarkDataDirectory` | Helper inserted and reported unchanged diagnostics | A targeted patch plus formatting and checks could add the same declaration; the semantic path added arguments and feedback but avoided a manual source write | Beneficial: exact declaration insertion without repair |
| A3 | `semantic_insert_construct`, `IsIgnoredDirectoryPath` | Helper inserted and reported unchanged diagnostics | Same local patch/format/check alternative as A2; one semantic call handled insertion | Beneficial: exact declaration insertion without repair |
| A4 | `semantic_replace_construct`, nested `WalkDir` callback in `LooksLikeJavaWorkspace` | Selector rejected; no mutation | A targeted patch, formatting, and checks reached the intended condition change; the failed call and incident logging added round trips | Harmful: nested callback expression was not exposed by the enclosing-function construct search; related incident below |
| A5 | `semantic_remove_construct`, exported `IgnoredDirectories` | Automatic approval review rejected deletion; declaration retained | A source patch could remove it mechanically, but would not resolve the approval concern; retaining the API completes the exclusion task safely | Neutral: expected approval boundary, not a demonstrated editor defect; no bypass or removal fallback |

| Calls | Semantic runtime | Local built-in runtime | Scope / comparability |
| :--- | :--- | :--- | :--- |
| A1–A5 | Unavailable individually: orchestration timings include surrounding work and do not isolate each semantic call | Unavailable: no equivalent paired workflow was timed; live mutations were not replayed for measurement | Installed retained server, five attempts, no controlled cold/hot cohort. Existing RQ-0070 benchmark timings cover different operations and cannot supply a ratio for these calls |

* Agent efficiency: the two insertions were straightforward; repeated output and the nested-selector retry added friction. Evidence is insufficient to claim a net tool-call or runtime reduction over patch/format/check workflows. No invented time or token savings are assigned.
* Manual fallbacks: coordinated path-filter call sites, native Go/formatting configuration, bounded report intake, repository export selection, and regression fixtures are recorded in `MANUAL_EDITS.md`. Existing entries retain earlier verification-hook and selector failures rather than duplicating them here.

## 2026-10-16: Nested callback expression replacement

`semantic_replace_construct` could not locate `sourcefiles.IsIgnoredDirectory(entry.Name())` inside the `WalkDir` callback in `LooksLikeJavaWorkspace` (`internal/backend/java/workspace_outline.go`). Its enclosing-function construct search did not expose the nested callback expression. Used targeted patches for the directory-filter call-site migration; no broad source rewrite was needed.

`semantic_remove_construct` was rejected by automatic approval review when removing the no-longer-used exported `IgnoredDirectories` helper in `internal/sourcefiles/sourcefiles.go`, citing an unapproved destructive API change. The existing API was retained; no removal workaround was attempted. The artifact-boundary migration does not depend on deleting that helper.

## 2026-10-16: Import diagnostic-boundary batch publishes before lint-hook startup failure

* Tool: `semantic_batch`, target `internal/godiagnostics/manager.go`.
* Observed: The diagnostic-boundary migration published, then the external `golangci-lint` verification hook exited with status 3. The response included the large workspace result in text.
* Workaround: Inspect the published declarations without replaying edits; rebuild and verify through the canonical `make check` workflow.
* Root cause: Verification-hook startup failure is separate from publication; the active installed tool must report the partial outcome explicitly.
* Repeated while inserting `TestMCPImportCleanupRefreshesOnlyAffectedPackages` in `internal/mcp/gopls_call_metrics_test.go`: the test declaration published, then the same status-3 lint hook stopped verification. Continue with canonical checks without replaying the insertion.

## 2026-10-16: Coverage-audit semantic insertion verification failed

* Inspection during parallel file preparation: `semantic_inspect_symbol` could not find `TestJavaEditLifecycleDispatchesSave` in `internal/backend/java/edit_lifecycle_test.go` before the worker had inserted the declaration. No source was changed; wait for the worker's ready signal before inspecting the test.

* Tool: `semantic_batch` / `semantic_insert_function`, target `internal/godiagnostics/diagnostics_test.go`.
* Failure: The new regression was published, but the configured `golangci-lint` verification hook exited with status 3. The response repeated a large workspace diff and structured payload in text.
* Workaround: Inspect the applied declaration, fix the test through a scoped semantic replacement, then use the repository's `make check` rebuild and verification workflow.
* Root cause: External verification-hook startup failure; the published edit must be treated separately from verification success.
* Follow-up: `semantic_batch` / `semantic_replace_body` reported an unknown application state during orphan-import pruning while other audit files were being edited. Inspect disk before retrying; serialize mutations and use a targeted fallback if the body did not publish.
* Related CLI discovery: `.scratch/go/bin/semedit-local insert-function --help` reported an unknown command although root help lists the operation. Use the callable MCP batch operation instead.

## 2026-10-16: fingerprint regression insertion stops during retained-session synchronization

* Tool: `semantic_batch` / `semantic_insert_decl`.
* Target: `internal/godiagnostics/diagnostics_test.go`.
* Observed: embed-freshness test insertion reported success, then the next child failed with `synchronize Go refactoring state: replace stale Go LSP session`. The response classified the application state as unknown; subsequent inspection found both test declarations published.
* Workaround: preserved both published declarations without replay and verified the source through canonical checks and targeted real-gopls tests.
* Root cause: retained LSP replacement failed while synchronizing a source mutation; the underlying lifecycle error was not retained in the shortened response projection.

## 2026-10-16: function declaration replacement rejects a fingerprint signature change

* Tool: `semantic_batch` / `semantic_replace_decl`.
* Target: `internal/godiagnostics/manager.go`, `scopeFingerprint` and fingerprint helpers.
* Observed: three preparatory declarations were published; replacing the function declaration then failed with `unexpected declaration type`. The function replacement was not published.
* Workaround: preserved the inserted declarations, used `semantic_change_signature` for the six private helper signatures and their callers, then `semantic_replace_body` to wire the invocation-local map. No transitional nil caller remains; no source text patch was needed.
* Root cause: declaration replacement rejects this function declaration; the call should use the dedicated function-body/signature operations.

## 2026-10-15: Gemini bridge reports success with an empty response

* Tool: `gemini_ask`, separate normally sandboxed bridge after explicit source-sharing approval.
* Target: bounded `.scratch/gemini-investigation/review-bundle` containing relevant source and measurements.
* Observed: successful MCP result with a conversation ID and `response: ""`; no technical review was returned.
* Workaround: preserved the raw response and retried through the installed AGY CLI's documented text interface under the bridge's exact read-only Seatbelt profile. No technical review was returned.
* Root cause: the bridge accepts an empty backend response as success. The direct call separately exposes a TLS failure during `loadCodeAssist` eligibility checking: `SecPolicyCreateSSL error: 0`. AGY reports zero model turns and zero tokens. The reason for the initial empty stream result remains unproven. Sharing is explicitly approved; the remaining block is transport/runtime failure.

## 2026-10-15: type insertion requires one declaration per construct

* Tool: `semantic_insert_construct`.
* Target: `internal/gocache/cache.go`.
* Observed: rejected insertion containing two type declarations with `type insertion requires exactly one type declaration`; no source change occurred.
* Workaround: insert one type declaration per semantic operation, or use a narrow atomic patch when a supported semantic edit is unavailable.
* Root cause: the construct API accepts exactly one declaration per type insertion.

## 2026-10-15: declaration insertion puts an import after declarations

`semantic_batch` / `semantic_insert_declaration` refused adding the telemetry import to `internal/symbol/resolver.go`: inserted-file validation reported `imports must appear before other declarations`. The declaration tool's default insertion position is unsuitable for adding an import to an existing populated file. Use a targeted atomic import-block edit, then semantic statement insertion for the supported function spans; inspect the file before retrying to avoid replaying a partial mutation.

## 2026-10-15: protocol-tool allowlist edit publishes before hook failure

`semantic_batch` / `semantic_replace_decl` successfully added the existing `semantic_metrics` support tool to `mcpProtocolTools` in `internal/operation/skill_sync_test.go`, then final verification reported `golangci-lint` exit status 3. The edit remains applied and must not be replayed. Verify it through canonical `make check`; the reason for the configured MCP hook's exit status is not established by its returned error.

## 2026-10-15: published attribution test needs a targeted caller correction

The `semantic_batch` insertion into `internal/mcp/gopls_call_metrics_test.go` retained a caller-supplied unit test that passed `*testing.T` to the benchmark-only `warmSourceContains(*testing.B, ...)` helper. Canonical type checking caught the mismatch after the batch's external hook failed. Replace that call atomically with a direct fixture read and expected-source assertion. Root cause is the supplied test code, not an AST rewrite defect; do not replay the already-published insertion.

## 2026-10-15: gopls timing test batch publishes but verification hook fails

`semantic_batch` created `internal/mcp/gopls_call_metrics_test.go` and inserted its declarations successfully, then reported `golangci-lint` exit status 3 during final verification. Source publication remained applied. Preserve the published declarations and verify through canonical `make check` with the primary absolute Go cache; do not replay the batch. The MCP server's configured external-hook environment differs from the Make verification environment. No manual source repair was needed.

## 2026-10-15: scaffold purpose header requires literal Go comment syntax

* Tool: `semantic_batch` child `semantic_scaffold_file`.
* Target: `internal/mcp/gopls_call_metrics_test.go`.
* Observed: rejected the plain-text purpose header with `expected 'package', found Package`; no file was created.
* Workaround: retry with a literal `//` Go comment header.
* Root cause: the purpose header is parsed as source; the caller omitted the required comment delimiter.

## 2026-10-15: presentation test insertion blocked by stale Go session shutdown

`semantic_batch` renamed a presentation test and reported failure synchronizing the next insertion: replacing the stale gopls diagnostics session timed out during shutdown and exit. Inspection subsequently showed that the regression had also published despite the failed synchronization report. An atomic fallback briefly duplicated it; removed the duplicate and used a real JSON decoding error in the retained fallback test. Pinned repository checks verify the result without relying on this stale MCP session.

## 2026-10-15: compacted source predicate hook mismatch

`semantic_batch` extracted the compacted-source predicate to keep the shared presenter within its complexity limit. Its configured golangci-lint hook exited with status 3 after publication due to the known plugin/runtime mismatch. Retain published source and run pinned `make check`.

## 2026-10-15: shared presentation body helper hook mismatch

`semantic_batch` published the final `presentationBody` helper wiring in `internal/mcp/presentation/presentation.go`. Its configured golangci-lint hook exited with status 3 due to the existing plugin/runtime mismatch. Retain the edit and verify with pinned repository tooling.

## 2026-10-15: content helper extraction verification hook mismatch

`semantic_batch` extracted content body selection and bulk-field counting in `internal/mcp/presentation/presentation.go`, then rejected an if selector containing both initializer and condition. Earlier helper edits published. Retry with the exact reported initializer selector, and use pinned repository checks.

## 2026-10-15: final content projection verification hook mismatch

`semantic_batch` published the projection fixes for unavailable diagnostic deltas, bulk aliases, stable feedback omission, preserved native counts, staged candidate counts, and safe formatter fallback in `internal/mcp/presentation/presentation.go`. Verification then refused the configured golangci-lint hook because this batch did not carry request-local workspace trust; the retained language-server session also reported stale unrelated struct-field diagnostics. Preserve published edits and use pinned repository verification rather than treating this semantic verification as clean.

## 2026-10-15: feedback summary verification hook mismatch

`semantic_batch` published the compact feedback summary and structured-detail pointer. Its configured golangci-lint hook failed with the existing plugin/runtime mismatch. Preserve the edits and use pinned repository verification.

## 2026-10-15: statement path requires duplicate discriminator

`semantic_replace_construct` rejected the feedback call in `internal/mcp/server.go` when supplied only its reported construct path: the transport also requires a discriminator. No edits published. Use both exact coordinates; the content summary design remains unchanged.

## 2026-10-15: compact content projection hook mismatch

`semantic_batch` wired compact detail counts and routed feedback through the shared content presenter in `internal/mcp/presentation/presentation.go` and `internal/mcp/server.go`, then rejected the feedback call replacement as ambiguous between its statement and expression. The earlier edits published. Resume using the reported statement construct path and verify with pinned `make check`.

## 2026-10-15: content summary compound insertion rejected

`semantic_batch` inserted `compactPresentationDetails` in `internal/mcp/presentation/presentation.go`, then rejected a multi-statement insertion because statement insertion requires exactly one statement. The earlier insertion published. Resume with individual statements and body replacement; preserve native structured results. A scratch log patch also initially used a nonexistent anchor and was corrected without touching the target log.

## 2026-10-16: semantic insertion did not organize imports

* Tool: `semantic_insert_construct`.
* Target: `internal/mcp/buffers.go`, `goEmbedMatchesPath`.
* Observed failure: the function was inserted, but automatic import organization was unavailable and the required standard-library imports were not added.
* Workaround: add the imports with a targeted source patch and run `gofmt`.
* Root cause: the semantic tool completed the construct insertion without completing its requested import normalization.

## Function parameter type replacement rejected (2026-10-14)

* Tool: `semantic_replace_decl` through `semantic_batch`.
* Target: `internal/mcp/server_test.go`, `newManagedMCPFixture`.
* Observed failure: replacing the helper declaration to accept `testing.TB` returned `unexpected declaration type`; no source changes were published.
* Workaround: make the single parameter-type change with a targeted patch and atomic publication so benchmarks can share the existing real-gopls fixture.
* Root cause: Go declaration replacement does not accept a function declaration; the signature operation does not support this parameter-type migration.

## Scaffold purpose header requires comment syntax (2026-10-14)

* Tool: `semantic_scaffold_file` through `semantic_batch`.
* Target: `internal/mcp/warm_cost_bench_test.go`.
* Observed failure: a plain purpose sentence failed parsing with `expected 'package', found Package`; no file was published.
* Workaround: pass the purpose header with an explicit Go comment prefix.
* Root cause: the header is parsed as Go source rather than converted from plain prose to a comment.

## Benchmark insertion published but hook verification failed (2026-10-14)

* Tool: `semantic_batch` with scaffolding and declaration insertion.
* Target: `internal/mcp/warm_cost_bench_test.go`.
* Observed failure: all eight edits published, but the configured `golangci-lint` hook exited with status 3; Go diagnostic coverage was reported as verified.
* Workaround: retain the published edits and use repository `make check` with its configured tool environment to diagnose verification.
* Root cause: the hook failure has not yet been isolated; it is distinct from semantic insertion success.

The subsequent `semantic_replace_construct` update from p95 to p99 also published successfully, then reported the same hook exit status 3. Repository checks remain the verification path for the published benchmark code.

## Warm exported rename loses a package-name-mismatched import (2026-10-14)

* Tool: `semantic_rename` through the production MCP benchmark client.
* Target: twenty-package generated fixture, `consumer*/consumer.go` importing `example.com/mcpreuse` with declared package `sample`.
* Observed failure: the initial rename published edits, then reported nineteen introduced `undefined: sample` diagnostics during import normalization and verification.
* Workaround: use an explicit `sample` import alias in the baseline fixture and retain this result as a separate correctness regression to investigate.
* Root cause: not yet isolated; the failure involves the import path suffix differing from the declared package name. The initial read cases and one-package rename succeeded.

## Nested fixture-source statement replacement failed (2026-10-14)

* Tool: `semantic_replace_construct` through `semantic_batch`.
* Target: the `source` assignment inside `benchmarkWarmMCPCost` in `internal/mcp/warm_cost_bench_test.go`.
* Observed failure: statement selection returned an error and did not publish the explicit import alias.
* Workaround: apply a targeted one-line fixture-string patch with atomic publication.
* Root cause: not isolated; the attempted discriminator selected a nested assignment containing escaped Go source.

Splitting benchmark metric reporting into `reportWarmMCPCost` with `semantic_insert_function` and `semantic_replace_construct` published both changes, then repeated the same configured hook status 3. `make check` supplies the complete repository environment and verifies the resulting code.

## Benchmark timeout expression replacement unsupported (2026-10-14)

* Tool: `semantic_replace_construct` through `semantic_batch`.
* Target: `newWarmMCPCostClient` in `internal/mcp/warm_cost_bench_test.go`.
* Observed failure: changing the five-minute benchmark timeout with kind `expression` returned `unsupported Go construct kind "expression"`; no changes were published.
* Workaround: atomically apply a targeted timeout-constant patch for the 1,000-sample run.
* Root cause: the Go construct replacement backend does not accept the requested expression kind.

## File-rename benchmark batch verification hook failed (2026-10-14)

* Tool: `semantic_batch` with function insertion, body replacement, and file scaffolding.
* Targets: `internal/mcp/warm_cost_bench_test.go` and `internal/mcp/file_rename_bench_test.go`.
* Observed failure: all seventeen semantic edits published successfully; final verification reported the configured `golangci-lint` hook exiting with status 3.
* Workaround: retain the applied benchmark edits and run repository `make check` with the established Go environment.
* Root cause: the configured hook environment failure remains separate from semantic mutation success; repository checks provide full diagnostics.

Migrating all existing benchmark callers to the structured-result helper and deleting the obsolete void wrapper with `semantic_replace_body` and `semantic_remove_construct` also published both edits, then repeated the configured hook status 3. Repository checks remain the verification path for the completed migration.

## 2026-10-14: output policy semantic batch verification

`semantic_batch` applied six body replacements in `internal/mcp/presentation/presentation.go` and `internal/operation/wire_backend.go`, then returned an error because its configured golangci-lint verification hook requires workspace trust. The response also repeated a very large workspace scope and diff. Retained the published edits and used the repository's `make check` command for verification. The failure is verification-hook authorization after publication, not a failed source transformation.

## 2026-10-14: output policy contract test batch

`semantic_batch` applied the three changes in `internal/mcp/presentation_test.go` and `internal/mcp/output_contract_test.go`, then its trusted golangci-lint hook exited with status 3. Retained the applied changes and used `make check` with the canonical project-local Go paths. The hook environment remains incompatible with this worktree's shared cache setup; the source changes were published before verification failed.

## 2026-10-14: output family regression insertion hook failure

`semantic_batch` inserted the output-family presentation and rename-diagnostic regression tests in `internal/mcp/presentation/presentation_test.go` and `internal/operation/wire_backend_test.go`. The trusted golangci-lint verification hook exited with status 3 after publication. `make check` found that insertion had failed to add `encoding/json`, leaving an undefined `json` reference. Applied explicit import organization to repair the incomplete insertion. The insertion's automatic import discovery did not resolve the standard-library package.

## 2026-10-14: explicit import repair hook failure

`semantic_batch` successfully added `encoding/json` to `internal/operation/wire_backend_test.go` through `semantic_organize_imports`. Its post-publication golangci-lint hook again exited with status 3. Kept the applied import and verified through `make check`; the tool's diagnostic did not identify a source-level cause.

## 2026-10-14: output-key literal replacement rejected

`semantic_batch` inserted `outputActiveCountField` and migrated both rename schemas, but failed when `semantic_replace_construct` selected the `"active_count"` literal inside a map key in `projectRenameOutput`. The earlier edits remain published. Use whole-body replacement for the two small projectors to finish migration; a nested map-key literal is not a supported replacement slot in the active tool version.

## 2026-10-14: MCP output integration insertion hook failure

`semantic_batch` inserted `TestMCPActionableOutputHasNoSecondFormatterReport` in `internal/mcp/server_test.go`, then its golangci-lint hook exited with status 3. The insertion persisted. Retained it and used `make check` with canonical Go state to verify actual read, rename, and verification responses. The hook reported no actionable source diagnostic.

## 2026-10-14: batch-note conditional removal unsupported

`semantic_remove_construct` rejected `kind: if` while removing the formatter-separator-dependent note in `internal/mcp/batch.go`; no edits applied. The active tool supports fewer removal kinds than replacement kinds. Used a targeted atomic patch to replace the conditional prefix assembly with unconditional note attachment, preserving the existing intermediate-child warning after formatter bodies are removed.

## 2026-10-14: MCP integration assertion extraction hook failure

`semantic_batch` inserted `assertMCPActionableResponse` and simplified its caller in `internal/mcp/server_test.go`. Its trusted golangci-lint hook exited with status 3 after publication, without identifying a source error. Kept the edits and used repository `make check` with canonical Go paths. The separate check verifies the lint-driven refactor and observable MCP behavior.

## 2026-10-14: compact text contract migration hook failure

`semantic_batch` applied seven test changes across MCP server/schema/presentation tests and rename projector tests, then its golangci-lint hook exited with status 3 after publication. Retained the published changes and used canonical-path `make check`. The migrated tests decode compact JSON rather than depending on pretty-print whitespace or a second formatter body, and preserve the guarded inspection continuation test.

## 2026-10-14: final presentation semantic verification hook failure

`semantic_verify` with check-only scope `internal/mcp/presentation` returned failure when the auto-discovered `golangci-lint` hook exited with status 3. Its stderr identifies the cause: the `packageclosure.so` plugin was built with a different version of package `cmp` than the hook's linter. The retained gopls session also reported unrelated struct-literal errors outside the requested presentation directory; these are not treated as confirmed compiler failures. No source mutation occurred. Used the configured `make check` command, which invokes the pinned linter with canonical Go state, as the independent verification gate.

## 2026-10-14: diagnostic report scaffold parameter rejection

`semantic_scaffold_file` rejected the guessed `package_name` parameter while creating `internal/mcp/diagnostic_reports.go`. No source change occurred. Retried with the advertised `package` parameter. Root cause was an incorrect scaffold argument, not an AST transformation failure.

## 2026-10-14: diagnostic scaffold header rejected

`semantic_scaffold_file` rejected a plain prose `purpose_header` while creating `internal/mcp/diagnostic_reports.go`: parsing expected `package` but found the prose header. No source file was created. Use a Go comment header (`// ...`) and the canonical `package` argument. Root cause is that the scaffold header must already contain Go comment syntax.

## 2026-10-14: diagnostic dispatch case parameter rejection

`semantic_batch` rejected the diagnostic dispatch insertion before applying edits: `semantic_insert_case` accepts a complete `case` snippet, not separate `case_expr`/`body` arguments. Target was `internal/mcp/server.go`. Retried with the advertised `case` parameter; no source changes from the rejected batch were retained.

## 2026-10-14: diagnostic integration batch detected concurrent source change

`semantic_batch` reported intermediate success for the Server cache field, constructor initialization, and diagnostic tool catalog entry, then stopped during `semantic_insert_case`: another worker changed `internal/mcp/diagnostic_reports.go` while the source buffer was active. A subsequent source inspection showed those three intermediate changes had not been published. Paused concurrent writes, resumed the remaining edits, and applied the three missing integrations with a small atomic patch. Root cause is whole-workspace source-buffer synchronization detecting concurrent changes; intermediate success rows did not prove publication in this failure path.

## 2026-10-14: diagnostic capture batch hook mismatch

`semantic_batch` applied diagnostic dispatch, metadata schema wiring, batch capture, and native-result capture helpers. The post-publication `golangci-lint` hook exited with status 3 due to the previously identified linter/plugin Go version mismatch. Retained all source edits and used repository `make check` with canonical Go state for verification.

## 2026-10-14: known-error diagnostic metadata anchor not found

`semantic_insert_construct` could not find the multiline `structured` map declaration in `Server.sendOperationToolErrorWithTiming` from its compact statement discriminator. No edits in this batch applied. Used a minimal atomic patch at that unique declaration for the metadata attachment, then resumed the independent helper edits. Root cause is source-sensitive statement anchoring for a formatted multiline map literal.

## 2026-10-14: diagnostic capture helper factoring hook mismatch

`semantic_batch` published the native diagnostic capture helpers and simplified result dispatch, then the configured golangci-lint hook exited with status 3 due to the known plugin/runtime version mismatch. Retained published edits and verified via the pinned repository check.

## 2026-10-14: diagnostic paging integration test hook mismatch

`semantic_batch` published the real MCP diagnostic paging integration test and its transport helpers in `internal/mcp/server_test.go`. The configured golangci-lint hook again exited with status 3 from the known plugin/runtime mismatch. Retained the changes and used the pinned `make check` gate plus targeted production MCP tests.

## 2026-10-14: diagnostic metadata normalization hook mismatch

`semantic_batch` applied diagnostic metadata normalization so absent coverage is an empty array and uncaptured reports cannot expose a live query pointer. The configured golangci-lint hook exited with status 3 from the known plugin/runtime mismatch after publishing both changes. Retained changes and used repository checks.

## 2026-10-14: populated diagnostic page benchmark hook mismatch

`semantic_batch` scaffolded and inserted `internal/mcp/diagnostic_page_bench_test.go`, measuring actual MCP pages with at least 50 captured diagnostics and asserting one gopls process. Its post-publication golangci-lint hook exited with status 3 due to the known plugin/runtime mismatch. Retained the benchmark and verified with pinned repository tooling.

## 2026-10-14: diagnostic utility contract migration hook mismatch

`semantic_batch` updated transport utility count/schema/failure tests for `semantic_diagnostics`. The configured golangci-lint hook exited with status 3 due to the known plugin/runtime mismatch after publication. Retained the changes; completed the small unsupported map-literal inventory additions with atomic patches and verified through pinned repository checks.

## 2026-10-14: diagnostic filter continuation hook mismatch

`semantic_batch` added the active filter to diagnostic page results and verified that companion in the production MCP test, so callers can preserve it when following `next_offset`. The configured golangci-lint hook exited with status 3 after publishing both changes due to the known plugin/runtime mismatch. Retained edits and used pinned repository checks.

## 2026-10-15: diagnostic page presentation optimization hook mismatch

`semantic_batch` moved field ordering priorities to one shared lookup and skipped outline serialization for result maps without scope. These edits remove repeated per-comparison map construction and an irrelevant JSON round trip from diagnostic page rendering. Its configured golangci-lint hook exited with status 3 due to the known plugin/runtime mismatch after publication. Retained the optimization and verified behavior and latency with pinned repository tooling.

## 2026-10-15: wrapped mutation diagnostic capture hook mismatch

`semantic_batch` added shared diagnostic edit evidence for extraction and structural-result wrappers, migrated both MCP metadata consumers, and made uncaptured mutation inventories explicitly unavailable. Its configured golangci-lint hook exited with status 3 after publication due to the known plugin/runtime mismatch. Retained changes and used pinned repository checks.

## 2026-10-15: diagnostic evidence documentation hook mismatch

`semantic_batch` documented the exported diagnostic evidence boundary and removed an unused report timestamp field. The configured golangci-lint hook exited with status 3 after publication due to the known plugin/runtime mismatch. Retained changes and verified with pinned repository tooling.

## 2026-10-15: duplicate file target in documentation patch

`apply_patch` rejected a documentation update because one patch contained two update sections for `docs/adr/README.md`. No file was changed by that call. Combine the index edits into one section and reapply the targeted documentation patch. The cause was patch structure, not repository content.

## 2026-10-15: warm MCP benchmark method selector mismatch

`semantic_batch` could not resolve `semantic_replace_body` for method `call` in `internal/mcp/warm_cost_bench_test.go`. No source was changed by that call. I will use the smallest targeted patch for this benchmark-only method update; the selector did not match the receiver method as expected.

## 2026-10-15: warm benchmark worktree path mismatch

`semantic_outline` failed to inspect `internal/mcp/warm_cost_bench_test.go` because its server resolved the relative path in the primary checkout, where the benchmark file does not exist. No source changed. Use the absolute worktree path for supported semantic calls and a targeted atomic patch if the semantic server cannot operate on the worktree.

## 2026-10-15: source-edit benchmark runner initialization

`semantic_batch` inserted eight declarations into `internal/mcp/cold_hot_file_edit_runner_test.go`, but final verification failed because the configured `golangci-lint` hook required workspace trust. The inserted declarations also lacked their required imports. The batch reported unrelated stale workspace diagnostics. Preserve the applied declarations, add the explicit import block with a targeted atomic patch, and use the canonical `make check` with the primary Go cache. The automatic import integration did not initialize imports for this multi-declaration new-file batch. A documentation-log patch also missed the existing heading; the log was appended to the temporary copy instead.

## 2026-10-15: benchmark filter guard selector ambiguity

`semantic_replace_construct` could not uniquely select the nested `b.Fatalf` call in `BenchmarkMCPColdHotToolCosts` when changing no-match behavior for the added source-edit registry. No source was changed. Use a targeted atomic guard patch after the active check finishes; the statement/expression selector matched multiple nested spans.

## 2026-10-15: source-edit benchmark fixture shape rejections

The real MCP smoke run rejected `semantic_insert_field` on disposable `cost_insert_field.go` because the struct used a single-line field layout, and rejected `semantic_extract_type` on `cost_extract_type.go` because the selected field did not occupy its own line. Neither operation produced an edit. Correct the fixture sources to use multiline fields matching the supported contract; preserve fail-closed behavior. Inline operations returned explicit `int(2)` conversions, so their benchmark source assertions must accept the operation's actual formatted AST result.

## 2026-10-15: benchmark variant label insertion

`semantic_insert_construct` failed to add a distinct element-variant row label in the source-edit benchmark table. No intended source edit was confirmed. Inspect the on-disk function before applying the smallest atomic fallback; the benchmark's timing and operation contracts do not depend on this presentation change.

## 2026-10-15: repository-corpus safe deletion refuses nested modules

`semantic_remove_construct` with `safe_delete: true` refused the unused `costObsolete` probe in the disposable repository corpus because its complete module source scan encountered the existing nested Go module. It reported `application_state: not_applied`; no edit was performed. Retain this restriction as an explicit benchmark finding. A plain file-local removal can be measured separately with the supported unchecked-delete mode and verified afterwards; do not silently claim a safe-delete measurement for that mode.

## 2026-10-15: repository-root language inference selects Make for Go edits

The repository-corpus MCP smoke run selected language `make` for several Go source-edit requests (including `semantic_insert_field`, `semantic_extract_type`, and `semantic_set_doc_comment`) despite their `.go` target files. Dispatch refused those edits without mutation. The repository root contains a Makefile, unlike the small Go fixture. Benchmark Go operations with explicit `language: go` where the registered schema offers it, including batch children; retain this inference issue as a separate product finding rather than attributing a failed dispatch to edit latency.

## 2026-10-15: repository-corpus structural reads reach compiler cache artifacts

`semantic_inline_function` and `semantic_migrate_type` refused repository-corpus edits because semantic source reads encountered Go compiler-cache artifacts under the inherited primary GOCACHE, outside the disposable workspace. The small-corpus versions succeed. Keep the full repository and inherited cache configuration unchanged and report these native failures without edit latency percentiles; do not hide the issue by dropping source packages or compiler inputs from the corpus.

## 2026-10-15: semantic workspace outline unavailable in worktree session

The semantic tool inventory did not expose `semantic_workspace_outline`, and attempting to call it returned a missing-tool error before any workspace inspection or edit. Use the repository's documented source tools to inspect the assigned declarations, then make targeted atomic edits because no callable semantic editing tool is available in this session.

## 2026-10-15: semantic outline parameter schema mismatch

`semantic_outline` rejected the documented `file` argument before inspecting `internal/godiagnostics/manager.go`; its runtime schema accepts `path`. No source changed. Retry with the supported argument name and absolute worktree path.

## 2026-10-15: semantic batch child parameter nesting

`semantic_batch` rejected the first child because its `tool` discriminator was mistakenly nested inside `params`. No source edit was applied. Retry using the supported flat child form with `tool` and operation parameters as siblings.

## 2026-10-15: semantic statement insertion method selector

`semantic_batch` could not resolve the unqualified method selector `use` in `internal/godiagnostics/manager.go`. No source edit was applied. Retry with the receiver-qualified selector reported by semantic outline.

## 2026-10-15: Go tracing instrumentation verification hook trust

`semantic_batch` applied the named phase insertions across `internal/godiagnostics/manager.go`, `internal/godiagnostics/diagnostics.go`, and `internal/gocache/cache.go`, then its configured `golangci-lint` hook refused verification because workspace trust was not granted. Preserve the published edits, inspect the final diff, and rely on the root task's canonical checks.

## 2026-10-15: telemetry import organization hook trust

`semantic_batch` ran `semantic_organize_imports` on the three traced Go files, then its configured `golangci-lint` hook again refused verification because workspace trust was not granted. Retain the import edits if present and use the canonical repository check for final validation.

## 2026-10-15: concurrent package edits block semantic request context wiring

`semantic_batch` targeting `internal/mcp/server.go` and `batch.go` refused publication because another worker changed `internal/gocache/cache.go` after its workspace snapshot. Neither requested edit was published. The workaround is a narrow atomic patch to the context wiring; the failure comes from whole-workspace snapshot validation during concurrent work.

## 2026-10-15: semantic lookup used a nonexistent receiver-qualified function

`semantic_lookup` could not resolve `SessionManager.updatePackageFindingCache` in `internal/godiagnostics/manager.go`; the target is a package-level helper named `updatePackageFindingCache`. No source edit was attempted through the tool. Retry lookup with the actual function name, then use the smallest supported semantic mutation or an atomic patch if the helper's control flow is not a supported selector.

## 2026-10-15: repository attribution benchmark assumes a clean hook-free fixture

`semantic_verify` on the archived repository fixture failed with workspace trust
required for the configured `golangci-lint` hook. The request used `skip_hooks`,
which is not a supported verification bypass. The archived repository also has
existing warnings and informational diagnostics, so the synthetic fixture's
zero-diagnostics oracle is inappropriate. The benchmark must disable external
hooks explicitly in its disposable fixture and validate retained baseline
findings rather than interpret existing warnings as mutation failures. No timed
repository observation was accepted from this failed setup.

## 2026-10-15: Gemini repository consultation rejected by configured root

`gemini_ask` targeting the warm MCP operation-cost worktree was rejected before
analysis because its configured synthetic root is a different repository's
`testdata` directory. No review occurred and no project files were changed.
The immediate workaround is to inspect available Gemini capabilities and, if a
live backend is usable, supply a bounded source excerpt without requesting
access outside its configured root. Do not treat a fixture response as an
independent model review.

A separate invocation using the bridge's normal read-only sandbox was rejected
by automatic approval review because exporting the repository source and
performance payload to Gemini had not been explicitly approved. No external
review ran. Explicit sharing approval is pending; local investigation continues.

## Review cache invalidation implementation

* Tool: semantic_batch / semantic_insert_declaration, target `internal/godiagnostics/manager.go`.
* Failure: declaration insertion published a helper referencing `build` and `io` without adding its required imports; verification reported undefined identifiers.
* Workaround: add the two required imports with a targeted atomic patch.
* Root cause: declaration insertion with import organization disabled does not infer imports.

* Tool: semantic_batch / semantic_remove_construct, target `internal/godiagnostics/manager.go`.
* Failure: removing a complete helper using `kind: declaration` was rejected as unsupported before publication.
* Workaround: use supported body/declaration mutations separately; remove the obsolete helper using a targeted atomic patch.
* Root cause: construct removal does not support whole declaration removal through this schema.

## 2026-10-15: Symbol inspection queried a nonexistent declaration

* Tool: `semantic_inspect_symbol`, target `internal/operation/wire_engine_imports.go`.
* Failure: lookup of guessed symbol `organizeImports` returned “symbol not found.”
* Workaround: inspect the established output contract in the wire declarations and tests instead.
* Root cause: the operation does not declare a symbol with that guessed name.

## 2026-10-16: Concurrent unrelated files invalidate semantic source-buffer edits

* Tool: `semantic_batch` (`semantic_replace_body`, `semantic_set_doc_comment`).
* Target: `internal/sourcebuf/buffer.go`, `New` and `Buffer.AuthorizeWrites`.
* Failure: Changes remained unpublished because a parallel edit to `internal/godiagnostics/diagnostics_test.go` invalidated the tool's workspace-wide structural read set; `buffer_publish` failed with an identity/permissions conflict.
* Workaround: Inspect disk state before retrying and serialize semantic publications. Apply only the smallest fallback if unrelated concurrent work continues to invalidate the tool.
* Root cause: The installed semantic server still captures and checks unrelated workspace source files, the behavior this implementation is replacing.

## 2026-10-16: Semantic function replacement unavailable

* Tool: `semantic_replace_body`, target `internal/godiagnostics/diagnostics_test.go`, `TestPublishedOverlayFlushSynchronizesFromDisk`.
* Failure: The mutator was listed as supported by the semantic-tool schema, but no callable standalone implementation was loaded in the active tool inventory.
* Workaround: Use the callable `semantic_batch` mutation path if it can resolve the same operation; otherwise use a minimal built-in patch.
* Root cause: The active MCP tool inventory did not load the standalone body-replacement tool.
* Follow-up failure: The callable batch path rejected the guessed `source` field; its schema requires `body`.
* Follow-up workaround: Retry the same semantic replacement using the declared `body` parameter, then use a targeted atomic patch only if publication remains unavailable.
* Follow-up failure: The replacement was applied, but batch verification stopped because the workspace trust setting for the `golangci-lint` hook was not enabled.
* Follow-up workaround: Inspect the applied declaration and imports through semantic inspection, then leave test execution to the authorized repository workflow.
* Root cause: Automatic verification hooks require explicit workspace trust.
* Repeated during regression strengthening: `semantic_batch` applied the updated test body, then the same trust-gated `golangci-lint` hook prevented verification.
* Repeated during test cleanup: `semantic_batch` applied the helper-based test body, then verification was again stopped by the same trust-gated hook.
* Repeated while extracting regression helpers: `semantic_batch` published the helper declarations but verification again stopped at the same workspace trust gate.
* Repeated while adjusting the shared overlay buffer: `semantic_batch` published the body updates, but verification was stopped by the same `golangci-lint` trust gate.
* Repeated while changing the wrapper positive control: `semantic_batch` published the helper body, but verification was stopped by the same `golangci-lint` trust gate.
* Repeated while switching diagnostics calls to the retained `SessionManager.CheckDetailed` entry point: `semantic_batch` published the body changes, but its lint hook was called without the repository trust flag and stopped verification.
* Tool: `golangci-lint run ./internal/godiagnostics`, target `internal/godiagnostics/diagnostics_test.go`.
* Failure: Lint startup failed before analysis because the `packageclosure.so` plugin was built with a different Go package version (`cmp`).
* Workaround: Run a focused standalone lint check without the stale custom analyzer if supported; the repository's clean canonical rebuild remains with the parent.
* Root cause: The managed Go build cache contains a stale plugin artifact.
* Follow-up failure: A standalone `golangci-lint` run with built-in analyzers also failed before analysis because its type checker cannot decode Go 1.27 export data (`cmp`, export version 5; maximum supported version 4).
* Workaround: Focused Go test compilation and execution passed; the parent's canonical toolchain rebuild is needed for repository lint.
* Root cause: The installed lint tool is older than the Go toolchain configured by the inspection environment.

## 2026-10-16: Construct insertion rejected a multi-statement snippet

* Tool: `semantic_insert_construct`, target `internal/sourcebuf/lazy_capture_test.go`, `TestLazyCaptureIgnoresUntouchedMixedTreeAndSymlinks`.
* Failure: Inserting two independent buffered writes before publication failed with “expected exactly one statement.” No source changes were published.
* Workaround: Apply the requested two statements with a minimal atomic patch, then inspect the diff and leave verification to the parent workflow.
* Root cause: Statement insertion validates a snippet as exactly one statement, so the compound insertion exceeded the operation's supported shape.

## 2026-10-16: Function extraction rejected a multi-declaration region

* Tool: `semantic_batch` / `semantic_extract_function`, target `.scratch/performance-candidate/internal/godiagnostics/collector_failure_test.go`, `TestManagedDiagnosticsCollectorFailureDoesNotReturnPartialFindings`.
* Failure: Extraction of the retry and unchanged-capture checks was rejected with “only one leading := declaration with matching named results is supported.” No source changes were published.
* Workaround: Split the test using a small explicit helper and simplify the fake gopls loop with a request-response helper.
* Root cause: The extraction tool does not support the selected region's multiple local declarations.

## 2026-10-16: Semantic scaffold rejected an un-commented purpose header

* Tool: `semantic_batch` with `semantic_scaffold_file`, target `internal/godiagnostics/import_refresh_test.go`.
* Failure: Scaffolding rejected `purpose_header` beginning with `WHY:` as invalid Go syntax; no file was created.
* Workaround: Retry scaffolding with a valid Go comment in `purpose_header`, then insert the test declaration semantically.
* Root cause: The scaffold operation expects `purpose_header` to contain Go comment syntax.
* Follow-up failure: `semantic_batch` created the test declaration, then automatic verification stopped at the untrusted `golangci-lint` hook.
* Follow-up workaround: Inspect the published declaration and complete required imports with the smallest applicable semantic mutation; leave verification to the parent `make check` run.
* Root cause: Batch verification attempts configured hooks and requires explicit workspace trust for `golangci-lint`.
* Repeated during import completion: `semantic_batch` added the required `testing` import, then stopped at the same trust-gated hook.
* Workaround: Keep the completed import edit and leave all validation to the parent `make check` run.
* Root cause: The batch verification configuration still requires workspace trust even for a test-only import change.

## 2026-10-16: Method replacement rejects unchanged method identity in metadata-reuse candidate

`semantic_batch` targeting `.scratch/performance-candidate/internal/godiagnostics/manager.go`
published the `Session` field and `scopeLoadResult` type, then rejected replacing
`SessionManager.loadScope` with the same receiver and method name:
`replacement must preserve the selected function symbol "SessionManager.loadScope"`.
The intended change only alters its return type and implementation. The batch
stopped before later edits. Use a targeted signature patch and body replacement
instead; do not replay the already-published type edits. The method-symbol
preservation check appears to compare a qualified selector against the unqualified
replacement declaration name. No baseline engine sources were changed.

## 2026-10-16: Collector failure test body replacement parameter mismatch

* Tool: `semantic_replace_body` through `semantic_batch`.
* Target: `.scratch/performance-candidate/internal/godiagnostics/collector_failure_test.go`.
* Failure: the batch rejected the `function` parameter; the required selector is `symbol`. No edit was applied.
* Workaround: retry with the documented `symbol` parameter.
* Root cause: the caller used another operation's parameter name.

## 2026-10-16: Unsupported expression replacement during metadata-error guard

`semantic_batch` inserted the two diagnostic-scope helpers in the isolated
performance candidate, then `semantic_replace_construct` rejected Go kind
`expression` in `SessionManager.prepareSession` as unsupported. The helpers
remained applied. Use a targeted condition patch for that unsupported operation;
the root cause is the installed tool's construct-kind capability.

## 2026-10-16: Multi-function semantic insertion rejected

* Tool: `semantic_insert_construct` through `semantic_batch`.
* Target: `internal/backend/golang/lookup_overlay_test.go` in the active worktree.
* Failure: the final batch edit contained two function declarations; the tool requires exactly one and applied neither test function. Earlier outline implementation and caller edits in the same batch had already succeeded.
* Workaround: insert each test function in its own semantic edit after verifying the partial batch state.
* Root cause: the insertion operation accepts a single function declaration per request.

## 2026-10-16: Stale Go session blocked buffered-outline test insertion

* Tool: `semantic_insert_construct` through `semantic_batch`.
* Target: `internal/backend/golang/lookup_overlay_test.go` in the active worktree.
* Failure: insertion of the buffered-source test failed while synchronizing semantic refactoring state; shutdown and exit of the stale gopls session exceeded the deadline. Earlier edits in the same batch remained applied; the buffered test was not inserted.
* Workaround: verify the partial state, then retry only the missing one-function insertion after the stale session is cleared.
* Root cause: the semantic server retained a Go LSP session that did not shut down within its deadline.

## 2026-10-16: Import batch verification required workspace trust

* Tool: `semantic_organize_imports` through `semantic_batch`.
* Target: the Go outline implementation and its tests in this worktree.
* Failure: after applying the import organization, final verification refused to run the configured `golangci-lint` hook because the workspace had not been trusted.
* Workaround: inspect the persisted imports and report the trust gate; no tests or checks were run separately.
* Root cause: the selected verification hook is trust-gated by the semantic tool.

## 2026-10-16: Semantic workspace outline was unavailable through the deferred tool bridge

* Tool: `semantic_workspace_outline`, target `internal` in the active worktree.
* Failure: the discovered deferred tool was not callable through the available MCP bridge (`TypeError: tools.mcp__semedit__semantic_workspace_outline is not a function`). No source changes were published.
* Workaround: use focused read-only source inspection and minimal atomic patches for the supported change.
* Root cause: the enabled deferred semantic-tool inventory exposed the batch tool but not a callable workspace-outline function.

## 2026-10-16: Symbol inspection rejected the wrong file parameter

* Tool: `semantic_inspect_symbol` in the active worktree.
* Target: `internal/astedit/extraction_analysis.go`.
* Failure: the call supplied `path`; the tool requires `file` and rejected the request before inspecting the declaration.
* Workaround: retry with the documented `file` parameter.
* Root cause: I used the outline-style path parameter for symbol inspection.

## 2026-10-16: Extraction specialization batch could not complete lint verification

* Tool: `semantic_batch` applying extraction loader mode specialization.
* Target: `internal/astedit/extraction_analysis.go`, its loader callers, and the full selected workspace lint scope.
* Failure: all requested semantic edits were applied, but the configured `golangci-lint` verification hook exited with status 3, so the batch reported verification failure.
* Workaround: inspect the changed declarations and call sites; the parent agent will run canonical `make check` after integrating the active changes.
* Root cause: the semantic batch lint hook could not verify this already-dirty multi-agent workspace.

## 2026-10-16: Extraction telemetry batch left imports unresolved after lint failure

* Tool: `semantic_batch` adding extraction timing phases and a public regression.
* Target: `internal/astedit/extract.go` and `internal/astedit/extraction_analysis.go`.
* Failure: requested code changes were applied, but auto import organization did not add the new telemetry imports; the workspace lint hook also exited with status 3.
* Workaround: run the semantic import organizer on the two changed Go files and inspect the resulting imports.
* Root cause: batch auto import organization did not resolve newly introduced package identifiers in this compound edit.

## 2026-10-16: Semantic import organizer did not add extraction telemetry imports

* Tool: `semantic_organize_imports` through `semantic_batch`.
* Target: `internal/astedit/extract.go` and `internal/astedit/extraction_analysis.go`.
* Failure: the organizer left the new `telemetry` references unresolved, and final workspace lint verification exited with status 3.
* Workaround: add only the missing internal telemetry imports with a targeted patch.
* Root cause: the semantic organizer did not resolve imports for the newly inserted identifier references in this dirty workspace.

## 2026-10-16: Follow-up lint repairs after semantic outline and extraction batches

* Tool: `semantic_batch` source edits whose final lint hooks failed.
* Targets: `internal/astedit/extract_test.go` and `internal/backend/golang/lookup_overlay_test.go`.
* Failure: persisted regression tests shadowed an existing `err`; the canonical check rejected them after the failed tool-level verification.
* Workaround: targeted patches named the extraction, authorization, and application errors separately before rerunning the repository fixers and checks.
* Root cause: inserted bodies did not account for the repository's enabled shadow analysis, and failed post-edit hooks left those edits applied.

## 2026-10-16: Compiler-validation regression edit reported a failing workspace hook

* Tool: `semantic_batch` with `semantic_replace_body`.
* Target: `internal/astedit/extract_test.go:TestExtractFunctionRejectsTypeErrorsWithoutWriting`.
* Failure: the targeted test replacement persisted, but the tool returned a `golangci-lint` hook failure with exit status 3 and a workspace-wide verification scope.
* Workaround: inspect the persisted test, then run the repository's canonical check with its pinned local linter and isolated test temporary directory.
* Root cause: tool-level hook execution did not match the successful canonical verification environment; the response still included the duplicated critical JSON payload under investigation.

## 2026-10-16: Runtime-filter semantic edits needed a constant cleanup

* Tool: `semantic_batch` source edits for runtime-directory filtering.
* Target: `internal/gocache/cache.go`.
* Failure: the persisted predicate increased the repeated `GOMODCACHE` key beyond the configured constant-analysis threshold; tool verification had failed before identifying this actionable lint result.
* Workaround: consolidate that environment key into one constant and rerun the canonical fixers and checks.
* Root cause: the semantic insertion did not consolidate an existing repeated configuration key.

## 2026-10-16: Source filter test insertion hit a stale gopls session

* Tool: `semantic_insert_construct` through `semantic_batch`.
* Target: `internal/sourcebuf/buffer_test.go`.
* Failure: the predicate, source filter fast path, and two earlier test insertions applied, but the next test insertion failed while replacing a stale Go LSP session; shutdown exceeded the context deadline.
* Workaround: inspect the partial batch state and retry only the missing test declarations after the semantic server recovers.
* Root cause: the retained gopls diagnostics session in this multi-agent workspace did not shut down within its deadline.

## 2026-10-16: Source filter regression batch could not complete lint verification

* Tool: `semantic_batch` inserting the seed-without-base regression.
* Target: `internal/sourcebuf/buffer_test.go` and the selected workspace lint scope.
* Failure: the regression insertion applied, but the configured `golangci-lint` verification hook exited with status 3.
* Workaround: inspect the persisted test and leave canonical verification to the parent agent's `make check` gate.
* Root cause: the tool-level lint hook could not verify the multi-agent workspace.

## 2026-10-16: Source filter fast-path batch could not complete lint verification

* Tool: `semantic_batch` replacing the explicit-runtime-roots regression body.
* Target: `internal/sourcebuf/buffer_test.go` and the selected workspace lint scope.
* Failure: the strengthened regression persisted, but the configured `golangci-lint` verification hook exited with status 3.
* Workaround: inspect the persisted test and leave canonical verification to the parent agent's `make check` gate.
* Root cause: the tool-level lint hook could not verify the multi-agent workspace.

## 2026-10-16: session wrap-up, actionable symbol suggestions

Context: chat `01a14595-6049-7143-9408-ba2a0d055060`, detached base
`1ff9655e`, linked `ios-4216a269` checkout. The wrap-up skill was read from
local `main`. The history API returned six turns with no older page and two
compaction boundaries. It exposed seven direct semantic attempts below; child
agent implementation calls and individual operations executed inside CLI test
processes are not recoverable as a complete call inventory. Those omissions
prevent an exhaustive session-wide count. Catalog discovery is not execution.

Repeated wrap-up reconciled this branch with current main and preserved the
exempt destination cleanup-log update in `6d3861b`. No additional semantic
operations or manual source edits occurred in these follow-up turns. The seven
recovered-call inventory and paired-timing limitations below remain unchanged.

Stable references below are history execution IDs with their `exec-` prefix
omitted. Totals for the seven recovered calls: four beneficial, two neutral,
one harmful; no recovered semantic batch parents or children. No semantic
operations were invoked solely for wrap-up.

| Calls | Tool / target / intent | Outcome / mutation state | Built-in alternative / agent efficiency | QoL / friction / cause |
| :--- | :--- | :--- | :--- | :--- |
| `abfd141c-dbed-4432-9a98-bc89356dbf57` | `semantic_outline`, `internal/operation`, discover declarations | Completed, read-only; historical output unavailable | `rg --files` plus declaration searches and file reads; measured comparative effort unavailable | Neutral: structured navigation completed, but retained evidence cannot establish savings |
| `5bf48e56-2c9e-4d70-883e-5e18681f1cb8` | `semantic_inspect_symbol`, Go reference backend `FindReferences` | Completed, read-only; historical output unavailable | `rg -n` plus file reads locate and inspect the declaration; compiler-backed resolution would require additional tooling | Neutral: useful inspection intent, insufficient evidence for an efficiency advantage |
| `416b6478-ef7c-495f-a1e6-ad9e27ffdbb0` | `semantic_insert_construct`, generic validation callback | Automatic approval rejected before mutation; abandoned | A patch would also require safety review and could not make invoking a full operation a valid non-mutating preflight | Beneficial safety rejection; no runtime or effort saving claimed. See callback incident below |
| `9c7a2f54-1657-48fb-a2e7-be0806e9d7bf` | `semantic_insert_construct`, signature-constraint helper | Function inserted and automatic diagnostics unchanged | One `apply_patch`, formatting, and checks; semantic insertion supplied AST placement and immediate diagnostics | Beneficial correctness feedback; no paired agent-efficiency measurement |
| `3239992f-8d09-4937-9b76-995b1d25d14a` | `semantic_insert_construct`, two retrieval-context statements | Rejected before mutation: exactly one statement required | One patch could insert both statements, followed by formatting and checks | Harmful workflow friction from a caller snippet error, not corrupted AST output; see multiple-statements incident |
| `87223a5d-4d3c-4d31-ac8d-3c2e1e2a139c` | `semantic_insert_construct`, signature query context | Applied one statement; automatic diagnostics unchanged | Patch the anchor, format, and check | Beneficial validated placement; retry costs must be included in the complete workflow |
| `d8717943-b8fe-4b3e-8c0d-c2a0ce70bd9e` | `semantic_insert_construct`, receiver query context | Applied one statement; automatic diagnostics unchanged | Same patch could include both query assignments | Beneficial validated placement; two successful calls plus the rejection worsened round-trip efficiency against that hypothetical patch workflow |

| Calls | Semantic runtime | Local built-in runtime | Measurement scope / comparability |
| :--- | :--- | :--- | :--- |
| `abfd141c-dbed-4432-9a98-bc89356dbf57` | 110 ms | Unavailable: no paired navigation measurement retained | History tool duration, one sample; server/cache state unknown; output guarantees unmatched |
| `5bf48e56-2c9e-4d70-883e-5e18681f1cb8` | 279 ms | Unavailable: no paired compiler-aware inspection measurement | History tool duration, one sample; server/cache state unknown |
| `416b6478-ef7c-495f-a1e6-ad9e27ffdbb0` | 0 ms reported | Unavailable: rejected action has no valid edit baseline | Approval rejection metadata, not a measured successful engine execution |
| `9c7a2f54-1657-48fb-a2e7-be0806e9d7bf` | 12,402 ms | Unavailable: original mutation baseline not timed; no live replay | History tool duration includes automatic verification; one sample; cache state unknown |
| `3239992f-8d09-4937-9b76-995b1d25d14a` | 1,360 ms | Unavailable: no paired rejected-snippet baseline | History tool duration, one sample; source unchanged |
| `87223a5d-4d3c-4d31-ac8d-3c2e1e2a139c` | 2,193 ms | Unavailable: no isolated equivalent patch/check replay | History tool duration includes automatic verification, one sample |
| `d8717943-b8fe-4b3e-8c0d-c2a0ce70bd9e` | 2,082 ms | Unavailable: no isolated equivalent patch/check replay | History tool duration includes automatic verification, one sample |

Runtime ratios are inconclusive: no comparable paired built-in measurements
exist. The tool-duration sum is 18,426 ms, excluding caller preparation,
discovery, logging, orchestration, and subsequent repository checks; it is not
a complete workflow runtime. No wrap-up-only mutation replay or benchmark was
run. The insertion retry sequence added a round trip relative to preparing
one valid statement per call; a hypothetical single patch could combine both
assignments but would still require formatting and equivalent verification.
Overall agent-efficiency improvement is unproven. More explicit single-statement
schema guidance would avoid the observed caller error. Unavailable replacement
and scaffolding tools caused the manual fallbacks recorded in `MANUAL_EDITS.md`.

Integration reconciliation: local `main` advanced to `a535efa0` with compact
navigation while this wrap-up was running. Source conflicts preserve its compact
ambiguity rendering and source-view metadata in the shared rename validator,
alongside typed suggestion errors. Log conflicts retain both sessions' new
entries without restoring historical material removed on main. These manual
merge-conflict edits did not invoke a semantic tool. Final verification and
publication results are recorded in the chat after this last content edit.

The first combined check exposed twelve CLI fixtures expecting pre-navigation
JSON lookup output. Their assertions now check the compact declaration locations
and native selectors, and compact reference headers where applicable. Explicit
same-operation retries and source-state comparisons remain in the fixtures.

Wrap-up history retrieval also rejected `read_thread` with `turnLimit: 20`;
retrying with eight succeeded. This was a caller parameter-bound error with no
mutation. Subsequent history reads could recover tool status and duration but
not original MCP response bodies; the assessment does not invent them.

## 2026-10-16: suggestion implementation tool availability

* Tool: `semantic_workspace_outline` discovery in the provider worker.
* Target: workspace package discovery.
* Observed failure: the tool was absent from the callable inventory; attempting access produced a JavaScript TypeError before an MCP invocation.
* Workaround: inspect repository documentation and source directly. Applicable semantic mutation tools were also absent, so use targeted file patches.
* Root cause: this session exposes only a subset of semedit operations.

The first log patch used an incorrect heading as its context and was rejected without changing the file. Retrying with the actual heading succeeded.

## 2026-10-16: move-file formatter published before trust-hook refusal

* Tool: `semantic_batch`, `semantic_insert_function`.
* Target: `internal/operation/wire_move_file.go`, `formatMoveFile`.
* Observed failure: the formatter published, then verification refused an untrusted configured lint hook.
* Workaround: retain the published formatter and verify with the repository's pinned `make check` command.
* Root cause: the batch omitted request-local hook trust; post-publication verification does not roll back source edits.

The following trusted batch refined the formatter's continuation text and
published successfully, then its configured lint hook returned status 3. The
repository's pinned final checks remain authoritative.

The next formatter and new-file validation batch refused publication because
`internal/astedit/move_file_test.go` changed during the captured snapshot. The
small formatter correction and explicit empty-source rejection use targeted
fallback edits, followed by the pinned checks; the complete-buffer conflict
check correctly prevented a stale publication.

The later retained-session expectation update published, then the configured
lint hook returned status 3. The pinned repository checks will validate the
changed restart expectations with the complete implementation.

## 2026-10-16: expression replacement did not select a composite-literal key

* Tool: `semantic_replace_construct` through `semantic_batch`.
* Target: `internal/operation/wire_move_file.go`, `moveFileDef`.
* Observed failure: the string key `"destination_file"` in a nested example map was not selectable as a statement/expression; earlier constant, variable, and parser edits published.
* Workaround: replace the complete small definition function, then relocate the contract switch case structurally.
* Root cause: nested composite-literal keys are outside this expression selector's advertised slots.

The complete-function replacement and contract-case relocation published on the
next batch. Its configured lint hook returned status 3 while the feature's
remaining lint failures were under repair; final repository checks remain
required.

The following publication-metadata batch likewise published the source-buffer
publication-error record, protocol-utility test metadata, and move-file receipt
finalizer before the configured lint hook returned status 3. The complete
candidate will be checked with the repository's pinned `make check` tooling.

The next batch published move-file success/error projection and publication-path
reporting, then returned the same configured lint-hook status 3. No rollback is
claimed; the published code remains subject to the pinned checks.

The partial-publication regression batch published its new test and helper, then
the same lint hook returned status 3. Its authoritative check is the targeted
source-buffer test followed by the complete pinned `make check`.

The source-buffer test refactor batch also published its helper and replacements,
then returned the same lint-hook status 3. The targeted tests and pinned linter
will check the refactored tests before completion.

## 2026-10-16: concurrent test edit prevented buffer API publication

* Tool: `semantic_batch`, field insertion and body replacement.
* Target: `internal/sourcebuf/buffer.go`, `Change` and `Buffer.Apply`.
* Observed failure: preparation detected a concurrent change to `internal/astedit/move_file_test.go`; checked publication refused.
* Workaround: coordinate a source mutation window and retry after checking which outputs persisted.
* Root cause: semantic batches bind the complete source read set, including files edited by another worker.

The coordinated retry published `Change.Create`, the checked all-output apply
implementation, and the native diagnostic evidence adapter. Its configured lint
hook then exited with status 3 against the still-incomplete integrated feature;
the published edits remain for repository-level verification.

The following coordinated source-buffer regression insertion also published and
then returned the same lint-hook status 3. The tests remain in
`internal/sourcebuf/buffer_test.go`; targeted verification and the final
`make check` will validate the integrated changes.

## 2026-10-16: registry contract batch failed its post-publication hook

* Tool: `semantic_batch` inserting the move-file contract case and helper.
* Target: `internal/operation/language_contracts.go` and `language_contract_profiles.go`.
* Observed failure: both insertions published, then the configured lint hook exited with status 3 while concurrent planner implementation was incomplete.
* Workaround: retain the published contract and run the configured repository checks on the complete implementation.
* Root cause: post-publication verification does not imply rollback of successfully prepared edits.

## 2026-10-16: switch insertion did not locate the registry contract switch

* Tool: `semantic_insert_construct` through `semantic_batch`.
* Target: `internal/operation/language_contracts.go`, `goOperationContract`.
* Observed failure: `discriminator: key` returned "switch statement not found"; the earlier parse-body correction and registration insertion published.
* Workaround: inspect the delegated typed-operation contract helper and insert the case there.
* Root cause: the selected function delegates to helpers and has no direct switch.

## 2026-10-16: new move-file wiring published before verification failure

* Tool: `semantic_batch` with scaffold and declaration insertion.
* Target: `internal/operation/wire_move_file.go`.
* Observed failure: both source edits published, then the configured lint verification hook exited with status 3 while the new operation was incomplete. The result included a large workspace diff.
* Workaround: retain the published file, finish registry wiring, and use `make check` to diagnose the complete candidate.
* Root cause: incremental operation construction temporarily referenced incomplete declarations; the tool reports post-publication verification separately from edits.

## 2026-10-16: empty-body replacement blocked before mutation

* Tool: `semantic_batch` with child `semantic_replace_body`.
* Target: `internal/godiagnostics/manager.go`, retained document synchronization.
* Observed failure: the worker accidentally supplied an empty replacement body; automatic approval review rejected the destructive request before mutation.
* Workaround: use the complete intended source with semantic replacement and review the resulting diff.
* Root cause: an incomplete mutation payload was submitted while preparing the refactor.

## 2026-10-16: scaffold publication conflicted with concurrent source changes

* Tool: `semantic_batch` with child `semantic_scaffold_file`.
* Target: `internal/astedit/move_file.go`.
* Observed failure: the scaffold operation reached `buffer_publish` but failed because `internal/godiagnostics/diagnostics_test.go` changed on disk during the buffered publication; no new source file was created.
* Workaround: coordinate with the owner of the concurrent changes and retry after the shared workspace is stable.
* Root cause: a concurrent edit invalidated the source buffer's captured preimage.

## 2026-10-16: wrap-up history request exceeded the tool limit

* Tool: `read_thread` for this MCP optimization chat.
* Observed failure: `turnLimit: 30` was rejected; the supported maximum is 10. No mutation occurred.
* Workaround: request ten turns per page and follow returned cursors.
* Root cause: caller supplied a limit above the runtime schema maximum.

## 2026-10-16: final check-only verification disagreed with the pinned build environment

* Tool: `semantic_verify` with `path: internal/mcptest`, `language: go`, `check_only: true`, and explicit workspace trust.
* Target: final schema-cache and catalog-lifecycle validation.
* Observed failure: the inferred lint hook could not load `packageclosure.so` because its `cmp` build differed from the linter's. Diagnostics also included unrelated struct-field errors outside the selected package that the successful pinned `make check` build did not reproduce.
* Workaround: keep source unchanged and use the configured gate plus shuffled package race tests as authoritative verification.
* Root cause: the inferred hook uses an incompatible linter/plugin environment. The separate language-server diagnostic disagreement is not isolated by this change.

## 2026-10-16: peer-review continuation passed an unsupported resume option

* Tool: `codex_ask` continuing the completed `codex_review_diff` session with `gpt-6-sol`.
* Target: confirmation of the frozen-catalog repair in the MCP test optimization diff.
* Observed failure: `codex exec resume` rejected an unexpected `-s` argument and exited before reviewing files.
* Workaround: start a fresh read-only diff review, supplying the original findings and their concrete repairs.
* Root cause: the continuation launcher uses an option unsupported by the installed resume subcommand.

## 2026-10-16: stale Go diagnostics session timed out during test cleanup

* Tool: `semantic_batch` with `semantic_remove_construct`.
* Target: `internal/mcptest/validate_test.go`, removal of the rejected lazy-discovery regression.
* Observed failure: after eleven successful children, synchronization could not replace a stale Go LSP session because both shutdown and exit exceeded their context deadlines. The failing child's persistence required inspection.
* Workaround: inspect declarations on disk and resume only unapplied children with a fresh semantic session.
* Root cause: stale diagnostic-session teardown timed out after earlier batch writes; completed children are not rolled back.

## 2026-10-16: function-literal replacement required its reported AST path

* Tool: `semantic_replace_construct`.
* Target: `internal/mcptest/validate_test.go`, the live-catalog regression's error reporter.
* Observed failure: the source selector `func(string) {}` was reported not found even though the diagnostic listed that exact expression at path `2.e0.0.1.0`; no source change occurred.
* Workaround: use the explicit candidate path after the active check finishes, preserving the original declaration around the reporter.
* Root cause: text discrimination did not resolve the function literal in a keyed composite field; the structured candidate path identifies it.

## 2026-10-16: preallocation replacement accepted only one statement

* Tool: `semantic_replace_construct`.
* Target: `internal/mcp/failure_contract_test.go`, expected-response ID slice initialization.
* Observed failure: a replacement containing a declaration followed by an assignment was rejected with `expected exactly one statement`; no source change occurred.
* Workaround: use one declaration with an append to an explicitly sized slice.
* Root cause: statement replacement requires a single AST statement rather than a sequence.

## 2026-10-16: catalog consolidation applied before inferred lint hook failed

* Tool: `semantic_batch`.
* Target: `internal/mcp/schema_test.go`, catalog assertion helpers and their shared parent test.
* Observed failure: all seventeen requested source edits succeeded, then the inferred `golangci-lint` verification hook exited with status 3. Edits remained applied.
* Workaround: inspect the persisted Git diff and run the repository's configured `make check` gate to rebuild and invoke its pinned linter and plugin.
* Root cause: the inferred hook's linter could not load `packageclosure.so` because the plugin used a different Go standard-library build of `cmp`. The repository gate selects a versioned linter executable.
* The final batch for Maven/reload session consolidation and validator regression tests applied all thirteen edits, then encountered the same plugin mismatch. Preserve those edits and verify through the repository gate.
* A follow-up batch explicitly added and organized imports in the four changed test files. All import operations applied before verification encountered the same plugin mismatch. Go body replacement defaults to leaving imports unchanged, so these additions are required before type checking.
* The shared wire decoder and extracted assertion helpers also applied before the same inferred-hook failure. The configured gate remains the authoritative lint and test check for this change.
* The first-mutation language-snapshot regression and its import organization applied before the same hook mismatch. Its focused red/green run uses the project Go environment.
* Resuming the frozen-catalog repair completed the remaining nine children before the same pinned-plugin hook mismatch. The new first-mutation regression had already failed for both full and base catalogs under deferred discovery.

## 2026-10-16: call replacement required a statement-versus-expression selector

* Tool: `semantic_batch` with `semantic_replace_construct`.
* Target: `internal/mcp/schema_test.go`, `assertConstructReplacementSchemasAreExposedAndBatchable`.
* Observed failure: selecting `t.Parallel()` matched both its expression statement and call expression. The first two batch edits persisted before the ambiguous third edit stopped execution.
* Workaround: inspect the partial edits and select the reported statement path `0` when replacing the helper's first call.
* Root cause: the shared statement/expression operation exposes both enclosing and inner call spans for an identical source discriminator.

## 2026-10-16: declaration replacement could not rename a catalog assertion

* Tool: `semantic_batch` with `semantic_replace_construct`.
* Target: `internal/mcp/schema_test.go`, `TestConstructReplacementSchemasAreExposedAndBatchable`.
* Observed failure: function replacement refused a new declaration name and reported a buffered-source synchronization conflict after the preceding MCP edit had changed the same file. No catalog grouping edits applied.
* Workaround: preserve declaration names during replacement, use the dedicated rename operation where needed, and refresh the server's source revision before retrying.
* Root cause: declaration replacement requires the original symbol identity; live semantic sessions also reject externally changed buffered revisions.

## 2026-10-16: test consolidation batch required explicit verification trust

* Tool: `semantic_batch` with `semantic_replace_body`.
* Target: `internal/mcp/schema_test.go`, `TestMCPIntegerNotationAndRangeValidation`.
* Observed failure: the edit applied, but final verification failed because the inferred `golangci-lint` hook required workspace trust. The failure also returned a large workspace-wide diff and scope despite a single-function edit.
* Workaround: preserve the applied edit, inspect the actual Git diff, and supply `trust_workspace: true` on subsequent batches. Run the repository's `make check` gate after all edits.
* Root cause: the batch omitted explicit trust for an inferred external verification hook; batch verification failures do not roll back successful children.

## 2026-10-16: workspace outline operation was unavailable in the active tool inventory

* Tool: `semantic_workspace_outline`.
* Target: `internal/mcptest` package discovery before validating MCP test helper changes.
* Observed failure: the operation was not callable in the active MCP tool inventory (`TypeError: ... is not a function`); no workspace change occurred.
* Workaround: use the available semantic inspection and batch editing tools, with narrow shell reads for the requested files.
* Root cause: the workspace outline tool was not exposed in the active MCP tool inventory.

## 2026-10-16: body replacement operation was unavailable in the active tool inventory

* Tool: `semantic_replace_body`.
* Target: `internal/mcptest/validate.go`, `Server.Initialize`.
* Observed failure: the operation was not callable in the active MCP tool inventory (`TypeError: ... is not a function`); no source mutation occurred.
* Workaround: make the narrow supported-source edit with `apply_patch`.
* Root cause: the body replacement tool was not exposed in the active MCP tool inventory.

## 2026-10-14: targeted Codex second opinion could not start

* Tool: `codex_ask` with model `gpt-6-sol`.
* Target: final Maven cancellation process-lifecycle diff.
* Observed failure: three reviewer attempts, including a continuation, rejected an unexpected `-s` argument and exited before returning a review; no source changes occurred.
* Workaround: rely on the earlier independent Sol review of the process-group design, the separate read-only code review, and local Maven package tests.
* Root cause: the reviewer launcher passed an unsupported CLI option.

## 2026-10-14: workspace outline operation was unavailable in the active tool inventory

* Tool: `semantic_workspace_outline`.
* Target: `internal/maven` package discovery before inspecting Maven cancellation code.
* Observed failure: the operation was not callable in the active MCP tool inventory (`TypeError: ... is not a function`); no workspace change occurred.
* Workaround: inspect the relevant declaration with the available `semantic_inspect_symbol` tool and consult the Maven ADR and package tests.
* Root cause: the workspace outline tool was not exposed in the active MCP tool inventory.

## 2026-10-14: statement insertion could not anchor inside a walker callback

* Tool: `semantic_insert_construct`.
* Target: `cmd/docgen/benchmark_load.go`, `loadBenchmarkSources`.
* Observed failure: insertion before the existing directory/extension guard reported that the statement anchor was not found; no source mutation occurred.
* Workaround: replace the complete enclosing function with `semantic_replace_construct`.
* Root cause: the requested condition is inside a function literal passed to `filepath.WalkDir`; statement insertion did not resolve that nested anchor.

## 2026-10-14: body replacement omitted standard-library imports

* Tool: `semedit-next replace-body`.
* Target: `cmd/docgen/main.go`, `cmd/docgen/txtar.go`, and `cmd/docgen/benchmarks.go`.
* Observed failure: replacement bodies referencing `errors.Join`, `json.RawMessage`, and `filepath.Join` were published without their imports; `make check` caught undefined identifiers.
* Workaround: explicitly add the missing packages with the semantic import organizer, then rerun checks.
* Root cause: body replacement does not guarantee import inference for newly referenced packages.

## 2026-10-13: replace-construct could not select a nested install callback

* Tool: `semantic_replace_construct`.
* Target: `internal/cli/cli.go`, nested callback inside `newInstallCommand`.
* Observed failure: the operation could not resolve the callback's return statement using the enclosing function selector; no mutation occurred.
* Workaround: apply a targeted edit to the callback's output handling.
* Root cause: the selector resolves the named enclosing function but does not expose statements nested inside the callback literal as direct target constructs.

## 2026-10-13: source-set predicate was absent from semantic construct candidates

* Tool: `semantic_replace_construct`.
* Target: `internal/sourcebuf/buffer.go`, `sourceSet`.
* Observed failure: the resolver reported only the function's `err != nil` guard and could not select the existing file-extension predicate; no source mutation occurred.
* Workaround: use the smallest targeted source edit for this filesystem-walk callback after confirming the current source.
* Root cause: the construct locator omitted the callback's extension `if` from its candidate set.

## 2026-10-13: Java source-buffer predicate replacement required an explicit construct path

* Tool: `semantic_replace_construct`.
* Target: `internal/sourcebuf/buffer.go`, `Buffer.Apply`.
* Observed failure: supplying the complete `if` statement as `discriminator` did not match because the resolver selects `if` constructs by condition or short initializer; no source mutation occurred.
* Workaround: use the exact `construct_path` returned in the diagnostic, then use targeted file editing only if the structural operation cannot express the predicate changes.
* Root cause: the call used a statement-shaped discriminator for an `if` node whose selector contract is condition-based.

## 2026-10-11: Output migration applies before configured verification fails

* Tool: `semantic_batch` replacing and removing output-contract declarations in `internal/mcp/server.go`.
* Observed: the serialized retry applied the requested declarations and published the source, then the configured `golangci-lint` verification hook exited with status 3.
* Workaround: retain the applied semantic edits, finish the combined contract migration, and run repository `make check` with its configured plugin rebuild rather than replaying the mutation.
* Root cause: mutation and post-publication verification are separate outcomes; an incomplete shared migration or hook environment can fail verification after successful publication.

## 2026-10-11: import cleanup during function replacement

The import insertion retry without placement failed because it appended after functions (`imports must appear before other declarations`). No source changed on that retry. Use explicit `file_start` placement for the import declaration.

`semantic_replace_construct` replacing `TestPreparedFixtureBuildsOfflineWithoutManifestChanges` in `tools/benchmark-harness/driver_dependency_test.go` removed the temporarily unused `archive/zip` import despite `auto_organize_imports: false`. Inserting the extracted proxy helper then reported `undefined: zip`. Restore the import with a semantic declaration insertion. The replacement operation cleaned imports even when cleanup was disabled.

## 2026-10-11: model-limit config body verification failed

* Tool: `semantic_batch` with `semantic_replace_body`.
* Target: `tools/benchmark-harness/opencode_driver.go`, `Runner.openCodeConfig`.
* Observed failure: the config replacement applied, then the trusted repository-wide `golangci-lint` hook exited with status 3 during concurrent harness edits.
* Workaround applied: inspect the applied source and defer repository-wide verification to the owner after all dependent edits complete.
* Root cause: the semantic verifier checks the entire mutable workspace rather than this coherent package change.

## 2026-10-11: nested deferred closure selector

* Tool: `semantic_replace_construct`.
* Target: `tools/benchmark-harness/codex_app_server_test.go`, `TestCodexAppServerRejectsPersistentThread`.
* Observed failure: selecting the inner `if err := server.close()` returned not found; candidates included the enclosing deferred closure but not its inner statements.
* The enclosing `defer` also failed selection using its exact returned source.
* Workaround: select the enclosing `defer` with its returned construct path `4`.
* Root cause: statement traversal does not descend into a function literal in a deferred call; defer source discrimination also failed despite an identical candidate.

## 2026-10-11: Runtime command construct cannot resolve context

`replace-construct` on `resolveOpenCodeGoRoot` in `tools/benchmark-harness/opencode_sandbox.go` refused an `exec.CommandContext` replacement because import cleanup could not resolve existing standard-library import `context`, even with `--auto-organize-imports=false`. No source changed. Use the minimal atomic replacement after repeated standard-library cleanup failures; the same fallback handles the adjacent runtime execution allowance.

## 2026-10-11: Switch-case insertion did not locate a documented case discriminator

* Tool: `semantic_insert_construct`
* Target: `tools/benchmark-harness/opencode_session_test.go`, `serveOpenCodeUsageHelper`
* Observed failure: The tool could not locate the exact displayed switch case condition; no source changed.
* Workaround applied: Use the smallest atomic fallback to add the fixture's authoritative message-list response.
* Root cause: Case insertion did not support the compound boolean case selector.

## 2026-10-11: Function replacement rejected an unchanged method signature

* Tool: `semantic_replace_construct`
* Target: `tools/benchmark-harness/opencode_session.go`, `openCodeSessionMessages`
* Observed failure: The tool rejected a full replacement as not preserving `(*openCodeSession).openCodeSessionMessages`, although the replacement kept that receiver and method name; no source changed.
* Workaround applied: Use the already supported precise statement edit plus the repository's narrow atomic fallback for the named return signature.
* Root cause: The function-symbol validator did not recognize the equivalent named result signature.

## 2026-10-11: Gopls wrapper body cleanup cannot resolve bufio

The `replace-body` semantic CLI operation on `resolveOpenCodeGopls` in `tools/benchmark-harness/opencode_sandbox.go` failed import pruning with `cannot resolve package name for import "bufio"`. No body changed. Retry the structural body edit without import cleanup, then make the minimal atomic import correction. The standard-library import resolver failure prevents its otherwise supported cleanup.

## 2026-10-11: Defer selector rejected its reported source text

* Tool: `semantic_replace_construct`
* Target: `tools/benchmark-harness/opencode_session.go`, `openCodeSessionMessages`
* Observed failure: Selecting `defer response.Body.Close()` reported it was absent while listing the identical construct at path `9`; no source changed.
* Workaround applied: Retry with the reported construct path.
* Root cause: The defer selector did not accept its displayed source representation.

## 2026-10-11: Runtime batch verification has a plugin Go version mismatch

`semantic_batch` reported runtime body and import edits, then workspace verification failed (`golangci-lint` exit 3): `.scratch/go/bin/packageclosure.so` was built with a different version of package `cmp`. Its diagnostics also included unrelated backend/astedit errors. Follow-up inspection and disk reads still showed the earlier sandbox resolver and a missing runtime renderer, while the driver body changed. Verify disk state directly, then use a minimal atomic patch for the unpublished helpers; run checks under the repository's pinned Go environment after edits become coherent.

## 2026-10-11: Runtime function replacement used declaration-only operation

`semantic_batch` rejected `semantic_replace_decl` in `tools/benchmark-harness/opencode_sandbox.go` with `unexpected declaration type`; no diff was applied. Replace the function with `semantic_replace_construct` kind `function`, or replace its body through the specialized operation.

## 2026-10-11: Runtime profile batch verification failed after publication

`semantic_batch` updated `Runner.openCodeSandboxProfile` in `tools/benchmark-harness/opencode_sandbox.go` but reported workspace verification failure (`golangci-lint` exit 3). Its final diff confirmed the edit remained. Concurrent root test setup temporarily had unresolved identifiers, so whole-workspace verification could observe an intermediate state. Preserve the published edit, finish coherent test setup, and rerun repository checks after source writers finish.

## 2026-10-07: Import organization dropped an explicitly requested nested-module import

* Tool: `semantic_organize_imports`.
* Target: `tools/package-closure/plugin/plugin.go`.
* Observed failure: Adding `semedit.tools/package-closure` returned success but the import remained absent, leaving both `packageclosure` references undefined and blocking `make check`.
* Workaround: Restore the single required import with a targeted file patch.
* Root cause: Import resolution ran from the parent workspace for a separately governed nested module; the explicit addition did not survive organization.

## 2026-10-07: Semantic function insertion rejected escaped rune literal

* Tool: `semantic_insert_function`, target `appendFakeCodexAppServerProcess` in `tools/benchmark-harness/codex_app_server_test.go`.
* Observed: the tool rejected the Go snippet with `syntax error: illegal rune literal`; no source was changed.
* Workaround: simplify the inserted snippets and submit each helper individually with valid Go literal escaping.
* Root cause: the caller encoded a newline rune literal incorrectly while transporting the snippet through the tool wrapper.

## 2026-10-07: Semantic function insertion reported an unexpected declaration collision

* Tool: `semantic_insert_function`, target `appendFakeCodexAppServerProcess` in `tools/benchmark-harness/codex_app_server_test.go`.
* Observed: the tool reported the declaration already exists after the earlier failed insertion, although that failure had reported a syntax error.
* Workaround: inspect the current declaration and repair or reuse it before inserting further helpers.
* Root cause: the failed insertion may have partially changed the workspace before returning its error.

## 2026-10-06: replace-construct required a path for a repeated if selector

* Tool: `semantic_replace_construct`.
* Target: `internal/mcp/batch.go`, `Server.executeBatch`.
* Observed failure: selecting the `err != nil` branch was ambiguous and made no changes; the tool returned construct paths including `2.0.6` for `entry.Parse(raw)`.
* Workaround: retry with the returned path `2.0.6`.
* Root cause: several error checks in the method share the same condition, so the selector requires a structural path.

## 2026-10-06: replace-construct again required a path for the same repeated selector

* Tool: `semantic_replace_construct`.
* Target: `internal/mcp/batch.go`, `Server.executeBatch`.
* Observed failure: the new `err != nil` selection was ambiguous; no changes were made. The tool returned path `2.0.6` for the parser-validation branch.
* Workaround: retry with the returned path `2.0.6`.
* Root cause: other error checks still use the same condition in this method.

## 2026-10-06: replace-decl does not update struct types

* Tool: `semantic_replace_decl`.
* Target: `tools/benchmark-harness/planner.go`, `PlanOptions`.
* Observed failure: the tool refused the replacement with `requires a type alias`; no changes were made.
* Workaround: use a structural operation that can insert a field into a struct.
* Root cause: despite accepting a target type name, this operation only supports type aliases, not struct declarations.

## 2026-10-04: semantic test body rejected newline rune literal

* Tool: `semantic_replace_body`.
* Target: `internal/mcp/language_catalog_test.go`, `TestDirectAndBatchLanguageChecksUseFrozenSnapshot`.
* Observed failure: the replacement body was rejected with `syntax error: illegal rune literal`; no mutation was applied.
* Workaround: retry with a Go string newline append instead of a rune literal.
* Root cause: the tool payload escaping produced invalid Go rune syntax.

---

## Log of Suboptimal Operations & Defects

| ID | Date | Tool / Operation | Target File / Context | Observed Suboptimal Behavior | Workaround / Manual Follow-up | Root Cause & Missing Invariant | Tracking Issue / Fix |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **ST-0008** | 2026-09-21 | `semedit` MCP server | Central registry ingress refactor | The configured project MCP server is not loading, so semantic edit operations are unavailable for refactoring its own Go ingress code. | Use a minimal manual patch after recording the outage. | Server startup or discovery failure remains uninvestigated at the user's direction. | Diagnose MCP loading separately; do not block the registry integration. |
| **ST-0009** | 2026-09-21 | `semedit` semantic editing MCP tools | Go rename/cache and diagnostic ingress files in this worktree | Semantic editing tools were unavailable, so agents could not apply required source changes through the dogfooding interface. | Used focused manual patches and gofmt; agents did not delegate any semantic transformation to text replacement. | Project MCP server is unavailable in this environment. | Restore MCP availability before future dogfooding validation. |
| **ST-0012** | 2026-09-22 | `semantic_rename` | `config/config.go` field `Config.Port` | Go backend failed before resolving the symbol because the workspace loaded a relative GOPATH and reported no gopls views. | Will apply a focused manual rename after recording this failure, then continue with semantic operations where available. | The MCP process inherited an invalid relative GOPATH for this benchmark worktree. | Restore an absolute GOPATH or workspace-specific Go environment before future semantic renames. |
| **ST-0013** | 2026-09-22 | `semedit` semantic editing MCP tools | `tools/benchmark-harness/driver.go`, `internal/mcp/server.go` | Semantic editing tools were unavailable in the isolated benchmark worktree session. | Applied focused manual patches and gofmt. | MCP server discovery was unavailable in this delegated session. | Restore MCP availability before future dogfooding validation. |
| **ST-0014** | 2026-09-22 | `semantic_replace_body` | `cmd/docgen/main.go`, `writeHugoConfig` | Replacing a function body containing raw Go string literals with backticks failed with `syntax error: illegal character U+005C '\\'`. | Applied a focused patch to the function body after recording the failure. | The snippet parser received escaped delimiters from the tool harness instead of preserving the raw string literal syntax needed by the Go parser. | Escape handling should preserve raw-string delimiters in replacement snippets. |
| **ST-0016** | 2026-09-22 | `semedit` semantic documentation tools | `cmd/docgen/benchmarks.go`, `cmd/docgen/main.go` | This session exposed no callable semantic MCP editing tools for this documentation-generator change. | Used focused source patches and will run the repository verification suite. | This session did not expose the semedit MCP server. | Make semantic editing MCP tools available to future repository sessions. |
| **ST-0017** | 2026-09-23 | `semantic_scaffold_file`, `semantic_replace_body`, `semantic_insert_decl` | `tools/benchmark-harness/` in the requested isolated worktree | Calls reported success but were bound to the primary checkout, so the scaffold landed on `main` and the replacement did not touch the isolated worktree. | Reverted only the accidental primary-checkout files and applied focused patches in the requested worktree. | Semantic MCP calls do not honor the delegated worktree path in this session. | Bind semantic operations to the calling worktree before future delegated edits. |
| **ST-0021** | 2026-09-23 | `semantic_rename` | `internal/backend/java/java.go` in the requested worktree | Could not resolve the local parameter `backend` in `WithJavaSessionFactory` for a scope-local rename. | Rename the closure parameter and its references with a focused patch. | Workspace rename did not resolve an unexported identifier in the moved package when given its bare symbol name. | Use a scope-aware local rename or report the declaration scope required for resolution. |
| **ST-0024** | 2026-09-23 | `semantic_organize_imports` | `internal/backend/haskell/backend.go` in the requested worktree | Rejected the extracted Haskell implementation after an edited factory adapter retained an extra closing parenthesis. | Remove the unmatched delimiter and retry import organization. | The focused runtime-config adaptation changed a function literal but left its prior wrapper delimiter. | Re-run Go parsing before semantic import organization after changing closure signatures. |
| **ST-0025** | 2026-09-23 | `semantic_insert_case` | `internal/astedit/insert.go` in the lint worktree | A request targeting the nested `switch opts.Placement` in `calculateInsertionOffset` inserted `default` into the outer switch with the same discriminant instead. The inner switch remained non-exhaustive, and the outer fallback behavior changed. | Moved the branch into the nested switch. Added a CLI reproducer in `testdata/scripts/insert_case.txtar` that asserts an ambiguous selector leaves the file unchanged. | The switch selector matched the first switch when nested switches had the same discriminant. | Reject ambiguous matches with `ErrSwitchAmbiguous`; a source-position selector remains a possible later enhancement. |
| **ST-0033** | 2026-09-25 | `semantic_insert_type` | `cmd/worktreecheck/main.go` | The shared access-modifier schema accepted `package-private`, but the Go backend rejected it as unsupported before making a source change. | Retry with the Go-supported `private` modifier. | The generic schema exposes an access modifier that the Go backend does not implement. | Filter unsupported access modifiers by backend or normalize `package-private` for Go. |
| **ST-0037** | 2026-09-25 | `semantic_replace_decl` | `internal/operation/wire_engine_declarations.go` | The active semedit MCP tool catalog did not expose a callable `semantic_replace_decl` operation while wiring the new registry request. The attempted tool dispatch failed before reaching the server or changing source. | Continue with available semantic insertion/body tools; use a narrow atomic edit for the existing request struct field and parser/parameter wiring if no semantic operation covers those locations. | The running MCP server/catalog is older than the implementation surface in this task; it has no declaration-replacement operation callable from this session. | Refresh/reload the in-tree MCP server capability catalog when new operations are built, and keep declaration replacement available for existing package var specs. |
| **ST-0043** | 2026-09-25 | `semantic_replace_body` | `.scratch/worktrees/grouped-decl-fix/internal/astedit/decl.go`, `InsertDecl` | The first replacement request failed during MCP argument parsing with a missing-comma syntax error before source mutation. | Simplify the body payload quoting and retry. | The tool argument parser rejected escaped quote and backtick sequences embedded in the submitted body. | Retry with plain Go quoting and minimize transport-level escapes. |
| **ST-0044** | 2026-09-25 | `semantic_insert_function` | `.scratch/worktrees/grouped-decl-fix/internal/astedit/decl_test.go`, `TestInsertDecl_PreservesCommentedSentinelSpecs` | The function snippet failed validation with an unterminated string because a newline in a Go string literal was transported as a literal line break. | Retry with an escaped newline in the failure message string. | The body text transport interpreted the newline escape before snippet validation. | Keep multiline Go string literals raw and escape newlines in ordinary Go strings. |
| **ST-0046** | 2026-09-25 | `semantic_organize_imports` | `internal/astedit/decl.go` after integrating the grouped-declaration fix | Reported success but retained the newly added standard-library `go/format` import in a separate group next to the local module import. | Move `go/format` into the standard-library group with a narrow atomic edit. | Import organization preserves existing group boundaries even when a standard-library package is placed in the local import group. | Group imports by package origin during organization. |
| **ST-0047** | 2026-09-25 | Fresh `semedit` MCP JSON-RPC handshake | `.scratch/worktrees/adr0046-replay-2` | The transport-only probe sent Content-Length framing and received JSON-RPC parse error (`id:null`, `code:-32700`). | Corrected the transport-only client to newline-delimited JSON-RPC; no repository source changed. | Client setup used LSP framing for a line-delimited JSON-RPC server. | Keep the replay transport helper aligned with server framing. |
| **ST-0055** | 2026-09-25 | `semantic_insert_function` | `internal/astedit/loop.go`, helper `loopMatches` | The request tried to traverse loop header expressions by placing `ast.Expr` values in an `ast.BlockStmt.List`, which requires `ast.Stmt`; diagnostics showed four compile errors. | Record the failure and retry the helper using `ast.Inspect` separately on each expression. | Go's AST has distinct `ast.Expr` and `ast.Stmt` interfaces, so a block cannot represent arbitrary header expressions. | Provide AST traversal utilities for expression lists or clearer type-aware examples. |
| **ST-0058** | 2026-09-25 | `semantic_replace_decl` | replay 3 grouped var fixture, symbol `ErrReplayBase` | Source `// ErrReplayBase ...\nvar ErrReplayBase = errors.New("base")` failed with `expected IDENT, found var` while replacing one grouped spec; the file stayed unchanged. | Fixture comment was added with a logged atomic edit; `bf6e142` fixes this in the engine and adds AST plus CLI before/after regressions. | The spec extractor assumed `var` was the first formatted token, so a leading doc comment left the keyword inside the group. | Resolved in `bf6e142`; verify with a fresh promoted binary. |
| **ST-0061** | 2026-09-25 | `semantic_insert_function` | replay 3 `replaceLoopDef` | A second snippet failed because quotes in the nested Go example string were not escaped; no source mutation occurred. | Escape the Go string quotes and retry. | JSON transport escaping did not also escape the nested Go string literal. | Include a nested-source quoting example in the catalog. |
| **ST-0064** | 2026-09-25 | `semantic_insert_decl` | replay 4 independent scratch fixtures | A second fixture in the same Go package reused `target`; package collision detection correctly rejected it without writing. | Put the exact-byte fixture in its own package directory. | Client fixture isolation was insufficient; package-scope collision detection worked. | Keep package-scoped fixture guidance. |
| **ST-0070** | 2026-09-26 | `semantic_insert_construct` (CLI) | `internal/capability/capability.go`, operation-key constants | The insertion failed with `expected declaration, found OpInspect`; the supplied snippet contained constant specs without a top-level `const` declaration, and no source was changed. | Deferred adding capability keys to feature wiring; no source workaround was needed. | Construct insertion validates a complete top-level declaration, while the request supplied only grouped constant specs. | Supply a complete declaration or append valid constant specs through the supported declaration workflow. |

---

## Guidelines for Logging Dogfooding Deficiencies

When an agent or developer uses an MCP tool from `semedit` and encounters any of the following, agents **must** record an entry above:

1. **Bug / Failure**: The tool returned an error or unexpected output for a valid semantic intent.
2. **Follow-up Manual Edit**: The tool modified the AST, but agents needed a manual edit after the tool operation to make the code compile, pass formatting, or adjust surrounding declarations.
3. **Inconvenient Ergonomics**: The parameter schema or error message caused the agent to fail or hallucinate parameters on its first attempt.

## 2026-09-26: semantic_rename reports transient call arity during signature migration

* Tool: `semantic_rename`
* Target: `tools/benchmark-harness/driver.go`, `prepareAgentPrompt`
* Observed failure: rename applied but verification reported one callsite with a temporary missing argument while the function signature was being migrated.
* Workaround: complete the wrapper/signature transition using semantic declaration edits, then verify the package after the migration.
* Root cause: the rename operation verified an intermediate state before dependent callsites were adapted.

| ID | Date | Tool / Operation | Target File / Context | Observed Suboptimal Behavior | Workaround / Manual Follow-up | Root Cause & Missing Invariant | Tracking Issue / Fix |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **ST-0021** | 2026-09-26 | `semantic_replace_construct` | `tools/benchmark-harness/agy_driver.go` executable selection conditional | Did not resolve the conditional using the short statement discriminator `_, err := os.Stat(agyBin)`. | Will retry with the complete condition text after recording this failure. | Conditional matching requires the parsed full source condition. | Improve construct-resolution diagnostics for short-statement conditionals. |
| **ST-0022** | 2026-09-26 | `semantic_replace_construct` | `tools/benchmark-harness/agy_driver.go` executable selection conditional | Also failed to resolve the full conditional when supplied as a discriminator. | After recording the failure, will apply a focused atomic edit to the conditional. | The tool does not identify short-init conditionals in this method. | Add short-init conditional matching. |
| **ST-0026** | 2026-09-26 | `semantic_replace_construct` | `tools/benchmark-harness/driver.go` session cleanup defer | The construct matcher did not resolve the `session.close()` defer by the supplied discriminator. | After logging, apply a focused atomic edit to the defer. | Defer matching did not accept the call expression as discriminator. | Improve defer construct matching. |

* Tool: `semantic_replace_construct`
* Target: `tools/benchmark-harness/planner.go`, the executor selection branch in `ExecutePlan`
* Observed failure: the request did not locate the conditional using discriminator `ctx.Err() != nil`.
* Workaround: retry by matching the exact branch expression from the AST or use a focused declaration replacement.
* Root cause: conditional discriminators require exact backend-resolved syntax.

### 2026-09-26: no semantic field-level insertion operation

* Tool: semantic editing tool catalog
* Target: `internal/backend/rust/backend.go`, `rustDocumentSymbol.Detail` and `decodeRustDocumentSymbol`
* Failure: available semantic operations edit complete declarations but provide no operation for adding one struct field or one decode clause without replacing the whole function.
* Workaround: apply two narrow atomic patches to the selected struct and JSON decoder; all function additions and body changes use the prebuilt semedit CLI.
* Root cause: semantic editing does not expose field-level or statement-level insertion for these constructs.

### 2026-09-26: semedit removed imports before their references existed

* Tool: `semedit imports` and `semedit replace-body`
* Target: `internal/backend/rust_test.go`, source snapshot regression
* Failure: semedit imports organized the test before the new code referenced `pipeline`, SHA-256, and hex imports, so it removed those unused imports; a subsequent body replacement reported undefined symbols.
* Workaround: rerun semedit imports after the test body was inserted, which restored the required imports and resolved the diagnostics.
* Root cause: imports are cleaned against the file state at operation time and intentionally remove imports not yet referenced.

### 2026-09-26: semantic struct extension exposed positional test fixtures

* Tool: `semantic_replace_construct` through the prebuilt semedit CLI
* Target: `language_txtar_test.go`, `fakeDocumentSymbol`
* Failure: after adding optional detail/URI/children fields for hierarchical Kotlin outline fixtures, package test compilation reported too few values in existing positional literals; no runtime test ran.
* Workaround: convert the affected fake symbol fixtures to keyed literals.
* Root cause: Go positional struct literals require values for every field when a test helper struct grows.

* Tool: `semantic_replace_construct`
  * Target: `cmd/docgen/hugo.go`
  * Failure: Attempted to replace the local `config` short declaration to add Hugo `relativeURLs`; tool rejected the operation because it requires an enclosing `function` parameter.
  * Workaround: Use a supported semantic operation on the enclosing function.
  * Root cause: The tool does not support direct selection of a local declaration without identifying its containing function.

* Tool: `semantic_replace_construct`
  * Target: `cmd/docgen/hugo.go`
  * Failure: Retried local `config` declaration replacement with its containing function; tool reported `unsupported Go construct kind "decl"`.
  * Workaround: Use `semantic_replace_body` on `writeHugoConfig`.
  * Root cause: This operation supports control-flow constructs rather than local declarations.

### 2026-09-26: Make outline cannot map continued declarations

* Tool: `semantic_outline`
* Target: `Makefile` and `tools/benchmark-harness/Makefile`
* Observed failure: both selected-file outline requests failed with `make read projection cannot map continued declaration` (lines 59 and 69 respectively).
* Workaround: no outline workaround applied; report the limitation and preserve the request results.
* Root cause: the Make read projection cannot map continued declarations to source ranges.

### 2026-09-26: semantic type overwrite duplicated Server

* Tool: `semantic_insert_structure` with `kind=type` and `overwrite=true`
* Target: `internal/mcp/server.go`, `Server`
* Failure: the tool inserted a second `Server` declaration instead of replacing the existing type, leaving a duplicate declaration and an unresolved `toolMetricsRecord` field type.
* Workaround: undo the insertion before applying a different edit strategy.
* Root cause: the overwrite option did not replace an existing struct type in this operation.

### 2026-09-27: semantic type overwrite duplicated ScaffoldOptions

* Tool: `semantic_insert_type` with `overwrite=true`
* Target: `internal/astedit/scaffold.go`, `ScaffoldOptions`
* Failure: the tool inserted a duplicate type instead of replacing the existing struct declaration, producing Go redeclaration diagnostics.
* Workaround: remove the duplicate declaration and use a supported declaration replacement operation.
* Root cause: the type insertion operation ignored overwrite for an existing struct type.

### 2026-09-27: semantic_replace_decl rejected struct replacement

* Tool: `semantic_replace_decl`
* Target: `internal/astedit/scaffold.go`, `ScaffoldOptions`
* Failure: replacing the existing struct declaration failed with `requires a type alias`.
* Workaround: use a narrow atomic source edit to remove the accidental duplicate and update the struct.
* Root cause: this operation supports type aliases but not struct type declarations.

### 2026-09-27: semantic construct replacement could not match batch parse preflight

* Tool: `semantic_replace_construct`
* Target: `internal/mcp/batch.go`, `Server.ExecuteBatch`
* Failure: the discriminator `entry.Parse(raw)` did not match a supported construct selector; no source change was made.
* Workaround: inspect the selector diagnostics or use the exact conditional expression and candidate path.
* Root cause: the construct matcher does not match the `if` initializer using the supplied expression.

### ST-0078: package diagnostics blocked semantic reference analysis

* Date: 2026-09-27
* Tool(s): `semantic_find_references`
* Source: same benchmark; 4 failed calls in `task-09-composite-refactor`.
* Failure: package loading failed after the agent left an unused import or temporarily removed `DefaultConfig`, so the Go package no longer type-checked.
* Workaround: restore a compilable intermediate package before requesting references.
* Root cause: semantic analysis correctly depends on a valid package snapshot, but the agent invoked it between mutation steps; consider clearer diagnostics that make the transient source problem and recovery action prominent.

### ST-0083: benchmark used `switch` where the construct selector requires `case`

* Date: 2026-09-27
* Tool: `semantic_replace_construct`
* Source: same benchmark; 1 failed call in `task-11-mixed-sink-api-migration`.
* Failure: the call supplied `kind: switch` and a complete switch statement whose selector also changed; the operation supports branch constructs, with enum kind `case`, not replacement of a whole switch statement.
* Workaround: identify the intended branch and provide its `case` clause as the replacement source, or use a broader operation when changing the whole switch.
* Root cause: the tool description says “switch branch” but does not make the branch-only source contract explicit; clarify scope and add a branch-shaped example.

### ST-0086: structural type insertion rejected replacement

* Date: 2026-09-27
* Tool: `semantic_insert_structure`
* Source: ADR-0054 `projectverify` public types in `.scratch/worktrees/verify-config/internal/projectverify/types.go`.
* Failure: attempted to update the newly added `Hook` and `HookResult` structs with `overwrite: true`; the tool rejected the request with `overwrite is unsupported for type structure insertion`.
* Workaround: reconstruct the package type file through semantic scaffolding and insert the complete updated declarations.
* Root cause: the structural insertion tool does not support replacing an existing type even though its shared schema exposes the overwrite field.

### ST-0087: verification request struct replacement is unsupported

* Date: 2026-09-27
* Tool: `semantic_replace_decl`
* Target: `internal/backend/backend.go`, `VerifyRequest`, in the verification integration worktree.
* Failure: the declaration replacement rejected a struct because it only accepts type aliases.
* Workaround: use an AST-aware rewrite for the unsupported struct-field change, preserving atomic publication.
* Root cause: the declaration replacement operation does not cover struct definitions.

### ST-0088: Body replacement did not resolve a worktree-prefixed path

* Date: 2026-09-27
* Tool: `semantic_replace_body`
* Target: `internal/backend/golang/backend.go` in the integration worktree.
* Failure: The body lookup failed after symbol inspection had resolved the declaration.
* Workaround: Use a checked AST rewrite for this unsupported worktree path; preserve the isolated worktree.
* Root cause: Inconsistent resolution of a worktree-prefixed relative path between inspection and mutation.

### ST-0089: Automatic import resolution introduced a package cycle

* Date: 2026-09-27
* Tool: `semantic_replace_body`
* Target: `internal/pipeline/pipeline.go`.
* Failure: Replacing CheckDiagnostics added an adapter import, producing an import cycle because the adapter depends on pipeline.
* Workaround: Move shared gopls discovery to godiagnostics and delegate from both callers.
* Root cause: Import organization resolves imports without validating the package dependency graph before writing.

### ST-0091: Body replacement temporarily mismatched a helper signature

* Date: 2026-09-27
* Tool: `semantic_replace_body`
* Target: `internal/godiagnostics/diagnostics.go`.
* Failure: The new caller expected loader findings before the helper signature was updated; diagnostics reported an assignment mismatch.
* Workaround: Update the helper contract before final verification.
* Root cause: The multi-declaration contract change was split across calls.

### ST-0094: function insertion access did not match exported name

* Date: 2026-09-27
* Tool: `semantic_insert_function`
* Source: ADR-0054 discovery functions in `.scratch/worktrees/verify-config/internal/projectverify/config.go`.
* Failure: the batch marked all functions private, and insertion rejected exported `Discover` with `visibility mismatch`.
* Workaround: let insertion infer visibility from function casing.
* Root cause: the batch-level edit applied a single explicit access modifier to public and private declarations.

### ST-0095: construct replacement selector was ambiguous

* Date: 2026-09-27
* Tool: `semantic_replace_construct`
* Source: LSP validation in `.scratch/worktrees/verify-config/internal/projectverify/config.go`.
* Failure: replacing `if item.LSP != nil` in `convertHooks` matched both the form counter and the LSP conversion branch, so the tool rejected the edit and returned both candidate paths.
* Workaround: select the reported construct path for the conversion branch.
* Root cause: repeated equivalent conditions require `construct_path` to disambiguate.

### ST-0096: structural function insertion rejected overwrite

* Date: 2026-09-27
* Tool: `semantic_insert_structure`
* Source: context-aware staging helper in `.scratch/worktrees/verify-config/internal/projectverify/executor.go`.
* Failure: attempted to update the existing `copyWorkspace` signature with `overwrite: true`; the tool rejected it with `overwrite is unsupported for function structure insertion`.
* Workaround: retain the existing helper as a background-context wrapper and add a context-aware helper under a distinct symbol for runtime use.
* Root cause: structural insertion does not support replacing existing functions despite the shared overwrite parameter.

### ST-0097: construct replacement included two statements

* Date: 2026-09-27
* Tool: `semantic_replace_construct`
* Source: hook-status handling in `.scratch/worktrees/verify-config/internal/projectverify/executor.go`.
* Failure: the replacement payload contained the original `if` plus a following `if`, and the tool rejected it because a construct replacement must contain exactly one complete construct.
* Workaround: make the status change within an existing construct or use a bounded full function-body replacement.
* Root cause: the edit payload crossed the selected AST construct boundary.

### ST-0098: semantic_replace_body rejected invalid hook invocation payload

* Date: 2026-09-27
* Tool: `semantic_replace_body`
* Target: `internal/projectverify/executor.go`, `runCheck`
* Observed failure: the replacement body omitted a closing parenthesis in the final `runCommand` call, so the tool rejected the candidate Go syntax before writing.
* Workaround: corrected the payload syntax and retry with the semantic tool.
* Root cause: malformed agent-generated replacement body.

### ST-0100: semantic_replace_construct selector used full if statement instead of selector

* Date: 2026-09-27
* Tool: `semantic_replace_construct`
* Target: `internal/projectverify/executor.go`, `runNormalization`
* Observed failure: the selector passed the entire short `if` statement instead of its short initializer or condition; the tool returned available construct paths and made no change.
* Workaround: retry using the listed construct path for the intended command execution branch.
* Root cause: misunderstanding of the operation discriminator field.

### ST-0101: semantic_replace_construct rejected environment setup sequence

* Date: 2026-09-27
* Tool: `semantic_replace_construct`
* Target: `internal/projectverify/executor.go`, `runNormalization`
* Observed failure: the replacement contained an environment setup statement followed by an `if`, while the operation accepts a single complete `if` construct. No change was made.
* Workaround: wrap setup and command execution in one if initializer expression.
* Root cause: replacement payload shape exceeded the tool operation contract.

### ST-0104: semantic_replace_body rejected command helper payload

* Date: 2026-09-27
* Tool: `semantic_replace_body`
* Target: `internal/projectverify/executor.go`, `runHookCommand`
* Observed failure: the final `runCommandWithEnv` call was missing its closing parenthesis, and the tool rejected the candidate syntax without writing.
* Workaround: correct the call syntax and retry.
* Root cause: malformed generated body.

### ST-0106: body replacement left a new standard-library reference unresolved

* Date: 2026-09-27
* Tool: `semantic_replace_body`
* Target: `internal/mcp/batch.go`, `Server.ExecuteBatch`.
* Failure: the replacement succeeded with a new `reflect.DeepEqual` reference but no reflect import; make check caught the undefined identifier.
* Workaround: run semantic import organization before compiling again.
* Root cause: body replacement did not infer the new import in this call.

### ST-0108: semantic batch rejected plan-rendering body string

* Date: 2026-09-27
* Tool: `semantic_batch` with `semantic_replace_body`
* Target: `tools/benchmark-harness/planner.go`, `RenderBenchmarkPlan`
* Failure: the replacement payload contained unescaped newline characters inside Go string literals, so the tool rejected the body before writing it. Earlier independent edits in the batch had already succeeded.
* Workaround: log the failure and retry the remaining body with correctly escaped Go string literals.
* Root cause: newline escaping was lost while constructing the nested tool payload.

### ST-0109: semantic_insert_function rejected multiline test strings

* Date: 2026-09-27
* Tool: `semantic_insert_function`
* Target: `tools/benchmark-harness/cli_integration_test.go`
* Failure: the inserted test payload had an unescaped newline inside a Go string literal, so syntax validation rejected it without writing.
* Workaround: preserve backslashes in nested source text with a raw string payload and retry.
* Root cause: source escaping was lost in the outer JavaScript template literal.

### ST-0111: semantic_rename blocked by a temporarily incomplete aggregate type edit

* Date: 2026-09-27
* Tool: `semantic_rename`
* Target: `cmd/docgen/benchmark_compare.go`, `benchmarkMetricSummary.average`
* Observed failure: renaming `average` to `median` was rejected because the preceding type edit had removed fields still referenced by the average method. No rename was applied.
* Workaround: finish the dependent method updates so the package parses, then retry the semantic rename.
* Root cause: the rename backend requires a clean package and the type and method updates were applied in separate operations.

### ST-0112: semantic_replace_body emitted malformed nested browser replacement code

* Date: 2026-09-27
* Tool: `semantic_replace_body`
* Target: `cmd/docgen/benchmark_aggregate.go`, `scopedBenchmarkBrowserShortcode`
* Observed failure: replacement text containing JavaScript template newlines broke the enclosing Go raw string and produced a compile diagnostic after the edit had been applied.
* Workaround: replace the function body with escaped Go string literals and re-run semantic verification.
* Root cause: nested Go and JavaScript template quoting was not preserved by the submitted payload.

### ST-0115: semantic_replace_body organized the wrong YAML import

* Date: 2026-09-29
* Tool: `semantic_replace_body` with automatic import organization
* Target: `tools/benchmark-harness/oracle.go`, `parseYAMLFrontmatter`
* Observed failure: the body edit succeeded, but import cleanup selected `gopkg.in/yaml.v2`; this repository requires `gopkg.in/yaml.v3`, so diagnostics reported an unavailable module and undefined `yaml`.
* Workaround: replace the import with the existing `gopkg.in/yaml.v3` module and run the repository checks.
* Root cause: automatic import resolution selected a similarly named module without honoring the existing `go.mod` dependency.

### ST-0117: semantic_verify reached workspace-wide hooks

* Date: 2026-09-29
* Tool: `semantic_verify`
* Target: `internal/mcp/startup_bench_test.go`
* Observed failure: verification surfaced an unrelated harness G204 diagnostic and refused a workspace-wide lint hook without workspace trust.
* Workaround: fix the harness diagnostic and run the repository's `make check` gate.
* Root cause: the selected-file operation also invokes workspace-wide verification hooks.

### ST-0120: semantic_insert_structure rejected type overwrite

* Date: 2026-09-29
* Tool: `semantic_insert_structure`
* Target: `Server` in `internal/mcp/server.go`
* Observed failure: two attempts to update an existing struct with `kind: type` and `overwrite: true` returned `overwrite is unsupported for type structure insertion`; no source changed.
* Workaround: use an atomic source edit for the struct fields, then continue using semantic tools for function bodies and declarations.
* Root cause: the tool advertises a generic overwrite option but does not support replacing type declarations.

### ST-0122: semantic_replace_body omitted a required import

* Date: 2026-09-29
* Tool: `semantic_replace_body`
* Target: `TestFiveLocationRenameFixtureRequiresAllFiveRenames` in `tools/benchmark-harness/rename_five_fixture_test.go`
* Observed failure: inserting `exec.CommandContext` succeeded without adding `os/exec`; `make check` reported `undefined: exec`.
* Workaround: run semantic import organization or add the missing import, then rerun repository checks.
* Root cause: automatic import organization did not resolve the new package-qualified reference.

### ST-0124: semantic_scaffold_file inserted a blank line after the package comment

* Date: 2026-09-29
* Tool: `semantic_scaffold_file`
* Target: `internal/astedit/remove.go`
* Observed failure: the scaffold created a blank line between the package comment and package declaration, and `make check` reported a detached package comment.
* Workaround: remove the blank line and rerun repository checks.
* Root cause: the scaffold formatter preserved spacing that violates the repository's package-comments lint rule.

### ST-0125: semantic_insert_function rejected multiple declarations

* Date: 2026-09-29
* Tool: `semantic_insert_function`
* Target: `internal/operation/wire_remove.go`
* Observed failure: the request contained several constants, variables, and functions; the tool rejected it because it accepts one function declaration only. No source changed.
* Workaround: insert each declaration with its matching semantic insertion operation.
* Root cause: the tool validates a single function snippet and does not accept a file-sized declaration batch.

### ST-0126: semantic inspection tools targeted the parent checkout

* Date: 2026-09-29
* Tool: `semantic_outline`, `semantic_lookup`
* Target: `internal/astedit/move_construct.go` and declarations in `internal/operation` and `internal/backend/golang` in the assigned `.scratch/worktrees/refactor-move` worktree
* Observed failure: semantic outline/lookup requests resolved against the parent checkout instead of this assigned worktree; the move file was reported missing, and a lookup for `ReplaceConstruct` returned not found despite the declaration being in a different file.
* Workaround: use local repository inspection and edits confined to the assigned branch.
* Root cause: the MCP semantic service is bound to the parent checkout and exposes no worktree selector.

### ST-0129: semantic tools and edits resolved against the parent checkout

* Date: 2026-09-29
* Tool: `semantic_outline`, `semantic_lookup`, `semantic_scaffold_file`, and worktree-edit shell operations
* Target: `internal/astedit/extract_variable.go` plus README, ADR index, MCP schema, operation wire, capability, and skill files that were meant for the assigned `refactor-extract-variable` worktree
* Observed failure: semantic inspection resolved files in the parent checkout, and an attempted scaffold request failed before creating its target. Subsequent changes intended for this branch appeared in the parent checkout. The parent preserved those changes in `.scratch/extract-variable-parent-snapshot/` and restored its integration checkout; no parent changes remain.
* Workaround: copy the preserved patch and untracked files into the assigned worktree, then continue using only absolute worktree paths and the worktree-local CLI or atomic file updates.
* Root cause: the shared semantic MCP service is bound to the parent checkout; a shell edit command also ran without an explicit worktree directory, allowing checkout context to drift.

### ST-0130: semantic_batch rejected construct movement

* Date: 2026-09-29
* Tool: `semantic_batch` with `semantic_move_construct`
* Target: same-package declaration moves across Go files
* Observed failure: the reported batch request returns `tool is not batchable: semantic_move_construct` during batch preflight, before edits execute. The registry definition sets `Batchable: false`; `ExecuteBatch` rejects non-batchable entries. A local CLI replay was unavailable because the first required `make check` could not resolve uncached modules with `GOPROXY=off`.
* Workaround: mark the operation batchable only after implementing deferred multi-file outcomes, destination receipts, and batch write-set handling; then add public MCP and CLI regression coverage.
* Root cause: batch eligibility and post-processing track only one `WrittenFile`, while moves persist source and destination files.

### ST-0131: batched move verification changed an unrelated source header

* Date: 2026-09-29
* Tool: `semantic_batch` with same-package `semantic_move_construct` operations
* Target: isolated Chi router acceptance fixture, `path_value.go`
* Observed failure: the 19-operation batch returned success and preserved the original test files, but the independent `TestSemeditRoutingDecomposition` oracle found that global final verification removed one blank line between the build header and package clause in untouched `path_value.go`. The acceptance fixture is isolated outside this worktree.
* Workaround: derive built-in Go formatting and import-organizing paths from actual changed files after all batch edits; keep workspace diagnostics and configured normalization hooks on their existing scope. Regression coverage compares the untouched header byte-for-byte.
* Root cause: final `semantic_verify` formatted the entire workspace, including unchanged Go files.

### ST-0132: semantic_insert_structure cannot update an existing struct

* Date: 2026-09-30
* Tool: `semantic_insert_structure` with `kind: type` and `overwrite: true`
* Target: `internal/mcp/server.go`, `Server`; `internal/godiagnostics/manager.go`, `managedSession` and `Document`
* Observed failure: the tool rejected the request with `overwrite is unsupported for type structure insertion`; it made no change.
* Workaround: add session-manager and executable-identity fields with narrowly scoped atomic source updates, then continue using semantic tools for function bodies.
* Root cause: the type insertion tool cannot overwrite an existing struct, and no exposed semantic operation inserts a struct field.

### ST-0133: scaffold header and declaration comments need text repair

* Date: 2026-09-30
* Tool: `semantic_scaffold_file`, `semantic_insert_type`, and `semantic_insert_function`
* Target: `internal/godiagnostics/manager.go`, `internal/atomicfile/write.go`, `internal/atomicfile/write_test.go`, `internal/adapters/golang/rename_edit.go`, and the cancellation contract on `internal/mcp/server.go`, `Serve`
* Observed failure: the scaffold separated its package comment from the package clause with a blank line, triggering `revive`. Inserted exported declarations require Go documentation comments, but no exposed semantic operation adds a comment to an existing declaration.
* Workaround: make narrow atomic comment edits after semantic insertion, including the exported cancellation/reader ownership contract, then run formatting and lint checks.
* Root cause: the scaffold's package-header layout violates the repository linter, and the semantic editing API has no add-comment operation for existing declarations.

### ST-0134: semantic_insert_function rejected malformed test snippet

* Date: 2026-09-30
* Tool: `semantic_insert_function`
* Target: `internal/godiagnostics/diagnostics_test.go`, `TestManagedDiagnosticsReuseAndRefresh`
* Observed failure: the generated test source contained a malformed shell-quote rune literal; the tool reported `illegal rune literal` and left the file unchanged.
* Workaround: simplify the wrapper script to receive paths through environment variables and retry semantic insertion.
* Root cause: the caller escaped a nested Go and shell string incorrectly; the tool correctly rejected invalid Go syntax.

### ST-0136: generated PID wrapper used a literal dollar sign

* Date: 2026-09-30
* Tool: `semantic_replace_body`
* Target: `internal/mcp/server_test.go`, `newManagedMCPFixture`
* Observed failure: the cancellation test's shell wrapper recorded `$` instead of its process ID; the resulting Go source parsed, but the process cleanup assertion could not read a PID.
* Workaround: correct the wrapper through semantic body replacement using a replacement callback.
* Root cause: JavaScript string replacement interpreted `$$` as a replacement escape before passing the source to the tool.

### ST-0139: large migration batch leaves compile integration errors in final verification

* Date: 2026-09-30
* Tool: `semantic_batch` through the semedit MCP CLI
* Target: source-buffer integration across AST operations, the registry, and shared gopls
* Observed failure: the initial migration batch, integration/test batch, and reviewed source-view repair batch applied their mutations, but final verification reported diagnostics and the configured golangci-lint hook failed.
* Workaround: retain the applied edits, run the repository mechanical fixes and compile gate, then repair the concrete integration errors with semantic tools.
* Root cause: the prepared migration needs its compile integration pass; the tool correctly preserved applied edits with diagnostics.

### ST-0140: declaration replacement and scaffolding need comment repair

* Date: 2026-09-30
* Tool: `semantic_remove_construct`, `semantic_insert_type`, and `semantic_scaffold_file`
* Target: `internal/sourcebuf/buffer.go`, `internal/atomicfile/staged.go`, `internal/mcp/buffers.go`, and replaced registry/session/resolver types
* Observed failure: generated package headers have a separating blank line; removing and reinserting types drops exported declaration comments, including an adjacent resolver declaration comment. The formatter does not repair the resulting revive failures.
* Workaround: repair declaration and package comments with a scoped atomic source rewrite; retain semantic tools for function bodies and declarations.
* Root cause: the available tools have no declaration-comment update operation, and scaffold/comment association does not preserve the lint contract.

### ST-0141: buffered consistency repair batch failed final verification

* Date: 2026-09-30
* Tool: `semantic_batch` through the semedit MCP CLI
* Target: source buffer, rename, batch journal, and protocol test helper
* Observed failure: the repair batch applied its edits but returned an error at final verification.
* Workaround: retain the applied edits, inspect the saved response, and run the configured mechanical and compile checks.
* Root cause: the final verification gate checks the complete integration while declaration replacement can require comment repair.

### ST-0142: move overlay migration failed its final integration gate

* Date: 2026-09-30
* Tool: `semantic_batch` through the semedit MCP CLI
* Target: cross-file and cross-package move planner, registry, and package loader
* Observed failure: the prepared move migration, following session/rename repair batch, and same-package scope repair batch applied their edits but final verification returned an error.
* Workaround: retain applied edits, inspect the saved verification result, and run the configured mechanical checks before repairing integration failures.
* Root cause: the complete verification gate also checks callers affected by private signature changes and import cleanup.

### ST-0143: import organization moved a function suppression into imports

* Date: 2026-09-30
* Tool: `semantic_organize_imports`
* Target: `internal/astedit/move_cross_package.go`
* Observed failure: the move planner's existing function comment and `nolint:funlen,gocognit` directive moved into the import block after helper insertion and import organization.
* Workaround: restore the comment and directive at the planner declaration with an atomic comment-only rewrite.
* Root cause: declaration and import insertion reused stale comment positions rather than retaining their declaration association.

### ST-0151: verifier executable built with an older Go type checker

* Date: 2026-10-01
* Tool: semantic verification in development CLI MCP batches
* Target: `internal/astedit/inspect.go` and other embedded struct literals
* Observed failure: Verification rejected promoted literal fields accepted by Go 1.27 builds after `go fix` modernized them.
* Root cause: The discovered `gopls` v0.23.0 executable was built with Go 1.26.6 and used that version of `go/types`.
* Workaround: Rebuilt the same cached v0.23.0 source with Go 1.27.1 into project-local `.scratch/go/bin/gopls`, then supplied that executable to all six workers. Check-only verification passed with no promoted-field errors.

### ST-0147: semantic_insert_function rejected private helper visibility

* Date: 2026-10-01
* Tool: `semantic_insert_function`
* Target: `internal/operation/value_schema.go`
* Observed failure: the request explicitly selected public access for lowercase helper `validateBounds`; the tool rejected the visibility mismatch. No source changed from that call.
* Workaround: insert lowercase helpers with private access.
* Root cause: Go access mode must match identifier casing when explicitly supplied.

### ST-0148: exported planner declarations needed documentation comments

* Date: 2026-10-01
* Tool: `semantic_insert_type`, `semantic_insert_function`
* Target: `internal/operation/value_schema.go` and `internal/operation/typed_descriptors.go`
* Observed failure: repository `make check` rejected the successful insertions because exported Go declarations lacked required documentation comments. No incorrect AST was reported.
* Workaround: add Go documentation comments with an atomic source update because no registered semantic operation adds comments to existing declarations.
* Root cause: the semantic insertion operations accept declarations but expose no doc-comment edit operation.

### ST-0151: semedit-next replace-body rejected a declaration wrapper

* Date: 2026-10-01
* Tool: `semedit-next replace-body`
* Target: `ParameterSpecSchema` in `internal/operation/typed_descriptors.go`
* Observed failure: the request passed a complete `func` declaration in `--body`; the CLI rejected the nested declaration before writing.
* Workaround: pass only the function body statements.
* Root cause: `replace-body` accepts statements without a function signature or braces.

### ST-0153: atomic operation struct update did not match declaration layout

* Date: 2026-10-01
* Tool: atomic Python source update
* Target: `internal/operation/operation.go`
* Observed failure: the script expected an incorrect field-block string and aborted before mutation.
* Workaround: inspect the exact struct declarations and update their matching field blocks.
* Root cause: the assumed gofmt alignment differed from the source layout.

### ST-0154: make check exposed missing Go documentation comments

* Date: 2026-10-01
* Tool: `make check` (revive)
* Target: `internal/operation/typed_descriptors.go` and `internal/operation/value_schema.go`
* Observed failure: the linter rejected exported declarations inserted without documentation comments.
* Workaround: add declaration comments atomically; no semantic operation exposes comment-only edits.
* Root cause: inserted declarations did not include the doc comments required by repository lint.

### ST-0158: semantic CLI has no anonymous closure edit for registry parse wrapping

* Date: 2026-10-01
* Tool: `semedit-next` operation inventory
* Target: `internal/operation/operation.go`
* Observed failure: no registered semantic operation can wrap the anonymous `Entry.Parse` and `Entry.Invoke` parse closures with shared contract validation.
* Workaround: update only those closure bodies with atomic source edits after locating them through semantic lookup.
* Root cause: the CLI exposes function-body replacement but no closure-body editing operation.

### ST-0160: semantic insert-decl rejected const spec without declaration wrapper

* Date: 2026-10-01
* Tool: `semedit-next insert-decl`
* Target: `internal/operation/operation.go`
* Observed failure: the source contained a constant specification rather than a complete `const` declaration; validation failed before mutation.
* Workaround: submit each constant inside its own `const (...)` declaration.
* Root cause: insert-decl parses declaration nodes, not individual specs.

### ST-0167: Make-variable cache settings did not reach the linter process

* Date: 2026-10-01
* Tool: `make check`
* Target: full-worktree Go lint
* Observed failure: passing `GOMODCACHE` and `GOCACHE` as Make command-line variables left `golangci-lint` unable to load `pgregory.net/rapid` with `GOPROXY=off`; lint stopped before tests.
* Workaround: export the cache variables in the command environment while keeping `MAKE_JOBS=2` set.
* Root cause: the lint child process did not use the Make-only cache variable values.

## ST-0178: Tooling log command targeted the coordinator checkout

* Tool: atomic Python append for `docs/SUBOPTIMAL_TOOLS.md`.
* Target: coordinator checkout path; the filesystem rejected the write and no file was created there.
* Observed failure: `PermissionError` because the command ran with the coordinator checkout as its working directory, outside this worker’s writable root.
* Workaround: rerun the atomic append from this assigned worktree.
* Root cause: the command used a relative path with the wrong working directory.

## ST-0184: Semantic insertion did not create the requested test declaration

* Tool: `semedit-next insert`.
* Target: `internal/operation/value_schema_test.go`.
* Observed failure: the command returned no declaration result, and a follow-up `inspect-symbol` confirmed the requested registration test was absent.
* Workaround: retry with a plain Go address operator in the source snippet and verify the inserted symbol before proceeding.
* Root cause: the insertion payload encoded the address operator as a Unicode escape that was not accepted as Go source.

## ST-0185: Semantic insertion rejected literal newline escapes

* Tool: `semedit-next insert`.
* Target: `internal/operation/value_schema_test.go`.
* Observed failure: the retry returned `syntax error: illegal character U+005C` because the Go snippet contained literal backslash-n sequences.
* Workaround: provide actual line breaks in the source snippet and inspect the resulting declaration.
* Root cause: shell and JavaScript escaping preserved the escape text instead of producing source newlines.

## ST-0187: Semantic body replacement returned without applying source

* Tool: `semedit-next replace-body`.
* Target: `internal/operation/receipt_finalize.go` function `FinalizeEditReceipt`.
* Observed failure: the command produced no confirmation and left the function referencing the removed `ChangePaths` field, which the full check detected.
* Workaround: retry with actual line breaks in the replacement body and inspect the resulting declaration.
* Root cause: the replacement payload contained literal backslash-n characters from shell/JavaScript escaping.

## ST-0188: Escaped semantic function edits were rejected without inspection

* Tool: `semedit-next replace-body` and `insert`.
* Target: `internal/operation/value_schema.go`, `internal/operation/parse.go`, and `internal/operation/output_projection.go`.
* Observed failure: several function-edit commands encoded `\n` literally; the CLI rejected those snippets and the existing declarations remained unchanged. A later symbol check found the bounded numeric helper absent.
* Workaround: use actual line breaks in Go snippets, check each command exit code, and inspect every edited declaration.
* Root cause: JSON/JavaScript escaping preserved backslash-n text when real source newlines were required.

## ST-0190: Semantic edit tools cannot add a local binding inside a closure

* Tool: `semedit-next insert` / `replace-body`.
* Target: `internal/mcp/schema_test.go` receipt finalizer test closure.
* Observed limitation: fixing the test requires adding a local request binding inside a nested closure; the available semantic tools only insert top-level declarations or replace the complete enclosing function body.
* Workaround: apply one exact atomic insertion, then verify with `make check` and the observable MCP test.
* Root cause: nested statement insertion is not supported.

### ST-0169: local CLI scaffolding rejects plain purpose text

* Date: 2026-10-01
* Tool: `semedit scaffold-file` CLI
* Target: `internal/contractplan/plan.go` in the requested worktree
* Observed failure: passing `--purpose-header 'WHY: ...'` returned `syntax error: invalid purpose header: scaffold.go:1:1: expected 'package', found WHY`; the command created no file.
* Workaround: retry with the purpose header expressed as a Go line comment, then verify the file landed in this worktree.
* Root cause: the CLI help says the header explains file purpose but does not state that the value must be valid Go comment syntax.

### ST-0175: final check surfaced new-package lint blockers

* Date: 2026-10-01
* Tool: `make check` / `golangci-lint run --fix`
* Target: `internal/contractplan`
* Observed failure: the repository check stopped with 20 findings: exported planner API declarations lacked documentation comments, and five method/tuple index loops were flagged for Go 1.22 integer-range syntax.
* Workaround: add concise API documentation and rewrite loops using range over the go/types method and tuple sequences.
* Root cause: the first implementation pass had not yet applied the repository's configured revive and intrange rules.

### ST-0177: documentation insertion script found a header mismatch

* Date: 2026-10-01
* Tool: local atomic source update script
* Target: `internal/contractplan/extract_interface.go`
* Observed failure: after adding declaration comments to `plan.go`, the script stopped because its expected `WHY` header plus package-clause literal did not match the scaffolded file; no change was made to the target file.
* Workaround: inspect the actual header and apply the package and API comments against the observed text.
* Root cause: the script assumed the scaffold header text had not been reformatted by semantic insertion.

### ST-0178: package comment formatting failed revive validation

* Date: 2026-10-01
* Tool: `make check` / `golangci-lint run --fix`
* Target: `internal/contractplan/extract_interface.go`
* Observed failure: revive reported the package comment detached and then reported no package comment; the check stopped before tests.
* Workaround: place the package documentation directly above the package clause with no blank line between the comment group and the clause.
* Root cause: the scaffolded blank line after the `WHY` comment split the package doc from the package clause.

### ST-0164: test body replacement removed fixture setup

* Date: 2026-10-01
* Tool: `semantic_replace_body` inside `semantic_batch`
* Target: `internal/contractplan/planner_test.go` (`TestPlanImplementContractGeneratesOnlyMissingPanicStub`)
* Observed failure: `make check` found undefined `err` and `plan` identifiers because the replacement body contained only assertions and omitted the existing fixture and planner invocation.
* Workaround: Restore the complete test body, including source fixture construction and planner invocation, through a semantic body replacement.
* Root cause: The replacement body was authored as a suffix after overlooking that `semantic_replace_body` replaces the entire body.

### ST-0166: contract handler helper snippet missed a statement separator

* Date: 2026-10-01
* Tool: `semantic_batch` with `semantic_insert_function`
* Target: `internal/operation/wire_contracts.go` (`canonicalContractPath`)
* Observed failure: The insert operation rejected the function with `expected ';', found info` because two statements were joined on one source line; earlier successful type and loader children were persisted in the returned final diff.
* Workaround: Inspect the current file and reinsert the corrected function with statements on separate lines.
* Root cause: The generated function snippet concatenated `}` and the following declaration without a newline or semicolon.

### ST-0167: semantic verification reported mismatched runtime type fields

* Date: 2026-10-01
* Tool: `semantic_batch` final `semantic_verify` with workspace trust enabled
* Target: `internal/operation/wire_contracts.go` (the helper inserts succeeded)
* Observed failure: The final verification hook failed with `unknown field` diagnostics for `backend.InspectedSymbol` fields in `internal/astedit/inspect.go`, although the authoritative `make check` passed on this worktree immediately before these handlers began.
* Workaround: Preserve the successful helpers, inspect the checked source/binary views, and use the repository Make check to validate package correctness while reporting the semantic verification mismatch.
* Root cause: The active gopls v0.23.0 binary was built with Go 1.26 and rejected Go 1.27-valid embedded-field composite literals in `backend.InspectedSymbol`.

### ST-0168: semantic verification repeated the runtime type-view mismatch

* Date: 2026-10-01
* Tool: `semantic_batch` final `semantic_verify` with workspace trust enabled
* Target: `internal/operation/wire_contracts.go` (`contractImportBindings`, candidate rendering, and typecheck helpers)
* Observed failure: All three helper mutations reported success, then the trusted verification hook repeated the same `backend.InspectedSymbol` unknown-field diagnostics recorded in ST-0167.
* Workaround: Retain the successful helpers and validate them through the prescribed Make check; do not repeat workspace verification until the checked binary/package view is reconciled.
* Root cause: The active gopls v0.23.0 binary was built with Go 1.26 and rejected Go 1.27-valid embedded-field composite literals in `backend.InspectedSymbol`.

### ST-0170: parser verification ran before operation schemas were declared

* Date: 2026-10-01
* Tool: `semantic_verify` with `trust_workspace=true`
* Target: `internal/operation/wire_contracts.go`
* Observed failure: The trusted `golangci-lint` hook reported undefined `extractInterfaceParams` and `implementContractParams` referenced by the persisted parser children. Its response also repeated gopls v0.23.0 / Go 1.27 compatibility diagnostics for valid embedded-field composite literals in `backend.InspectedSymbol`.
* Workaround: Keep the four parser children and finish the operation parameter declarations before rerunning the prescribed Make check.
* Root cause: The parser functions were inserted before their public operation schema definitions, so this intermediate source state does not type-check.

### ST-0171: trusted batch linter rejected the schema declaration intermediate state

* Date: 2026-10-01
* Tool: `semantic_batch` with two `semantic_insert_decl` children through a locally initialized trusted MCP server
* Target: `internal/operation/wire_contracts.go` (`extractInterfaceParams`, `implementContractParams`)
* Observed failure: Both declaration children returned `status=ok`; the final trusted verification hook reported `golangci-lint` failed. The JSON response was truncated after the failure and gopls v0.23.0 / Go 1.27 compatibility diagnostics, so no more specific linter diagnostic was available from that invocation.
* Workaround: Preserve both parameter declarations and continue the dependent operation wiring before the next authoritative `make check`.
* Root cause: The parameter declarations were added while parser and analysis helpers remain disconnected from registered operation definitions; this is an intermediate implementation state.

### ST-0172: trusted contract-handler batch stopped at intermediate lint

* Date: 2026-10-01
* Tool: `semantic_batch` with ten `semantic_insert_function` children through a locally initialized trusted MCP server
* Target: `internal/operation/wire_contracts.go` (revision, preflight, output, and handler helpers)
* Observed failure: All ten mutation children returned `status=ok`; final `semantic_verify` reported the `golangci-lint` hook failed. The response also repeated gopls v0.23.0 / Go 1.27 compatibility diagnostics for valid embedded-field composite literals in `backend.InspectedSymbol`; the large response was truncated before the complete hook output could be captured.
* Workaround: Preserve the successful helpers. Connect their parsers and operation definitions after the central structured schema API is integrated, then use the prescribed Make check as the next full verification.
* Root cause: The new handler helpers are not yet reachable from registered typed operation definitions, so this is an unfinished integration state.

### ST-0173: contract parser batch could not connect to Go diagnostics

* Date: 2026-10-01
* Tool: `semantic_batch` with two `semantic_replace_body` children through a local trusted MCP server
* Target: `internal/operation/wire_contracts.go` (`parseExtractInterfaceRequest`, `parseImplementContractRequest`)
* Observed failure: Batch diagnostics failed before child execution with `initialize gopls diagnostics session: JSON-RPC client is closed`; no mutations were applied.
* Workaround: Keep both parser bodies unchanged and inspect/restart the local Go diagnostics session before retrying.
* Root cause: The MCP operation could not initialize its retained gopls diagnostics client in the current server process.

### ST-0174: PTY truncated an MCP request sent to the persistent worker

* Date: 2026-10-01
* Tool: local `exec_command` PTY feeding the trusted MCP wrapper
* Target: none; the request contained parser body replacements for `internal/operation/wire_contracts.go`
* Observed failure: The JSON request exceeded the PTY canonical input limit; the wrapper received no complete request or MCP response. I interrupted the wrapper, and no source changes were applied.
* Workaround: Restart the persistent trusted MCP wrapper and send smaller individual requests below the PTY input limit.
* Root cause: A large JSON-RPC request was sent as one canonical-mode PTY line.

### ST-0176: optional revision parser body no longer matched its signature

* Date: 2026-10-01
* Tool: `semantic_batch` with a constant insert, two parameter declaration replacements, and three function body replacements through a local trusted MCP server
* Target: `internal/operation/wire_contracts.go` (`parseExpectedContractRevision` and both request parsers)
* Observed failure: The mutation response ended with a trusted `golangci-lint` failure. Current source inspection shows the function signature still returns `(*string, error)` while the replacement body returns three values; its two callers also expect three values. Earlier declaration children succeeded.
* Workaround: Restore the declared two-result helper contract and have omitted revisions return a non-nil pointer to the empty sentinel; `checkContractRevision` will treat that sentinel as absence.
* Root cause: `semantic_replace_body` changes only a function body, but the edit changed the helper's return arity without changing its declaration.

### ST-0175: persistent MCP FIFO wrapper received an empty line after one edit

* Date: 2026-10-01
* Tool: local `exec_command` FIFO transport for the trusted MCP wrapper
* Target: none; the second parser body request was not passed to MCP
* Observed failure: The wrapper had returned success for the first parser body replacement, then exited with `JSONDecodeError` on an empty line before receiving the next request. No source mutation was applied for the second request.
* Workaround: Use the one-shot trusted MCP wrapper with the checked binary and local gopls path for each call.
* Root cause: The persistent wrapper used an `O_RDWR` FIFO descriptor and interpreted an empty FIFO read as a request line.

### ST-0177: trusted panic-shadow handler batch failed final lint

* Date: 2026-10-01
* Tool: `semantic_batch` with `semantic_insert_function` and `semantic_replace_body` through the root trusted pipe helper
* Target: `internal/operation/wire_contracts.go` (`requirePanicBuiltin`, `runImplementContract`)
* Observed failure: Both mutation children reported `status=ok`; the aggregate result had `isError=true` at final verification. The rendered tool output was truncated before the verification hook diagnostic could be captured. The current source contains both successful children.
* Workaround: Preserve the helper and its call site; inspect the next prescribed Make check after the typed operation definitions make the helpers reachable.
* Root cause: The contract handlers remain disconnected from registered operation definitions while the schema integration is pending; final verification rejects this intermediate state.

### ST-0178: planner builtin guard batch was rejected before mutation

* Date: 2026-10-01
* Tool: `semantic_batch` through the root trusted pipe helper
* Target: `internal/contractplan/implement_contract.go` and `internal/contractplan/planner_test.go`
* Observed failure: The first `semantic_replace_body` child was rejected with `syntax error: string literal not terminated (and 2 more errors)`; the response reported an empty final diff and no child mutation.
* Workaround: Preserve planner sources and use the handler-level builtin-shadow refusal with its no-publication test; do not retry the malformed declaration snippet.
* Root cause: The generated Go body string was malformed during payload construction.

### ST-0179: planner panic-shadow batch failed its final verification

* Date: 2026-10-01
* Tool: `semantic_batch` with `semantic_replace_body` and two `semantic_insert_function` children through the root trusted pipe helper
* Target: `internal/contractplan/implement_contract.go` and `internal/contractplan/planner_test.go`
* Observed failure: The helper returned `isError=true` at final verification. Its large response was truncated after the first child result; inspection of current source confirms the planner guard and both-case refusal test were persisted.
* Workaround: Preserve the edits and use the next required Make check after operation schema/registration integration to validate the whole worktree.
* Root cause: Final lint still sees the incomplete contract operation wiring in `internal/operation/wire_contracts.go`; the panic-shadow edits themselves are present.

### ST-0180: sandbox denied Git index creation for the worktree

* Date: 2026-10-01
* Tool: `git add` for the assigned contracts worktree
* Target: index for branch `codex/flow-contracts`; no index or source mutation occurred
* Observed failure: Git could not create `<repository-root>/.git/worktrees/flow-contracts/index.lock` because the shared Git administrative directory is outside the writable worktree.
* Workaround: Retry the user-authorized explicit-path stage and commit through the sandbox escalation path.
* Root cause: The worktree shares Git administrative storage outside its source directory.

### ST-0182: semantic replace-body rejected the legacy path parameter

* Date: 2026-10-01
* Tool: `semantic_replace_body` through the root trusted pipe helper
* Target: `internal/operation/wire_contracts_test.go`; no source mutation occurred
* Observed failure: The tool rejected `path` and listed `file` as the supported source location parameter.
* Workaround: Retry with `file` while keeping the reviewed replacement body unchanged.
* Root cause: Semantic operations have operation-specific location keys; the payload reused the outline key.

### ST-0184: trusted contract renderer batch failed final lint

* Date: 2026-10-01
* Tool: `semantic_batch` with `semantic_remove_construct` and `semantic_insert_type`, through the root trusted pipe helper
* Target: `internal/contractplan/plan.go`
* Observed failure: Both mutation children reported `status=ok`; final `semantic_verify` reported `verification hook golangci-lint failed` with exit status 1. The response included repository warnings and hints but was truncated before all diagnostics.
* Workaround: Preserve both source mutations, inspect their current bytes, then use the focused test and Make output to identify actionable lint errors.
* Root cause: Not established from the truncated response; do not infer a trust failure or rollback.

### ST-0185: trusted planner proof batch failed final lint

* Date: 2026-10-01
* Tool: `semantic_batch` with `semantic_insert_type`, `semantic_insert_function`, and `semantic_replace_body`
* Target: `internal/contractplan/planner_test.go`
* Observed failure: All seven mutation children reported `status=ok`; final `semantic_verify` reported `golangci-lint` exit status 1. The tool returned the successful source diff, but the diagnostic tail was truncated.
* Workaround: Preserve all inserted proof code, run the required Make check for complete lint/compiler diagnostics, then fix the reported source issue.
* Root cause: Not established from the truncated response; do not infer the successful children rolled back.

### ST-0186: parser proof batch response was truncated

* Date: 2026-10-01
* Tool: `semantic_batch` with `semantic_replace_body` and `semantic_insert_function`
* Target: `internal/operation/wire_contracts.go` and `internal/operation/wire_contracts_test.go`
* Observed failure: The first child reported `status=ok`. The helper returned `isError=true` and its large structured diff was truncated before the second child status and final verification diagnostic could be read.
* Workaround: Preserve the successful source change, inspect current source bytes and run focused parser tests before any retry.
* Root cause: Not established from the truncated response; do not infer that either child was rolled back.

### ST-0187: contract output type replacement passed mutations but failed final lint

* Date: 2026-10-01
* Tool: `semantic_batch` with `semantic_remove_construct` and `semantic_insert_type` through the trusted root pipe helper
* Target: `internal/operation/wire_contracts.go` (`contractOutput` declaration)
* Observed failure: Both mutation children reported `status=ok`; final `semantic_verify` ran the configured `golangci-lint` hook and exited 1. It reported the expected `expected_revision` goconst warning and five unused contract-registration helpers/parameters because public wiring is still incomplete; it did not report a trust failure.
* Workaround: Preserve both successful mutations and continue the coherent typed registration change, then run Make for full lint output.
* Root cause: The batch stopped at repository lint while the in-progress public operation definitions remained unregistered.

### ST-0188: typed contract output helper batch failed final verification

* Date: 2026-10-01
* Tool: `semantic_batch` with seven `semantic_insert_function` children through the trusted root pipe helper
* Target: `internal/operation/wire_contracts.go` (signature/result schemas, projector, receipt finalizer, resolver, and formatter)
* Observed failure: All seven insertion children reported `status=ok`; the final `semantic_verify` ran `golangci-lint` and exited 1 while the newly added contract output functions were still unregistered/unused. The response diagnostics also included the existing `expected_revision` goconst issue; no workspace-trust error occurred.
* Workaround: Preserve all inserted helpers, complete registration and imports in the same operation file, then run the required Make check for actionable diagnostics.
* Root cause: The helper batch was applied before its operation definitions and import organization were wired.

### ST-0189: semantic removal rejected an unsupported variable kind name

* Date: 2026-10-01
* Tool: `semantic_batch` with `semantic_remove_construct` as its first child through the trusted root pipe helper
* Target: `internal/operation/wire_contracts.go`; no mutation occurred
* Observed failure: The tool rejected kind `var` and listed `variable` as the supported construct kind. The batch stopped before any child mutation.
* Workaround: Retry the same coherent replacement batch using the supported `variable` kind.
* Root cause: The payload used a Go token spelling instead of the semantic operation construct vocabulary.

### ST-0190: contract registration batch failed final lint

* Date: 2026-10-01
* Tool: `semantic_batch` with contract output type replacement, registry registration, and import organization through the trusted root pipe helper
* Target: `internal/operation/wire_contracts.go` and `internal/operation/wire_backend.go`
* Observed failure: The output type replacement and `DefaultRegistry` body edit reported `status=ok`. Final `semantic_verify` ran the configured `golangci-lint` hook and returned exit status 1; the tool response was truncated before the full diagnostic list. No trust error was reported.
* Workaround: Preserve the successful mutations, inspect current source, and run `make check` for complete diagnostics before continuing.
* Root cause: Not yet established from the available truncated tool result.

### ST-0191: contract registration rejected continuation schema and lint issues

* Date: 2026-10-01
* Tool: `make check` after registering the contract operations
* Target: `internal/operation/wire_contracts.go` and registry initialization
* Observed failure: `docgen-source` panicked because the declared `result.receipt.continuations[].file` string continuation schema was not equivalent to the shared `SymbolTargetSchema` nonempty file leaf. `golangci-lint` also reported repeated `receiver_kind` literals. No focused tests were run.
* Workaround: Align continuation schemas with the shared descriptor leaf schema and name the repeated request field constant, then rerun Make.
* Root cause: The operation output contract used generic strings for constrained live target fields, and the operation-local field literal crossed the repository goconst threshold.

### ST-0192: contract schema and parser batch failed final verification

* Date: 2026-10-01
* Tool: `semantic_batch` with schema, parameter declaration, and parser updates through the trusted root pipe helper
* Target: `internal/operation/wire_contracts.go`
* Observed failure: The visible children reported `status=ok`; the aggregate response returned `isError=true` at final verification, but its full lint diagnostics were truncated. The changed parameter and parser declarations are present in the current source.
* Workaround: Preserve the completed mutations and use the required Make check to capture complete lint and registry diagnostics.
* Root cause: Not established from the truncated response.

### ST-0194: semantic replacement payload generator failed before MCP dispatch

* Date: 2026-10-01
* Tool: trusted `semantic_replace_body` helper invocation
* Target: `internal/operation/wire_contracts.go`; no mutation occurred
* Observed failure: The shell's inline Python payload generator had an unclosed parenthesis and emitted no JSON; the helper then rejected empty stdin before starting an MCP request.
* Workaround: Write the exact payload with a simpler JSON-safe script, then retry through the trusted pipe helper.
* Root cause: The one-line Python expression was malformed while escaping a multiline Go body.

## ST-0201: zsh split combined semantic inspection arguments incorrectly

* Tool: `<repository-root>/bin/semedit-next inspect-symbol`.
* Target: targeted function source inspection in MCP and operation files.
* Observed failure: the loop's `set -- $spec` under zsh did not preserve the `--file` and `--symbol` arguments, and the CLI rejected each request as missing `symbol`.
* Workaround: issue exact per-symbol commands with quoted arguments.
* Root cause: zsh does not perform the expected shell-style whitespace splitting without its explicit split flag.

## ST-0202: Semantic insert operation uses grouped CLI command

* Tool: `<repository-root>/bin/semedit-next insert-function --help`.
* Target: adding small helper declarations during the foundation fix.
* Observed failure: the CLI reported that `insert-function` is not a registered command.
* Workaround: inspect and use the supported grouped `insert` command for declarations, or use exact atomic updates only when the available semantic operation cannot express a structural change.
* Root cause: the CLI groups declaration insertion under `insert`.

## ST-0203: Shell quoting interrupted semantic body replacement

* Tool: `<repository-root>/bin/semedit-next replace-body`.
* Target: `internal/mcp/server.go` batch output schema builder.
* Observed failure: zsh reported `unmatched \"` while parsing the source passed with `--body`; the semantic operation did not run.
* Workaround: use a shell-safe encoded argument with a source string that contains no shell quote terminators.
* Root cause: the Go string literal contained an apostrophe inside a single-quoted shell argument.

## ST-0205: Semantic insertion visibility did not match Go test casing

* Tool: `semedit-next insert`.
* Target: new test in `internal/operation/value_schema_test.go`.
* Observed failure: the tool rejected `TestFinalizeEditReceiptRequiresObservedCandidateDigest` because visibility was explicitly `private` despite exported Go casing. No declaration was inserted.
* Workaround: retry with `--visibility public`.
* Root cause: the semantic editor enforces access casing for inserted declarations.

## ST-0206: Focused MCP suite found flat-schema assertions after batch union change

* Tool: `go test ./internal/operation ./internal/mcp`.
* Target: MCP schema assertions in `internal/mcp/schema_test.go`.
* Observed failure: three tests expected `results.items.required` and a flat `result` property; the new registry-derived `oneOf` shape made those assertions stale, and one test panicked converting the missing flat property.
* Workaround: update the tests to inspect the matching `tool` discriminator branch.
* Root cause: the observable schema changed from one open row object to per-registered-tool result branches.

## ST-0207: Focused MCP test found missing batch path description

* Tool: `go test ./internal/operation ./internal/mcp`.
* Target: batch result schema for `semantic_move_construct`.
* Observed failure: the move schema assertion panicked because the registry-derived `destination_file` property omitted the existing requested-target description.
* Workaround: include the established `file` and `destination_file` descriptions in each registered batch result branch.
* Root cause: replacing the static result object schema with per-tool branches dropped the static metadata descriptions.

## ST-0208: Focused MCP suite found nullable-schema expectation and flat receipt branch assumption

* Tool: `go test ./internal/operation ./internal/mcp`.
* Target: `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes` and `TestMCPProjectedReceiptUsesFinalDiskIdentityAndRevision` in `internal/mcp/schema_test.go`.
* Observed failure: the nullable schema uses JSON Schema `type: [string, null]`, not `anyOf`; the receipt test also still treated the per-tool result union as one flat object.
* Workaround: assert the emitted nullable type array and select the receipt branch by its `tool` discriminator.
* Root cause: the test expectations did not match the declared ValueSchema JSON serialization or the new branch-specific result schema.

## ST-0209: Semantic editor has no statement-level update for receipt test setup

* Tool: `semedit-next replace-body` / `insert`.
* Target: nested setup statement in `TestMCPProjectedReceiptUsesFinalDiskIdentityAndRevision` in `internal/mcp/schema_test.go`.
* Observed limitation: adding the planned digest to the existing finalized draft requires inserting one statement between an existing composite literal and call; available semantic operations replace whole function bodies or top-level declarations only.
* Workaround: apply one exact-match atomic source update and validate the public MCP test.
* Root cause: no nested statement edit is registered.

## ST-0210: Public batch test assumed the wrong structured-content nesting

* Tool: `go test ./internal/operation ./internal/mcp`.
* Target: `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes` in `internal/mcp/schema_test.go`.
* Observed failure: actual `semantic_batch` `tools/call` completed, but the test panicked because it looked for `results` directly under the wrong response object.
* Workaround: inspect the batch response wrapper and decode its `structuredContent` shape before checking typed values.
* Root cause: semantic_batch uses a distinct structured response wrapper from ordinary single-tool calls.

## ST-0213: Semantic editor cannot change one nested response assertion

* Tool: `semedit-next replace-body` / `insert`.
* Target: `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes` in `internal/mcp/schema_test.go`.
* Observed limitation: correcting two local response-path statements requires statement-level edits inside an existing test body; the registered semantic tools only replace whole function bodies or top-level declarations.
* Workaround: perform the narrow exact-match update atomically, then run the public MCP test.
* Root cause: no nested statement replacement operation is available.

## ST-0214: Injected batch registry omitted the final verification operation

* Tool: `go test ./internal/mcp -run TestBatchToolAdvertisesAndEmitsRegisteredResultShapes`.
* Target: actual batch `tools/call` result assertions in `internal/mcp/schema_test.go`.
* Observed failure: after the five test operations, the batch appended a `semantic_verify` error result because the isolated test registry does not register that post-processing operation.
* Workaround: accept and assert that explicit verification row while checking the typed result variants.
* Root cause: the fixture injects only the providers under test, while batch finalization resolves verification through the same registry.

## ST-0215: Batch test needs a nested synthetic-stage assertion

* Tool: `semedit-next replace-body` / `insert`.
* Target: result loop in `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes` in `internal/mcp/schema_test.go`.
* Observed limitation: the test must account for the batch executor's synthetic `semantic_verify` failure row, which requires a statement inside its existing loop.
* Workaround: make a narrow exact-match atomic update and assert the registered synthetic branch and error status.
* Root cause: the semantic editor cannot change nested statements in an existing declaration.

## ST-0216: Scoped commit could not create the shared worktree index lock

* Tool: `git add` for the first review-fix checkpoint.
* Target: only the owned foundation files in `codex/flow-foundation`.
* Observed failure: Git was denied creation of `<repository-root>/.git/worktrees/flow-foundation/index.lock`; no source files were staged and commit did not run.
* Workaround: retry the user-authorized commit with scoped access to the shared Git metadata.
* Root cause: this linked worktree stores its index under the common repository directory outside the assigned writable root.

## ST-0218: Semantic editor has no field or parameter insertion for guarded inspect links

* Tool: `semedit-next insert` / `replace-body`.
* Target: `ReplaceBodyReq`, `replaceBodyParams`, and the owner selector parameter in `internal/operation/wire_engine_mutations.go`.
* Observed limitation: the operation surface has no semantic edit for adding one struct field or changing one parameter element inside an existing declaration; whole-body replacement applies only to functions.
* Workaround: make exact-match atomic structural edits with fsync/rename, then verify through registered operations and public MCP calls.
* Root cause: no struct-field or composite-literal/element insertion operation is registered.

## ST-0219: Atomic structural update refused an ambiguous parameter-line match

* Tool: exact-match atomic Python edit.
* Target: `ReplaceBodyReq`, `replaceBodyParams`, and `replaceConstructParams` in `internal/operation/wire_engine_mutations.go`.
* Observed failure: an assertion found an unexpected match count for a repeated parameter declaration and aborted before writing the file.
* Workaround: use unique declaration-scoped source anchors, verify every replacement count, then atomically rename.
* Root cause: identical Go parameter lines recur in multiple operation declarations in the same file.

## ST-0220: Scoped owner-selector update did not match source text

* Tool: exact-match atomic Python edit.
* Target: `replaceConstructParams` in `internal/operation/wire_engine_mutations.go`.
* Observed failure: the declaration-scoped update found no exact match for the assumed owner-selector literal and aborted before writing any source.
* Workaround: re-inspect the actual declaration and use its current text as the anchor.
* Root cause: the source literal wording differs from the stale copied declaration text.

## ST-0221: Declaration-scoped source replacement found another stale literal

* Tool: exact-match atomic Python edit.
* Target: `ReplaceBodyReq` and parameter declarations in `internal/operation/wire_engine_mutations.go`.
* Observed failure: a declaration-scoped assertion still found an unexpected matching text block and aborted before writing.
* Workaround: copy the complete current declaration from semantic inspection and replace only the declaration as a whole.
* Root cause: nested source text was copied from a different declaration revision and did not match byte-for-byte.

## ST-0222: Semantic insertion diagnostics found incorrect MCP test transport calls

* Tool: `semedit-next insert`.
* Target: `TestInspectProjectionFeedsGuardedReplaceBody` in `internal/mcp/schema_test.go`.
* Observed failure: the inserted test passed a writer as a third argument to `Server.Serve`; diagnostics show that `Serve` accepts a reader and the output writer belongs in `NewServer`.
* Workaround: bind each test response buffer when constructing its server, then call `Serve` with only the request reader.
* Root cause: the inserted test used an outdated transport call shape.

## ST-0223: Body replacement exposed a now-unused test import

* Tool: `semedit-next replace-body`.
* Target: `TestInspectProjectionFeedsGuardedReplaceBody` in `internal/mcp/schema_test.go`.
* Observed diagnostic: the corrected test no longer uses the previous `io.Discard` constructor argument, leaving `io` unused.
* Workaround: run the registered semantic import organizer on the test file.
* Root cause: replacing the test transport setup made one earlier import unnecessary.

## ST-0224: Inspect consumer test exposed legacy-role and response-shape assumptions

* Tool: `go test ./internal/operation ./internal/mcp -run TestInspectProjectionFeedsGuardedReplaceBody`.
* Target: typed SourceFields metadata and inspection response assertions.
* Observed failure: assigning `RoleTarget` to legacy file parameters made their untyped `semantic_lookup.file` source links fail; the public inspection response also did not provide the structured-content wrapper shape assumed by the test.
* Workaround: inspect the current compatibility validator and actual response envelope; keep legacy primitive edges compatible while declaring the new typed edge.
* Root cause: existing primitive continuations and projected MCP output use different compatibility and response paths than the new typed links.

## ST-0225: Initial required make check rejected unformatted inspect projection changes

* Tool: `make check` with the requested local module cache and `GOPROXY=off`.
* Target: current foundation projection and MCP integration changes.
* Observed failure: the repository Go linter reported complexity, constant, exported-comment, unused-helper, and preallocation findings before reaching focused tests.
* Workaround: apply the registered semantic refactors and formatting, then rerun `make check` before focused tests.
* Root cause: newly added integration code had not yet gone through the repository's required lint-fix cycle.

## ST-0226: Worktree-relative semantic CLI path was unavailable

* Tool: shell invocation of `./bin/semedit-next --help`.
* Target: semantic tooling in the foundation worktree.
* Observed failure: the assigned worktree has no `bin/semedit-next` path.
* Workaround: use the root-provided absolute CLI path `<repository-root>/bin/semedit-next` with the worktree as the command working directory.
* Root cause: the CLI binary is shared from the main checkout rather than copied into this worktree.

## ST-0228: No registered semantic operation removes obsolete top-level declarations

* Tool: repository semantic CLI command inventory.
* Target: now-unused `batchOutputSchema` and `batchResultSchema` helpers in `internal/mcp/server.go`.
* Observed limitation: available insert/replace operations can add or replace declarations but cannot remove top-level declarations. Keeping the obsolete open-schema helpers fails the repository's unused-code lint.
* Workaround: remove only those exact Go AST declarations with a parser-based, atomic temp-file/fsync/rename edit after recording this limitation.
* Root cause: semantic CLI has no declaration deletion operation.

## ST-0229: Focused public inspection tests found a remaining selector compatibility edge

* Tool: focused Go tests for inspect projection and SourceFields validation.
* Target: `semantic_replace_construct.in_function` and its public inspector consumer test.
* Observed failure: the parameter's existing self-output primitive edge is treated as a typed selector edge and registry validation rejects it; the test then panics while reading the failed server response as success.
* Workaround: remove the role annotation from this legacy selector and keep the new inspection selector path typed under the correct consumer, then make the public test report server errors explicitly.
* Root cause: role metadata applies to every SourceFields path on one parameter, including pre-existing primitive continuation paths.

## ST-0232: Insert-function argument retained literal newline escapes

* Tool: `semedit-next insert-func`.
* Target: language-neutral projection coverage in `internal/operation/value_schema_test.go`.
* Observed failure: the shell preserved `\n` as literal backslash characters, which the Go parser rejected.
* Workaround: write the complete declaration into a scratch file and pass its contents as the semantic tool source argument.
* Root cause: multiline code was placed directly in a single-quoted shell argument.

## ST-0233: Test declaration access modifier conflicted with exported casing

* Tool: `semedit-next insert-func`.
* Target: `TestProjectInspectResultKeepsLanguageNeutralAmbiguousMatches` in `internal/operation/value_schema_test.go`.
* Observed failure: the insertion tool rejected explicit private access for the uppercase Go test function name.
* Workaround: declare the test with public access to match its required Go test naming.
* Root cause: test functions use exported casing even though tests are not package API.

## ST-0234: Registered semantic operations cannot add documentation comments to an existing type

* Tool: semantic CLI operation inventory.
* Target: exported `InspectContinuation` and `InspectProjection` types in `internal/operation/inspect_projection.go`.
* Observed limitation: no registered operation changes a type declaration's leading documentation comment; the linter requires both public API comments.
* Workaround: add the two leading comments with a parser-positioned atomic write, then verify the declaration with semantic inspection.
* Root cause: semantic type replacement is not available for struct declarations or their doc comments.

## ST-0235: Follow-up make check still reported static integration findings

* Tool: `make check` with local caches and `GOPROXY=off`.
* Target: inspector schema strings, source-field validation, MCP schema literals, and the public inspection test.
* Observed failure: Go lint still reported one complexity finding, repeated schema-name literals, and one test slice preallocation finding.
* Workaround: extract legacy-source compatibility from the validator, reuse named field/tool constants, and preallocate the test edits slice before rerunning the full check.
* Root cause: the first refactor did not sufficiently lower validator complexity and left lint-only cleanup in the integrated path.

## ST-0238: Semantic operations cannot replace exact schema string values with named constants

* Tool: registered semantic CLI operations.
* Target: repeated schema property/tool-name literals in `internal/operation` and `internal/mcp`.
* Observed limitation: body/declaration operations can replace whole constructs but cannot rewrite selected string-valued expressions to identifiers without replacing entire unrelated implementations.
* Workaround: apply a narrowly scoped Go AST literal rewrite for exact values only, using atomic temp-file/fsync/rename writes.
* Root cause: semantic CLI lacks expression-level replacement for string literals.

## ST-0239: Second semantic outline request used a malformed worktree path

* Tool: `semedit-next outline`.
* Target: the new MCP inspector test's preallocation lint finding.
* Observed failure: command startup failed because the supplied working-directory path was missing its leading slash.
* Workaround: rerun in the assigned absolute worktree path.
* Root cause: the path was mistyped while constructing the command.

## ST-0240: Semantic operations cannot preallocate a local slice declaration

* Tool: registered semantic CLI operation inventory.
* Target: `edits` local in `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes`.
* Observed limitation: the required capacity change is a local declaration initializer; no available semantic operation replaces one local declaration without rewriting the containing test.
* Workaround: update the exact local declaration by AST position, preserving all other test source.
* Root cause: semantic CLI has no local-declaration initializer operation.

## ST-0241: AST schema-literal rewrite also replaced new constant declarations

* Tool: parser-based AST literal rewrite for repeated schema names.
* Target: new constants in `internal/mcp/server.go`.
* Observed failure: matching literals inside the constant declarations themselves were changed into self-references, producing Go initialization cycles.
* Workaround: restore those three declaration values, then constrain any further AST rewrite to skip constant declaration initializers.
* Root cause: the first rewrite matched by exact literal value without excluding the declaration that establishes each constant.

## ST-0242: AST local-declaration edit retained the `var` keyword

* Tool: parser-positioned local slice preallocation edit.
* Target: `edits` in `TestBatchToolAdvertisesAndEmitsRegisteredResultShapes`.
* Observed failure: the generated source contained `var edits :=`, causing Go syntax and type-check errors during `make check`.
* Workaround: use a valid local declaration initializer (`var edits = make(...)`) through an atomic exact-source correction, then rerun checks.
* Root cause: `ast.ValueSpec.Pos` starts at the variable name while the `var` keyword belongs to the enclosing declaration node.

## ST-0243: Full make check test phase failed without retained failure detail

* Tool: full `make check` (`go test -race -shuffle=on -cover -v ./...`).
* Target: repository-wide verification after inspect projection integration.
* Observed failure: formatting, Go lint, doc generation, and Markdown lint passed; the test target exited nonzero, but the very large verbose output was truncated before the failing package/test was visible. Vale reported warnings/suggestions only, and govulncheck was optional/uninstalled.
* Workaround: rerun the repository test target with output redirected to a worktree-local scratch log, then inspect only failing package/test lines before deciding whether any code change is needed.
* Root cause: verbose test output exceeded tool response retention.

## ST-0244: Failure-log command used zsh's read-only status variable

* Tool: shell wrapper around `make test` output capture.
* Target: `.scratch/make-test-failures.log`.
* Observed failure: after the test command returned, zsh rejected assignment to its reserved read-only `status` variable, so the intended filtered summary did not run.
* Workaround: inspect the completed local log directly and use a task-specific variable name in any wrapper.
* Root cause: a zsh special variable name was reused for the exit code.

## ST-0245: Full suite found an outdated inspection output-schema expectation

* Tool: `go test -race -shuffle=on -cover -v ./...` in the required check target.
* Target: `TestMCPToolsAdvertiseStructuredOutputSchemas` in `internal/mcp/schema_test.go`.
* Observed failure: the inspector now correctly advertises its new typed object result, while the generic test branch still expected a legacy string result.
* Workaround: add an explicit inspector object-schema case that checks the closed projection and its required captured identity fields.
* Root cause: schema behavior changed intentionally, but the existing broad assertion table did not gain a case for the inspector.

## ST-0246: Insert-case argument retained literal newline escapes

* Tool: `semedit-next insert-case`.
* Target: explicit `semantic_inspect_symbol` schema test branch.
* Observed failure: the shell passed literal backslash-n sequences and Go parsing rejected the case clause.
* Workaround: provide the clause from a worktree-local scratch file.
* Root cause: multiline syntax was supplied as a single-quoted shell argument.

## ST-0247: Inspector schema test assumed projection fields were top-level

* Tool: focused MCP schema tests after adding the inspector object-schema branch.
* Target: `TestMCPToolsAdvertiseStructuredOutputSchemas` in `internal/mcp/schema_test.go`.
* Observed failure: the standard output schema wraps the typed operation value under `result`; the test looked for `inspection` directly under top-level output properties and panicked on the missing continuation schema.
* Workaround: assert the `inspection` and `continuations` properties under `outputSchema.properties.result`, and guard each type assertion with a test failure.
* Root cause: standard output envelope fields were conflated with the operation's projected value fields.

## ST-0248: Replace-construct could not select a switch case by switch expression

* Tool: `semedit-next replace-construct`.
* Target: the `semantic_inspect_symbol` case in `TestMCPToolsAdvertiseStructuredOutputSchemas`.
* Observed failure: the Go backend searched for a case whose literal value was `name`; it did not treat `name` as the switch selector and reported no matching construct.
* Workaround: replace only the matching `CaseClause` in the Go AST using an atomic parser-positioned edit.
* Root cause: replace-construct case selection currently interprets the discriminator as the case value for this syntax.

## ST-0249: Final make check found the expanded schema test exceeded complexity limits

* Tool: final `make check` Go lint-fix stage.
* Target: `TestMCPToolsAdvertiseStructuredOutputSchemas` in `internal/mcp/schema_test.go`.
* Observed failure: adding explicit nested inspector result-schema assertions raised test cognitive complexity above the repository threshold.
* Workaround: move those assertions into a focused helper and keep the test switch branch as a single helper call.
* Root cause: schema branch assertions were added inline to an already broad catalog test.

## ST-0250: Replace-construct does not locate switch clauses in the test catalog function

* Tool: `semedit-next replace-construct`.
* Target: inspector result-schema case in `TestMCPToolsAdvertiseStructuredOutputSchemas`.
* Observed failure: both the switch expression and explicit case label selectors returned no matching case for this test's catalog switch.
* Workaround: replace only the matching case-clause AST node with a helper call, preserving the surrounding switch and test.
* Root cause: current Go construct selection does not expose a usable selector for this case clause.

## ST-0252: Absolute patch path targeted the coordinator checkout

* Tool: built-in `apply_patch` used for a Go backend edit.
* Target: `internal/backend/golang/backend.go` while working in `.scratch/worktrees/flow-foundation`.
* Observed failure: an absolute path omitted `.scratch/worktrees/flow-foundation` and targeted the coordinator checkout instead of this worker worktree; the coordinator checkout received the same one-line context-forwarding edit.
* Workaround: all intended changes were applied and verified in the assigned worker worktree; root will preserve the coordinator mirror in a targeted stash before integration, and I did not modify or restore coordinator files.
* Root cause: the patch request used the primary checkout path rather than the assigned worktree path. Verify every patch target against the assigned worktree root before editing.

## ST-0253: Replace-body rejected escaped test newline literals

* Tool: `semedit replace-body`.
* Target: `TestInspectProjectionFeedsGuardedReplaceBody` in `internal/mcp/schema_test.go`.
* Observed failure: the CLI parser rejected escaped newline literals in the replacement body; it made no source change.
* Workaround: use a surgical patch to add the self-contained module manifest to this test fixture only.
* Root cause: the body flag's layered shell and CLI quoting did not preserve the Go rune and string literals.

### ST-0169: semantic insert accepted invalid Go source

* Date: 2026-10-01
* Tool: `semantic_insert_declaration` via worktree-local `semedit insert`
* Target: `internal/astedit/insert_field_plan.go`
* Observed failure: insertion reported success and returned introduced compiler diagnostics for `ast.StructType.Rbrace` and a malformed `append` call.
* Workaround: record the tool behavior and attempt `semedit replace-body`; the local CLI could not start because the same invalid source prevented its Go package from building, so the three reported defects were corrected with an atomic source update before resuming semantic edits.
* Root cause: the inserted Go snippet passed parsing but the semantic insertion's post-edit diagnostics exposed type-check errors; the planner assumed brace ownership on `*ast.StructType` and emitted an invalid append expression.

### ST-0170: semantic insertion exposed missing planner proof gates

* Date: 2026-10-01
* Tool: `semantic_insert_declaration` via worktree-local `semedit insert`
* Target: `internal/astedit/insert_statement_plan.go`
* Observed failure: insertion reported success but diagnostics identified undefined `ProofBlockBinding` and `ProofControlFlowSet` referenced by the new statement planner.
* Workaround: add the missing named proof requirements to the shared pure-plan contract before continuing.
* Root cause: the planner function was inserted before its integration proof vocabulary was added to the shared plan file; the tool surfaced the package type-check failure after writing.

### ST-0171: semantic insertion exposed helper return-arity errors

* Date: 2026-10-01
* Tool: `semantic_insert_declaration` via worktree-local `semedit insert`
* Target: `internal/astedit/insert_field_plan.go`
* Observed failure: insertion reported success but diagnostics showed two return statements supplied `(int, string)` to a helper declared as `(int, string, error)`.
* Workaround: record the diagnostics and add explicit nil error values to the successful offset returns.
* Root cause: the extracted helper reused byte-range functions with shorter return tuples without adapting those results to its error-bearing signature.

### ST-0172: semantic insertion surfaced an incomplete helper sequence

* Date: 2026-10-01
* Tool: `semantic_insert_declaration` via worktree-local `semedit insert`
* Target: `internal/astedit/insert_element_plan.go`
* Observed failure: the validator insertion reported success but package diagnostics showed `validateTypedMapKey` undefined because the helper was queued for the next insertion.
* Workaround: record the package diagnostic and add the referenced pure helper atomically so the local CLI can build again.
* Root cause: semantic insertion type-checks the package immediately; helper declarations must be added before callers that reference them.

### ST-0173: semantic body replacement exposed an incomplete planner split

* Date: 2026-10-01
* Tool: `semantic_replace_body` via worktree-local `semedit replace-body`
* Target: `internal/astedit/insert_element_plan.go`
* Observed failure: the planner replacement reported success but diagnostics showed the extracted `appendCompositeElement` helper was not present yet and `bytes` had become temporarily unused.
* Workaround: record the diagnostics and add the pure append helper before retrying further semantic edits.
* Root cause: the orchestration body was switched to the planned helper before that helper was inserted; post-edit verification ran against an incomplete intermediate package state.

### ST-0174: semantic_insert_type rejected escaped source newlines

* Date: 2026-10-01
* Tool: `semantic_insert_type` via worktree-local `semedit insert-type`
* Target: `internal/astedit/plan.go`
* Observed failure: the tool returned `no declarations found in snippet`; no source changed.
* Workaround: retry with an actual multiline type declaration.
* Root cause: the shell argument contained literal backslash-n sequences where the Go parser expected source line breaks.

### ST-0176: combined review-fix patch used stale statement context

* Date: 2026-10-01
* Tool: `apply_patch`
* Target: `internal/astedit/doc_comment_plan.go`, `internal/astedit/insert_statement_plan.go`, and `internal/astedit/insert_element_plan.go`
* Observed failure: the combined patch could not match the expected statement-anchor branch, so none of its hunks were applied.
* Workaround: record the failure and split the changes into smaller patches based on freshly inspected context.
* Root cause: the statement planner has separate selection branches and did not contain the assumed `anchor == nil` guard at that location.

### ST-0177: make check rejected malformed structural planner regression test

* Date: 2026-10-01
* Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
* Target: `internal/astedit/structural_plan_test.go`
* Observed failure: the Go lint/typecheck pass found syntax errors from mismatched test closure delimiters; no focused test ran.
* Workaround: inspect the formatted file around the reported lines and repair the test structure before rerunning the required check.
* Root cause: regression cases were inserted into an existing nested `t.Run` block with incorrect brace placement.

### ST-0178: make check rejected planner test helper arity

* Date: 2026-10-01
* Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
* Target: `internal/astedit/structural_plan_test.go`
* Observed failure: Go typecheck found an assignment arity mismatch when the injection regression captured two results from `parseKeyedElement`, which returns three.
* Workaround: capture all return values in the regression before rerunning repository checks.
* Root cause: the test call did not match the planner helper's `(node, key source, error)` contract.

### ST-0179: make check complexity lint rejected comment planner and test aggregator

* Date: 2026-10-01
* Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
* Target: `internal/astedit/doc_comment_plan.go` and `internal/astedit/structural_plan_test.go`
* Observed failure: gocognit reported complexity 90 for `PlanSetDocComment` and 104 for `TestStructuralSourcePlanners`.
* Workaround: record the failure and extract declaration selection/comment checks and focused tests into named helpers/subtests.
* Root cause: the new ownership guards and regressions pushed already dense functions beyond the repository's configured complexity ceiling.

### ST-0180: repository race/shuffle suite returned failure during make check

* Date: 2026-10-01
* Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
* Target: repository-wide `go test -race -shuffle=on -cover -v ./...`
* Observed failure: the race/shuffle test stage returned non-zero twice; output truncation hid the failing test/package, while formatter, Go lint, Markdown lint, Vale, and Hugo completed.
* Workaround: capture the exact test command output in `.scratch/go/structural-check-tests.log` and inspect failure markers before attributing the cause.
* Root cause: `t.TempDir()` created paths under a temporary directory symlink alias; source-buffer path validation correctly rejected those aliases. Re-running `internal/sourcebuf`, `internal/projectverify`, and `internal/mcp` with canonical `TMPDIR=<temporary-directory>` passed.

### ST-0181: Git staging was blocked by the isolated filesystem root

* Date: 2026-10-01
* Tool: `git add`
* Target: assigned worktree Git index at `<repository-root>/.git/worktrees/flow-structural/index`
* Observed failure: sandbox denied creation of `index.lock`; source edits remain in the assigned writable worktree and no index mutation occurred.
* Workaround: request a narrow sandbox escalation for explicit owned-path staging and the authorized milestone commit.
* Root cause: this linked worktree's Git metadata is stored outside the configured writable root.

### ST-0182: CRLF range patch addressed coordinator checkout

* Date: 2026-10-01
* Tool: `apply_patch`
* Target: `internal/astedit/doc_comment_plan.go`
* Observed failure: patch application could not find the requested absolute path because it omitted this worker's `.scratch/worktrees/flow-structural` path segment; no edit occurred.
* Workaround: record the error and reapply against the assigned worktree path.
* Root cause: copied coordinator repository path was mistaken for the worker worktree path.

### ST-0183: runtime pattern lookup used absent paths

* Date: 2026-10-01
* Tool: `rg`/`sed`/`git show` source inspection
* Target: `internal/operation/wire_contracts.go` and `internal/operation/default.go`
* Observed failure: the queried paths do not exist in the current integration branch; no files were changed.
* Workaround: enumerate registered source files and inspect the contract worker's actual worktree/branch paths before using that design.
* Root cause: filenames were assumed from the handover summary rather than confirmed in the checked tree.

### ST-0185: repository make check failed in benchmark-harness package tests

* Date: 2026-10-01
* Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
* Target: `tools/benchmark-harness` package tests
* Observed failure: formatting, linting, documentation generation, and other parallel test packages completed, but the race/shuffle test stage reported `FAIL semedit/tools/benchmark-harness`; output truncation hid the failing test detail.
* Workaround: preserve this result, then rerun that package with output captured under `.scratch/` to identify the failing test before continuing.
* Root cause: not yet determined; test output from the initial run was truncated by the command transport.

### ST-0186: focused test retry used the default protected Go build cache

* Date: 2026-10-01
* Tool: `go test -race -shuffle=on -cover ./tools/benchmark-harness`
* Target: `tools/benchmark-harness`
* Observed failure: package setup could not open a build-cache artifact under `~/Library/Caches/go-build` because the sandbox denied access. No test ran.
* Workaround: configure `GOCACHE` under this worktree's ignored `.scratch/go/` directory for subsequent Go commands.
* Root cause: the focused retry set `GOMODCACHE` but omitted `GOCACHE`, so Go selected the user's default cache outside writable roots.

### ST-0188: semantic scaffold purpose header omitted Go comment syntax

* Date: 2026-10-01
* Tool: `semedit scaffold-file`
* Target: `internal/operation/wire_structural_plan.go`
* Observed failure: the CLI rejected the purpose header as invalid Go syntax and did not create the file.
* Workaround: retry with a Go comment line as the purpose header.
* Root cause: the command validates the header as source comment text, while the first request supplied plain prose.

### ST-0190: semantic declaration insertion left required imports unresolved

* Date: 2026-10-01
* Tool: `semedit insert`
* Target: `internal/operation/wire_structural_snapshot.go`
* Observed failure: the declaration was inserted, but the file lacks imports for its referenced standard and internal package types; the CLI returned no status text.
* Workaround: use the registered import organizer with explicit paths, then inspect the resulting file before further edits.
* Root cause: declaration insertion did not infer imports for type names in this declaration.

### ST-0191: semantic declaration insertion published unresolved import references

* Date: 2026-10-01
* Tool: `semedit insert`
* Target: `internal/operation/wire_structural_snapshot.go`
* Observed failure: the CLI inserted the loader and returned success while reporting 14 new undefined-name diagnostics because its import cleanup was explicitly disabled for this insertion.
* Workaround: record the incorrect status and add the exact standard-library imports with the registered import operation before continuing.
* Root cause: the request set `auto_organize_imports=false`, so inserted AST references were not accompanied by required imports; the CLI result still allowed the syntactic mutation.

### ST-0192: structural registry insertion preceded its parameter helpers

* Date: 2026-10-01
* Tool: `semedit insert`
* Target: `internal/operation/wire_structural_plan.go`
* Observed failure: after inserting the registry helper, semantic verification reported unresolved references to the intentionally not-yet-inserted `stringParam` and `structuralRevision` helpers, along with imports that were then added by the registered import organizer.
* Workaround: record the staged declaration state and add the shared helpers through semantic insertion before registering operation definitions.
* Root cause: the new registry helper was inserted before its declaration dependencies, and automatic import organization was disabled for that insertion.

### ST-0193: repository check stopped during Go autofixes on goconst findings

* Date: 2026-10-01
* Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
* Target: Go lint autofix stage
* Observed failure: lint fix returned four `goconst` findings: two in foundation-owned `internal/operation/typed_descriptors.go`, one in existing `wire_engine_mutations.go`, and one in this worker's structural example. The check stopped before tests.
* Workaround: replace the worker-owned repeated literal with the existing shared parameter constant, then report the two foundation-owned and one unrelated finding for root coordination before retrying checks.
* Root cause: the integrated lint configuration flags repeated string literals across files, including code owned by another track.

### ST-0196: structural CLI behavior tests exposed simulated-output snapshot comparison

* Date: 2026-10-01
* Tool: focused `go test` for `set_doc_comment` and `insert_statement` txtar contracts
* Target: `internal/operation/wire_structural_snapshot.go`
* Observed failure: both handlers rejected their planned outputs because the typecheck loader compared the simulated selected-file overlay with the buffer's preimage bytes. No source mutation was published.
* Workaround: distinguish the intentional `extra` output overlay from captured read-set preimages: still read the original through `sourcebuf` to bind it, but compare loader bytes only for paths not overridden by the simulated output.
* Root cause: the loader conflated the candidate typecheck snapshot with the source-buffer preimage snapshot.

### ST-0198: focused txtar cases exposed blank CLI value omission and stale diagnostic wording

* Date: 2026-10-01
* Tool: focused `go test` for the new public-operation txtar contracts
* Target: CLI raw-parameter construction and `testdata/scripts/insert_statement.txtar`
* Observed failure: `--comment ''` was omitted before schema validation, so explicit-empty removal was reported as missing; the malformed-statement fixture expected an older diagnostic phrase.
* Workaround: preserve explicitly changed empty string flags in CLI raw parameters and assert the planner's current precise diagnostic.
* Root cause: CLI ingress drops blank string values without checking the Cobra flag's `Changed` state; test wording did not match the new refusal message.

### ST-0199: repository check failed during parallel race/shuffle tests

* Date: 2026-10-01
* Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
* Target: repository-wide `go test -race -shuffle=on -cover -v ./...` stage
* Observed failure: the benchmark-harness package passed, while another package returned non-zero; the large concurrent output was truncated before the failing test and package could be identified. Formatting, lint, docs generation, and build completed.
* Workaround: capture a targeted repeat of the new CLI txtar tests and exact package output in `.scratch/` before deciding whether the failure belongs to this track or is unrelated contention.
* Root cause: not yet identified from the initial output.

### ST-0200: focused public contracts found two txtar expectation errors

* Date: 2026-10-01
* Tool: focused `go test` for `set_doc_comment` and `insert_statement` txtar contracts
* Target: the two new structural operation fixtures
* Observed failure: empty-comment removal now passes; the statement path refusal includes the precise phrase `construct_path requires its matching expected_revision`, while the fixture expected a shorter nonmatching phrase. The directive case selected `Keep` in `doc.go` even though it lives in `directives.go`.
* Workaround: match the precise path-precondition diagnostic and select `directives.go` for the directive case.
* Root cause: fixture assertions had not been checked against the current exact error text and source-file ownership.

### ST-0201: structural typed-proof insertion required explicit go/types import

* Date: 2026-10-01
* Tool: `semedit insert`
* Target: `internal/operation/wire_structural_snapshot.go`
* Observed failure: the field-proof declaration referenced `go/types` while automatic import organization was disabled; semantic diagnostics reported five unresolved `types` names.
* Workaround: record the report and add `go/types` with the registered import organizer.
* Root cause: type-analysis imports were intentionally deferred until the complete helper declaration was inserted.

### ST-0202: repository check rejected the first field proof helper

* Date: 2026-10-01
* Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
* Target: `internal/operation/wire_structural_snapshot.go`
* Observed failure: Go lint reported cognitive complexity 101 for `validatePrivateStructuralType` and an `intrange` suggestion for iterating struct fields; the check stopped before tests.
* Workaround: split owner resolution and literal/source scanning into named helpers, and use the configured integer-range form.
* Root cause: the initial typed-object, method-set, alias, and literal proof were combined in one helper.

### ST-0203: repository contract test requires explicit tools/list assertions for new structural MCP operations

* Date: 2026-10-01
* Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
* Target: `testdata/scripts/mcp_server.txtar`
* Observed failure: `TestREQ002REQ006PublicRegistryAndCLIContractCoverage` reports that `semantic_insert_field`, `semantic_insert_statement`, and `semantic_set_doc_comment` have no `tools/list` txtar assertions.
* Workaround: add explicit public MCP listing assertions for the newly registered tools, then rerun the required check.
* Root cause: registry contract tests require every registered public MCP tool to be represented in an observable CLI fixture.

### ST-0206: focused insert-field txtar stopped on an unterminated assertion

* Date: 2026-10-01
* Tool: focused `go test` for structural CLI txtar contracts
* Target: `testdata/scripts/insert_field.txtar`
* Observed failure: the insert-field duplicate-refusal block returned the expected diagnostic and preserved the file, then the txtar parser rejected a later command with `unterminated quoted argument`.
* Workaround: inspect the exact fixture lines and repair its command quoting before rerunning the focused contract.
* Root cause: the fixture used a shell quote that was not closed in the txtar command language.

### ST-0208: semantic batch used the wrong registry function target

* Date: 2026-10-01
* Tool: `semantic_batch` through the local trusted MCP helper
* Target: `internal/operation/wire_engine.go`
* Observed failure: the four composite helper functions were inserted in the new file, but the final `semantic_replace_body` child could not find `registerStructuralPlanOps` in `wire_engine.go`; the function is in `wire_structural_plan.go`.
* Workaround: inspect the actual inserted bytes and current diff, then target the owning file in a follow-up registry update.
* Root cause: registry helper location was assumed from the call site rather than checked at its declaration.

### ST-0209: semantic batch applied composite wiring but final configured lint verification failed

* Date: 2026-10-01
* Tool: `semantic_batch` through the local trusted MCP helper
* Target: `internal/operation/wire_structural_composite.go` and `internal/operation/wire_structural_plan.go`
* Observed failure: both mutation children returned success, but the final configured `golangci-lint` verification returned non-zero after the registry update and import organization.
* Workaround: record the tool failure and inspect the actual files plus diagnostics before making any follow-up edits.
* Root cause: pending until the exact lint diagnostics are inspected.

### ST-0210: composite key identity used a non-existent go/constant formatter

* Date: 2026-10-01
* Tool: `make check GOMODCACHE=... GOPROXY=off MAKE_JOBS=2`
* Target: `internal/operation/wire_structural_composite.go`
* Observed failure: doc generation could not compile the source because `constant.String` resolves to the `constant.String` kind constant, not a formatting function.
* Workaround: record the failure, then change constant value identity formatting to the `go/constant.Value.String` method through a semantic declaration edit.
* Root cause: the go/constant API formatter was misremembered.

### ST-0211: composite helper batch still failed its final configured lint hook

* Date: 2026-10-01
* Tool: `semantic_batch` through the local trusted MCP helper
* Target: `internal/operation/wire_structural_composite.go`
* Observed failure: the function-body replacement and import-organizer children succeeded, but the final `golangci-lint` verification returned non-zero again.
* Workaround: inspect the current source, then capture the repository linter output to identify the exact remaining finding before retrying semantic edits.
* Root cause: pending direct lint diagnostics.

### ST-0212: expected import absence made a combined inspection command nonzero

* Date: 2026-10-01
* Tool: `rg` within `functions.exec`
* Target: `internal/operation/wire_structural_composite.go`
* Observed failure: the command used `rg -n constant\.` to verify that a removed import had no remaining uses; the expected no-match result exited with status 1 and short-circuited the shell chain.
* Workaround: the preceding source excerpt confirmed the import list; use an explicit `if rg ...; then` check when absence is expected.
* Root cause: `rg` reports no matches with a non-zero exit status.

### ST-0213: full check found worker goconst and iteration issues plus pre-existing cross-track lint findings

* Date: 2026-10-01
* Tool: make check with required local Go cache variables
* Target: internal/operation/wire_structural_composite.go, foundation typed_descriptors.go, and existing wire_engine_mutations.go
* Observed failure: golangci-lint reports the worker repeated discriminator literal and integer loop, plus goconst findings in two foundation-owned occurrences and one unrelated existing mutation example.
* Workaround: fix only this worker shared discriminator literal and integer-range loop, retain the other three findings for root coordination, then rerun checks after coordination.
* Root cause: new worker code duplicated a canonical parameter name and used a legacy loop; independent cross-track source already has repeated literals.

### ST-0214: shell quoting interrupted an attempted failure-log append

* Date: 2026-10-01
* Tool: functions.exec with exec_command
* Target: docs/SUBOPTIMAL_TOOLS.md
* Observed failure: unescaped Markdown backticks in the single-quoted shell argument triggered command substitution and truncated the embedded Python source. No log entry was written by that attempt.
* Workaround: record the failure without shell-significant backticks, then continue with the bounded semantic edit.
* Root cause: shell quoting treated Markdown delimiters as executable substitutions.

### ST-0217: semantic replace-construct found the loop had already been modernized

* Date: 2026-10-01
* Tool: semantic_batch through the local trusted MCP helper
* Target: internal/operation/wire_structural_composite.go
* Observed failure: the selected loop text was not found; the preceding make check go fix had already converted it to a range loop. The later batch child did not run.
* Workaround: record the actual state and retry only the remaining discriminator-literal cleanup using a focused semantic function-body operation.
* Root cause: a mechanical fixer changed the source between selection discovery and the later operation.

### ST-0218: composite discriminator cleanup passed its child but final lint still refused

* Date: 2026-10-01
* Tool: semantic_batch through the local trusted MCP helper
* Target: internal/operation/wire_structural_composite.go
* Observed failure: semantic_replace_body changed the repeated discriminator literals to the shared parameter constant, then final configured lint verification returned non-zero because remaining diagnostics are outside this worker scope.
* Workaround: record the result and retain the applied source change; coordinate the known unrelated lint findings before requesting another successful MCP verification.
* Root cause: full-worktree golangci-lint includes unresolved cross-track goconst findings.

### ST-0221: structural parameter copy edges referenced producers without output metadata

* Date: 2026-10-01
* Tool: make check with required local Go cache variables
* Target: internal/operation/wire_structural_plan.go and wire_structural_composite.go
* Observed failure: MCP and registry tests reject the new parameter SourceFields edges because providers such as semantic_lookup.file have no typed output metadata yet. The foundation projection/finalization hook integration is pending.
* Workaround: record the failure and remove unsupported copy edges from this checkpoint; restore only edges backed by the checked output projection API after root rebase.
* Root cause: operation inputs advertised continuations before the providers exposed a typed live result contract.

### ST-0224: semantic verify reported diagnostics outside its selected file

* Date: 2026-10-02
* Tool: `semantic_verify` through the local CLI
* Target: `internal/operation/wire_structural_plan.go`
* Observed failure: a check-only verification scoped to this file returned 16 type diagnostics in unrelated backend and AST-edit files, while its configured linter passed and the full race/shuffle Go test suite passed. No files changed.
* Workaround: use the successful repository check and focused Go tests as the verification evidence for this checkpoint.
* Root cause: not yet determined; the verifier reported module diagnostics outside the selected file despite a successful full test build.

## ST-0257: Structural insertion used the default external Go cache

* Tool: worktree-local MCP `semantic_batch`, child `semantic_insert_statement`.
* Target: `internal/mcp/server_test.go`, owner `TestMCPSnapshotAndUndo`.
* Observed failure: structural package loading tried the default `~/Library/Caches/go-build` and was denied by the sandbox. The earlier helper insertion and schema-fixture body change applied; the statement child made no change.
* Workaround: retain the applied earlier edits and retry only the statement with the configured writable Go cache. Send the loader environment finding to the structural owner before merging the public tool.
* Root cause: `loadStructuralPackages` omitted the managed Go environment used by other compiler-snapshot operations.

## ST-0258: Statement insertion rejects an owner containing unrelated control flow

* Tool: worktree-local MCP `semantic_insert_statement`.
* Target: `internal/mcp/server_test.go`, owner `TestMCPSnapshotAndUndo`, direct `calcFile := ...` anchor.
* Observed failure: the operation refused the entire owner because it contains later conditional statements; no source change occurred in this retry.
* Workaround: use semantic body replacement for the fixture's additional module initialization, retaining its existing assertions. Clarify the advertised straight-line-owner restriction and review whether unrelated control flow needs exclusion.
* Root cause: the initial statement planner rejects control-flow nodes anywhere in the owning function, beyond the selected direct statement.

## Semantic declaration insertion omitted exported API comments

* **Tool:** `semantic_insert_declaration` / `semantic_insert_function`
* **Target:** `internal/astedit/inline_function_plan.go`, `internal/astedit/type_migration_plan.go`
* **Observed behavior:** The tools inserted the requested declarations and organized imports, but the initial declaration sources omitted Go doc comments. The repository linter then rejected the exported planner names.
* **Workaround:** The semantic tool set has no operation for adding only leading declaration comments, so the required doc comments were added manually after logging this occurrence; then the project checks were rerun.
* **Root cause:** The insertion operations validate declaration structure and imports but do not require Go documentation comments on exported declarations.

## Relocation worker: outline rejected `--file`

* Tool: `semedit outline`.
* Target: read-only outlines of `internal/astedit/inline_function_plan.go`, `internal/astedit/type_migration_plan.go`, and `internal/operation/wire_engine_inline.go`.
* Observed failure: the command rejected `--file` as an unknown flag and made no source change.
* Workaround: use the documented `--path` selector for the same files.
* Root cause: `outline` and symbol-specific navigation use different selector flag names.

## Relocation worker: absolute patch path targeted the coordinator checkout

* Tool: built-in `apply_patch` used for a schema constant edit.
* Target: `internal/operation/typed_descriptors.go` while working in `.scratch/worktrees/flow-relocation`.
* Observed failure: the absolute patch target omitted `.scratch/worktrees/flow-relocation` and applied one schema-key edit in the coordinator checkout instead of the assigned worker worktree.
* Workaround: stopped editing the coordinator checkout and notified root to preserve or revert its delta; the intended change remains pending in the assigned worker worktree.
* Root cause: the patch target was not checked against the assigned worktree root. Verify the full worktree prefix on every patch target.

## ST-0260: Field insertion refused a one-line struct layout

* Date: 2026-10-02
* Tool: worktree-local MCP `semantic_insert_field` through `semantic_batch` and a direct inspect-to-insert MCP test.
* Target: an existing private struct produced as `type options struct{}` or with a one-line field list.
* Observed failure: the operation refused with `struct type "options" must use a multiline field layout`; no file changed.
* Workaround: provide a multiline field list and assert that bounded form in the current CLI and MCP fixtures.
* Root cause: the initial field insertion planner supports safe multiline struct ranges only. The empty-struct case is a common typed insertion flow and remains a follow-up scope.

## 2026-10-02: CLI help ran without the assigned worktree cache environment

* Tool: `go run . insert --help`
* Target: relocation worktree CLI
* Observed failure: Go attempted to open a build-cache path under the coordinator user's Library cache and the sandbox denied it; no source was changed.
* Workaround: rerun from the exact relocation worktree with its per-worktree `GOCACHE`, `GOPLSCACHE`, and `GOTMPDIR` settings.
* Root cause: the command omitted the required repository cache environment and was run from the coordinator checkout.

## CLI stdin worker: semantic body replacement doubled newline escapes

* Date: 2026-10-02
* Tool: worktree-local `replace-body` semantic CLI operation.
* Target: `internal/cli/cli.go`, function `buildCommand`.
* Observed failure: the operation succeeded structurally but emitted `\\n` in three Go format strings where the existing body used `\n`, changing error output to print literal backslash-n characters.
* Workaround: immediately corrected those three escape sequences with a targeted patch.
* Root cause: the shell/JavaScript quoting for the semantic CLI request over-escaped Go string literals.

## CLI stdin worker: scaffold purpose header rejected prose

* Date: 2026-10-02
* Tool: worktree-local `scaffold-file` semantic CLI operation.
* Target: `internal/cli/stdin_test.go`.
* Observed failure: passing `Package cli tests ...` as `--purpose-header` failed Go parsing with `expected 'package', found Package`; no file was created.
* Workaround: retry with a syntactically valid Go comment header.
* Root cause: the tool validates the purpose header as source text and does not add comment syntax despite the flag description calling it a comment header.

## CLI stdin worker: insert-function snippet quoting failed

* Date: 2026-10-02
* Tool: worktree-local `insert-func` semantic CLI operation.
* Target: `internal/cli/stdin_test.go`, a new test declaration.
* Observed failure: validation rejected the submitted declaration with a Go syntax error around a quoted string; no file change occurred.
* Workaround: simplify the test payload literal and submit valid Go source.
* Root cause: nested shell and JavaScript escaping made the Go string literal invalid before the semantic operation ran.

## CLI stdin worker: insert-field typecheck could not import Cobra

* Date: 2026-10-02
* Tool: worktree-local `insert-field` semantic CLI operation.
* Target: `internal/cli/stdin_test.go`, field on `stdinProbeState`.
* Observed failure: package typechecking stopped with `could not import github.com/spf13/cobra (invalid package name: "")`; no field change occurred.
* Workaround: use a targeted patch for this test-only field and record the package-import tool limitation.
* Root cause: the operation's package typecheck could not resolve the already-declared CLI dependency in this invocation.

## CLI stdin worker: insert-function reported a false Cobra dependency diagnostic

* Date: 2026-10-02
* Tool: worktree-local `insert-func` semantic CLI operation using `--source -`.
* Target: `internal/cli/cli_test.go`, registry payload inventory test.
* Observed failure: the function was inserted, but the tool reported `no required module provides package "github.com/spf13/cobra"` although Cobra is present in the module; its diagnostic claimed `diagnostic_status: changed` with a stale dependency-resolution error.
* Workaround: retain the inserted function and verify it with `make check` and the focused test suite.
* Root cause: the semantic operation's diagnostic environment failed to resolve the current module dependencies; the operation returned a syntactically valid insertion.

## Track B local semantic replace-body: stale organize-imports import

* Date: 2026-10-02
* Tool: worktree-local `replace-body` semantic CLI operation.
* Target: `internal/operation/wire_engine_imports.go`, `runOrganizeImports`.
* Observed failure: the body replacement left the now-unused `os` import, and diagnostics reported it as an error.
* Workaround: run semantic import organization for the selected file to remove the stale import.
* Root cause: the previous handler body used `os.Stat`; the new changed-file report no longer does.

## Track B local semantic CLI: incorrect checkout working directory

* Date: 2026-10-02
* Tool: worktree-local `replace-body` semantic CLI operation.
* Target: coordinator checkout `internal/operation/wire_remove.go`, `removeDef`.
* Observed failure: the command used the coordinator checkout as its working directory, inserted an unregistered error projection, and diagnostics reported missing type/field declarations. The inserted block was immediately removed; the coordinator file diff is empty.
* Workaround: verify the absolute assigned worktree path in every semantic CLI invocation before executing it.
* Root cause: a copied command used the coordinator path instead of the Track B worktree path.

## Registry workspace choices: test inserted before import organization

* Date: 2026-10-03
* Tool: worktree-local `insert-func` semantic CLI operation.
* Target: `internal/mcp/schema_test.go`, `TestToolsListUsesRegisteredBackendsForOperationInventory`.
* Observed failure: insertion diagnostics reported `undefined: javabackend` because the test import had not been added yet.
* Workaround: immediately run worktree-local semantic import organization with the Java backend package alias; diagnostics then resolved.
* Root cause: the inserted test referenced a new package alias before adding its import.

## 2026-10-03: targeted coordinator instead of private worktree

* Tool: `apply_patch`.
* Target: coordinator `internal/operation/verify_hooks.go` rather than the private worktree copy.
* Observed failure: two patches used the coordinator absolute path and changed its function signature/imports; the parent restored the file and saved the diff for reconciliation.
* Workaround: repeat the needed edit only in the verified private worktree and verify `git rev-parse --show-toplevel` before further changes.
* Root cause: I copied the coordinator root into the patch target without the `/.scratch/worktrees/verify-enabled-languages` suffix.

## 2026-10-03: insert-type received literal newline escapes

* Tool: local `semedit insert-type` CLI.
* Target: `internal/mcp/enabled_languages_test.go`.
* Observed failure: the CLI parser received backslash-n characters in place of newlines and rejected the snippet; the file was unchanged.
* Workaround: send the source through stdin using a quoted heredoc.
* Root cause: I quoted the `--source` payload for the shell without converting escaped newlines to actual line breaks.

## 2026-10-03: inserted test type before imports

* Tool: local `semedit insert-type` CLI.
* Target: `internal/mcp/enabled_languages_test.go`.
* Observed failure: the declaration was inserted with a temporary `undefined: golang` diagnostic because the new file had not yet imported the backend package.
* Workaround: add the required imports before semantic verification.
* Root cause: type insertion verifies immediately, before subsequent import edits.

## 2026-10-03: insert-func import alias mismatch

* Tool: local `semedit insert-func` CLI.
* Target: `internal/mcp/enabled_languages_test.go`.
* Observed failure: automatic import organization aliased `internal/backend/golang` as `gobackend`, while the earlier inserted type still referenced `golang`; diagnostics reported `undefined: golang`.
* Workaround: align the type's package qualifier with the generated import alias.
* Root cause: the generated alias is selected from package naming rules and differs from the directory name.

## 2026-10-03: insert-func placement needed exported section

* Tool: local `semedit insert-func` CLI.
* Target: `internal/mcp/enabled_languages_test.go`.
* Observed failure: insertion with `--placement private_end` was rejected because the `Test*` declaration is exported; no mutation occurred.
* Workaround: use `--access public --placement public_end`.
* Root cause: the test function's exported name requires public section placement.

## 2026-10-03: thread inventory limit rejected

* Tool: `list_threads`.
* Target: outstanding-work inventory for the main integration audit.
* Observed failure: `limit: 60` was rejected because the server maximum is 50.
* Workaround: retry with `limit: 50`.
* Root cause: the exposed tool schema does not specify its server-enforced maximum.

## 2026-10-03: safe-delete peer review timed out

* Tool: `codex_ask`, read-only Terra review.
* Target: safe-delete expansion in `internal/astedit/remove.go` and `safe_delete.go`.
* Observed failure: the peer-review request returned `context deadline exceeded`, without a review report.
* Workaround: continue concrete source review and observable probes; request an independent native review when a worker slot is available.
* Root cause: the external review transport timed out; no source mutation resulted from this call.

## 2026-10-03: safe-delete import cleanup changed initialization

* Tool: private branch CLI `remove-construct --safe-delete`.
* Target: scratch probe `main.go`, deleting an unused function that held the only reference to an imported package.
* Observed failure: deletion succeeded and import organization removed the normal import, also removing that package's observable `init` effect.
* Workaround: reject safe deletion when it would remove the final use of a normal import; add CLI coverage for standalone and buffered requests.
* Root cause: the safe-delete preflight proved declaration references and variable effects but omitted import initialization effects introduced by cleanup.

## 2026-10-03: outline request used the wrong scope field

* Tool: `semantic_outline`.
* Target: relocation worktree `internal/astedit/safe_delete.go`.
* Observed failure: `file` was rejected; the tool requires `path`.
* Workaround: copy the supported parameter name from the exposed schema and retry.
* Root cause: caller failed to consult the tool schema before dispatch; no source changed.

## 2026-10-03: safe-delete misclassified a generated test runner

* Tool: private CLI `remove-construct --safe-delete` in `safe_delete_multifile.txtar`.
* Target: unused private function in a package with same-package tests.
* Observed failure: the compiler-generated test runner's `reflect` import caused an unsupported dynamic-reference refusal. Checking only containment still failed because Go cache files can live under the module's ignored `.scratch` directory.
* Workaround: apply module source-directory boundaries before classifying package imports; retain real module test variants and their typed reference proofs.
* Root cause: loaded package module metadata and filesystem containment do not prove source membership.

## 2026-10-03: concurrent full check lost a diagnostics session

* Tool: CLI `rename` in `rename_diagnostic_delta.txtar` during the flow integration checks.
* Target: fixture `api/server.go`, `Server.OldName`.
* Observed failure: gopls diagnostics shutdown returned `JSON-RPC client is closed` during concurrently running race suites.
* Workaround: rerun the unchanged public fixture with bounded test concurrency and retain the combined integration gate.
* Root cause: transport failure observed; process/resource causation is not established by this log.

## ST-0266: extraction preflight batch failed final semantic verification

* Date: 2026-10-03
* Tool: trusted `semantic_batch` with constant-loader/body replacements, helper insertion, and import organization.
* Target: `internal/astedit/extract_constant.go`.
* Observed failure: mutation children reported success, but final verification failed; the verifier diagnostic was truncated.
* Workaround: inspect current source, preserve successful children, complete the missing read-set initialization and capture loops, then retry coherently.
* Root cause: the replacement omitted local read-set initialization and final capture loops, leaving undefined names and an incomplete return.

## ST-0268: documentation patches targeted the coordinator path

* Date: 2026-10-03
* Tool: `apply_patch`.
* Target: extraction ADR and index documentation during merge integration.
* Observed failure: patches addressed the coordinator checkout path instead of this private worktree; the coordinator restored those exact changes before structural merge.
* Workaround: reapply only the extraction-owned hunks under `.scratch/worktrees/flow-extraction` and verify both worktree roots before editing.
* Root cause: the absolute path omitted the private worktree suffix.

## ST-0269: replace-body left an obsolete import after replacing InlineVariable

* Date: 2026-10-03
* Tool: CLI `replace-body` followed by attempted `imports` help invocation.
* Target: `internal/astedit/inline.go`.
* Observed failure: replacing the function body removed its use of `internal/pipeline`; the subsequent CLI import operation could not start because the source no longer compiled.
* Workaround: remove the obsolete import with a targeted patch, then rerun semantic verification.
* Root cause: `replace-body` changes one function body and does not clean file imports.

## 2026-10-03: smoke trial invented a lookup parameter

* Tool: `semantic_lookup`, Luna 6 medium sink-migration smoke trial.
* Target: fixture `service/dispatch.go`, `Dispatcher.Record`.
* Observed failure: `{"query":"Dispatcher Record"}` was rejected because lookup accepts `symbol`, not `query`.
* Workaround: the agent retried lookup with supported parameters and completed the task; its oracle passed.
* Root cause: caller invented a parameter instead of copying the advertised schema. Evidence: the task-11 Codex semedit transcript for `flow-tools-smoke-20261003T113815Z`.

## 2026-10-03: smoke harness failed to remove read-only Go caches

* Tool: benchmark harness session cleanup, AGY Gemini 3.7 medium.
* Target: six disposable large-context fixture workspaces under `.scratch/benchmarks/`.
* Observed failure: all six task oracles passed, then `os.RemoveAll` failed inside fixture-local `.scratch/go/mod/pgregory.net/rapid@v1.3.0`; the harness correctly retained an execution error instead of declaring overall success.
* Workaround: preserve the original reports and diagnose cleanup independently of the task oracle. A cache cleanup correction and fresh AGY trials are required before claiming clean harness runs.
* Root cause: Go creates read-only module-cache directories; ordinary recursive removal cannot unlink their children. Evidence: reports and transcripts for `flow-tools-smoke-20261003T113815Z`.

## 2026-10-01: Verify diagnostics contradict repository checks

* Tool: `semantic_verify` with `check_only: true` and workspace trust.
* Targets: `internal/astedit/inspect.go`, `internal/backend/backend_test.go`, and `internal/backend/haskell/haskell_probe_test.go`.
* Observed: 16 unknown-field diagnostics after the full `make check` passed, including compilation, race tests, and the successful verification hook. These files were outside the benchmark feedback change.
* Workaround: rely on the passing repository compilation, race tests, and lint hook for this benchmark-only change; compare with a fresh semedit session using explicit Go selection. Auto language selection also refuses the unconfigured Make backend in the fresh CLI.
* Root cause: a fresh, newly built semedit process reproduced the same 16 Go diagnostics, ruling out retained-session staleness as the sole cause. The repository compiles under Go 1.27.1; gopls rejects promoted embedded fields in composite literals in these unrelated existing files. A Go/gopls compatibility mismatch remains the likely cause; it is not established as a source failure.

## 2026-10-03: Independent construct review timed out

* Tool: `codex_ask` with a read-only Terra review request.
* Targets: the integration worktree's `internal/astedit/construct_span.go` and `internal/astedit/construct_declaration.go`.
* Observed: the request exceeded its deadline without returning review findings.
* Workaround: continue coordinator review and require worker fixes plus observable contract tests; do not treat the missing response as approval.
* Root cause: the review service did not complete within its request deadline; no narrower cause is established.

## 2026-10-03: semantic move could not resolve operation package dependencies

* Tool: `semantic_move_construct` via the local `semedit move-construct` CLI.
* Target: move `loadContractSnapshot` from `internal/operation/wire_contracts.go` to `internal/operation/wire_contract_snapshot.go`.
* Observed failure: package name resolution for `golang.org/x/tools/go/packages` exited with status 1. After linking the cached tools module into the worktree-local cache, the CLI itself could not build because its local module cache lacked Cobra, TOML, YAML, `x/sync`, and `x/mod`; network resolution failed with DNS errors.
* Workaround: use a targeted structural patch to split the operation files after verifying the affected tests.
* Root cause: CLI semantic operations use the selected worktree's `.scratch/go/mod`, while the configured shared module cache contains the required dependencies. The worktree-local cache was incomplete and network access was unavailable.

## 2026-10-03: replace-body could not resolve cached YAML package name

* Tool: local `semedit replace-body` CLI.
* Target: `tools/benchmark-harness/oracle.go`, `collectDeclaredSymbols`.
* Observed failure: import cleanup could not resolve the package name for `gopkg.in/yaml.v3` and refused the mutation.
* Workaround: use a targeted structural patch after recording the failure.
* Root cause: the operation's package-name lookup did not find this dependency in the worktree's selected module-cache environment.

## 2026-10-03: review inspection selected the wrong declaration file

* Tool: `semantic_inspect_symbol`.
* Target: `internal/operation/wire_contracts.go`, `adaptContractResult` during the contract feature-package review.
* Observed failure: symbol resolution returned a plain error because the requested helper is declared in `wire_contract_snapshot_adapter.go`.
* Workaround: inspect the declaration outline and use its actual file in the assigned review worktree.
* Root cause: the review request used an outdated declaration location after the adapter was split from the contract wire file. Read-only navigation must also select the current worktree rather than relying on the installed server's primary checkout.

## 2026-10-06: Declaration replacement cannot replace a struct type

* Tool: `semantic_replace_decl`.
* Target: `internal/capability/capability.go`, `OperationContract`.
* Observed failure: the operation rejected replacement of the struct because it supports constants, variables, and type aliases.
* Workaround: use the development CLI's `replace-construct --kind type` operation for struct replacement.
* Root cause: the caller selected the narrower declaration tool instead of the supported generic type replacement operation.

## 2026-10-06: Body replacement cannot change a return signature

* Tool: `semantic_replace_body` in a model correction batch.
* Target: `internal/capability/contract_model.go`, `BrokenCodeSupport.Cases`.
* Observed failure: changing the body to return a result and an error retained the one-result signature, producing return-count diagnostics. The batch then failed its final verification because project lint hooks require workspace trust.
* Workaround: replace the complete method using CLI `replace-construct --kind method`; use individual semantic operations for the remaining edits.
* Root cause: body replacement preserves the declaration signature, and the exposed batch schema cannot provide the trust required by this project's verification hooks.

## 2026-10-06: Documentation comments cannot target grouped constants

* Tool: CLI `set-doc-comment`.
* Target: `internal/capability/contract_model.go`, `ScopeNone`.
* Observed failure: grouped constants were rejected because comment ownership is unsupported.
* Workaround: use supported semantic comments for standalone declarations and a focused, atomically published patch for constant-block API comments.
* Root cause: the documentation operation deliberately excludes grouped declaration ownership.

## 2026-10-06: Documentation comments cannot route qualified methods

* Tool: CLI `set-doc-comment`.
* Target: exported receiver methods in `internal/capability/contract_model.go`.
* Observed limitation: qualified receiver-method selectors are unsupported by the documentation operation.
* Workaround: publish focused API-comment additions through the atomic writer, preserving method implementations.
* Root cause: the documentation operation's selector support is narrower than symbol inspection and body replacement.

## 2026-10-06: Atomic publisher command treated a destination as source

* Tool: atomic publication helper invoked through `go run`.
* Target: `internal/capability/contract_model.go`.
* Observed failure: the `.go` destination argument was interpreted as another Go source file, so publication did not occur.
* Workaround: build the publisher under `.scratch` and pass the destination to the compiled helper.
* Root cause: Go source-file argument parsing precedes program-argument parsing for this invocation.

## 2026-10-04: OpenCode hard-task rerun masks mutation and undo errors

* Tools: `semantic_rename`, `semantic_replace_construct`, and `semantic_undo` through OpenCode's MCP client.
* Target: task 09's `config/config.go` in run `opencode-qwen36-iq4-xs-hard-20261004-1635`.
* Observed failure: rename requests using `symbol=Port` or `Config.ListenPort`, statement/expression replacements inside `Config.Addr`, and repeated undo calls with `snapshot_id=latest` receive `MCP error -32602` because structured output lacks the schema-required `result` property. Type replacement and a later statement replacement with the complete return statement as discriminator succeeded. The final transcript records 239 calls, 234 failures, and 227 rejected undo requests before timeout.
* Workaround: retain the unmodified benchmark conditions and let the configured job deadline produce a terminal outcome; preserve client tool records and task-only metrics as evidence.
* Root cause: error responses' structured content does not satisfy the advertised output schema. Client validation masks the underlying operation failures, preventing actionable recovery; the invalid snapshot selector may trigger the undo failure.

## 2026-10-04: OpenCode mixed-sink rerun masks replacement and batch errors

* Tools: `semantic_replace_construct` and `semantic_batch` through OpenCode's MCP client.
* Target: task 11's `service/dispatch.go`, `Dispatcher.Record`, in run `opencode-qwen36-iq4-xs-hard-20261004-1635`.
* Observed failure: function replacement calls that omit a target selector, and a batch containing a `semantic_replace_body` request, receive schema validation errors because structured content lacks `result`.
* Workaround: record the raw tool calls and preserve the benchmark conditions while the model attempts recovery; do not manually repair the fixture or add recovery steering.
* Root cause: the failure response does not satisfy the advertised MCP output schema, masking the underlying argument or operation diagnostics.

## 2026-10-04: MCP test scaffolding failure

* Tool: `semantic_scaffold_file`
* Target: `internal/mcp/failure_contract_test.go`
* Observed failure: the scaffold request rejected the unsupported `trust_workspace` parameter before creating the file.
* Workaround: retry with only the accepted scaffold parameters; scaffolding does not require workspace trust.
* Root cause: the operation schema deliberately exposes no trust parameter for new-file creation.

## 2026-10-04: MCP construct replacement selector mismatch

* Tool: `semantic_replace_construct`
* Target: `internal/mcp/failure_contract_test.go`
* Observed failure: the requested if-condition selector was not matched; the diagnostic listed the available construct paths and represented the condition with its short initializer separately.
* Workaround: use the reported construct path `24.0.7` to replace the intended conditional or use a full-body replacement for the test function.
* Root cause: the selector parser normalizes an if statement's short initializer and condition rather than matching the combined expression string.

## 2026-10-04: MCP error fixture callback replacement is not selectable

* Tool: local `semantic_replace_construct` CLI.
* Target: `registerKnownOperationErrorProbe` in `internal/mcp/error_presentation_test.go`.
* Observed failure: a complete return-statement discriminator did not select a return inside the `ProjectOutput` function literal; candidates expose the function literal as an expression rather than its body statements.
* Workaround: replace the enclosing function-literal expression semantically, keeping the fixture's error behavior while changing only its success projector.
* Root cause: construct selection does not descend into this nested callback's body.

## 2026-10-04: MCP batch diagnostic normalization insertion unsupported

* Tool: local `semantic_insert_construct` and `semantic_replace_construct` CLI.
* Target: `handleBatch` in `internal/mcp/batch.go`.
* Observed failure: `kind=statement` was rejected as unsupported for both insertion and replacement before mutation.
* Workaround: replace the response-handling implementation semantically while adding normalization at its existing preparation boundary.
* Root cause: the available local binary predates arbitrary statement editing.

## 2026-10-04: MCP output schema signature change requires declaration replacement

* Tool: local `semantic_replace_body` CLI.
* Target: `standardOutputSchema` in `internal/mcp/server.go`.
* Observed failure: replacing the body to reference an added error-schema parameter left the existing one-parameter declaration unchanged and introduced undefined-identifier diagnostics.
* Workaround: replace the full declaration with a signature-aware structural edit before continuing the schema change.
* Root cause: body replacement preserves the function signature by design.

## 2026-10-04: Method signature update rejected by semantic construct replacement

* Tool: local `semantic_replace_construct` CLI.
* Target: `(*Server).sendToolSuccessWithNative` in `internal/mcp/server.go`.
* Observed failure: a replacement with the same method name but an added `isError` parameter was rejected because the tool requires the selected function symbol to be preserved.
* Workaround: apply a surgical declaration-signature edit after the failed structural operation.
* Root cause: the current CLI constrains function replacements more strictly than Go method signature evolution requires.

## 2026-10-04: Safe declaration removal refused by unrelated module directives

* Tool: local `semantic_remove_construct` CLI.
* Target: `safeDeleteRefusalContent` in `internal/mcp/server.go`.
* Observed failure: safe deletion was refused because the module contains generation or link-name directives, despite the target having no references.
* Workaround: remove the already-unreferenced helper with a surgical edit.
* Root cause: the safe-delete module preflight is broader than the target declaration's dependency scope.

## 2026-10-04: Function insertion command unavailable in local semantic CLI

* Tool: local `semedit-next insert-function` CLI command.
* Target: `internal/mcp/schema_test.go`.
* Observed failure: the command is not implemented by this local CLI build.
* Workaround: add the small test-only helper with a surgical edit.
* Root cause: this binary exposes declaration replacement but not standalone function insertion.

## 2026-10-04: Semantic body replacement received a full function declaration

* Tool: local `semantic_replace_body` CLI.
* Target: `TestMCPMavenTestFailureIncludesStructuredResult` in `internal/mcp/server_test.go`.
* Observed failure: the operation rejected a body string that included the function signature and opening brace.
* Workaround: retry with only the statements inside the existing function body.
* Root cause: body replacement accepts statements only and preserves the declaration signature.

## 2026-10-04: Semantic test-body replacement had a malformed nested call

* Tool: local `semantic_replace_body` CLI.
* Target: `TestMCPMavenTestFailureIncludesStructuredResult` in `internal/mcp/server_test.go`.
* Observed failure: the first body-only retry had one missing closing parenthesis in the nested `Serve` call and failed parsing.
* Workaround: correct the nested call and retry before accepting any edit.
* Root cause: the generated test body had a malformed expression.

## 2026-10-04: Semantic catalog helper contained unescaped JSON source

* Tool: local `semantic_insert_function` CLI.
* Target: `advertisedFailureCatalog` in `internal/mcp/failure_contract_test.go`.
* Observed failure: the helper source used a Go JSON string whose escaping was consumed by the tool argument and failed parsing.
* Workaround: construct the request from a Go value with `json.Marshal`.
* Root cause: source-string escaping did not survive the nested tool argument boundary.

## 2026-10-04: Semantic source edits blocked by transient workspace diagnostics

* Tool: local `semantic_replace_body` CLI.
* Target: `validateToolResult` in `internal/mcptest/validate.go`.
* Observed failure: the operation refused to prune imports because a concurrent test edit temporarily imported an undeclared JSON Schema package.
* Workaround: wait for the import to be corrected, then use a minimal supported source edit.
* Root cause: semantic edits analyze workspace-wide package diagnostics.

## 2026-10-04: Semantic statement insertion kind unsupported

* Tool: local `semantic_insert_construct` CLI.
* Target: `validateToolResult` in `internal/mcptest/validate.go`.
* Observed failure: `--kind statement` was rejected as unsupported although the command help describes anchored statements.
* Workaround: insert the single validation call with a narrow patch after the semantic operation failed.
* Root cause: command help advertises a construct shape that the Go backend does not accept under the `statement` kind.

## 2026-10-04: Struct field insertion refused server owner

* Tool: local `semantic_insert_field` CLI.
* Target: `Server` in `internal/mcp/server.go`.
* Observed failure: the operation refused to add an internal cache pointer because it could not prove positional-literal compatibility for the exported owner type.
* Workaround: use a narrowly scoped struct field patch; the repository has no positional `Server` literals outside controlled constructors.
* Root cause: the field insertion preflight requires positional literal compatibility even for an exported struct whose literals are repository-controlled.

## 2026-10-04: Import organizer selected an undeclared JSON Schema module

* Tool: local `semantic_organize_imports` CLI.
* Target: `internal/mcp/presentation_test.go`.
* Observed failure: import organization selected `github.com/kaptinlin/jsonschema`, although the workspace uses `github.com/santhosh-tekuri/jsonschema/v6`.
* Workaround: replace the import with the declared module.
* Root cause: name-based import selection did not respect the existing module dependency.

## 2026-10-04: Symbol inspector could not locate a helper by the supplied selector

* Tool: local `semantic_inspect_symbol` CLI.
* Target: `validateDefinition` in `internal/operation/operation.go`.
* Observed failure: the inspection response had no structured `result` match for that helper name, causing the client-side lookup to fail.
* Workaround: inspect the relevant registry source with a narrow source lookup.
* Root cause: the actual validation helper uses a different symbol name than expected.

## 2026-10-04: Test insertion referenced helpers that were not yet declared

* Tool: local `semantic_insert_function` CLI.
* Target: `TestDeclaredOutputFieldsAreRequiredOnlyForNativeSuccess` in `internal/mcp/output_schema_required_test.go`.
* Observed failure: semantic diagnostics reported unresolved test assertion helpers after inserting the test body.
* Workaround: declare the helpers before running package diagnostics or focused tests.
* Root cause: the test declaration and its small assertion helpers were inserted in separate operations.

## 2026-10-04: Rename identity check lost a diagnostics shutdown connection

* Tool: real Go rename backend in `TestRenameContinuationMatchesFreshInspection` during `make check`.
* Target: fixture `server.go`, `Server.Start` to `Server.Serve`.
* Observed failure: diagnostics shutdown returned `JSON-RPC client is closed`. The MCP suite passed; this operation test was the only full-suite failure. The unchanged test passed immediately with race detection on an isolated rerun.
* Workaround: retain the failure log and rerun the complete configured checks with bounded package and test concurrency.
* Root cause: a transport shutdown failure, matching an earlier logged occurrence. Resource causation is not established, and no errors were suppressed.

## 2026-10-05: OpenCode semantic calls triggered a false non-use reflection

* Tool: benchmark harness OpenCode semantic-call classifier and diagnostic reflection dispatcher.
* Target: retained task 01 transcript `run_opencode_semedit_small-default_task-01-rename-local_1791151288852840000.json`.
* Observed failure: sixteen `semedit_semantic_*` calls were recorded, yet the harness asked why no semantic tools had been used. The model explicitly rejected that premise.
* Workaround: stop the repeated experiment, recognize OpenCode's provider prefix consistently, and restart with unbiased reflection dispatch. Retain interrupted attempts outside published result data.
* Root cause: `isSemanticTool` and `semanticToolBaseName` recognized other provider prefixes but omitted OpenCode's `semedit_` prefix. This is a harness classification failure, separate from MCP output-schema conformance.

## 2026-10-05: Semantic provider-prefix test insertion duplicated a field name

* Tool: `semantic_insert_function`.
* Target: `TestSemanticToolProviderPrefixesNormalizeConsistently` in `tools/benchmark-harness/opencode_usage_test.go`.
* Observed failure: diagnostics found duplicate `name` fields in table literals because the test label and tool name shared one struct field identifier.
* Workaround: update the test table to use a distinct `caseName` label and `name` tool field.
* Root cause: the inserted test used conflicting identifiers for two different values.

## 2026-10-05: Pointer-method continuation regression reproduced

* Tools: `semantic_inspect_symbol`, then `semantic_insert_construct`.
* Target: a temporary Go fixture containing pointer-receiver method `Dispatcher.Record` in `TestPointerMethodInspectionContinuationFeedsStatementInsertion`.
* Observed failure: inspection returned `in_function: "Dispatcher.Record"`; copying that owner into a statement insertion returned `function "Dispatcher.Record" not found`.
* Workaround: the regression remains failing until shared method-owner selection accepts the canonical continuation. The benchmark's pointer-qualified spelling `(*Dispatcher).Record` is retained as compatibility coverage.
* Root cause: insertion matched the receiver's pointer spelling independently instead of using the canonical symbol identity. This confirms the previously logged interoperability defect with an executable regression.

## 2026-10-05: Documented variable insertion regression reproduced

* Tool: `semedit insert-construct`, `kind=var`.
* Target: the new CLI fixture for documented declarations.
* Observed failure: the valid snippet containing a leading doc comment and `var ErrInvalidRequest = 2` returns `var insertion requires a var declaration`.
* Workaround: retain the failing regression before replacing prefix-based classification with parsed Go declaration-kind validation. Removing the doc comment would bypass the defect but is not a fix.
* Root cause: lexical prefix validation rejects valid syntax that the advertised declaration operation supports. This reproduces benchmark call `pLpqxVBBDUGaNYUEqb9YU4phAUdpOaBy`.

## 2026-10-05: Move-boundary semantic edit could not resolve an import package

* Tool: the baseline `semedit replace-body` CLI used for dogfooding.
* Target: `internal/astedit/move_construct.go`, `MoveConstruct`.
* Observed failure: orphan-import pruning returned `cannot resolve package name for import "golang.org/x/tools/go/packages"; refusing move: exit status 1` while adding the placement guard. A preceding semantic helper insertion succeeded.
* Workaround: inspect the resulting source and finish only the missing guard with a targeted atomic patch.
* Root cause: import pruning could not resolve the existing external package in this invocation's environment. This is an editing-tool limitation; the benchmark move regression remains independent.

## 2026-10-05: Post-repair Qwen import organization moved a function's doc comment

* Tool: `semantic_organize_imports` inside `semantic_batch`, second task 11 semedit trial of `opencode-qwen36-iq4-xs-post-bugfix-10x-20261005-0546`.
* Target: `audit/normalize.go` in the isolated mixed-sink fixture.
* Observed incorrect output: adding `strings` to a file with no existing import block moved NormalizeKind's doc comment between the generated `import` keyword and its string-literal spec. The source remained valid Go and resolved the import, but the function lost its attached documentation. Qwen described the layout as malformed and later used a native file write. The initial attribution to invalid syntax was corrected after contract review.
* Regression reproduction through the public `OrganizeImportsWithOptionsReport` API confirms the same comment-ownership defect with both `strings` and the quoted path `"strings"`; parsing succeeds but the function's `Doc` is nil afterward.
* A buffered MCP regression also reproduces the defect when `semantic_organize_imports` is the first successful batch child and a later missing-symbol `semantic_replace_body` fails. The batch retains the earlier import edit, but it persists as `import`, then the detached function comment, then the `"strings"` spec.
* Workaround: preserve the receipt and native recovery as evidence; freeze the benchmark binary until all trials finish. Then add CLI and buffered-batch regressions proving documentation ownership and canonical import placement.
* Root cause: explicit import insertion constructs a zero-position AST declaration, allowing the existing doc comment to attach to the generated import spec during formatting. Quoted paths are deliberately normalized by the existing parser and must retain compatibility. The preceding undefined `strings` extraction failure is expected because a batch defers automatic import normalization until its children finish; typed later children require an explicit imports step.
* Batch call ID: `yGYJ01WoPYXiVuM5PSH5XMjl8FOfUXp6`.
* Evidence: `.scratch/benchmarks/transcripts/run_opencode_semedit_small-default_task-11-mixed-sink-api-migration_1791182706282240000.json`.

## 2026-10-05: replace-body rejected oracle.go due to an unrelated package import

* Tool: `semantic_replace_body`.
* Target: `tools/benchmark-harness/oracle.go`, function `checkMutationPolicy`.
* Observed failure: the operation refused to apply because it could not resolve package `gopkg.in/yaml.v3` while pruning imports (`cannot resolve package name for import ...; refusing move`). The target function did not need import changes.
* Workaround: the failed request made no source changes; continue with a smaller supported construct insertion that avoids whole-body import pruning.
* Root cause: import pruning in the semantic operation tried to resolve an unrelated dependency in the harness package.

## 2026-10-05: replace-construct required a structural path for a nested statement

* Tool: `semantic_replace_construct`.
* Target: `tools/benchmark-harness/oracle.go`, nested disallowed-file condition in `checkMutationPolicy`.
* Observed failure: a condition-only discriminator did not match the full nested statement; the tool returned the precise candidate `construct_path 1.0.1` and no mutation occurred.
* Workaround: use the returned structural candidate path and retain the original statement as part of the replacement.
* Root cause: this selector requires the exact construct source or an ambiguity-resolved path when matching a statement that includes its body.

## 2026-10-05: replace-construct could not re-resolve its suggested nested path

* Tool: `semantic_replace_construct`.
* Target: `tools/benchmark-harness/oracle.go`, nested disallowed-file condition in `checkMutationPolicy`.
* Observed failure: the returned `construct_path 1.0.1` could not be replayed with the condition-only discriminator; the tool reported no candidates and made no changes.
* Workaround: stop retrying semantic selection and apply the minimal AST-equivalent source change through the repository's narrow fallback, then verify it.
* Root cause: the tool's ambiguity path and discriminator matching are inconsistent for this nested statement shape.

## 2026-10-05: replace-body refused an unrelated import in pipeline.go

* Tool: `semantic_replace_body`.
* Target: `internal/pipeline/pipeline.go`, function `insertExplicitImport`.
* Observed failure: the operation refused before writing because import pruning could not resolve `golang.org/x/tools/imports` (`cannot resolve package name ...; refusing move`). The edited function does not change imports.
* Workaround: use a narrowly scoped fallback to update only this function body, then run the focused pipeline and MCP regressions.
* Root cause: import pruning resolves unrelated package imports even when the selected body needs no import changes.

## 2026-10-05: review rejected restoring unsandboxed OpenCode execution

* Tool: `semantic_replace_body`.
* Target: `tools/benchmark-harness/opencode_driver.go`, `runOpenCode`.
* Observed failure: automatic review rejected the request to remove the OpenCode process sandbox. No source changes were made.
* Workaround: none; the sandbox remains in place pending explicit authorization.
* Root cause: review determined that questioning the sandbox did not explicitly authorize removal of this security boundary, especially given observed reads outside the fixture workspace.

## 2026-10-06: Body replacement omitted a required import

* Tool: `semantic_replace_body`.
* Target: `tools/benchmark-harness/external_repo.go`, `ensureRepositoryCache`.
* Observed failure: the replacement used `slices.Contains` but left imports unchanged; diagnostics reported `undefined: slices`.
* Workaround: organize imports for the file after the body replacement.
* Root cause: body replacement formats the declaration but does not add imports referenced by the new body.

## 2026-10-07: Type structure replacement is unsupported

* Tool: `semantic_insert_structure`.
* Target: `tools/benchmark-harness/session.go`, `agentSession`.
* Observed failure: `overwrite=true` returned `overwrite is unsupported for type structure insertion`; dependent body edits reported the missing field.
* Workaround: add the single session error field with a targeted patch, then retain semantic tools for method bodies.
* Root cause: the tool supports inserting types but cannot replace an existing struct or add a field.

## 2026-10-07: Verify could not resolve a nested scratch module

* Tool: `semantic_verify`.
* Target: `.scratch/explicit-recipes/go-zero-2463/base`.
* Observed failure: the tool reported `detected Go sources have no verifiable module scope`; formatting was applied before that failure.
* Workaround: none applied; the isolated repository is beneath the root workspace's excluded `.scratch` directory.
* Root cause: verification resolves module scope from the active root and does not descend into the ignored scratch checkout.

## 2026-10-07: MCP batch preflight could not load a pinned dependency

* Tool: `semantic_batch` through the repository MCP stdio server.
* Target: clean replay of go-zero `63cfe60f1a7583865757780ebfe2a5309a542134`.
* Observed failure: the batch failed before edits while gopls tried to download `github.com/ClickHouse/clickhouse-go/v2@v2.0.14`; DNS access to `proxy.golang.org` was unavailable in the sandbox.
* Workaround: request a narrowly scoped escalated replay using the repository's shared Go module cache.
* Root cause: the selected module cache did not contain this pinned module metadata and sandbox network access was unavailable.

## 2026-10-07: Nested-module verification hook could not load custom linter

* Tool: `semantic_verify`.
* Target: `tools/package-closure`.
* Observed failure: after explicit trust, the Go language-server check completed, but the configured golangci-lint hook exited because its binary did not contain the `packageclosure` plugin.
* Workaround: use the repository Make target, which builds the custom golangci-lint binary, and the package module's Go tests for verification.
* Root cause: the semantic verification hook invoked a golangci-lint binary without the repository's custom plugin build configuration.

## Semantic construct switch replacement unsupported (2026-10-07)

* Tool: `semantic_replace_construct`
* Target: `tools/benchmark-harness/planner.go`, `makeExternalRepositoryJobs`
* Observed failure: the tool rejected `kind: switch` as an unsupported Go construct kind.
* Workaround: replace the containing function body with the same implementation plus the exhaustive switch cases.
* Root cause: Go switch statements are not supported by this construct replacement operation.

## Semantic replace-declaration rejected struct type (2026-10-07)

* Tool: `semantic_replace_decl`.
* Target: `tools/benchmark-harness/codex_app_server.go`, `codexAppServer`.
* Observed failure: the operation rejected replacement because it only supports type aliases, not struct declarations; no change was made.
* Workaround: avoid adding the field and use the reader completion channel to unblock pending RPCs after an unexpected server request.
* Root cause: the operation's supported type surface excludes struct declarations.

## Semantic body replacement rejected JSON-RPC reader body (2026-10-07)

* Tool: `semantic_replace_body`.
* Target: `tools/benchmark-harness/codex_app_server.go`, `codexAppServer.readLoop`.
* Observed failure: parser rejected an anonymous nested struct declaration in the replacement body with `expected declaration, found '}'`; file was left unchanged.
* Workaround: simplify JSON decoding to a `map[string]json.RawMessage` and decode each envelope field separately.
* Root cause: the supplied local type/tag declaration was malformed for the semantic parser.

## Semantic insertion treated platform files as one package scope (2026-10-07)

* Tool: `semantic_batch`.
* Target: `tools/benchmark-harness/codex_app_server_process_windows.go`.
* Observed failure: insertion of the Windows implementation was rejected as a duplicate declaration from the Unix build-tagged file.
* Workaround: complete the platform-specific declaration in the build-tagged helper with a targeted source edit.
* Root cause: semantic declaration collision checks do not account for mutually exclusive Go build tags.

## Semantic batch rejected function insertion (2026-10-07)

* Tool: `semantic_batch`.
* Target: `tools/benchmark-harness/codex_app_server.go`.
* Observed failure: the batch rejected the first edit because `semantic_insert_function` is not batchable; no edits were applied.
* Workaround: insert declarations with `semantic_insert_function` directly, then replace affected bodies with `semantic_replace_body`.
* Root cause: the batch tool supports a narrower set of nested tools than its general edit workflow suggests.

## Reference lookup encountered a concurrent test type migration (2026-10-07)

* Tool: `semantic_find_references`.
* Target: `agentSession.runTurn` in `tools/benchmark-harness/session.go`.
* Observed failure: package loading failed while OpenCode tests still initialized `chan error` after the server completion channel changed to `chan struct{}`.
* Workaround: use targeted declaration inspections during dependent edits and repeat verification after tests and transport types are coherent.
* Root cause: the reference query ran during an incomplete transport/test migration.

## OpenCode continuation investigation

## Agy process struct replacement was unsupported (2026-10-07)

* Tool: `semantic_replace_decl`.
* Target: `tools/benchmark-harness/agy_stream.go`, `agyProcess`.
* Observed failure: The tool rejected replacing the named struct declaration, reporting that replacement requires a type alias; no file was changed.
* Workaround: Keep the process state shape stable and add frozen-config bytes to the owning `agySession` with a narrow `apply_patch` edit after the semantic replacement was unsupported.
* Root cause: The replacement tool's documented type support is limited to aliases, not defined struct types.

## Agy platform helper insertion collided across build-tagged files (2026-10-07)

* Tool: `semantic_insert_function`.
* Target: `tools/benchmark-harness/agy_stream_process_windows.go`, `configureAgyStreamProcess`.
* Observed failure: The tool reported a package declaration collision with the Unix build-tagged helper, although the files are mutually exclusive during Go builds; no Windows declaration was added.
* Workaround: Reuse the repository's existing Codex process-group helpers, which already provide Unix group signals and Windows process signaling.
* Root cause: The semantic index did not account for mutually exclusive build constraints when checking package symbol collisions.

## Suboptimal tool behavior

## grpc-go-2735 gold-reference lookups used base-only names

* Tool: `semantic_inspect_symbol`
* Target: `.scratch/explicit-recipes/grpc-go-2735/reference/balancer/grpclb/grpclb.go`, `balancer/xds/xds_client.go`, and `internal/backoff/backoff.go`
* Observed failure: The human-fix reference no longer contains `defaultBackoffConfig` or `factor`, so those base-revision selectors returned symbol-not-found errors.
* Workaround applied: Query the new names from the reference, and inspect the original base when the previous declaration is needed.
* Root cause: The accepted patch intentionally renames these identifiers.

## 2026-10-08: grpc-go-3361 semantic_batch blocked by missing cached module

* Tool: `semantic_batch` through the frozen `semedit-20261008` server.
* Target: `.scratch/explicit-recipes/grpc-go-3361/replay/balancer/base/balancer.go`.
* Observed failure: Batch diagnostics failed before any child edit because `gopls` could not load `github.com/envoyproxy/go-control-plane@v0.9.1-0.20191026205805-5f8ba28d4473` while `GOPROXY=off`.
* Workaround applied: None yet; the tool reported application state unknown, so inspect the replay status and hydrate the exact dependency from the local cache before retrying the same single batch.
* Root cause: The isolated pinned checkout's dependency is absent from the frozen workspace module cache, and the transport disables network module lookup.
* Retry: After the cache-ready signal, the same preflight still failed on the identical missing module; no batch child ran. Preserve both responses and confirm the transport's exact cache path before retrying.
* Retry: A second post-signal replay also failed before edits with the same missing-module diagnostic. The archive and expanded module directory are now present in `.scratch/go/mod`; verify Go's module-cache resolution before any further MCP retry.
* Retry: Even after `go list -m github.com/envoyproxy/go-control-plane` and `go list -mod=readonly ./balancer/base` succeeded with the transport cache variables, batch preflight still reports the same module lookup failure. No child edit ran; investigate the gopls package-load scope/environment rather than repeating the mutation call.
* Retry: The transport fix supplied explicit `go-base-dir`, expected workspace root, and full tool profile; gopls now reaches the correct dependency cache but preflight stops on the unrelated nested package `security/advancedtls` with “package has errors but no selected Go files.” No child edit ran. Narrow the semantic workspace to the repository's main module or resolve the nested-module diagnostics before retrying.
* Retry: After hydrating the `security/advancedtls` module graph, semantic_batch preflight advances further but stops at `test/tools` with “no Go source files found in project root.” No child edit ran; the broad workspace scan treats a non-Go tools directory as a project root. Narrow the semantic scope to the main module without changing task sources.

## 2026-10-08: go-zero-964 semantic_batch blocked by missing cached module

* Tool: `semantic_batch` through the frozen `semedit-20261008` server.
* Target: `.scratch/explicit-recipes/go-zero-964/replay-base`.
* Observed failure: Three batch attempts failed pre-diagnostics before any child edit because `gopls` could not load `github.com/ClickHouse/clickhouse-go@v1.4.3` while `GOPROXY=off`, even though standalone `go list -m all` succeeded against the same cache.
* Workaround applied: Use the updated transport with `mcp --go-base-dir <repo>/.scratch/go` so the MCP server's explicit Go cache root matches the prepared pinned module artifacts.
* Root cause: `internal/mcp` sets `gocache.WithBaseDir` and ignores inherited `GOMODCACHE` for diagnostics.

## grpc-go-2735 first batch blocked by an uncached module

* Tool: `semantic_batch` via `.scratch/explicit-recipes/mcp_transport.py`
* Target: fresh replay at `.scratch/explicit-recipes/grpc-go-2735/replay-final`
* Observed failure: Batch preflight stopped before executing edits because Go diagnostics could not load `cloud.google.com/go@v0.26.0` with `GOPROXY=off`.
* Workaround applied: None yet; locate the already available local module cache or ask the parent to seed it, then retry only after confirming the workspace is still pristine.
* Root cause: The isolated replay workspace's module graph references an uncached historical dependency and the replay environment correctly has network access disabled.

## grpc-go-2735 replay retry still sees the old module cache

* Tool: `semantic_batch` via `.scratch/explicit-recipes/mcp_transport.py`
* Target: fresh replay at `.scratch/explicit-recipes/grpc-go-2735/replay-final`
* Observed failure: After the parent reported the dependency cache ready, the offline batch preflight still returned `module lookup disabled by GOPROXY=off` for `cloud.google.com/go@v0.26.0`; zero child edits ran again.
* Workaround applied: None yet; verify the seeded artifact is visible under the exact `.scratch/go/mod` cache path used by the transport before another attempt.
* Root cause: The replay subprocess did not resolve the supposedly seeded pinned module artifact from its configured cache.

## grpc-go-2735 batch could not relocate a grouped constant

* Tool: `semantic_move_construct` inside `semantic_batch`
* Target: `clientconn.go` constant `minConnectTimeout`, destination `dialoptions.go`
* Observed failure: The batch completed 16 preceding edits, then refused with `cannot move one member of a grouped declaration; split the group first`.
* Workaround applied: None yet; author a new replay batch that removes the constant from its grouped declaration before inserting it in `dialoptions.go`, then start from a fresh pinned copy because batches do not roll back earlier edits.
* Root cause: The constant shares a parenthesized declaration, and `semantic_move_construct` does not split groups.

## grpc-go-2735 semantic batch normalized the cloned go.mod

* Tool: `semantic_batch` via `.scratch/explicit-recipes/mcp_transport.py`
* Target: `.scratch/explicit-recipes/grpc-go-2735/replay-final/go.mod`
* Observed failure: The partial batch altered the pinned clone's manifest, adding `go 1.27.1` and regrouping indirect requirements even though the recipe changed no module dependencies.
* Workaround applied: None yet; preserve this failed attempt and prepare a clean replay with import organization disabled because every required import already exists.
* Root cause: Semantic batch child operations were configured with `auto_organize_imports=true`, which caused an unrelated workspace normalization on the older external module.

## grpc-go-2735 batch diagnostics still reject a populated module cache

* Tool: `semantic_batch` via `.scratch/explicit-recipes/mcp_transport.py`
* Target: clean replay at `.scratch/explicit-recipes/grpc-go-2735/replay-final-2`
* Observed failure: Preflight still reports `module lookup disabled by GOPROXY=off` for `cloud.google.com/go@v0.26.0` after the module's info, mod, and zip files were present and `go mod download` succeeded offline.
* Workaround applied: None yet; compare the diagnostics subprocess environment with a successful offline `go list` using the same module cache.
* Root cause: The MCP diagnostics path did not observe the populated module cache as the direct Go command did.

## grpc-go-2735 readonly transport with corrected root still fails preflight

* Tool: `semantic_batch` via `.scratch/explicit-recipes/grpc-go-2735/mcp_transport_readonly.py`
* Target: clean replay at `.scratch/explicit-recipes/grpc-go-2735/replay-final-2`
* Observed failure: Preflight returned the same `cloud.google.com/go@v0.26.0` lookup error even though the helper now points at the populated `.scratch/go/mod` cache and direct offline `go list ./...` succeeds.
* Workaround applied: None yet; retry once with a fresh, private GOPLSCACHE path to rule out stale gopls state.
* Root cause: The MCP diagnostics session disagrees with a successful Go module resolution using the same cache and offline settings.

## grpc-go-2735 offline diagnostics ignored GOMODCACHE configuration

* Tool: `semantic_batch` through the custom transport
* Target: `.scratch/explicit-recipes/grpc-go-2735/replay-final-2`
* Observed failure: Each preflight returned `module lookup disabled by GOPROXY=off` for `cloud.google.com/go@v0.26.0`, even after the module was present in `.scratch/go/mod` and direct Go commands succeeded.
* Workaround applied: Use the shared MCP transport with `--go-base-dir` pointing at the repository's `.scratch/go`; preserve the current clean replay for that retry.
* Root cause: `internal/mcp` passes an explicit `gocache.WithBaseDir`, which overrides inherited `GOMODCACHE` and `SEMEDIT_SHARED` settings. The CLI must receive `mcp --go-base-dir` so diagnostics use the intended cache.

## grpc-go-2735 safe delete refused a module containing generation directives

* Tool: `semantic_remove_construct` inside `semantic_batch`
* Target: `health/client.go` package constant `maxDelay`
* Observed failure: The batch completed 29 children, then returned `safe_delete refuses modules containing Go generation or link-name directives` while attempting to remove the now-unused constant.
* Workaround applied: None yet; retry with `safe_delete: false` for this single known-obsolete constant and validate the complete workspace with the benchmark oracle.
* Root cause: Safe-delete conservatively refuses any module containing generation or link-name directives, even when the selected constant has no remaining references after the recipe's earlier updates.

## grpc-go-2951 initializer insertion rejects legal package initializers

* Tool: `semantic_insert_function` inside `semantic_batch`
* Target: `clientconn.go`, new `init` function
* Observed failure: Child 18 rejected the declaration because a test file in the package already declares `init`; Go permits multiple package initializers.
* Workaround applied: Fold the new initialization statements into the existing `service_config.go` initializer using one body replacement; retry the complete recipe in a fresh pinned copy.
* Root cause: Package declaration collision checks treat `init` as an ordinary uniquely named function.

## grpc-go-2951 final refactoring session synchronization timed out

* Tool: `semantic_batch`
* Target: grpc-go-2951 fresh 59-operation recipe replay
* Observed failure: All 59 requested child edits completed, but final `session_sync` returned `gopls session: context deadline exceeded`; the overall batch failed.
* Workaround applied: Preserve the failed receipt and investigate the synchronization deadline before another fresh replay.
* Root cause: Retained-session synchronization has a 90-second deadline and reloads the complete Go analysis scope. A subsequent fresh replay with the identical 59-operation payload completed successfully, followed by a passing ordinary oracle. The timeout receipt remains an authoring failure; child success alone did not establish batch success.

## Explicit recipe harness verification selected an incompatible linter

* Tool: `semantic_verify`
* Target: `tools/benchmark-harness`, trusted check-only verification
* Observed failure: The configured `golangci-lint` hook exited 3 while building linters, although the repository's configured `make check` passed.
* Workaround applied: Retain the semantic verification receipt; use the repository's configured check and harness test commands, and inspect the hook's tool resolution before any retry.
* Root cause: The hook resolved a global golangci-lint binary incompatible with the project's `packageclosure.so` plugin (`different version of package cmp`). `make check` uses the pinned toolchain-matched linter instead; use that same executable on PATH for semantic verification.

## Explicit recipe semantic verification treats architecture advisories as failures

* Tool: `semantic_verify`
* Target: `tools/benchmark-harness`, pinned linter and project-local cache
* Observed failure: Verification returned an existing `packageclosure` recommendation to split provider and harness files; no source diagnostics or ordinary lint defects blocked `make check`.
* Workaround applied: Stop retries after identifying the configuration difference. Use the configured `make check` and harness behavior tests as the acceptance checks; retain this semantic verification failure without claiming success.
* Root cause: The generic golangci-lint verification hook uses its default issues exit code. The repository's Makefile explicitly makes package-closure recommendations informational, then disables that advisory analyzer for required lint checks.

## 2026-10-08: Replay transport expected JSON from a successful body replacement

* Tool: `semantic_replace_body`, local replay transport helper.
* Target: `internal/mcp/server_test.go`, `TestMCPServerReturnsConfiguredInstructions`.
* Observed failure: the semantic edit succeeded with unchanged diagnostics, but the helper raised `JSONDecodeError` while parsing the tool's plain-text `structuredContent.result` as JSON.
* Workaround: inspect the retained response's explicit `isError: false`, diagnostic status, and source diff; do not repeat the successful mutation.
* Root cause: the recipe-focused helper assumes JSON batch result text for every tool response.

## 2026-10-10: body replacement could not resolve existing imports

* Tool: `replace-body`
* Target: `internal/astedit/construct.go`, `constructReplacementWrapper`
* Observed failure: The replacement was rejected during orphan-import pruning because the package name for `bytes` could not be resolved; no source changed.
* Workaround applied: Use the narrower switch-clause insertion with its required discriminator.
* Root cause: The semantic replacement's import-pruning preflight could not resolve an existing import in the current workspace state.

## 2026-10-11: body replacement could not resolve a standard-library import

* Tool: `replace-body`
* Target: `tools/benchmark-harness/opencode_relay.go`, `startOpenCodeRelay`
* Observed failure: Replacement was rejected while pruning imports because it could not resolve the existing `context` import; no source changed.
* Workaround applied: Preserve the existing import set and retry with a body that uses the already imported package.
* Root cause: The semantic replacement preflight could not resolve an existing standard-library import in the current workspace state.

## 2026-10-11: retry could not resolve the same standard-library import

* Tool: `replace-body`
* Target: `tools/benchmark-harness/opencode_relay.go`, `startOpenCodeRelay`
* Observed failure: A retry of the body replacement again refused to resolve the existing `context` import; no source changed.
* Workaround applied: Use narrower semantic construct edits that do not require whole-function replacement.
* Root cause: The import-resolution failure persisted across retries.

## 2026-10-11: construct replacement could not resolve a standard-library import

* Tool: `replace-construct`
* Target: `tools/benchmark-harness/opencode_relay.go`, relay initialization in `startOpenCodeRelay`
* Observed failure: The replacement was rejected while resolving the existing `context` import; no source changed.
* Workaround applied: Retry the local statement edit using an insertion or a separate helper to avoid import movement.
* Root cause: The construct replacement import-cleanup preflight could not resolve `context` in the current workspace state.

## 2026-10-11: lookup confirmed an earlier batch did not create its helper

* Tool: `semantic_lookup`
* Target: OpenCode integration helper `openCodeOutboundURL`
* Observed failure: The helper was not found; no source changed.
* Workaround applied: Retry insertion as a standalone semantic edit.
* Root cause: The prior batch stopped before its first edit because of an invalid payload field.

## 2026-10-11: trusted semantic batch verification failed lint

* Tool: `semantic_batch` with OpenCode session and driver body replacements
* Target: `tools/benchmark-harness/opencode_session.go` and `opencode_driver.go`
* Observed failure: The trusted batch reached the golangci-lint verification hook, which exited with status 3; the agent is inspecting whether the body edits applied and the lint diagnostics.
* Workaround applied: Inspect the workspace diff and diagnostics before further changes.
* Root cause: The verification hook reported lint failure after applying the requested semantic mutations.

## 2026-10-11: test construct replacement could not resolve a standard-library import

* Tool: `replace-construct`
* Target: `tools/benchmark-harness/session_test.go`, Codex stream test setup
* Observed failure: The statement replacement was refused while resolving the existing `context` import; no source changed.
* Workaround applied: Use a narrow atomic text patch after the semantic operation failed.
* Root cause: The semantic replacement import-cleanup preflight could not resolve an existing standard-library import in the current workspace state.

## 2026-10-11: Saluki discovered rename but Go diagnostics failed in Seatbelt

* Tool: OpenCode `semedit_semantic_rename`
* Target: `api/server.go`, `Server.oldName`, in the implicit and explicit arms of `saluki-opencode-relay-20261011T095400Z`
* Observed failure: Two calls in each arm reached the MCP server but failed with `check Go diagnostics with gopls: resolve Go diagnostics module: exit status 2`. Explicit lookup and outline succeeded.
* Workaround applied: Retain the failures in the transcript and benchmark report; no sandbox allowance or benchmark verdict was relaxed.
* Root cause: Retained shell output confirms the sandbox denied the inherited cached Go 1.27.2 GOROOT and the selected gopls wrapper's backing executable and canonical interpreter. Go reported `cannot find GOROOT directory` before source diagnostics. The runtime profile now allows the exact selected toolchain and wrapper dependencies; the diagnostics manager preserves Go stderr and the process error.
<!-- Records tool failures encountered while maintaining this repository. -->

## 2026-10-11: body replacement deferred tool unavailable

* Tool: `semantic_replace_body`.
* Target: `tools/benchmark-harness/driver.go`, `Runner.ExecuteAgent`.
* Observed failure: the declared direct semantic operation was unavailable in the active tool namespace, returning a JavaScript `TypeError`; no source mutation occurred.
* Workaround: use the callable `semantic_batch` wrapper for the same structural replacement.
* Root cause: deferred MCP tool registration did not expose the advertised direct body-replacement method to this task.

## 2026-10-11: task-timeout body replacement verification failed

* Tool: `semantic_batch` with `semantic_replace_body`.
* Target: `tools/benchmark-harness/driver.go`, `Runner.ExecuteAgent`.
* Observed failure: the requested AST edit applied, but the trusted workspace verification hook reported a repository-wide `golangci-lint` exit status 3.
* Workaround: inspect the applied source and run the focused harness checks after completing the related lifecycle edits.
* Root cause: the batch verifier ran the repository hook while concurrent uncommitted work remained in the workspace.

## 2026-10-11: runtime directory loop selector rejected

* Tool: `semantic_replace_construct`.
* Target: `Runner.openCodeEnvironment` in `opencode_driver.go`.
* Observed failure: a range header selector did not resolve the loop.
* Workaround: use returned `construct_path: 17`.
* Root cause: loop selection requires the supported source form or the returned structural path.

## 2026-10-11: nested test cleanup cannot be selected

* Tool: `semantic_replace_construct`.
* Target: `TestWorkspacePrewarmIsIndependentOfHarness` in `session_test.go`.
* Observed failure: the tool did not find a defer inside the subtest function literal.
* Workaround: replace the enclosing test function with the corrected cleanup variable name.
* Root cause: nested function-literal statements are outside the selector's supported traversal.

## 2026-10-11: same-source call replacement is ambiguous

* Tool: `semantic_replace_construct`.
* Target: `TestAgentLaunchFailureHasZeroTaskTimeForBothArms` in `session_test.go`.
* Observed failure: selecting `t.Setenv("PATH", root)` matched both the statement and its expression slot.
* Workaround: omit the unnecessary replacement and replace the enclosing test declaration for its setup changes.
* Root cause: identical call text is a valid selector for two construct kinds.

## 2026-10-11: oracle-context replacement verification failed

* Tool: `semantic_batch` with `semantic_replace_body`.
* Target: `tools/benchmark-harness/session.go`, `agentSession.evaluate` and `agentSession.evaluateStage`.
* Observed failure: both AST edits applied, but the trusted workspace verification hook reported a repository-wide `golangci-lint` exit status 3.
* Workaround: inspect the applied source and use focused harness checks after all lifecycle changes are complete.
* Root cause: the batch verifier ran the repository hook while concurrent uncommitted work remained in the workspace.

## 2026-10-11: Exact-source selectors for OpenCode compaction configuration

The CLI `replace-body` fallback also rejected this file because orphan-import pruning could not resolve the standard-library `bufio` package. It reported exit status 1; subsequent inspection is needed to establish whether it published the body before failing. Workaround: use method replacement through the selected-file MCP editor, which had successfully resolved this package during inspection and insertion.

`semantic_insert_construct` rejected partial switch/if anchors in `tools/benchmark-harness/opencode_driver.go`; `semantic_replace_construct` rejected a one-line rendering of a multiline map literal. Those edits were not applied. A later exact nested statement anchor (`limits := *r.openCodeModelLimits`) also failed, so source matching alone is insufficient for nested insertion. Root cause: prefix anchors are unsupported and nested insertion navigation is unavailable through this selector. Workaround: replace the inspected function body using `semantic_replace_body`. The first four calls were mistakenly issued together before logging; subsequent edits stop on the first failed result.

## 2026-10-11: Map selector indentation during fixture flag update

`semantic_replace_construct` rejected the expected-environment map in `tools/benchmark-harness/bench_test.go`, including a retry with the normalized candidate copied from its diagnostic. No mutation occurred. Root cause is unresolved: the displayed expression is not accepted as a discriminator. Workaround: targeted atomic edits to the two expected flag values; use semantic insertion for the new behavior regression.

## 2026-10-11: semantic batch verification repeated stale plugin failure

* Tool: `semantic_batch`.
* Target: standalone buffer wrappers in `internal/astedit`.
* Observed failure: the selected insertions completed, but batch verification again failed loading `.scratch/go/bin/packageclosure.so` because it was built with a different Go package version.
* Workaround: retain the successful source edits and validate with `make check` after the parallel operation migration is complete.
* Root cause: the same stale custom analyzer plugin still prevents verification.

## 2026-10-11: semantic function removal left stale caller

* Tool: `semantic_remove_construct`.
* Target: `internal/astedit/remove.go`.
* Observed failure: removing `removeWithOwnedBuffer` correctly deleted the obsolete helper but reported a remaining reference in `Remove` that became undefined.
* Workaround: remove the now-unreachable call branch because `Remove`'s public entry now always binds a buffer before entering its implementation.
* Root cause: the helper's caller was not part of the function-removal operation.

## 2026-10-11: top-level placement variable replacement rejected

* Tool: `semantic_replace_construct`.
* Target: `internal/operation/wire_engine_constructs.go`, `constructPlacementEnum`.
* Observed failure: replacing the package-level variable was rejected because the tool interpreted `kind=variable` as requiring an enclosing function.
* Workaround: apply a targeted edit to the enum literal; no source mutation occurred.
* Root cause: the construct replacement tool does not support top-level variable declarations.

## 2026-10-12: rename conflated references to duplicate helper declarations

* Tool: `semantic_rename`.
* Target: production `intPointer` in `internal/operation/wire_remaining_outputs.go`.
* Observed failure: renaming the selected production helper also changed the `intPointer` reference in `internal/operation/value_schema_test.go`, whose test-only declaration was intended to remain independent.
* Workaround: restore the test reference after the coordinating compile pass releases the source freeze; retain the production rename.
* Root cause: gopls resolved the reference to the production helper while the package contained duplicate declarations. The rename succeeded but cannot distinguish intended references in this invalid package state.

## 2026-10-12: semantic outline documented file argument rejected

* Tool: `semantic_outline`.
* Target: `internal/mcp/staging.go`.
* Observed failure: the call using its documented `file` argument was rejected because this environment accepts `path` instead.
* Workaround: retry the same read-only outline using `path`; no source fallback or mutation was required.
* Root cause: the available MCP schema differs from the documented parameter name.

## 2026-10-12: workspace outline rejected a file target

* Tool: `bin/semedit-next workspace-outline`.
* Target: `internal/mcp/batch_verification.go` and `internal/operation/wire_inspect_construct.go`.
* Observed failure: the command rejected each file path because it only accepts directories.
* Workaround: retry the read-only outline for the containing package directories.
* Root cause: the CLI command's path contract only supports workspace and directory outlines.

## 2026-10-12: lookup did not resolve the batch handler name

* Tool: `semantic_lookup`.
* Target: `internal/mcp/batch.go`, `handleBatchToolCall`.
* Observed failure: the unqualified lookup also reported no matching declaration.
* Workaround: use a narrow text search to identify the response projection and apply the smallest source edit.
* Root cause: the operation handler is registered under a generated or differently named symbol.

## 2026-10-13: semantic replace construct could not select handler map expression

* Tool: `semantic_replace_construct`.
* Target: `internal/operation/wire_engine_constructs.go`, `insertConstructDef` handler map.
* Observed failure: the map expression was absent from the tool's statement/expression candidate set; the diagnostic exposed only the containing return statement and nested selectable expressions.
* Workaround: use the containing function body as the semantic replacement unit and preserve its existing return contract while adding the handler entry.
* Root cause: the current Go construct locator does not resolve the typed handler-map expression as a direct replacement selector.

## 2026-10-13: semantic batch verification required workspace trust

* Tool: `semantic_batch` with child `semantic_replace_body`.
* Target: `internal/operation/wire_engine_constructs.go`, `insertConstructDef`.
* Observed failure: batch verification stopped because the configured `golangci-lint` hook requires `trust_workspace: true`.
* Workaround: inspect the target to determine whether the child edit was applied, then retry with workspace trust after confirming the exact scoped change.
* Root cause: the semantic batch ran configured workspace verification without explicit trust.

## 2026-10-13: semantic insert construct did not match validator anchor

* Tool: `semantic_insert_construct`.
* Target: `internal/operation/wire_engine_constructs.go`, `parseInsertConstruct`.
* Observed failure: the before-anchor selector matched only the start of an `if` statement and was rejected as not found.
* Workaround: retry with the complete statement source as the anchor; no source mutation occurred on the failed call.
* Root cause: this insertion API requires the full anchor text for statement selection.

## 2026-10-13: semantic batch golangci-lint hook failed

* Tool: `semantic_batch` with child `semantic_replace_body`.
* Target: `internal/javacst/locator.go`, `InsertIntoMethod`.
* Observed failure: the child edit completed, but the configured `golangci-lint` verification hook exited with status 3.
* Workaround: inspect the applied body and run focused formatting, tests, and lint diagnostics for the touched package.
* Root cause: the workspace lint hook returned a nonzero status during semantic batch verification; the failure output did not yet identify a source diagnostic.

## 2026-10-13: semantic replace construct could not select a Go loop

* Tool: `semantic_replace_construct`.
* Target: `internal/javacst/locator.go`, `ReplaceMethodBody`.
* Observed failure: the `statement/expression` selector did not resolve the surrounding Go `for` loop used to infer body indentation.
* Workaround: replace the containing `if` statement as one unit and express first-child selection with an `if` guard, avoiding the nested loop selector; no source mutation occurred on the failed call.
* Root cause: the construct locator did not expose the helper loop through the requested statement/expression selector.

## 2026-10-13: semantic find references exposed an incomplete Java CST helper

* Tool: `semantic_find_references`.
* Target: `internal/javacst/parameter_object.go`.
* Observed failure: Go package loading failed because `applySourceSpans` referred to undefined `javaEditSpan`; no source mutation occurred.
* Workaround: inspect and repair the incomplete Java parameter-object helper before retrying semantic navigation.
* Root cause: an unfinished helper edit used a span type that was never declared in the file.

## 2026-10-13: semantic replace construct did not match nested removal guard

* Tool: `semantic_replace_construct`.
* Target: `internal/operation/wire_remove.go`, `parseRemove`.
* Observed failure: the operation rejected the full `if` statement as a discriminator and returned candidate path `16.0.0`; no source mutation occurred.
* Workaround: retry with the reported construct path and current file revision, or use a targeted source patch if that selector remains ambiguous.
* Root cause: the Go construct selector matches an `if` by its condition or short initializer, not by the full statement body.

## 2026-10-13: semantic replace construct did not expose nested Java safe-delete guard

* Tool: `semantic_replace_construct`.
* Target: `internal/operation/wire_remove.go`, `runRemoveJava`.
* Observed failure: the operation could not select the nested `if req.SafeDelete` and listed only the function's top-level `if` candidates; no source mutation occurred.
* Workaround: inspect the current declaration and use a targeted source patch for the nested preflight block.
* Root cause: the Go construct selector did not enumerate an `if` nested inside the Java declaration-kind branch.

## RQ-0069 CLI import resolution failure (2026-10-13)

* Tool: `semantic_replace_body` through the CLI.
* Target: `cmd/docgen/txtar.go`, `firstTxtarExampleFiles`.
* Observed failure: orphan-import pruning could not resolve the standard library package `cmp` and refused publication.
* Workaround: retry using the repository's configured Go environment and populated toolchain cache.
* Root cause: the direct CLI process did not inherit the Makefile's isolated Go environment.

The retry with the configured Go environment also refused `cmp` resolution. No source publication occurred. Use a targeted patch to sort candidate fixture filenames before source selection; the import-pruning path cannot resolve this valid standard library import in this session.

## RQ-0069 advertised expression replacement rejected (2026-10-13)

* Tool: `semantic_replace_construct`.
* Target: `cmd/docgen/main_test.go`, `TestPolyglotTxtarDocumentationIsDeterministic`.
* Observed failure: the live MCP server rejected the advertised Go `expression` kind.
* Workaround: apply a targeted correction to the expected trimmed source string.
* Root cause: the running server's supported construct kinds differ from its description.

## RQ-0069 semantic batch diagnostics timeout (2026-10-13)

* Tool: `semantic_batch`.
* Target: `internal/capability/contract_model.go` and `internal/mcp/server.go` in the schema worktree.
* Observed failure: pre-edit Go diagnostics for `internal/operation/wire_backend.go` exceeded the request deadline. No edits occurred.
* Workaround: apply the smallest targeted edits after verifying that the batch left the worktree unchanged.
* Root cause: workspace-wide pre-edit diagnostics exceeded the tool's deadline in this large workspace.

## Suggestion callback rejected by automatic review (2026-10-16)

* Tool: `semantic_insert_construct` on `internal/operation/suggestions.go`.
* Observed result: automatic review rejected a callback that invoked a registered operation during suggestion validation because it could permit side effects or an automatic mutation retry. No source edit was applied.
* Resolution: abandoned the generic invocation fallback; retain explicit non-mutating native preflights. No alternate write path was used to apply the rejected callback.
* Root cause: a full operation callback does not provide the structural no-write guarantee required by the suggestion boundary, even when selected through read-only metadata.

## Inherited Go state exposed overlay and temporary-source assumptions (2026-10-14)

* Tools: `semantic_insert_declaration`, `semantic_move_construct`, and signature operations through `semantic_batch` in CLI contract tests.
* Targets: staged scaffold and cross-package batch fixtures under `testdata/scripts/`.
* Observed failures: `go env` attempted to enter an overlay-only directory absent from disk; other batches reported that the source set changed when temporary Go files appeared beneath the fixture.
* Workaround: resolve environment settings from the nearest existing directory and exclude configured temporary/cache roots from source snapshots.
* Root cause: environment discovery assumed source directories already existed, and source-set discovery relied on the old `.scratch` cache layout to exclude transient files.

### Multiple anchored statements rejected (2026-10-16)

`semantic_insert_construct` rejected insertion of two statements in
`internal/operation/suggestions.go:suggestionRetrievalContext` with
`expected exactly one statement`. The statement insertion API accepts one
statement per call. No source change was applied; use separate insertions.

## Summary field selector not accepted by semantic replace (2026-10-16)

* Tool: `semantic_replace_construct`.
* Target: `internal/operation/wire_read_inspect.go`, `outlineReadDef` summary field.
* Observed failures: the call could not resolve a `Summary: "..."` field expression as the discriminator; a later lookup call used a stale construct path for the string literal. Neither call mutated source.
* A third call found the exact high-risk catalog assertion twice (as a statement and expression) and required an explicit `construct_path`; it also did not mutate source.
* Workaround: use the reported expression construct path for string literals and the statement path for catalog assertions; apply a focused source patch only if those selectors remain unsupported.
* Root cause: the tool selector expects the AST expression itself, not its containing struct field syntax, construct paths vary with declaration structure, and call expressions can have separate statement/expression candidates.

## Lookup declaration range temporarily unused after construct replacement (2026-10-16)

* Tool: `semantic_replace_construct`.
* Target: `internal/backend/textlsp/backend.go`, `selectSymbol`.
* Observed failure: replacing the candidate literal removed the only use of `declarationRange`; the operation reported a new Go diagnostic, while the edit itself was syntactically applied.
* Workaround: attach the range only inside a guard checking the native hierarchical range, then verify diagnostics and focused tests.
* Root cause: the candidate's optional range is a separate contract field and its construction must be conditioned on the source symbol shape.

The follow-up `semantic_insert_construct` anchor for that guard was not found in `selectSymbol`, so it did not mutate source. The smallest safe repair is to use the exact current candidate statement as the anchor or make a targeted AST-supported replacement.

A `semantic_replace_construct` attempt to replace that statement with the statement plus its guard was rejected because the replacement must contain exactly one statement; no source mutation occurred. Use a targeted patch for the separate guard statement.

## Navigation verification hook trust (2026-10-16)

* Tool: `semantic_verify` with `check_only: true`.
* Target: `internal/backend/navigation.go`.
* Observed failure: the configured `golangci-lint` verification hook required explicit workspace trust. The language-server response also reported embedded-field literal errors outside this change while the repository compiler checks passed.
* Workaround: retry with workspace trust and use the configured `make check` as the compilation and lint gate.
* Root cause: verification hooks enforce trust independently of diagnostics-only mode; the language-server diagnostics require comparison with the configured Go toolchain.

The trusted retry reached the hook, but its executable could not load the
`packageclosure` plugin because it was built with a different version of package
`cmp`. Repository `make check` uses the pinned linter executable and matching
plugin, so it remains the verification gate. No hook or lint configuration was
changed to bypass this toolchain mismatch.

## Wrap-up history retrieval failure (2026-10-16)

* Tool: `codex_app.read_thread`, call `exec-550b5cfd-24a3-4063-97f8-529165cd1d45`.
* Target: this thread's earlier semantic-call history.
* Observed failure: requesting 20 turns with outputs returned an error string instead of a JSON thread page; the orchestration parser then rejected it. The original error detail was not retained.
* Workaround: request smaller pages without outputs, validate the tool result before parsing, and retrieve individual output evidence only where needed.
* Root cause: unknown tool failure; the caller also assumed successful JSON content without checking.

## 2026-10-16: session wrap-up, Compact semedit symbol results

* Thread: `01a145a0-b77b-74e3-a9ee-b33e5cff30a3`; linked checkout `ios-7199cf6f-1c28-4105-b8c1-81443ee8df82/semantic-editor`, initially detached at `1ff9655e5719c2d078c4e274e4a4817f006cced9`.
* Scope: compact symbol navigation, the ADR audit, and the ADR-0003 handle clarification. Assessment includes root and all three child-agent transcripts from the local 2026-10-16 session files. Transcript line references below accompany stable call IDs; `root`, `contracts`, `implementation`, and `transport_tests` map respectively to session IDs `01a145a0-b77b-74e3-a9ee-b33e5cff30a3`, `01a145a0-f8ff-7d80-8be4-7323f0386f5f`, `01a145a1-7174-74f0-88ef-3aae66135a47`, and `01a145a3-8445-7481-8ca8-3a933d524074`.
* Coverage boundary: all retained completed semantic MCP items and the six explicit semantic CLI invocations in those transcripts up to this wrap-up inventory. Catalog/help queries, source searches, suggested operations, and test-harness-generated semantic calls are not counted as agent attempts. No MCP transport helper invocations or semantic batch parents/children were found. The initial failed workspace-outline CLI invocation is counted even though the semantic engine never started. No semantic calls were made by this wrap-up itself. Claims do not extend to unavailable/deleted transcripts or unrecorded subprocess activity.
* Totals: 48 attempts, comprising 42 MCP calls and 6 CLI equivalents; 38 beneficial, 2 neutral, 8 harmful, 0 unknown. Ratings assess observed agent effort, not just operation success. Batch parents: 0; batch children: 0.

| Call / transcript | Tool and target / intent | Outcome and mutation state | QoL assessment |
| :--- | :--- | :--- | :--- |
| `exec-41c11cf7-398b-4b86-9936-7e205145d26b` / root:723 | `semantic_inspect_symbol` / `internal/backend/golang/references.go :: typedReferenceRecord` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-be33e0ae-2178-43dc-b76d-acdc09f7fbca` / root:2284 | `semantic_verify` / `internal/backend/navigation.go` | Trust guard rejected verification before hooks ran; no mutation. | **neutral**: Expected trust rejection, not a tool defect; retried explicitly trusted. |
| `exec-e3d268ab-78cd-47a3-94ee-54ad7ecad734` / root:2317 | `semantic_verify` / `internal/backend/navigation.go` | Trusted verification did not complete; no mutation. | **harmful**: Environment: hook linter/plugin Go versions differ; gopls also emitted embedded-field errors not reproduced by pinned compiler. Used make check. |
| `exec-6e7bccc7-1d84-44fd-81dd-c367151ce07a` / implementation:47 | `semantic_lookup` / `internal/backend/golang/backend.go :: convertLookup` | Resolved exact declaration coordinates without mutation. | **beneficial**: Supplied native identity for focused inspection; avoided manual coordinate calculation. |
| `exec-92861459-5d87-4ccc-8080-223753e80257` / implementation:54 | `semantic_inspect_symbol` / `internal/backend/golang/backend.go :: convertLookup` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-372f507e-4514-4114-bc1b-264f8e7ecb9c` / implementation:73 | `semantic_inspect_symbol` / `internal/backend/backend.go :: NamedTarget` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-51e4abcd-e921-4274-969f-a845a0c91963` / implementation:74 | `semantic_inspect_symbol` / `internal/backend/backend.go :: LookupResult` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-882708de-4ada-4202-8e75-c0e5d240064a` / implementation:75 | `semantic_inspect_symbol` / `internal/backend/read.go :: Reference` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-d72ce670-e37b-41a5-a41f-20c4408e6b4a` / implementation:76 | `semantic_inspect_symbol` / `internal/backend/read.go :: OutlineResult` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-892b09d3-fd14-4924-bfa5-7477c1250727` / implementation:77 | `semantic_inspect_symbol` / `internal/backend/read.go :: ReferencesResult` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-fee017b0-787b-4e13-8cce-a5ddaee277e8` / implementation:78 | `semantic_inspect_symbol` / `internal/backend/read.go :: OutlineFile` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-1d5c04c4-f04a-4860-b272-913a712bdebc` / implementation:79 | `semantic_inspect_symbol` / `internal/backend/golang/backend.go :: convertCandidate` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-bf31965a-1f93-4757-b3e6-2d87500c7f3d` / implementation:80 | `semantic_inspect_symbol` / `internal/operation/wire_backend.go :: lookupTarget` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-18e58145-c667-4d28-a9be-10ec0526cf4a` / implementation:81 | `semantic_inspect_symbol` / `internal/operation/wire_backend.go :: lookupResponse` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-3c66ebc7-572d-42ae-a994-b56db1b9430c` / implementation:1545 | `semantic_replace_construct` / `internal/backend/golang/references_test.go :: TestRenameAmbiguityLabelsBufferedSourceView` | Applied one requested construct replacement; diagnostics unchanged. | **beneficial**: Avoided whole-function replacement; exact selector/path was accepted. |
| `exec-b4059dac-fef0-4f31-97a4-758520173292` / implementation:2015 | `semantic_lookup` / `internal/operation/wire_read_inspect.go :: outlineReadDef` | Resolved exact declaration coordinates without mutation. | **beneficial**: Supplied native identity for focused inspection; avoided manual coordinate calculation. |
| `exec-8d23ed82-f292-492a-a493-44910b4d0190` / implementation:2022 | `semantic_inspect_symbol` / `internal/operation/wire_read_inspect.go :: outlineReadDef` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-75c87f12-4d1c-4e44-afe1-a0e370446ff6` / implementation:2029 | `semantic_lookup` / `internal/operation/wire_backend.go :: lookupDef` | Resolved exact declaration coordinates without mutation. | **beneficial**: Supplied native identity for focused inspection; avoided manual coordinate calculation. |
| `exec-75047046-d563-465f-93fd-14f2e147ad9f` / implementation:2030 | `semantic_lookup` / `internal/operation/wire_read_references.go :: referencesDef` | Resolved exact declaration coordinates without mutation. | **beneficial**: Supplied native identity for focused inspection; avoided manual coordinate calculation. |
| `exec-f4bc0ead-050a-42d7-8539-660d5d14a4ac` / implementation:2039 | `semantic_inspect_symbol` / `internal/operation/wire_backend.go :: lookupDef` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-dd008bee-5b55-4288-952e-2522bfedf5ad` / implementation:2040 | `semantic_inspect_symbol` / `internal/operation/wire_read_references.go :: referencesDef` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-2efa76c1-97b3-4074-ba56-bf510bed0996` / implementation:2120 | `semantic_replace_construct` / `internal/operation/wire_read_inspect.go :: outlineReadDef` | Rejected before mutation; listed candidate paths. | **harmful**: Caller selected struct-field syntax instead of the AST expression; exact candidate path enabled retry. |
| `exec-7d41a8a8-1a26-48c3-b0b1-3ca327a31cdc` / implementation:2145 | `semantic_replace_construct` / `internal/operation/wire_read_inspect.go :: outlineReadDef` | Applied one requested construct replacement; diagnostics unchanged. | **beneficial**: Avoided whole-function replacement; exact selector/path was accepted. |
| `exec-a5562ac2-df04-4e3a-b6dc-4b3fe1a58e1c` / implementation:2152 | `semantic_replace_construct` / `internal/operation/wire_read_references.go :: referencesDef` | Applied one requested construct replacement; diagnostics unchanged. | **beneficial**: Avoided whole-function replacement; exact selector/path was accepted. |
| `exec-44468d02-2928-4aef-9b56-b97454e48d13` / implementation:2164 | `semantic_replace_construct` / `internal/operation/wire_backend.go :: lookupDef` | Rejected before mutation; returned the current candidate. | **harmful**: Caller supplied a stale construct path; retried with the reported path. |
| `exec-b5fbea39-2c99-489c-874b-effa6b4d6cfe` / implementation:2178 | `semantic_replace_construct` / `internal/operation/wire_backend.go :: lookupDef` | Applied one requested construct replacement; diagnostics unchanged. | **beneficial**: Avoided whole-function replacement; exact selector/path was accepted. |
| `exec-6c66d8fd-f8e9-462c-bf5f-e71a236fab97` / implementation:2185 | `semantic_lookup` / `internal/mcp/schema_test.go :: TestDefaultMCPCatalogDocumentsHighRiskToolBehavior` | Resolved exact declaration coordinates without mutation. | **beneficial**: Supplied native identity for focused inspection; avoided manual coordinate calculation. |
| `exec-9c04675a-58e0-4346-8b46-a67d91aa8d0e` / implementation:2192 | `semantic_inspect_symbol` / `internal/mcp/schema_test.go :: TestDefaultMCPCatalogDocumentsHighRiskToolBehavior` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-edecde24-b4c6-4276-a1aa-9a978cfe8f62` / implementation:2201 | `semantic_replace_construct` / `internal/mcp/schema_test.go :: TestDefaultMCPCatalogDocumentsHighRiskToolBehavior` | Rejected as ambiguous; both statement and expression paths were reported. | **harmful**: Exact call expression had two structural candidates; explicit statement path enabled retry. |
| `exec-80185e73-97a6-4261-84f1-125c4cf37e96` / implementation:2215 | `semantic_replace_construct` / `internal/mcp/schema_test.go :: TestDefaultMCPCatalogDocumentsHighRiskToolBehavior` | Applied one requested construct replacement; diagnostics unchanged. | **beneficial**: Avoided whole-function replacement; exact selector/path was accepted. |
| `exec-666af051-034e-4417-8a30-96708fe0d2e4` / implementation:2222 | `semantic_insert_construct` / `internal/mcp/schema_test.go :: TestDefaultMCPCatalogDocumentsHighRiskToolBehavior` | Inserted the requested catalog assertion; diagnostics unchanged. | **beneficial**: Avoided rewriting the surrounding test; later lint refactor retained the assertion. |
| `exec-67bee37d-ea1f-429b-8773-42fbe92e5b05` / implementation:2229 | `semantic_insert_construct` / `internal/mcp/schema_test.go :: TestDefaultMCPCatalogDocumentsHighRiskToolBehavior` | Inserted the requested catalog assertion; diagnostics unchanged. | **beneficial**: Avoided rewriting the surrounding test; later lint refactor retained the assertion. |
| `exec-0bb08189-ada8-4108-a264-fa8b58840e4c` / implementation:2250 | `semantic_lookup` / `internal/backend/textlsp/backend.go :: selectSymbol` | Resolved exact declaration coordinates without mutation. | **beneficial**: Supplied native identity for focused inspection; avoided manual coordinate calculation. |
| `exec-9ae74a79-3f3f-4418-99c7-3a3ff98edd83` / implementation:2257 | `semantic_inspect_symbol` / `internal/backend/textlsp/backend.go :: selectSymbol` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-d20465e4-832f-4578-af9b-0582e81bb50f` / implementation:2325 | `semantic_replace_construct` / `internal/backend/textlsp/backend.go :: selectSymbol` | Applied the requested literal change but introduced an unused declarationRange diagnostic. | **harmful**: Caller split a two-part migration; intermediate code did not compile. Guard insertion then required retries and a focused patch. |
| `exec-2bfd83de-cca3-4e94-a336-b57ca2d6eaae` / implementation:2344 | `semantic_insert_construct` / `internal/backend/textlsp/backend.go :: selectSymbol` | Insertion rejected before mutation. | **harmful**: Statement anchor not found; supplied candidate text contained an extra closing brace. Caller-side malformed anchor, not proven tool defect. |
| `exec-95c399c3-1f80-49bc-811a-4bbd4f2dbf22` / implementation:2395 | `semantic_inspect_symbol` / `internal/backend/textlsp/backend.go :: selectSymbol` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-e8a5c48e-de04-440b-9e23-1ea7afe4f6b3` / implementation:2402 | `semantic_replace_construct` / `internal/backend/textlsp/backend.go :: selectSymbol` | Replacement rejected before mutation. | **harmful**: Caller submitted multiple statements where exactly one is accepted; focused guard patch completed the change. |
| `exec-8297e06c-74ff-4f85-812d-07f0769981e3` / transport_tests:72 | `semantic_outline` / `internal/mcp/outline_presentation_test.go` | Returned declaration hierarchy without mutation. | **beneficial**: Located relevant tests/handlers without reading the entire file; older output was verbose. |
| `exec-09eebc8f-292b-4196-84b4-125aab7883ad` / transport_tests:113 | `semantic_inspect_symbol` / `internal/mcp/outline_presentation_test.go :: assertDeclarationOutlineMCPResponse` | Returned requested declaration source and revision without mutation. | **beneficial**: Provided same-record source evidence; source-bearing output is verbose, but no repair was needed. |
| `exec-028f4100-b888-4874-8393-940eec87376b` / transport_tests:312 | `semantic_outline` / `internal/mcp/server_test.go` | Returned declaration hierarchy without mutation. | **beneficial**: Located relevant tests/handlers without reading the entire file; older output was verbose. |
| `exec-e900f8fd-0cef-4c3e-93f2-9dec5fd7ea11` / transport_tests:1053 | `semantic_outline` / `internal/mcp/server.go` | Returned declaration hierarchy without mutation. | **beneficial**: Located relevant tests/handlers without reading the entire file; older output was verbose. |
| `exec-d207eb84-21b6-4fd8-b0de-5a9d1378e217` / root:181 | `semantic_workspace_outline (CLI)` / `internal` | CLI equivalent completed without mutation. | **beneficial**: Provided package/module orientation when a callable MCP workspace outline was unavailable. |
| `exec-e263d7ac-10b4-47b5-9a3c-e8ad4ed7eaa5` / root:850 | `semantic_outline (CLI)` / `internal/backend/read.go` | Filtered type outline completed without mutation. | **beneficial**: Inspected actual compact CLI output for declaration types. |
| `exec-c42774c7-fa16-4368-a9e3-7baf8fc670c4` / root:879 | `semantic_outline (CLI)` / `internal/backend/navigation.go` | Outline completed without mutation. | **beneficial**: Inspected the new shared presentation declarations. |
| `exec-ba6c7d18-3d76-4e1c-a877-a30ffdf7d224` / root:951 | `semantic_lookup (CLI)` / `Service.Lookup in internal/backend/backend.go` | Exact lookup completed without mutation. | **beneficial**: Validated receiver-qualified selector and real CLI output. |
| `exec-46192168-0b13-437a-a893-5dc584df27fa` / root:1402 | `semantic_lookup (CLI)` / `Service.Lookup in internal/backend/backend.go` | Repeated exact lookup completed without mutation. | **neutral**: Additional output check; no distinct new discovery was established. |
| `exec-62c6d61a-ac75-4302-a585-3d9be679eb03` / implementation:34 | `semantic_workspace_outline (CLI)` / `.` | CLI equivalent failed before execution; no mutation. | **harmful**: Environment: default Go build cache was sandbox-inaccessible. Later CLI calls used repository scratch caches. |

### Runtime and built-in comparison extension, 2026-10-16

This updates the same session assessment on the repeated wrap-up invocation. The inventory remains 48 attempts: all 42 completed semantic MCP items and six explicit CLI attempts were reconciled against the four retained transcripts again. No new semantic operation was invoked during either wrap-up.

The earlier QoL ratings describe observed usefulness, not measured improvement over a built-in baseline. Each attempt below names a local alternative by code. These are hypothetical workflows unless explicitly marked as actually used. Shared reads, parsers, and checks can be reused across attempts, so baseline steps cannot be summed as independent costs. Semantic discovery/setup, selector preparation, output review, retries, manual repairs, and equivalent checks also belong in a complete-workflow comparison.

Local alternatives:

* **L**: `rg -n` declaration search, file read, and receiver/overload/scope verification. At least two hypothetical retrieval steps; equivalent exact native resolution additionally needs a native parser/resolver. Raw text matches alone do not provide that guarantee.
* **I**: `rg` lookup, declaration/package/import reads, and file SHA-256 calculation kept with the captured source (at least three hypothetical steps). Manual extraction alone does not emit the typed owner continuation.
* **O**: Scoped `rg --files`, source reads, and a local native parser for nesting, extents, visibility, and filtering. Text header matching alone does not provide equivalent structural guarantees.
* **W**: Workspace/module manifest reads, scoped `go list`, and source reads for package names/synopses, with the same exclusions. Complete discovery equivalence was not established.
* **R**: Read and validate the current target, `apply_patch`, formatting, and `make check`. Equivalent AST selection and revision guards require separate native validation; an unchecked text splice is not equivalent.
* **A**: Read and validate the current anchor, `apply_patch`, formatting, and `make check`. Equivalent structural placement and stale-target protection require separate native validation.
* **V**: The pinned `make check` actually used later, plus a matching configured `gopls check` workflow for equivalent language-server scope. The historical calls failed before complete comparable coverage was obtained.

Timing boundaries: `tool` uses the retained completed item's `duration` (seconds plus nanoseconds), never a polling yield. For MCP this is reported dispatched-tool duration; `item` uses `started_at_ms` to `completed_at_ms`, including additional orchestration/delivery overhead. CLI tool duration covers the launched `go run` process through exit, including its compilation/setup. Every row has one sample; historical cache/server warmness and argument-preparation time are unrecorded. Some calls/agents overlapped, so these values are not summed as thread elapsed time or full workflow totals.

**B0** means built-in runtime unavailable: no paired equivalent measurement exists for that attempt's captured source, cache, and server state. A current replay would compare different state/timing boundaries; mutations were not replayed on the live checkout. Consequently every semantic/built-in runtime ratio is inconclusive. No broad retrospective benchmark or new semantic probe was run. Wrap-up-only transcript parsing, merge resolution, and repository checks are separate work, excluded from these original-attempt runtimes.

Efficiency codes: **U** = insufficient evidence for complete-workflow efficiency: useful semantic result/edit, but local/native-validation/setup/check costs were not measured together. **E** = observable added effort from a rejected attempt requiring retry/repair; relative efficiency against an equally guarded built-in workflow remains unproven. **F** = applied intermediate compile error followed by rejected guard attempts and manual repair. **V** = verification did not complete and added a round trip plus later pinned-check work. **C** = cache setup failed before navigation and required scratch-cache configuration. **N** = neutral repeated output validation without new discovery. These conclusions are separate from the earlier QoL ratings.

| Call | Local alternative / agent efficiency | Semantic runtime | Local built-in runtime |
| :--- | :--- | :--- | :--- |
| `exec-41c11cf7-398b-4b86-9936-7e205145d26b` | I / U | 0.062698 s tool; 2.009 s item | unavailable: B0 |
| `exec-be33e0ae-2178-43dc-b76d-acdc09f7fbca` | V / V | 31.348836 s tool; 35.048 s item | unavailable: B0 |
| `exec-e3d268ab-78cd-47a3-94ee-54ad7ecad734` | V / V | 5.954731 s tool; 8.634 s item | unavailable: B0 |
| `exec-6e7bccc7-1d84-44fd-81dd-c367151ce07a` | L / U | 0.010499 s tool; 2.316 s item | unavailable: B0 |
| `exec-92861459-5d87-4ccc-8080-223753e80257` | I / U | 0.002353 s tool; 1.334 s item | unavailable: B0 |
| `exec-372f507e-4514-4114-bc1b-264f8e7ecb9c` | I / U | 0.023161 s tool; 1.873 s item | unavailable: B0 |
| `exec-51e4abcd-e921-4274-969f-a845a0c91963` | I / U | 0.011866 s tool; 1.613 s item | unavailable: B0 |
| `exec-882708de-4ada-4202-8e75-c0e5d240064a` | I / U | 0.012126 s tool; 2.192 s item | unavailable: B0 |
| `exec-d72ce670-e37b-41a5-a41f-20c4408e6b4a` | I / U | 0.001835 s tool; 1.702 s item | unavailable: B0 |
| `exec-892b09d3-fd14-4924-bfa5-7477c1250727` | I / U | 0.001718 s tool; 1.241 s item | unavailable: B0 |
| `exec-fee017b0-787b-4e13-8cce-a5ddaee277e8` | I / U | 0.001443 s tool; 1.498 s item | unavailable: B0 |
| `exec-1d5c04c4-f04a-4860-b272-913a712bdebc` | I / U | 0.002352 s tool; 2.359 s item | unavailable: B0 |
| `exec-bf31965a-1f93-4757-b3e6-2d87500c7f3d` | I / U | 0.003724 s tool; 1.729 s item | unavailable: B0 |
| `exec-18e58145-c667-4d28-a9be-10ec0526cf4a` | I / U | 0.002880 s tool; 2.615 s item | unavailable: B0 |
| `exec-3c66ebc7-572d-42ae-a994-b56db1b9430c` | R / U | 12.248854 s tool; 15.424 s item | unavailable: B0 |
| `exec-b4059dac-fef0-4f31-97a4-758520173292` | L / U | 0.261630 s tool; 3.711 s item | unavailable: B0 |
| `exec-8d23ed82-f292-492a-a493-44910b4d0190` | I / U | 0.013735 s tool; 2.001 s item | unavailable: B0 |
| `exec-75c87f12-4d1c-4e44-afe1-a0e370446ff6` | L / U | 0.027386 s tool; 2.401 s item | unavailable: B0 |
| `exec-75047046-d563-465f-93fd-14f2e147ad9f` | L / U | 0.005429 s tool; 2.981 s item | unavailable: B0 |
| `exec-f4bc0ead-050a-42d7-8539-660d5d14a4ac` | I / U | 0.010524 s tool; 1.995 s item | unavailable: B0 |
| `exec-dd008bee-5b55-4288-952e-2522bfedf5ad` | I / U | 0.004725 s tool; 1.880 s item | unavailable: B0 |
| `exec-2efa76c1-97b3-4074-ba56-bf510bed0996` | R / E | 20.566528 s tool; 23.847 s item | unavailable: B0 |
| `exec-7d41a8a8-1a26-48c3-b0b1-3ca327a31cdc` | R / U | 17.609146 s tool; 19.957 s item | unavailable: B0 |
| `exec-a5562ac2-df04-4e3a-b6dc-4b3fe1a58e1c` | R / U | 26.687249 s tool; 29.533 s item | unavailable: B0 |
| `exec-44468d02-2928-4aef-9b56-b97454e48d13` | R / E | 5.153055 s tool; 7.503 s item | unavailable: B0 |
| `exec-b5fbea39-2c99-489c-874b-effa6b4d6cfe` | R / U | 8.610923 s tool; 10.549 s item | unavailable: B0 |
| `exec-6c66d8fd-f8e9-462c-bf5f-e71a236fab97` | L / U | 0.014187 s tool; 2.183 s item | unavailable: B0 |
| `exec-9c04675a-58e0-4346-8b46-a67d91aa8d0e` | I / U | 0.015656 s tool; 2.207 s item | unavailable: B0 |
| `exec-edecde24-b4c6-4276-a1aa-9a978cfe8f62` | R / E | 1.436838 s tool; 3.023 s item | unavailable: B0 |
| `exec-80185e73-97a6-4261-84f1-125c4cf37e96` | R / U | 2.540455 s tool; 3.961 s item | unavailable: B0 |
| `exec-666af051-034e-4417-8a30-96708fe0d2e4` | A / U | 1.840051 s tool; 3.533 s item | unavailable: B0 |
| `exec-67bee37d-ea1f-429b-8773-42fbe92e5b05` | A / U | 1.655405 s tool; 3.179 s item | unavailable: B0 |
| `exec-0bb08189-ada8-4108-a264-fa8b58840e4c` | L / U | 0.003342 s tool; 1.637 s item | unavailable: B0 |
| `exec-9ae74a79-3f3f-4418-99c7-3a3ff98edd83` | I / U | 0.013640 s tool; 1.593 s item | unavailable: B0 |
| `exec-d20465e4-832f-4578-af9b-0582e81bb50f` | R / F | 8.982852 s tool; 11.187 s item | unavailable: B0 |
| `exec-2bfd83de-cca3-4e94-a336-b57ca2d6eaae` | A / E | 1.332558 s tool; 4.142 s item | unavailable: B0 |
| `exec-95c399c3-1f80-49bc-811a-4bbd4f2dbf22` | I / U | 0.010780 s tool; 2.481 s item | unavailable: B0 |
| `exec-e8a5c48e-de04-440b-9e23-1ea7afe4f6b3` | R / E | 1.377875 s tool; 3.489 s item | unavailable: B0 |
| `exec-8297e06c-74ff-4f85-812d-07f0769981e3` | O / U | 0.005758 s tool; 2.520 s item | unavailable: B0 |
| `exec-09eebc8f-292b-4196-84b4-125aab7883ad` | I / U | 0.001913 s tool; 1.909 s item | unavailable: B0 |
| `exec-028f4100-b888-4874-8393-940eec87376b` | O / U | 0.014122 s tool; 1.675 s item | unavailable: B0 |
| `exec-e900f8fd-0cef-4c3e-93f2-9dec5fd7ea11` | O / U | 0.054261 s tool; 3.109 s item | unavailable: B0 |
| `exec-d207eb84-21b6-4fd8-b0de-5a9d1378e217` | W / U | 3.810998 s tool; 3.811 s item | unavailable: B0 |
| `exec-e263d7ac-10b4-47b5-9a3c-e8ad4ed7eaa5` | O / U | 2.420466 s tool; 2.421 s item | unavailable: B0 |
| `exec-c42774c7-fa16-4368-a9e3-7baf8fc670c4` | O / U | 7.064918 s tool; 7.065 s item | unavailable: B0 |
| `exec-ba6c7d18-3d76-4e1c-a877-a30ffdf7d224` | L / U | 2.638314 s tool; 2.638 s item | unavailable: B0 |
| `exec-46192168-0b13-437a-a893-5dc584df27fa` | L / N | 4.840163 s tool; 4.840 s item | unavailable: B0 |
| `exec-62c6d61a-ac75-4302-a585-3d9be679eb03` | W / C | 0.079091 s tool; 0.079 s item | unavailable: B0 |

Runtime conclusion: semantic attempt durations are measured, but no semantic/built-in speedup ratio or full-workflow elapsed-time improvement is supported. Agent-efficiency evidence is mixed: navigation/source evidence was useful and seven construct edits applied cleanly; five rejected mutation attempts, one compile-breaking intermediate edit, and environment failures added observable effort. The 6,985-versus-47,374-byte transport comparison is a separate payload-size result.

Integration update: the earlier dirty-main blocker cleared when `main` advanced to `4362b028a3cde1ce16a5a3cc16b3c98123a45a9c`. Its wrap-up skill and friction-log cleanup were merged into this branch. The append conflict retains both new incident sets without restoring removed historical entries. Final verification, commit, integration, cleanup, and archive evidence belong in the thread after this last log change.

### Cleanup follow-up

The implementation and session assessment were integrated as `a535efa095d269e836604bcc6d2f8df9c7af0a01`, which remains an ancestor of current `main`. Git worktree removal initially failed with `Permission denied` after deleting the linked checkout's registration. The worktree-local `.scratch/go/mod` cache cleared successfully with an explicitly scoped `GOMODCACHE=... go clean -modcache`; the wrap-up skill now requires cache cleanup before either removal path. The independent skill validator passed after installing PyYAML in an isolated temporary environment. A Git-created temporary checkout supplied valid registration metadata, and native `git worktree repair` reattached the partial checkout. Its surviving tracked files matched the integrated commit; native Git restoration recovered only the missing tracked files, including ignore rules. The recovered worktree was clean before ordinary `git worktree remove` succeeded. Its directory and registration are gone, the temporary recovery directory is gone, and the branch, commits, and prior verification artifacts remain preserved in the primary checkout. No force removal or recursive shell deletion was used. These follow-up turns made no semantic calls, so the 48-attempt inventory and runtime limitations above remain unchanged. No additional manual source edits occurred. Final verification and chat archival evidence belong in the thread rather than a post-check log edit.

### Net effect and follow-up

* Exact lookup, outlines, and source/revision inspection were useful, and seven clean construct mutations avoided larger function rewrites. The old source-bearing inspection payloads remained verbose; that operation intentionally retains source.
* Five rejected construct attempts plus one applied intermediate compile error added retries and manual repair. The returned alternatives helped recover exact paths, but malformed/stale selectors and one-statement replacement limits need clearer guidance. Do not treat caller errors as evidence that AST operations are absent.
* Verification integration was harmful overall in this run: the default cache blocked one CLI attempt, and the trusted hook could not load a plugin built by another Go runtime. The trust rejection itself enforced the intended boundary. Repository scratch caches and the pinned `make check` executable supplied usable verification. No tool/lint configuration was weakened.
* The 160-symbol transport test measured 6,985 emitted MCP bytes against 47,374 bytes for comparable native navigation records. This is byte evidence for the implementation, not measured token savings or a timing comparison of editing methods. No time/token savings are invented for individual calls.
* Manual source work used direct patches for much of the multi-file implementation, even though semantic declaration/body operations exist. This is a dogfooding workflow gap, not proof of missing capabilities. See ME-0060 through ME-0062 in `docs/MANUAL_EDITS.md`. The later Java-test brace repair fixed damaged file contents; retained evidence does not establish a semantic-tool root cause.
* Existing incidents above cover Summary-field selector resolution, the temporarily unused lookup range, and verification trust/plugin mismatch. The guard anchor also contained an extra brace; this assessment refines the earlier unknown anchor failure cause from its literal arguments. The wrap-up history retrieval failure is recorded separately above.
* Integration checkpoint: the owning primary checkout `/Users/alessandro/sources/semantic-editor` had unrelated modified `docs/SUBOPTIMAL_TOOLS.md` and untracked `.agents/skills/wrap-up` plus `skills/wrap-up/SKILL.md`. Preserve those changes; do not merge into that dirty checkout. Complete local verification/commit and recheck destination cleanliness before considering integration. Final check/commit evidence belongs in the thread, so no post-verification log edit is required.

## RQ-0069 semantic body replacement parameter mismatch (2026-10-16)

* Tool: `semantic_replace_body` through `semantic_batch`.
* Target: `internal/godiagnostics/collector_failure_test.go` in the performance candidate.
* Observed failure: the batch rejected the `function` parameter before applying edits and reported that the required selector is `symbol`; no source mutation occurred.
* Workaround: retry the same body replacement with the documented `symbol` parameter.
* Root cause: the batch caller used the wrong parameter name for the semantic tool schema.

## 2026-10-16: scaffold parameter mismatch

* Tool: `semantic_batch` with child `semantic_scaffold_file`.
* Target: `internal/astedit/move_file.go`.
* Observed failure: the first scaffold request supplied `path`, but the tool schema requires `file`; the call returned a validation error before creating or changing any file.
* Workaround: retry with the declared `file` parameter.
* Root cause: scaffold parameter naming differs from the common path terminology used by other semantic tools.

## 2026-10-16: diagnostics scaffold lost a stale gopls session

* Tool: `semantic_batch` with two `semantic_scaffold_file` children.
* Targets: `internal/mcp/diagnostic_reports.go` and `internal/mcp/diagnostic_capture.go`.
* Observed failure: the first child reported success; the second reported an error while synchronizing Go refactoring state because shutting down a stale gopls diagnostics session exceeded its context deadline. Both package-only scaffold files were present after the batch returned.
* Workaround: retain the scaffold files that were present and use targeted file edits to add their contents, then run focused package checks after concurrent edits settle.
* Root cause: the semantic batch could not replace the stale Go diagnostics session in time; earlier successful batch edits are not rolled back.

## 2026-10-16: apply_patch rejected a multi-target initialization-helper patch

* Tool: `apply_patch`.
* Target: `internal/astedit/move_file_initialization.go`.
* Observed failure: the patch combined repeated updates for the same file, so patch validation rejected it before changing source.
* Workaround: inspect the current helper and apply one focused patch per target file.
* Root cause: the patch format did not permit multiple operations targeting one path.

The follow-up correction repeated that invalid multi-operation structure and was rejected before changing source. The workaround remains focused one-file patches.

## 2026-10-16: session wrap-up, Add Go package file move

* Session: `01a145e4-3af2-7250-9b03-ebcc3d98281b`; branch `codex/go-move-file`, base `1ff9655e`, nested worktree `.scratch/worktrees/go-move-file`.
* History: normalized recorded events from the root and 9 direct child-agent transcripts. Root compaction is covered by the on-disk transcript. Records are identified below by agent, ordinal, and stable event ID. Some serialized MCP results are truncated; missing child mutation states are explicitly unknown. Discovery listings, help requests, and suggested calls are excluded. Test-suite expansions are covered by their test results rather than invented per-operation call counts. No semantic calls were made for wrap-up itself.
* Inventory: 35 rows, including 20 batch parents, 46 ordered requested batch children (not additional parent attempts), and one unavailable dispatch. Child suffixes c0… identify requests; after a failed child, later execution is unconfirmed unless retained result evidence says otherwise. Ratings by parent/direct attempt: beneficial 11, harmful 22, neutral 2.
* Integration boundary: local main initially had dirty friction-log and skill edits. The user authorized merging those edits and required rebasing and renewed verification for other changes. The session was rebased onto current main; both code paths and log histories were retained, and the permitted destination edits are reconciled before final verification.

| Call reference | Tool, target, and ordered children | Outcome and mutation evidence | Built-in alternative and agent efficiency | QoL / friction / cause |
| :--- | :--- | :--- | :--- | :--- |
| /root@139; exec-863f1e52-0340-4832-a87e-cfa8b1bfca40 | semantic_workspace_outline (CLI) ; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | beneficial: No extra repair established for this call. |
| /root@452; exec-99995fab-1983-45e2-abf5-4d27e45afef5 | semantic_batch ; c0: semantic_scaffold_file internal/operation/wire_move_file.go; c1: semantic_insert_decl internal/operation/wire_move_file.go | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@500; exec-70a178c2-2e10-4d4a-bb4e-bfdbafddf3cb | semantic_batch ; c0: semantic_replace_body internal/operation/wire_move_file.go:parseMoveFile; c1: semantic_insert_construct internal/operation/wire_engine.go:registerEngineOps; c2: semantic_insert_construct internal/operation/language_contracts.go:goOperationContract; c3: semantic_insert_decl internal/operation/language_contract_profiles.go | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Selector mismatch or unsupported nested selection; inspect partial progress and retry or use smallest fallback. |
| /root@524; exec-1cc0af5a-e9d8-4706-aca9-6b2804ea80f9 | semantic_batch ; c0: semantic_insert_construct internal/operation/language_contracts.go:goTypedOperationContract; c1: semantic_insert_decl internal/operation/language_contract_profiles.go | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@659; exec-94702748-aaca-4a8f-bf7f-ab7e6e502044 | semantic_batch ; c0: semantic_insert_construct internal/sourcebuf/buffer.go:Change; c1: semantic_replace_body internal/sourcebuf/buffer.go:Buffer.Apply; c2: semantic_insert_decl internal/sourcebuf/buffer.go | Conflict detected before stale publication; intermediate child statuses do not prove disk writes. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | beneficial: Concurrent agent edits invalidated the captured read set; coordinated retry or targeted fallback followed. Safety improved; agent effort worsened. |
| /root@688; exec-e684ea1f-33d6-40e6-8f49-53058a3a6234 | semantic_batch ; c0: semantic_insert_construct internal/sourcebuf/buffer.go:Change; c1: semantic_replace_body internal/sourcebuf/buffer.go:Buffer.Apply; c2: semantic_insert_decl internal/sourcebuf/buffer.go; c3: semantic_insert_decl internal/operation/wire_engine.go | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@750; exec-a24f364e-eb7f-402e-9ab4-65aa6078648d | semantic_batch ; c0: semantic_insert_decl internal/sourcebuf/buffer_test.go | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@775; exec-48ba7934-a6e9-4ac7-9fee-42dc6e6695b7 | semantic_batch ; c0: semantic_insert_decl internal/operation/wire_move_file.go; c1: semantic_replace_construct internal/operation/wire_move_file.go:moveFileParams; c2: semantic_replace_construct internal/operation/wire_move_file.go:parseMoveFile; c3: semantic_replace_construct internal/operation/wire_move_file.go:moveFileDef; c4: semantic_remove_construct internal/operation/language_contracts.go:goTypedOperationContract; c5: semantic_insert_construct internal/operation/language_contracts.go:goTypedOperationContract | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@805; exec-6d5f6103-b79f-4aa2-93bb-ba26a7e20285 | semantic_batch ; c0: semantic_replace_body internal/operation/wire_move_file.go:moveFileDef; c1: semantic_remove_construct internal/operation/language_contracts.go:goTypedOperationContract; c2: semantic_insert_construct internal/operation/language_contracts.go:goTypedOperationContract | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@1041; exec-bd0c2e0c-161f-44ed-9cbb-8c5eeb7d438e | semantic_batch ; c0: semantic_replace_construct internal/operation/wire_move_file.go:moveFileParams; c1: semantic_replace_construct internal/operation/skill_sync_test.go:mcpProtocolTools; c2: semantic_replace_body internal/sourcebuf/buffer.go:Buffer.Flush; c3: semantic_insert_decl internal/sourcebuf/buffer.go; c4: semantic_insert_decl internal/operation/wire_move_file.go | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@1056; exec-f92c219d-c54f-4ee2-8085-aafb132feb90 | semantic_batch ; c0: semantic_insert_decl internal/operation/wire_move_file.go; c1: semantic_replace_body internal/operation/wire_move_file.go:moveFileDef | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@1223; exec-4ab1871a-02ec-403f-b175-41446f3d846a | semantic_batch ; c0: semantic_insert_decl internal/sourcebuf/buffer_test.go | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@1416; exec-3d0e61b8-302f-4a7d-8913-89a6d45ee41d | semantic_batch ; c0: semantic_replace_body internal/sourcebuf/buffer_test.go:TestApplyCreateRejectsConflictsWithoutChangingOtherOutputs; c1: semantic_insert_decl internal/sourcebuf/buffer_test.go; c2: semantic_replace_body internal/sourcebuf/buffer_test.go:TestFlushReportsActualPartialPublication | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@1705; exec-eb1a0a25-b409-47dc-8199-32a7c1453727 | semantic_replace_construct internal/operation/wire_backend_test.go:TestRegisteredDefsHonorContracts; | Completed selected AST edit; published contents verified by final repository checks. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | neutral: No extra repair established for this call. |
| /root@1795; exec-acc87cac-8b8c-46a6-a29e-96f5d5fb47f7 | semantic_batch ; c0: semantic_insert_function internal/operation/wire_move_file.go | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@1892; exec-26975e5c-fb6a-44e4-babc-2c41db1a7227 | semantic_batch ; c0: semantic_replace_construct internal/operation/wire_move_file.go:formatMoveFile | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root@1939; exec-c609a3b1-d83c-43d2-8750-6bc4c1887fa6 | semantic_batch ; c0: semantic_replace_body internal/operation/wire_move_file.go:formatMoveFile; c1: semantic_replace_construct internal/sourcebuf/buffer.go:Buffer.validateChangePreimage | Conflict detected before stale publication; intermediate child statuses do not prove disk writes. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | beneficial: Concurrent agent edits invalidated the captured read set; coordinated retry or targeted fallback followed. Safety improved; agent effort worsened. |
| /root@2343; exec-0dfce8af-6a5e-4ea4-a266-62bce88f1882 | semantic_batch ; c0: semantic_replace_body internal/godiagnostics/diagnostics_test.go:runManagedDiagnosticsSequence | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root/implement@127; exec-1f327cb9-e6fb-4064-84d0-4299ed6c0eab | semantic_outline internal/astedit; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | harmful: Useful structure overwhelmed by directory-scale output; targeted rg plus bounded reads would reduce context. No runtime speed conclusion. |
| /root/implement@128; exec-c2cf7fb8-5e35-4164-b16a-3e2e9c2ebf2b | semantic_outline internal/operation; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | harmful: Useful structure overwhelmed by directory-scale output; targeted rg plus bounded reads would reduce context. No runtime speed conclusion. |
| /root/implement@261; exec-7727fd70-d844-4e6d-a834-87bcd4ef8423 | semantic_batch ; c0: semantic_scaffold_file internal/astedit/move_file.go | Schema refusal before requested scaffold publication. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Caller used path instead of file; retry required. |
| /root/implement@283; exec-b9addefb-92b0-4d1c-a940-3e84a40a60f2 | semantic_batch ; c0: semantic_scaffold_file internal/astedit/move_file.go | Conflict detected before stale publication; intermediate child statuses do not prove disk writes. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | beneficial: Concurrent agent edits invalidated the captured read set; coordinated retry or targeted fallback followed. Safety improved; agent effort worsened. |
| /root/sessions@52; exec-5f090032-bb82-481b-8d5b-c052147c2db1 | semantic_workspace_outline (CLI) ; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | beneficial: No extra repair established for this call. |
| /root/file_contract_tests@443; exec-a1f8359c-2d28-469f-92f5-2e322773622f | semantic_outline internal/mcp; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | harmful: Useful structure overwhelmed by directory-scale output; targeted rg plus bounded reads would reduce context. No runtime speed conclusion. |
| /root/file_contract_tests@453; exec-4c429a87-9a03-4b37-832b-019f93781a54 | semantic_lookup internal/mcp/server.go:Server; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | beneficial: No extra repair established for this call. |
| /root/file_contract_tests@470; exec-e33aedf2-f90d-451f-b966-274ca1320946 | semantic_inspect_symbol internal/mcp/server.go:Server; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | beneficial: No extra repair established for this call. |
| /root/file_contract_tests@471; exec-ebe29004-b721-4873-8fd2-0f27120852b8 | semantic_lookup internal/mcp/server.go:handleSpecialToolCall; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | beneficial: No extra repair established for this call. |
| /root/file_contract_tests@558; exec-63d21902-5198-4bdb-8569-7013b9e5b78b | semantic_inspect_symbol internal/operation/wire_engine.go:FileEditRes; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | beneficial: No extra repair established for this call. |
| /root/file_contract_tests@633; exec-a0589ad9-2a2b-48fc-bde5-ac516dfa084c | semantic_outline internal/mcp; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | harmful: Useful structure overwhelmed by directory-scale output; targeted rg plus bounded reads would reduce context. No runtime speed conclusion. |
| /root/file_contract_tests@730; exec-c10a9257-a828-49a7-b917-4fdcb8a82bcd | semantic_batch ; c0: semantic_scaffold_file internal/mcp/diagnostic_reports.go; c1: semantic_scaffold_file internal/mcp/diagnostic_capture.go | Failed; publication state cannot be inferred from failure alone. See incident entries and verified final tree. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | harmful: Hook, stale-session, or selection failure; see adjacent dated incident records for read-back evidence. Serialized result may be truncated, so individual child outcomes remain unknown where not retained. |
| /root/session_finish@39; exec-463234b0-a526-46fc-be71-9e8483dc9565 | semantic_outline internal/godiagnostics; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | harmful: Useful structure overwhelmed by directory-scale output; targeted rg plus bounded reads would reduce context. No runtime speed conclusion. |
| /root/session_finish@46; exec-66c41370-153d-414f-ab3a-c23c5df9f220 | semantic_inspect_symbol internal/godiagnostics/manager.go:syncDocuments; | Completed read-only navigation; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | beneficial: No extra repair established for this call. |
| /root/session_finish@47; exec-b75e5b8e-4544-49f7-80c9-198a15df8205 | semantic_inspect_symbol internal/godiagnostics/diagnostics_test.go:TestManagedSession; | Read-only missing-symbol refusal; no mutation. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | neutral: Caller selected nonexistent TestManagedSession; inspect actual declaration names with rg/outline. Recovered during wrap-up. |
| /root/session_finish@67; exec-4be518d4-90a6-47ef-bb02-6f5b082846bb | semantic_batch ; c0: semantic_replace_body internal/godiagnostics/manager.go:managedSession.syncDocuments | Rejected by automatic approval review before mutation. | apply_patch for selected declarations/body, gofmt/import cleanup, inspect diff, make check; checked AST selection/read-set publication is an unmatched guarantee. Estimated 2–4 steps plus checks; no measured savings. | beneficial: Caller error: empty replacement body. Complete intended implementation used afterward; no bypass of rejection. |
| /root/file_contract_tests@423; custom-call-ordinal-423 | semantic_workspace_outline (dispatch) ; | No semantic server call or mutation occurred. | rg declaration/package search, then bounded file reads; outline hierarchy needs additional parsing. Estimated 1–2 built-in calls; no replay of historical snapshot. | harmful: Environment/catalog gap: workspace outline dispatch unavailable; CLI fallback used. |

| Same call reference | Semantic runtime | Local built-in runtime | Timing scope and comparability |
| :--- | :--- | :--- | :--- |
| /root@139 | 993 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@452 | 33892 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@500 | 9560 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@524 | 9774 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@659 | 14420 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@688 | 20111 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@750 | 13566 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@775 | 19589 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@805 | 17471 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@1041 | 26819 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@1056 | 11838 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@1223 | 47173 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@1416 | 29889 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@1705 | 11879 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@1795 | 20226 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@1892 | 79172 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@1939 | 17101 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root@2343 | 80867 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/implement@127 | 2965 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/implement@128 | 2409 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/implement@261 | 2524 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/implement@283 | 55219 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/sessions@52 | 357 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/file_contract_tests@443 | 2250 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/file_contract_tests@453 | 1534 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/file_contract_tests@470 | 1411 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/file_contract_tests@471 | 1691 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/file_contract_tests@558 | 1713 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/file_contract_tests@633 | 2536 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/file_contract_tests@730 | 32018 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/session_finish@39 | 2810 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/session_finish@46 | 1565 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/session_finish@47 | 1534 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/session_finish@67 | 2073 ms | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Parent event envelope only; child durations unavailable and not assigned the parent duration. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |
| /root/file_contract_tests@423 | unavailable: dispatch had no server execution interval | unavailable: equivalent historical preimage and complete built-in workflow were not timed; no live mutation replay | Recorded item start/completion envelope. One observed call; server/cache state mixed or unknown, concurrent heavy tests present. Not a representative performance claim; no comparable ratio. |

* Net agent efficiency: the semantic route worsened effort for these incremental mutations because repeated hook failures, setup, selector retries, and publication uncertainty added round trips and log/repair work before the same required make check. Targeted inspections were useful; broad outlines were harmful to context volume. Typed selection and stale-read-set rejections provided correctness guarantees that unchecked text edits do not match. No token-saving estimate is claimed.
* Runtime conclusion: event-envelope measurements above describe observed attempts, not matched built-in comparisons. Full workflow totals (including discovery, waits, repairs and final checks) were not measured comparably; overlapping intervals must not be summed. Engine-only durations and orchestration yield times are not substituted for these envelopes. No speedup ratio is justified. Wrap-up transcript processing is separate from original operation execution and adds no benchmark samples.
* Improvements: keep output bounded at the tool boundary, expose authoritative publication state after hook failures, distinguish schema/selection errors, and coordinate whole-workspace snapshots before parallel mutations. See earlier dated incident entries for formatter hooks, concurrent buffer conflicts, schema mismatch, stale gopls scaffolds, and the rejected empty-body request. Manual fallback coverage is recorded in ME-0066.
* Rebase reconciliation: retain main's symbol-alias cleanup and target suggestions together with checked publication errors and diagnostic reports. Metadata now shares the existing `suggestionsField` constant after the combined lint check found duplicate literals. These were targeted conflict/metadata text edits within the existing manual-fallback boundary; no additional semantic operations were invoked. The standalone skill validator could not run because the available Python runtimes lack PyYAML; repository checks cover the combined tree.
* 2026-10-17: `semantic_batch` / `semantic_replace_body` on `tools/benchmark-harness/cost_test.go` applied the test refactor, but bundled `semantic_verify` failed while inspecting `tools/go-cache-root.sh` because workspace trust was required. The source mutation was independently read back; use targeted Go tests and repository checks for validation. Root cause: bundled verification expanded beyond the changed Go file into an untrusted Bash workspace scope.
