import test from 'node:test';
import assert from 'node:assert/strict';
import {requireToolUse, requireTaskSuccess, acceptedModelResponse} from './tool-result.mjs';

const valid = {content: [{type: 'tool_use', id: 'call-1', name: 'write_file', input: {path: 'demo.txt', text: 'ok'}}]};
const accept = response => requireToolUse(response, 'write_file', input =>
  input.path === 'demo.txt' && typeof input.text === 'string' && Object.keys(input).length === 2);

test('200 with prose is not accepted as a required tool call', () => {
  assert.throws(() => accept({content: [{type: 'text', text: 'I would edit it.'}]}), /missing/);
  assert.throws(() => accept({}), /Invalid message/);
});
test('argument and duplicate-call failures remain visible', () => {
  assert.throws(() => accept({content: [{...valid.content[0], input: {path: '../secret', text: 'x'}}]}), /arguments/);
  assert.throws(() => accept({content: [valid.content[0], valid.content[0]]}), /ambiguous/);
  assert.throws(() => requireToolUse(valid, 'write_file', async () => true), /arguments/);
  assert.throws(() => requireToolUse(valid, 'write_file', () => 'yes'), /arguments/);
});
test('accepted model output does not mark a failed tool/task complete', () => {
  assert.equal(accept(valid).id, 'call-1');
  assert.throws(() => requireTaskSuccess({toolSucceeded: false, taskAssertionPassed: true}), /acceptance/);
  assert.throws(() => requireTaskSuccess({toolSucceeded: true, taskAssertionPassed: false}), /acceptance/);
  assert.equal(requireTaskSuccess({toolSucceeded: true, taskAssertionPassed: true}), true);
});
test('provider error and semantic refusal both reach only configured alternatives', async () => {
  for (const primary of [() => {throw new Error('503');}, () => ({content: [{type: 'text', text: 'no tool'}]})]) {
    let attempts = 0;
    const output = await acceptedModelResponse([
      async () => {attempts++; return primary();}, async () => {attempts++; return valid;},
    ], accept);
    assert.equal(output.id, 'call-1'); assert.equal(attempts, 2);
  }
  await assert.rejects(acceptedModelResponse([async () => ({content: []})], accept), /No accepted/);
});
test('shared deadline prevents a hanging primary from starting a fallback later', async () => {
  let fallback = false, signal;
  await assert.rejects(acceptedModelResponse([
    async value => {signal = value; return new Promise(() => {});},
    async () => {fallback = true; return valid;},
  ], accept, {timeoutMs: 20}), /deadline/);
  assert.equal(signal.aborted, true); assert.equal(fallback, false);
});
