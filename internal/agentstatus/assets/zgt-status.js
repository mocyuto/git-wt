// ZgtStatusPlugin reports opencode session status to zgt so that
// `zgt tmux ls` and `zgt agent status` can show whether this worktree's
// opencode session is working, idle, or asking for permission/input.
//
// It shells out to `zgt agent set-status` / `zgt agent clear-status` on
// session lifecycle events. A light in-memory throttle prevents spawning
// zgt on every high-frequency message.part.updated tick while still
// refreshing the status every few seconds during long runs.
//
// "ask" is reported in two situations:
//   - a permission prompt. opencode emits either `permission.asked` (per the
//     plugin docs) or `permission.updated` (per the SDK EventListResponse
//     union), depending on version; both are handled so the status flips to
//     ask regardless of which opencode release is installed.
//   - the built-in `question` tool is invoked (e.g. plan / implementation
//     choice confirmation), detected via message.part.updated when the
//     tool part has tool === "question" and state.status is pending/running.
//
// "idle" is reported when the session goes idle (session.idle or
// session.status with type=idle) and is protected from late-arriving
// message.updated events that would otherwise clobber it back to
// "working". The primary turn-start signal is session.status:busy/retry,
// but a user message.updated arriving more than idleDebounceMs after
// idle is also treated as a new turn (fallback for versions/flows
// where session.status:busy might not fire). On plugin init (agent
// restart) the status is reset to "idle" so a stale "working" from a
// previous/crashed session doesn't linger on disk.
const idleDebounceMs = 500;

let lastStatus = "";
let lastWrite = 0;

export const ZgtStatusPlugin = async ({ directory, worktree, $ }) => {
  const cwd = worktree || directory;
  if (!cwd) return {};
  let awaitingUser = false;
  let sessionIdle = false;
  let idleSince = 0;
  let writeQueue = Promise.resolve();

  const write = (status) => {
    const now = Date.now();
    if (status === lastStatus && now - lastWrite < 5000) return Promise.resolve();
    lastStatus = status;
    lastWrite = now;
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
  // from a previous/crashed session doesn't linger on disk.
  sessionIdle = true;
  idleSince = Date.now();
  write("idle");

  return {
    event: async ({ event }) => {
      if (!event || !event.type) return;
      switch (event.type) {
        // --- Tool part updates: detect the question tool ---
        case "message.part.updated": {
          const part = event.properties && event.properties.part;
          if (!part || part.type !== "tool") return;
          if (part.tool === "question") {
            const st = part.state && part.state.status;
            if (st === "pending" || st === "running") {
              awaitingUser = true;
              sessionIdle = false;
              return write("ask");
            }
            if (st === "completed" || st === "error") {
              awaitingUser = false;
              sessionIdle = false;
              return write("working");
            }
          }
          return;
        }

        // --- Message updates: track turn transitions ---
        case "message.updated": {
          if (awaitingUser) return;
          if (sessionIdle) {
            // After session.idle, late message.updated events can arrive
            // and must not clobber the idle status. A user message after
            // the debounce window is treated as a genuine new-turn signal
            // (fallback when session.status:busy doesn't fire). Assistant
            // messages and anything within the window are ignored.
            const info = event.properties && event.properties.info;
            if (info && info.role === "user" && Date.now() - idleSince > idleDebounceMs) {
              sessionIdle = false;
              return write("working");
            }
            return;
          }
          return write("working");
        }

        // --- Permission events ---
        // opencode emits `permission.asked` (plugin docs) or
        // `permission.updated` (SDK EventListResponse) depending on
        // version; accept both so the badge flips to `ask` regardless.
        case "permission.asked":
        case "permission.updated":
          awaitingUser = true;
          sessionIdle = false;
          return write("ask");
        case "permission.replied":
          awaitingUser = false;
          sessionIdle = false;
          return write("working");

        // --- Session lifecycle ---
        case "session.created":
          sessionIdle = true;
          idleSince = Date.now();
          awaitingUser = false;
          return write("idle");
        case "session.status": {
          const t = event.properties && event.properties.status && event.properties.status.type;
          if (t === "busy" || t === "retry") {
            sessionIdle = false;
            if (!awaitingUser) return write("working");
          } else if (t === "idle") {
            sessionIdle = true;
            idleSince = Date.now();
            awaitingUser = false;
            return write("idle");
          }
          return;
        }
        case "session.idle":
          sessionIdle = true;
          idleSince = Date.now();
          awaitingUser = false;
          return write("idle");
        case "session.compacted":
          awaitingUser = false;
          return;
        case "session.error":
          awaitingUser = false;
          sessionIdle = true;
          idleSince = Date.now();
          return write("idle");
        case "session.deleted":
          awaitingUser = false;
          sessionIdle = false;
          return clear();
      }
    },
  };
};
