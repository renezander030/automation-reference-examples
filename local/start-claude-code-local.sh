#!/usr/bin/env bash
# Own a fresh local Ollama server. Refuse to reuse a server whose context is unknown.
# Model choice is explicit; this helper never downloads a model.
set -euo pipefail
export OLLAMA_CONTEXT_LENGTH="${OLLAMA_CONTEXT_LENGTH:-64000}"
export ANTHROPIC_BASE_URL="${ANTHROPIC_BASE_URL:-http://localhost:11434}"
export ANTHROPIC_AUTH_TOKEN=ollama
export ANTHROPIC_API_KEY=''
: "${LOCAL_MODEL:?Set LOCAL_MODEL to a model already installed by ollama}"
case "$ANTHROPIC_BASE_URL" in http://localhost:11434|http://127.0.0.1:11434) ;; *) echo 'This helper owns a localhost:11434 server only.' >&2; exit 2;; esac
if curl -fsS --max-time 2 "$ANTHROPIC_BASE_URL/api/version" >/dev/null 2>&1; then
 echo 'Ollama is already running. Configure that serving process, verify ollama ps, and launch Claude manually.' >&2; exit 2
fi
ollama serve &
runner_ollama_pid=$!
cleanup() { kill "$runner_ollama_pid" 2>/dev/null || true; wait "$runner_ollama_pid" 2>/dev/null || true; }
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
ready=0
for _ in {1..60}; do
 if ! kill -0 "$runner_ollama_pid" 2>/dev/null; then echo 'Ollama failed to start.' >&2; exit 1; fi
 if curl -fsS --max-time 2 "$ANTHROPIC_BASE_URL/api/version" >/dev/null; then ready=1; break; fi
 sleep 0.5
done
if [ "$ready" -ne 1 ]; then echo 'Ollama readiness deadline exceeded.' >&2; exit 1; fi
# Request uses the same actual served ID for each Claude model alias.
export ANTHROPIC_DEFAULT_OPUS_MODEL="$LOCAL_MODEL"
export ANTHROPIC_DEFAULT_SONNET_MODEL="$LOCAL_MODEL"
export ANTHROPIC_DEFAULT_HAIKU_MODEL="$LOCAL_MODEL"
claude --model "$LOCAL_MODEL" "$@"
