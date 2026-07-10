import { test, expect } from "bun:test";
import { ZgtStatusPlugin } from "./zgt-status.js";

// Minimal mock `$` (the opencode shell helper): records the last command
// argv and resolves. `.quiet()` and `.catch()` are chainable to mirror the
// real API surface used by the plugin.
function makeMock$() {
  const calls = [];
  const mk = () => {
    const fn = (strings, ...vals) => {
      calls.push({ argv: strings, vals });
      const p = Promise.resolve();
      p.quiet = () => p;
      p.catch = () => p;
      return p;
    };
    return fn;
  };
  const $ = mk();
  $.$ = calls;
  return [$, calls];
}

function lastStatus(calls) {
  for (let i = calls.length - 1; i >= 0; i--) {
    // TemplateStringsArray is not a real array; materialize it so
    // findIndex works on every runtime.
    const argv = Array.from(calls[i].argv);
    const idx = argv.findIndex((s) => s.includes("--status"));
    if (idx >= 0) return calls[i].vals[idx];
  }
  return undefined;
}

function makeEvent(type, properties) {
  return { event: { type, properties: properties || {} } };
}

async function setup() {
  const [$, calls] = makeMock$();
  const dir = "/tmp/zgt-test-wt";
  const plugin = await ZgtStatusPlugin({ directory: dir, worktree: dir, $ });
  // Drain the init "idle" write so it doesn't pollute later assertions.
  await new Promise((r) => setTimeout(r, 0));
  return { plugin, calls, dir };
}

test("init writes idle", async () => {
  const { calls } = await setup();
  expect(lastStatus(calls)).toBe("idle");
});

test("session.created -> idle", async () => {
  const { plugin, calls } = await setup();
  const before = lastStatus([...calls]) || "idle";
  await plugin.event(makeEvent("session.created"));
  await new Promise((r) => setTimeout(r, 0));
  // Throttle skips duplicate writes within 5s; since both init and this
  // event write "idle", the recorded status must remain "idle".
  const after = lastStatus([...calls]) || before;
  expect(after).toBe("idle");
});

test("session.status busy -> working", async () => {
  const { plugin, calls } = await setup();
  calls.length = 0;
  await plugin.event(
    makeEvent("session.status", { status: { type: "busy" } })
  );
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus(calls)).toBe("working");
});

test("permission.asked -> ask (plugin docs event name)", async () => {
  const { plugin, calls } = await setup();
  calls.length = 0;
  await plugin.event(makeEvent("permission.asked"));
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus(calls)).toBe("ask");
});

test("permission.updated -> ask (SDK event name)", async () => {
  const { plugin, calls } = await setup();
  calls.length = 0;
  await plugin.event(makeEvent("permission.updated"));
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus(calls)).toBe("ask");
});

test("permission.replied -> working (after ask)", async () => {
  const { plugin, calls } = await setup();
  calls.length = 0;
  await plugin.event(makeEvent("permission.asked"));
  await new Promise((r) => setTimeout(r, 0));
  await plugin.event(makeEvent("permission.replied"));
  await new Promise((r) => setTimeout(r, 0));
  // last written status should be working
  expect(lastStatus(calls)).toBe("working");
});

test("question tool pending -> ask; completed -> working", async () => {
  const { plugin, calls } = await setup();
  calls.length = 0;
  await plugin.event(
    makeEvent("message.part.updated", {
      part: { type: "tool", tool: "question", state: { status: "pending" } },
    })
  );
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus(calls)).toBe("ask");
  await plugin.event(
    makeEvent("message.part.updated", {
      part: { type: "tool", tool: "question", state: { status: "completed" } },
    })
  );
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus(calls)).toBe("working");
});

test("session.idle -> idle (and late message.updated ignored)", async () => {
  const { plugin, calls } = await setup();
  await plugin.event(makeEvent("session.idle"));
  await new Promise((r) => setTimeout(r, 0));
  // Throttle skips duplicate idle writes within 5s; stays "idle".
  expect(lastStatus([...calls]) || "idle").toBe("idle");
  // A late assistant message.updated should NOT clobber idle back to working.
  await plugin.event(makeEvent("message.updated", { info: { role: "assistant" } }));
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus([...calls]) || "idle").toBe("idle");
  // A late user message.updated within the debounce window should NOT
  // clobber idle back to working.
  await plugin.event(makeEvent("message.updated", { info: { role: "user" } }));
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus([...calls]) || "idle").toBe("idle");
});

