# Stop silent failures in small automation helpers

Executable Go, Node and Python examples behind the corrected [Production AI Automation Notes index](https://gist.github.com/renezander030/9069db775e494ffd2cdd5a09adf83add). Each package demonstrates a narrow contract and its failure fixtures. These examples are independent of a production service; no provider credentials are required to test them.

## Run the checks

Requirements: Go 1.24+, Node 22+, Python 3.13+, Bash 4+, Linux/Unix for flock. Go dependencies are pinned in go.mod/go.sum. Tests use temporary files and localhost endpoints. They never send external messages, invoke an attack command or contact paid inference APIs.

```bash
go test -race ./...
node --test node/regression.test.mjs
python3 -m unittest discover -s tests -v
```

## Contracts and limits

| Example | What the fixtures check | Limits |
|---|---|---|
| node/serve.mjs | JSON remains argv, invalid jobs/child errors fail the batch | Fixed operator CLI/allowlist; destination permissions remain separate |
| python/drain.py | Immutable batches, durable completed IDs, uncertain starts held | One host; no automatic replay of ambiguous external effects |
| python/exec_guard.py | Malformed input blocks, DNS patterns, package commands, canonical cwd | Pattern tripwire, not a shell parser or sandbox |
| python/approval.py | Action/payload/target/version/expiry binding, revocation, insert failure | Local authorization/outbox, not proof of external delivery; provider CAS needed |
| python/budget.py | Atomic concurrent token/currency reservations and settlement | Caller must bound all billed components; not integrated into draftcat |
| node/retrieval.mjs | Hung/rejected branches settle by deadlines; signals cancel | Ignored cancellation may leave work running; semantic-only results may be lost |
| python/acl.py | Current permission checks, revoked provenance and visible counts | Remote permission/load atomicity requires provider snapshots |
| python/embedding_width.py | Actual vector width and malformed-shape rejection | Ollama endpoint; equal width is not equal vector space |
| shell/ndjson-deploy.sh | Quotes/newlines are valid JSON; monotonic durations | Source as Bash functions; service list is whitespace-delimited |
| shell/health-check-cron.sh | Checks during cooldown, immediate recovery, unknown status unhealthy | Private operator-owned state and endpoint configuration |
| go/* | Rate capacity, parse failures, cancellation, bounded handlers, publisher failures, UTC keys | See each package's source; helper guarantees do not imply durable delivery |

Redis Pub/Sub is at-most-once. A retained failed batch can duplicate if a failed command reached Redis. The publisher reports errors; it does not create durable retries. Redis quota tests cover validation/UTC keys; a live Redis deployment is not exercised by the offline suite.

SQLite FULL relies on a filesystem honoring sync/locking. Unit tests do not simulate a hardware power cut. An approval/outbox row is distinct from an external effect, and a caller-attested outcome is distinct from independently observed execution. Keep uncertain outcomes until reconciled.

## Related notes

- [JSONL CLI runners and immutable batches](https://gist.github.com/renezander030/807559488f523892fc25870bf9501d29)
- [Claude Code PreToolUse tripwire](https://gist.github.com/renezander030/a6761638d44a08748cfb45cd61bfa6e4)
- [Action-bound approval evidence](https://gist.github.com/renezander030/ad81c7a805a09a844983f881e2c487e5)
- [Bounded retrieval branches](https://gist.github.com/renezander030/77b1e95ae3a7b4460db0714b7dcf35d6)
- [Permission-aware retrieval](https://gist.github.com/renezander030/4c97bc473db055f2aa6d71be4c4551ce)

## Contributing

Include a small failing input, runtime/version and desired contract. Use synthetic data; remove credentials and customer records. Follow [@renezander030](https://github.com/renezander030) for the accompanying tested notes. MIT licensed.

## Local Claude startup helper

The [startup helper](local/start-claude-code-local.sh) and [mock process-ownership tests](local/startup.test.sh) maintain the corrected example from the archived local-ai-coding-stack repository. It never downloads a model, requires LOCAL_MODEL, clears a conflicting Anthropic API key and refuses to reuse an existing server with unknown context. Run `bash local/startup.test.sh`. No successful fresh hardware coding benchmark is claimed.
