// ZgtStatusPlugin reports opencode session status to zgt so that
// `zgt tmux ls` and `zgt agent status` can show whether this worktree's
// opencode session is working, idle, or asking for permission/input.
//
// It shells out to `zgt agent set-status` / `zgt agent clear-status` on
// session lifecycle events. A light in-memory throttle prevents spawning
// zgt on every high-frequency message.updated tick while still refreshing
// the status every few seconds during long runs.
//
// "ask" is reported in two situations:
//   - a permission prompt (permission.asked event)
//   - the built-in `question` tool is invoked (e.g. plan / implementation
//     choice confirmation), detected via the tool.execute.before hook.
//
// "idle" is reported when the session goes idle (session.idle or
// session.status with type=idle) and is protected from late-arriving
// message.updated events that would otherwise clobber it back to
// "working". On plugin init (agent restart) the status is reset to
// "idle" so a stale "working" from a previous/crashed session doesn't
// linger on disk; a "low watermark" timestamp ensures a just-queued
// "working" event is never overwritten by the init idle write.
let lastStatus = "";
let lastWrite = 0;

export const ZgtStatusPlugin = async ({ directory, worktree, $ }) => {
  const cwd = worktree || directory;
  if (!cwd) return {};
  // awaitingUser stays true while the session is blocked on user input
  // (either a permission prompt or the built-in `question` tool), so
  // high-frequency message.updated events don't flip the status back to
  // "working" prematurely.
  let awaitingUser = false;
  // sessionIdle stays true after the session goes idle until a new user
  // message or tool call restarts a turn. This prevents late-arriving
  // message.updated (assistant message finalization) from clobbering
  // "idle" back to "working".
  let sessionIdle = false;
  // writeQueue serializes subprocess invocations so that later writes
  // never land on disk before earlier ones, even across concurrent event
  // ticks.
  let writeQueue = Promise.resolve();
  // lowWatermark is the timestamp of the most recent non-idle write
  // (working/ask). The init reset only writes "idle" if no working/ask
  // write has been queued after it, so a resumed session that immediately
  // reports busy isn't clobbered back to idle.
  let lowWatermark = 0;

  const write = (status) => {
    const now = Date.now();
    if (status === lastStatus && now - lastWrite < 5000) return Promise.resolve();
    lastStatus = status;
    lastWrite = now;
    if (status !== "idle") lowWatermark = now;
    const p = writeQueue.then(() =>
      $`zgt agent set-status --agent opencode --status ${status} --cwd ${cwd}`.quiet().catch(() => {})
    );
    writeQueue = p;
    return p;
  };

  const clear = () => {
    lastStatus = "";
    const p = writeQueue.then(() =>
      $`zgt agent clear-status --cwd ${cwd}`.quiet().catch(() => {})
    );
    writeQueue = p;
    return p;
  };

  // On plugin init (agent restart), reset to idle so a stale "working"
  // from a previous/crashed session doesn't linger on disk. The write is
  // queued (not awaited) so plugin load isn't blocked. If a real
  // working/ask event arrives before this init write flushes, that event
  // bumps lowWatermark and we cancel the idle write below.
  const initStartedAt = Date.now();
  sessionIdle = true;
  const initP = writeQueue.then(() => {
    if (lowWatermark > initStartedAt) return; // a working/ask event won; skip idle
    return write("idle");
  });
  writeQueue = initP;

  return {
    // Named hooks receive (input, output) with input.tool available, which
    // is more reliable than inspecting event payloads. write() is queued
    // so subprocess invocations stay ordered across concurrent hooks.
    "tool.execute.before": async (input) => {
      if (input && input.tool === "question") {
        awaitingUser = true;
        sessionIdle = false;
        await write("ask");
      } else {
        awaitingUser = false;
        sessionIdle = false;
        await write("working");
      }
    },
    "tool.execute.after": async (input) => {
      if (input && input.tool === "question") {
        awaitingUser = false;
        sessionIdle = false;
        await write("working");
      }
    },
    event: async ({ event }) => {
      switch (event && event.type) {
        case "session.created":
          // Only reset to idle if no working/ask event has been seen
          // since init. This avoids clobbering a resumed session that
          // immediately reports busy.
          if (lowWatermark > initStartedAt) return;
          sessionIdle = true;
          awaitingUser = false;
          return write("idle");
        case "session.status": {
          const t = event.properties && event.properties.status && event.properties.status.type;
          if (t === "busy" || t === "retry") {
            sessionIdle = false;
            if (!awaitingUser) return write("working");
          } else if (t === "idle") {
            sessionIdle = true;
            awaitingUser = false;
            return write("idle");
          }
          return;
        }
        case "session.idle":
          sessionIdle = true;
          awaitingUser = false;
          return write("idle");
        case "message.updated": {
          if (awaitingUser) return;
          const info = event.properties && event.properties.info;
          // A user message marks the start of a new turn: leave idle and
          // flip to working. An assistant message arriving while idle is
          // a late finalization event and should be ignored.
          if (info && info.role === "user") {
            sessionIdle = false;
            return write("working");
          }
          if (sessionIdle) return;
          return write("working");
        }
        case "permission.asked":
          awaitingUser = true;
          sessionIdle = false;
          return write("ask");
        case "permission.replied":
          awaitingUser = false;
          sessionIdle = false;
          return write("working");
        case "session.compacted":
          // Compaction is a maintenance event; the session continues.
          // Reset transient flags so status updates resume normally.
          awaitingUser = false;
          return;
        case "session.error":
          // The session errored; flip to idle so a stale "working" badge
          // doesn't linger until the 1h stale prune.
          awaitingUser = false;
          sessionIdle = false;
          return write("idle");
        case "session.deleted":
          awaitingUser = false;
          sessionIdle = false;
          return clear();
      }
    },
  };
};