test("session.status busy after idle -> working (new turn)", async () => {
  const { plugin, calls } = await setup();
  await plugin.event(makeEvent("session.idle"));
  await new Promise((r) => setTimeout(r, 0));
  calls.length = 0;
  // A new turn is signaled by session.status:busy, not message.updated.
  await plugin.event(
    makeEvent("session.status", { status: { type: "busy" } })
  );
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus(calls)).toBe("working");
});

test("session.status retry -> working", async () => {
  const { plugin, calls } = await setup();
  calls.length = 0;
  await plugin.event(
    makeEvent("session.status", { status: { type: "retry", attempt: 1, message: "rate limit", next: 0 } })
  );
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus(calls)).toBe("working");
});

test("message.updated during active turn -> working", async () => {
  const { plugin, calls } = await setup();
  // Start a turn via session.status:busy (writes "working").
  await plugin.event(makeEvent("session.status", { status: { type: "busy" } }));
  await new Promise((r) => setTimeout(r, 0));
  // Any message.updated (user or assistant) during an active turn → working.
  // The 5s throttle suppresses the duplicate "working" write, so the
  // last recorded status is still "working" from session.status:busy.
  await plugin.event(makeEvent("message.updated", { info: { role: "assistant" } }));
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus(calls)).toBe("working");
});

test("user message.updated after debounce -> working (fallback)", async () => {
  const { plugin, calls } = await setup();
  await plugin.event(makeEvent("session.idle"));
  await new Promise((r) => setTimeout(r, 0));
  calls.length = 0;
  // Wait beyond the 500ms debounce window so a user message.updated
  // is treated as a genuine new-turn signal.
  await new Promise((r) => setTimeout(r, 550));
  await plugin.event(makeEvent("message.updated", { info: { role: "user" } }));
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus(calls)).toBe("working");
});

test("session.error -> idle (and late message.updated ignored)", async () => {
  const { plugin, calls } = await setup();
  await plugin.event(makeEvent("session.status", { status: { type: "busy" } }));
  await new Promise((r) => setTimeout(r, 0));
  calls.length = 0;
  await plugin.event(makeEvent("session.error"));
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus(calls)).toBe("idle");
  // A late message.updated after session.error should not clobber idle.
  await plugin.event(makeEvent("message.updated", { info: { role: "user" } }));
  await new Promise((r) => setTimeout(r, 0));
  expect(lastStatus([...calls]) || "idle").toBe("idle");
});

test("permission.replied after session.idle is ignored", async () => {
  const { plugin, calls } = await setup();
  await plugin.event(makeEvent("session.idle"));
  await new Promise((r) => setTimeout(r, 0));
  const beforeCount = calls.length;
  // A late permission.replied within the debounce window should NOT
  // clobber idle back to working.
  await plugin.event(makeEvent("permission.replied"));
  await new Promise((r) => setTimeout(r, 0));
  expect(calls.length).toBe(beforeCount);
  expect(lastStatus(calls) || "idle").toBe("idle");
});

test("question tool completed after session.idle is ignored", async () => {
  const { plugin, calls } = await setup();
  await plugin.event(makeEvent("session.idle"));
  await new Promise((r) => setTimeout(r, 0));
  const beforeCount = calls.length;
  // A late question-tool completed within the debounce window should
  // NOT clobber idle back to working.
  await plugin.event(
    makeEvent("message.part.updated", {
      part: { type: "tool", tool: "question", state: { status: "completed" } },
    })
  );
  await new Promise((r) => setTimeout(r, 0));
  expect(calls.length).toBe(beforeCount);
  expect(lastStatus(calls) || "idle").toBe("idle");
});

test("session.deleted clears the record", async () => {
  const { plugin, calls } = await setup();
  calls.length = 0;
  await plugin.event(makeEvent("session.deleted"));
  await new Promise((r) => setTimeout(r, 0));
  const cleared = calls.some(
    (c) => c.argv.some((s) => s.includes("clear-status"))
  );
  expect(cleared).toBe(true);
});