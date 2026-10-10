/** A response-level acceptance boundary; never executes a tool or repeats an effect. */
export function requireToolUse(message, name, validInput) {
  if (!message || !Array.isArray(message.content)) throw new Error('Invalid message content');
  const calls = message.content.filter(block => block?.type === 'tool_use');
  if (calls.length !== 1 || calls[0].name !== name) throw new Error('Required tool call missing or ambiguous');
  const call = calls[0];
  if (typeof call.id !== 'string' || !call.id || !call.input ||
      typeof call.input !== 'object' || Array.isArray(call.input) ||
      typeof validInput !== 'function' || validInput(call.input) !== true) {
    throw new Error('Invalid tool arguments');
  }
  return call;
}

export function requireTaskSuccess({toolSucceeded, taskAssertionPassed}) {
  if (toolSucceeded !== true || taskAssertionPassed !== true) throw new Error('Task acceptance failed');
  return true;
}

/**
 * At most one model-only request per configured destination. Destination allowance
 * belongs to the caller. No tool invocation has happened at this phase, so a
 * missing tool response can be retried without repeating an external effect.
 * This is not a LiteLLM callback/plugin or a completed-task verifier.
 */
export async function acceptedModelResponse(destinations, accept, {timeoutMs = 1000} = {}) {
  if (!Number.isFinite(timeoutMs) || timeoutMs <= 0 || !Array.isArray(destinations) ||
      destinations.length < 1 || destinations.length > 2) throw new Error('Invalid response budget');
  const controller = new AbortController();
  let timer;
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => {
      controller.abort();
      reject(new Error('Response deadline exceeded'));
    }, timeoutMs);
  });
  const failures = [];
  try {
    return await Promise.race([timeout, (async () => {
      for (const destination of destinations) {
        if (controller.signal.aborted) throw new Error('Response deadline exceeded');
        try {
          const response = await destination(controller.signal);
          if (controller.signal.aborted) throw new Error('Response deadline exceeded');
          return accept(response);
        } catch (error) {
          failures.push(error);
        }
      }
      throw new AggregateError(failures, 'No accepted model response');
    })()]);
  } finally {
    clearTimeout(timer);
  }
}
