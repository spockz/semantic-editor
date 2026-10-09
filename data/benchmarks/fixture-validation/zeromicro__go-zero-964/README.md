# Explicit recipe validation: go-zero-964

The archived prompt is in `explicit.md`. It asks for one `semantic_batch` with 17 ordered child operations and explicit parameters. The authoritative batch request, response, and per-child outcomes are saved alongside it. The successful clean replay returned `status=ok`, all 17 child outcomes succeeded, and the source changes touched five Go files.

## Acceptance fixture integrity

The original private test patch and human reference patch are preserved as `original-test.patch` and `original-fix.patch`. Their SHA-256 hashes are:

- `original-test.patch`: `dce1f0e0a007aef02fb5ea0a4223591f45dc61cb6931615bb99816f66d2ef505`
- `original-fix.patch`: `0445d581eb4a9f6bb863ec5489cd910961ffb5124776ca27838fd75ac08aa383`

The original acceptance test called `UnaryCrashInterceptor()` even though the symbol is a four-argument interceptor function. It also imported the legacy `github.com/agiledragon/gomonkey` package, which fails to compile on arm64 (`undefined: buildJmpDirective`). The original test patch did not declare its gomonkey dependency; those dependency hunks were only in the human fix patch. The original failures are captured in `native-arm64-uncorrected-deps.log`.

The corrected private test patch passes the interceptor function value, pins the test-only dependency `github.com/agiledragon/gomonkey/v2 v2.14.3`, and carries the required Go module checksums. The human fix patch no longer adds test-only module dependencies. `corrected-test.patch` and `corrected-fix.patch` preserve the final patches. Their hashes are:

- `corrected-test.patch`: `8653659afc05f22fa860b9dda8b01f4f6e1fc337d61fbafce500bcd9657d2b66`
- `corrected-fix.patch`: `ea29a444c6ba2cb2b7d7b1a782f4870c1c088c1bd9c1727d650e7f0bcf00cfac`

The test patch and human fix patch both apply to a fresh copy of the pinned base. No production dependency was added to the semantic recipe.

## Frozen tools and fixture identity

- Fixture SHA-256: `05afb1868456ec2ad0eb654da2b4303a8ae53729aac586ee3fe6dbc465710175`
- Frozen semedit SHA-256: `443e097ae55ce7649c6f46202aba9b9e6fa5dffa96cf5a6a6ba782d156db9253`
- Frozen benchmark-v2 SHA-256: `64194c02cc39c224056182ae4a8f1c3f352982a691e5222f8aaf8f278d9b7b9b`
- Corrected test patch SHA-256: `8653659afc05f22fa860b9dda8b01f4f6e1fc337d61fbafce500bcd9657d2b66`

## Oracle evidence

The repaired suite passed natively on darwin/arm64 with all assertions retained. Gomonkey's documented no-inlining condition is represented directly in the fixture's test command as `-gcflags=all=-l`.

The ordinary benchmark oracle was run with:

```sh
GOPROXY=file:///Users/alessandro/sources/semantic-editor/.scratch/go/mod/cache/download \
GOSUMDB=off \
.scratch/explicit-recipes/benchmark-20261008-v2 \
  --task zeromicro__go-zero-964 \
  --eval-dir .scratch/explicit-recipes/go-zero-964/replay-clean
```

Result: `Oracle passed for zeromicro__go-zero-964` with policy, AST, build, and test levels all OK. The full log is `benchmark-oracle.log`; native repaired package output is `native-arm64-repaired.log`. The fixture test metadata itself supplies `-gcflags=all=-l`; no environment-only Go flag workaround was used for the benchmark oracle.

The harness sets an explicit Go cache base for its isolated oracle copy. The local file proxy supplied the pinned module graph without direct network access. The shared Go cache problem observed before this setup was caused by `internal/mcp` explicitly setting `gocache.WithBaseDir`, which ignores inherited `GOMODCACHE`.
