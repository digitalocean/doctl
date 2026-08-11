# Codex TUI Parity Plan — Raw Request Relay

Goal: get the hosted codex TUI (via `doctl agents start-proxy`) working like the
native codex CLI, building on the shipped raw event passthrough work
(v1.5 outbound, v2 inbound, unmapped-event catch-all).

Repos / branches:

- `doctl` — `juliaye/codex-proxy-raw`
- `cthulhu` (hosted-agents) — `juliaye/codex-proxy`
- `plano` — `juliaye/codex-prxy`

Background: the codex app-server protocol distinguishes **notifications**
(fire-and-forget, no `id`) from **requests** (carry an `id`, block until
answered). Raw passthrough already covers notifications outbound and
`turn/start` params inbound. Requests need explicit plumbing because they
create a reply obligation scoped to a single JSON-RPC session — this plan adds
that plumbing.

---

## M0 — Unblock testing (half day)

Fixes for the capability-mismatch bug found in manual E2E (2026-08-07):
`turn/start.runtimeWorkspaceRoots requires experimentalApi capability` →
instant `run.failed` that never reached the TUI (eternal spinner).

1. **plano**: adapter declares `"experimentalApi": true` in its `initialize`
   handshake with the in-sandbox codex app-server (`agent_adapters/src/codex/mod.rs`,
   capabilities block). One line; prerequisite for everything after.
2. **plano**: if a raw-template `turn/start` is rejected by the app-server,
   retry once with params synthesized from text before failing the run
   (capability mismatch degrades to v1 behavior instead of killing the turn).
3. **doctl**: facade treats `run.failed` as turn-ending even when its run ID
   doesn't match the tracked turn (fail-safe close), so the TUI gets
   `turn/completed(failed)` instead of spinning forever.

**Exit test:** "Create hello.txt containing 'hello world', then edit it to say
'goodbye world'" completes in the TUI with real diff stats (+1 −1, no hang).

---

## M1 — Inbound request relay — code complete, pending manual E2E

Generic client→VM request pipe. Mechanically repeats the v2 send-input
threading, plus a response path (raw bytes flowing back up).

- **cthulhu**: `POST /v2/agents/sessions/{id}/request` with body
  `{source_raw: <client frame>}`, response `{source_raw: <app-server response>}`.
  Bytes stay opaque through the control plane (same contract as v2).
- **cthulhu + plano stubs**: new `RelayRequest` ControlFrame variant through the
  existing `AttachManager.roundTrip` command/ack machinery. Edit `ohr.proto` and
  regenerate — never hand-edit generated stubs. cthulhu: `make proto-ohr`
  (verified reproducible). plano: `crates/harness_proto/proto/regen.sh`.
- **plano**: adapter relay — denylist check (`initialize`, `thread/start`,
  `thread/resume`, `thread/unsubscribe`, `turn/start` keeps its dedicated path),
  rewrite `threadId`/`turnId` to VM-scoped IDs, mint own JSON-RPC `id`, forward
  to app-server, await with timeout, rewrite IDs back, return raw response.
- **doctl + godo vendor patch**: facade relays unhandled client requests
  instead of erroring; replace the `turn/interrupt` no-op stub
  (`facade.go` — currently acks optimistically while the run keeps going)
  with a real relay call.

**Exit test:** Esc mid-turn stops the run; TUI renders the abort cleanly.

**Expedite:** the cthulhu/plano legs and the doctl leg are independent until
integration — build in parallel, meet at the fake-harness tests.

### As built

Shape as planned, with three decisions worth recording:

- The RPC is `OHR.Relay` (`RelayRequest`/`RelayResponse`), added as a unary RPC
  *and* an Attach `ControlFrame`/`StreamFrame` variant, matching `SendInput`.
  Production forwards over the unary `ohr.Client`; Attach stays the opt-in
  alternative.
- **`turnId` is rewritten by the adapter, not the proxy.** The outbound path
  replaces the VM's turn id with the harness run id rather than remembering it,
  so neither side of doctl can reverse the mapping. The adapter now records
  Codex's turn id from `turn/start`'s reply and substitutes it on relay —
  without this, an interrupt is rejected as an unknown turn.
- **The adapter answers with a JSON-RPC error frame wherever the protocol can
  express the failure** (app-server rejection, timeout, dead transport) instead
  of failing the relay. The client is blocked on a specific request id, and a
  transport-level error gives it nothing to resolve that id with — which is the
  hang M1 exists to remove.

Denylist landed in the adapter (`RELAY_DENIED_METHODS` + `RELAY_DENIED_PREFIXES`),
covering the planned lifecycle methods plus `account/login|logout` by prefix,
since a relay forwards whatever it does not recognize.

Also fixed en route: `make proto-public` had been failing for everyone (the
pinned v1 `protoc-gen-grpc-gateway` aborts on the proto3 `optional` fields in
`events.proto`), which is why public stubs were being hand-edited. The
generated gateway was dead code — nothing ever called
`RegisterHarnessAPIHandler*` — so the target no longer generates it. First
regeneration picked up an `AGENT_KIND_HERMES` enum value that had been missing
from the Go stub. This was listed as an M3 merge blocker; it's done.

