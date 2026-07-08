// ZgtStatusPlugin reports opencode session status to zgt so that
// `zgt tmux ls` and `zgt agent status` can show whether this worktree's
// opencode session is working, idle, or waiting.
//
// It shells out to `zgt agent set-status` / `zgt agent clear-status` on
// session lifecycle events. A light in-memory throttle prevents spawning
// zgt on every high-frequency message.part.updated tick while still
// refreshing the status every few seconds during long runs.
let lastStatus = "";
let lastWrite = 0;

export const ZgtStatusPlugin = async ({ directory, worktree, $ }) => {
  const cwd = worktree || directory;
  if (!cwd) return {};
  const write = async (status) => {
    const now = Date.now();
    if (status === lastStatus && now - lastWrite < 5000) return;
    lastStatus = status;
    lastWrite = now;
    try {
      await $`zgt agent set-status --agent opencode --status ${status} --cwd ${cwd}`.quiet();
    } catch (_) {
      // zgt may not be installed; silently ignore so the session is unaffected.
    }
  };
  return {
    event: async ({ event }) => {
      switch (event && event.type) {
        case "tool.execute.before":
        case "message.updated":
          return write("working");
        case "session.idle":
          return write("idle");
        case "session.deleted":
          lastStatus = "";
          try {
            await $`zgt agent clear-status --cwd ${cwd}`.quiet();
          } catch (_) {}
          break;
      }
    },
  };
};
