# OpenRouter benchmark targets

The benchmark harness includes a dated, coding-oriented top-ten snapshot from
the [OpenRouter free-model collection](https://openrouter.ai/collections/free-models/).
The snapshot keeps explicit OpenRouter model IDs in
`tools/benchmark-harness/openrouter.go` so a changed upstream catalog cannot
silently change an experiment. Inspect the current pinned entries with:

```sh
go run ./tools/benchmark-harness --list-openrouter-free
```

Use the complete snapshot as a matrix target, or select one model explicitly:

```sh
OPENROUTER_API_KEY="$OPENROUTER_API_KEY" \
go run ./tools/benchmark-harness --matrix \
  --target opencode/openrouter/free-top10 \
  --task task-01-rename-local --variants small --repeats 2 \
  --run-id openrouter-free-task01-r2
```

An explicit model target has the form
`opencode/openrouter/<provider>/<model>:free`. OpenCode receives
`OPENROUTER_API_KEY` through its child-process environment. The generated
fixture configuration contains only an environment reference, and the key is
redacted from captured OpenCode diagnostics.

Every repeat is recorded with a `repeat` field. Per-target reports use a
`-repeat-N` suffix, and comparison grouping includes the repeat index, so
independent trials remain separate.