### Request inventory (per codex app-server docs)

Already handled by the facade today: `initialize`/`initialized`,
`thread/start`, `thread/resume`, `thread/unsubscribe`, `turn/start` (raw
params via v2), and local answers for `account/read`, `model/list`,
`hooks/list`, `skills/list`, `plugin/list`, `app/list`. `turn/interrupt`
exists but is a no-op stub.

Client→server requests M1's generic relay wires up:

- **High-impact, day one:** `turn/interrupt` (replace stub), `turn/steer`,
  `review/start`, `thread/compact/start`, `thread/fork`, `thread/rollback`
- **Thread bookkeeping:** `thread/read`, `thread/list`, `thread/turns/list`,
  `thread/items/list`, `thread/loaded/list`, `thread/name/set`,
  `thread/goal/set|get|clear`, `thread/metadata/update`, `thread/archive`,
  `thread/unarchive`, `thread/delete`
- **Execution surfaces (relay works; each is a policy decision — they run
  code in the guest):** `command/exec` + `command/exec/write|resize|terminate`
  (output streams back as notifications the catch-all already carries),
  `thread/shellCommand` (docs: runs *outside* codex's sandbox with full
  access — inside the VM that means the guest container; decide deliberately),
  `process/spawn|writeStdin|resizePty|kill` (experimental, same caveat),
  `thread/backgroundTerminals/clean|list|terminate` (experimental)
- **MCP / discovery / config (relay works; semantics are VM-side):**
  `mcpServerStatus/list`, `mcpServer/resource/read`, `mcpServer/tool/call`,
  `config/mcpServer/reload`, `mcpServer/oauth/login`,
  `modelProvider/capabilities/read`, `experimentalFeature/list`,
  `experimentalFeature/enablement/set`, `environment/info`,
  `permissionProfile/list`, `collaborationMode/list`, `skills/extraRoots/set`,
  `skills/config/write`, `plugin/read|install|uninstall`, `plugin/skill/read`,
  `marketplace/add|remove|upgrade`, `feedback/upload`,
  `config/read|value/write|batchWrite` (edits the **VM's** config.toml),
  `configRequirements/read`. `windowsSandbox/*` is N/A on Linux guests.

Denylist — never relay:

- `initialize`, `thread/start`, `thread/resume`, `thread/unsubscribe`
  (session lifecycle belongs to the adapter/facade)
- `turn/start` (keeps its dedicated send-input path with durable side effects)
- `account/login/*` (auth belongs to the harness, not the user's ChatGPT
  account — answer locally or reject)

---

## M2 — Outbound request catch-all (3–4 days)

Unknown server→client requests (approval types / interactive prompts the HITL
flow doesn't model, e.g. `item/tool/requestUserInput`) stop killing runs and
round-trip natively.

### Server→client request inventory

- Working today via HITL: `commandExecution` and `fileChange` approvals
- Need this milestone's catch-all: `item/tool/requestUserInput`,
  `mcpServer/elicitation/request`, `attestation/generate`, and anything codex
  adds later

- **plano**: wrap unknown ServerRequests in the pending-HITL flow with
  `source_raw` attached; timeout/default-decline policy so headless runs never
  block forever.
- **cthulhu + plano**: generalized resolve endpoint accepting raw response
  bytes, forwarded to the app-server; honor codex's `serverRequest/resolved`
  cleanup semantics (pending requests are cleared by turn start/completion/
  interruption — M1's interrupt doubles as the escape hatch for stuck requests).
- **doctl**: proxy forwards the native request frame to the TUI under its own
  minted `id`; routes the TUI's raw answer back through the resolve endpoint.

**Exit test:** an approval/prompt type we never hand-mapped round-trips through
the TUI.

---

## M3 — Hardening + mergeability (2 days, can overlap M2)

- Reconnect/replay mid-turn with interleaved raw/canonical events;
  multi-client attach behavior.
- Merge blockers: upstream the godo vendor patch; fix cthulhu's `proto-public`
  toolchain (`protoc-gen-grpc-gateway` rejects proto3 optional), the last
  codegen path still requiring a workaround.

**Exit test:** kill/restart the proxy mid-turn without TUI desync; CI green on
all three branches.

---

## Critical path

**M0 → M1 (~3 days) is where the TUI crosses from demo to daily-drivable**
(working interrupt + protocol-backed slash commands). M2/M3 proceed in
parallel afterward.

## Known structural differences (not protocol gaps — out of scope)

- Workspace is remote (VM files, not local checkout) — editor/git integrations
  point at the sandbox.
- Auth belongs to the harness, not the user's ChatGPT account (`/login`,
  rate-limit displays).
- Session lifecycle is managed server-side (persistence + reattach; inactivity
  timeout self-heals on next input).
- Added network latency on every round-trip.
