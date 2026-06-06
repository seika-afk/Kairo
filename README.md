
# Kairo

A real-time collaborative text editing server written in Go. Uses WebSockets and Operational Transformation (OT) to keep multiple clients in sync on a shared document, with live cursor tracking per session.

---

## Getting started

**Requirements:** Go 1.21+

```bash
git clone https://github.com/seika-afk/Kairo.git
cd Kairo
go mod tidy
go run .
```

By default it listens on `:4000`. You can change that:

```bash
go run . -addr :8080
```

The WebSocket endpoint will be at `ws://localhost:4000/ws`.

---

## How it works

There's one WebSocket endpoint. Everything — joining a session, sending edits, moving your cursor — happens over that single connection. Sessions are identified by a string ID; they're created on-demand the first time someone joins with that ID.

The server uses OT to reconcile concurrent edits. Every op carries the document version it was based on, and the server transforms it against any ops that arrived in between before applying it.

---

## WebSocket endpoint

### `ws://localhost:4000/ws`

All communication is JSON over a persistent WebSocket connection. The CORS check currently allows `http://localhost:3000` and connections with no Origin header.

---

## Message reference

### Joining a session

The very first message you send after connecting must be a join. Everything else is ignored until this succeeds.

```json
{
  "kind": "join",
  "session_id": "my-doc-room",
  "client_id": "user-abc",
  "doc": "optional initial content"
}
```

`doc` is optional — only relevant if you're the first one in and want to seed the document. If the session already exists, it's ignored.

**Server responds immediately with:**

```json
{
  "kind": "init",
  "doc": "current document content"
}
```

**Other clients in the room get:**

```json
{
  "kind": "join",
  "session_id": "my-doc-room",
  "client_id": "user-abc"
}
```

---

### Sending an insert

```json
{
  "type": "insert",
  "pos": 5,
  "text": "hello",
  "version": 3,
  "user_id": "user-abc"
}
```

| Field | Description |
|---|---|
| `type` | `"insert"` |
| `pos` | Character index where the text is inserted |
| `text` | The string to insert |
| `version` | The document version this op is based on |
| `user_id` | Your client ID |

---

### Sending a delete

```json
{
  "type": "delete",
  "pos": 2,
  "length": 4,
  "version": 3,
  "user_id": "user-abc"
}
```

| Field | Description |
|---|---|
| `type` | `"delete"` |
| `pos` | Start index of the range to delete |
| `length` | Number of characters to remove |
| `version` | The document version this op is based on |
| `user_id` | Your client ID |

**The server broadcasts the transformed op to all other clients in the session.** You don't get your own op echoed back.

---

### Cursor updates

Send this whenever your cursor moves. You won't receive your own cursor back; everyone else in the session will.

```json
{
  "kind": "cursor",
  "client_id": "user-abc",
  "line": 4,
  "position": 12
}
```

Note: `client_id` in the payload gets overwritten by the server with the actual sender's ID, so you can't spoof someone else's cursor.

**Other clients receive:**

```json
{
  "kind": "cursor",
  "client_id": "user-abc",
  "line": 4,
  "position": 12
}
```

---

### Client disconnect

When a client disconnects, the server broadcasts a cursor clear to everyone remaining in the session:

```json
{
  "kind": "cursor_clear",
  "client_id": "user-abc"
}
```

Use this to remove that client's cursor from your UI.

---

## Session behaviour

- Sessions are created automatically on first join — no setup needed.
- Document state is in-memory only; nothing is persisted. If the server restarts, all sessions are gone.
- Multiple sessions can run concurrently; clients in different sessions are completely isolated.

---

## Quick example flow

```
Client A connects → sends join (session_id: "doc1", client_id: "alice", doc: "hello")
  ← receives: { "kind": "init", "doc": "hello" }

Client B connects → sends join (session_id: "doc1", client_id: "bob")
  ← receives: { "kind": "init", "doc": "hello" }
  → Client A receives: { "kind": "join", "session_id": "doc1", "client_id": "bob" }

Client A sends: { "type": "insert", "pos": 5, "text": " world", "version": 0, "user_id": "alice" }
  → Client B receives the transformed op

Client A moves cursor: { "kind": "cursor", "client_id": "alice", "line": 0, "position": 11 }
  → Client B receives the cursor update

Client A disconnects
  → Client B receives: { "kind": "cursor_clear", "client_id": "alice" }
```
