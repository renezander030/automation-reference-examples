#!/usr/bin/env bash
set -euo pipefail
repo_root=$(cd "$(dirname "$0")" && pwd)
test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' EXIT
mkdir "$test_dir/bin"
cat > "$test_dir/bin/curl" <<'MOCK'
#!/usr/bin/env bash
if [ "${MOCK_EXISTING:-0}" = 1 ] || [ -f "$MOCK_DIR/started" ]; then exit 0; fi
exit 7
MOCK
cat > "$test_dir/bin/ollama" <<'MOCK'
#!/usr/bin/env bash
printf '%s\n' "$OLLAMA_CONTEXT_LENGTH" > "$MOCK_DIR/context"
touch "$MOCK_DIR/started"
trap 'exit 0' TERM
while :; do sleep 0.1; done
MOCK
cat > "$test_dir/bin/claude" <<'MOCK'
#!/usr/bin/env bash
[ "$ANTHROPIC_API_KEY" = '' ]
[ "$ANTHROPIC_DEFAULT_OPUS_MODEL" = 'served-id' ]
printf '%s\n' "$*" > "$MOCK_DIR/args"
MOCK
chmod +x "$test_dir/bin/"*
export MOCK_DIR="$test_dir" PATH="$test_dir/bin:$PATH" LOCAL_MODEL=served-id ANTHROPIC_API_KEY=conflicting
bash "$repo_root/start-claude-code-local.sh" -p test
[ "$(cat "$test_dir/context")" = 64000 ]
[ "$(cat "$test_dir/args")" = '--model served-id -p test' ]
rm "$test_dir/started"
export MOCK_EXISTING=1
if bash "$repo_root/start-claude-code-local.sh" > /dev/null 2>&1; then echo 'failed: existing server reused' >&2; exit 1; fi
printf '%s\n' 'startup ownership, context, credentials and existing-server refusal: PASS'
