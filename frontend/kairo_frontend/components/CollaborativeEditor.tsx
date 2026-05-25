"use client";
import { useCallback, useEffect, useRef, useState } from "react";

const COLORS = [
  "#3dd6c8",
  "#34d399",
  "#60a5fa",
  "#f472b6",
  "#fb923c",
  "#a78bfa",
];
const colorMap = new Map<string, string>();

function colorFor(id: string): string {
  if (!colorMap.has(id))
    colorMap.set(id, COLORS[colorMap.size % COLORS.length]);
  return colorMap.get(id)!;
}

function flatToLineCol(doc: string, pos: number) {
  const clamped = Math.max(0, Math.min(pos, doc.length));
  const before = doc.slice(0, clamped);
  const lines = before.split("\n");
  return { line: lines.length - 1, col: lines[lines.length - 1].length };
}

function applyOp(doc: string, op: any): string {
  const p = Math.max(0, Math.min(op.pos, doc.length));
  if (op.type === "insert" && op.text)
    return doc.slice(0, p) + op.text + doc.slice(p);
  if (op.type === "delete" && op.length)
    return doc.slice(0, p) + doc.slice(Math.min(p + op.length, doc.length));
  return doc;
}

function parseMessages(raw: string): any[] {
  const out: any[] = [];
  let depth = 0;
  let start = -1;
  for (let i = 0; i < raw.length; i++) {
    const ch = raw[i];
    if (ch === "{") {
      if (depth === 0) start = i;
      depth++;
    } else if (ch === "}") {
      depth--;
      if (depth === 0 && start !== -1) {
        try {
          out.push(JSON.parse(raw.slice(start, i + 1)));
        } catch (_) {}
        start = -1;
      }
    }
  }
  return out;
}

const CHAR_WIDTH = 8.4;
const LINE_HEIGHT = 22;

function RemoteCursor({
  id,
  line,
  col,
  color,
}: {
  id: string;
  line: number;
  col: number;
  color: string;
}) {
  const top = 16 + line * LINE_HEIGHT;
  const left = 16 + col * CHAR_WIDTH;
  return (
    <div style={{ position: "absolute", top, left, pointerEvents: "none" }}>
      <div
        style={{
          position: "absolute",
          width: 2,
          height: LINE_HEIGHT,
          background: color,
          opacity: 0.9,
        }}
      />
      <div
        style={{
          position: "absolute",
          bottom: "100%",
          left: 0,
          background: color,
          color: "#000",
          padding: "1px 5px",
          fontSize: 10,
          fontFamily: "'JetBrains Mono', monospace",
          borderRadius: "3px 3px 3px 0",
          whiteSpace: "nowrap",
          marginBottom: 3,
          fontWeight: 600,
          letterSpacing: "0.02em",
        }}
      >
        {id}
      </div>
    </div>
  );
}

function UserDot({ id, color }: { id: string; color: string }) {
  const initials = id.slice(0, 2).toUpperCase();
  return (
    <div
      title={id}
      style={{
        width: 22,
        height: 22,
        borderRadius: "50%",
        background: "#1e1e1e",
        border: "1px solid #2a2a2a",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontSize: 9,
        fontWeight: 700,
        color: color,
        fontFamily: "'JetBrains Mono', monospace",
        letterSpacing: "0.05em",
        cursor: "default",
      }}
    >
      {initials}
    </div>
  );
}

export default function CollaborativeEditor() {
  const [wsUrl, setWsUrl] = useState("ws://localhost:4000/ws");
  const [sessionInput, setSessionInput] = useState("");
  const [userIdInput, setUserIdInput] = useState("");
  const [clientId, setClientId] = useState("");
  const [joined, setJoined] = useState(false);
  const [status, setStatus] = useState("");
  const [peers, setPeers] = useState<string[]>([]);
  const [remoteCursors, setRemoteCursors] = useState<
    Map<string, { line: number; col: number; color: string }>
  >(new Map());

  const taRef = useRef<HTMLTextAreaElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const docRef = useRef("");
  const versionRef = useRef(0);
  const cursorRafRef = useRef<number | null>(null);
  const clientIdRef = useRef(clientId);
  const sessionInputRef = useRef("");
  const userIdInputRef = useRef("");
  useEffect(() => {
    clientIdRef.current = clientId;
  }, [clientId]);
  useEffect(() => {
    sessionInputRef.current = sessionInput;
  }, [sessionInput]);
  useEffect(() => {
    userIdInputRef.current = userIdInput;
  }, [userIdInput]);

  const sendCursor = useCallback(() => {
    const ta = taRef.current;
    const ws = wsRef.current;
    if (!ta || !ws || ws.readyState !== WebSocket.OPEN) return;
    const pos = ta.selectionStart;
    const line = ta.value.slice(0, pos).split("\n").length - 1;
    ws.send(
      JSON.stringify({
        kind: "cursor",
        client_id: clientIdRef.current,
        line,
        position: pos,
      }),
    );
  }, []);

  const scheduleCursor = useCallback(() => {
    if (cursorRafRef.current !== null)
      cancelAnimationFrame(cursorRafRef.current);
    cursorRafRef.current = requestAnimationFrame(() => {
      cursorRafRef.current = null;
      sendCursor();
    });
  }, [sendCursor]);

  const connect = useCallback(
    (sid: string, uid: string) => {
      if (wsRef.current) wsRef.current.close();
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        ws.send(
          JSON.stringify({ kind: "join", session_id: sid, client_id: uid }),
        );
      };

      ws.onmessage = (ev) => {
        const msgs = parseMessages(ev.data as string);
        for (const msg of msgs) {
          const myId = clientIdRef.current;

          if (msg.kind === "init") {
            docRef.current = msg.doc ?? "";
            versionRef.current = 0;
            if (taRef.current) taRef.current.value = docRef.current;
            setJoined(true);
            setStatus(sid);
            // announce ourselves so all existing peers see us
            ws.send(
              JSON.stringify({
                kind: "join",
                session_id: sid,
                client_id: clientIdRef.current,
              }),
            );
            continue;
          }
          if (msg.kind === "join" && msg.client_id && msg.client_id !== myId) {
            setPeers((p) =>
              p.includes(msg.client_id) ? p : [...p, msg.client_id],
            );
            continue;
          }
          if (msg.type === "insert" || msg.type === "delete") {
            const ta = taRef.current;
            if (!ta) continue;
            const curPos = ta.selectionStart;
            let newPos = curPos;
            if (msg.type === "insert" && msg.pos <= curPos)
              newPos = curPos + (msg.text?.length ?? 0);
            else if (msg.type === "delete" && msg.pos < curPos)
              newPos = Math.max(msg.pos, curPos - (msg.length ?? 0));
            docRef.current = applyOp(docRef.current, msg);
            if (msg.version !== undefined) versionRef.current = msg.version;
            ta.value = docRef.current;
            try {
              ta.setSelectionRange(newPos, newPos);
            } catch (_) {}
            continue;
          }
          if (
            msg.kind === "cursor" &&
            msg.client_id &&
            msg.client_id !== myId
          ) {
            const { line, col } = flatToLineCol(
              docRef.current,
              msg.position ?? 0,
            );
            const color = colorFor(msg.client_id);
            setRemoteCursors((prev) => {
              const next = new Map(prev);
              next.set(msg.client_id, { line, col, color });
              return next;
            });
            continue;
          }
          if (msg.kind === "cursor_clear" && msg.client_id) {
            setRemoteCursors((prev) => {
              const n = new Map(prev);
              n.delete(msg.client_id);
              return n;
            });
            setPeers((p) => p.filter((x) => x !== msg.client_id));
          }
        }
      };

      ws.onclose = () => {
        setStatus("disconnected");
        setJoined(false);
      };
      ws.onerror = () => {
        setStatus("connection error");
      };
    },
    [wsUrl],
  );

  function handleConnect() {
    const sid =
      sessionInputRef.current.trim() ||
      "room-" + Math.random().toString(36).slice(2, 7);
    const uid =
      userIdInputRef.current.trim() ||
      "user-" + Math.random().toString(36).slice(2, 8);
    setClientId(uid);
    clientIdRef.current = uid;
    connect(sid, uid);
  }

  function handleInput() {
    const ta = taRef.current;
    if (!ta) return;
    const newVal = ta.value;
    const oldVal = docRef.current;
    if (newVal === oldVal) return;
    let s = 0;
    while (s < oldVal.length && s < newVal.length && oldVal[s] === newVal[s])
      s++;
    let oe = oldVal.length,
      ne = newVal.length;
    while (oe > s && ne > s && oldVal[oe - 1] === newVal[ne - 1]) {
      oe--;
      ne--;
    }
    docRef.current = newVal;
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    if (oe > s) {
      ws.send(
        JSON.stringify({
          type: "delete",
          pos: s,
          length: oe - s,
          user_id: clientId,
          version: versionRef.current,
        }),
      );
      versionRef.current++;
    }
    if (ne > s) {
      ws.send(
        JSON.stringify({
          type: "insert",
          pos: s,
          text: newVal.slice(s, ne),
          length: 0,
          user_id: clientId,
          version: versionRef.current,
        }),
      );
      versionRef.current++;
    }
    scheduleCursor();
  }

  if (!joined) {
    return (
      <>
        <style>{`
          @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&family=Playfair+Display:ital,wght@0,700;1,400&display=swap');

          *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

          .ce-root {
            display: flex;
            height: 100vh;
            width: 100vw;
            align-items: center;
            justify-content: center;
            background: #111;
            font-family: 'JetBrains Mono', monospace;
            overflow: hidden;
          }

          /* Subtle background texture */
          .ce-root::before {
            content: '';
            position: fixed;
            inset: 0;
            background:
              radial-gradient(ellipse at 20% 50%, rgba(163,230,53,0.04) 0%, transparent 60%),
              radial-gradient(ellipse at 80% 50%, rgba(96,165,250,0.04) 0%, transparent 60%);
            pointer-events: none;
          }

          /* THE CARD */
          .ce-card {
            display: flex;
            width: 820px;
            height: 520px;
            border-radius: 16px;
            overflow: hidden;
            box-shadow:
              0 0 0 1px rgba(255,255,255,0.06),
              0 32px 80px rgba(0,0,0,0.6),
              0 8px 24px rgba(0,0,0,0.4);
            position: relative;
            z-index: 1;
          }

          /* LEFT — image half */
          .ce-card-image {
            width: 50%;
            flex-shrink: 0;
            position: relative;
            overflow: hidden;
            background: #ffffff;
          }
          .ce-card-image img {
            position: absolute;
            inset: 0;
            width: 100%;
            height: 100%;
            object-fit: contain;
            object-position: center;
          }

          /* RIGHT — form half */
          .ce-card-form {
            width: 50%;
            background: #161616;
            display: flex;
            flex-direction: column;
            justify-content: center;
            padding: 40px 36px;
            border-left: 1px solid rgba(255,255,255,0.05);
          }

          .ce-logo {
            font-family: 'Playfair Display', serif;
            font-style: italic;
            font-size: 13px;
            color: #4a4440;
            letter-spacing: 0.03em;
            margin-bottom: 28px;
          }

          .ce-heading {
            font-family: 'Playfair Display', serif;
            font-weight: 700;
            font-size: 22px;
            color: #e8e2da;
            line-height: 1.2;
            margin-bottom: 4px;
          }
          .ce-subheading {
            font-size: 10px;
            color: #4a4440;
            letter-spacing: 0.08em;
            text-transform: uppercase;
            margin-bottom: 28px;
          }

          .ce-field { margin-bottom: 14px; }
          .ce-label {
            display: block;
            font-size: 9px;
            font-weight: 600;
            letter-spacing: 0.12em;
            text-transform: uppercase;
            color: #5a5248;
            margin-bottom: 6px;
          }
          .ce-input {
            width: 100%;
            padding: 9px 12px;
            background: #1e1e1e;
            border: 1px solid #2a2a2a;
            border-radius: 6px;
            font-family: 'JetBrains Mono', monospace;
            font-size: 12px;
            color: #c9c3b8;
            outline: none;
            transition: border-color 0.15s;
          }
          .ce-input::placeholder { color: #3a3630; }
          .ce-input:focus {
            border-color: #3a3630;
          }

          .ce-divider {
            height: 1px;
            background: #222;
            margin: 10px 0 14px;
          }

          .ce-btn {
            width: 100%;
            padding: 11px;
            background: #e8e2da;
            color: #111;
            border: none;
            border-radius: 6px;
            font-family: 'JetBrains Mono', monospace;
            font-size: 11px;
            font-weight: 600;
            letter-spacing: 0.1em;
            text-transform: uppercase;
            cursor: pointer;
            transition: background 0.15s, transform 0.1s;
            margin-top: 6px;
          }
          .ce-btn:hover { background: #fff; }
          .ce-btn:active { transform: scale(0.98); }
        `}</style>

        <div className="ce-root">
          <div className="ce-card">
            <div className="ce-card-image">
              <img src="/imp_.png" alt="" />
            </div>

            {/* Form half */}
            <div className="ce-card-form">
              <div className="ce-heading">Join a session</div>
              <div className="ce-subheading">
                Real-time collaborative editing
              </div>

              <div className="ce-field">
                <label className="ce-label">Your name</label>
                <input
                  className="ce-input"
                  value={userIdInput}
                  onChange={(e) => setUserIdInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handleConnect()}
                  placeholder="alice"
                  autoFocus
                />
              </div>

              <div className="ce-divider" />

              <div className="ce-field">
                <label className="ce-label">Session ID</label>
                <input
                  className="ce-input"
                  value={sessionInput}
                  onChange={(e) => setSessionInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handleConnect()}
                  placeholder="room-abc123"
                />
              </div>

              <div className="ce-field">
                <label className="ce-label">WebSocket URL</label>
                <input
                  className="ce-input"
                  value={wsUrl}
                  onChange={(e) => setWsUrl(e.target.value)}
                />
              </div>

              <button className="ce-btn" onClick={handleConnect}>
                Connect →
              </button>
            </div>
          </div>
        </div>
      </>
    );
  }

  /////////////////////EDITOR
  const myColor = colorFor(clientId) || "#3dd6c8";
  const allUsers = [
    { id: clientId, color: myColor },
    ...peers.map((p) => ({ id: p, color: colorFor(p) })),
  ];

  return (
    <>
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&display=swap');
        *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

        .ed-root {
          display: flex;
          flex-direction: column;
          height: 100vh;
          width: 100vw;
          background: #0e0e0e;
          font-family: 'JetBrains Mono', monospace;
          color: #c9c3b8;
        }

        /* editor fills all space above the bottom bar */
        .ed-wrap {
          flex: 1;
          position: relative;
          overflow: hidden;
        }
        .ed-bg {
          position: absolute;
          inset: 0;
          background: #0e0e0e;
          z-index: 1;
        }
        .ed-cursor-layer {
          position: absolute;
          inset: 0;
          pointer-events: none;
          z-index: 5;
        }
        .ed-textarea {
          position: absolute;
          inset: 0;
          width: 100%;
          height: 100%;
          padding: 16px;
          font-size: 14px;
          line-height: 22px;
          font-family: 'JetBrains Mono', monospace;
          background: transparent;
          color: #d4cec6;
          border: none;
          outline: none;
          resize: none;
          caret-color: #3dd6c8;
          z-index: 10;
          letter-spacing: 0.01em;
        }
        .ed-textarea::selection {
          background: rgba(163,230,53,0.15);
        }

        /* BOTTOM BAR — minimal, centered, narrow */
        .ed-bar {
          display: flex;
          justify-content: center;
          align-items: center;
          padding: 10px 0 14px;
          background: transparent;
          flex-shrink: 0;
        }
        .ed-bar-inner {
          display: flex;
          align-items: center;
          gap: 14px;
          padding: 6px 16px;
          background: #161616;
          border: 1px solid #222;
          border-radius: 999px;
        }
        .ed-status-dot {
          width: 5px;
          height: 5px;
          border-radius: 50%;
          background: #3a3a3a;
          flex-shrink: 0;
        }
        .ed-session {
          font-size: 10px;
          color: #4a4440;
          letter-spacing: 0.05em;
        }
        .ed-session span {
          color: #6a6055;
        }
        .ed-sep {
          width: 1px;
          height: 12px;
          background: #2a2a2a;
        }
        .ed-users {
          display: flex;
          align-items: center;
          gap: 4px;
        }
      `}</style>

      <div className="ed-root">
        <div className="ed-wrap">
          <div className="ed-bg" />

          <div className="ed-cursor-layer">
            {[...remoteCursors.entries()].map(([id, { line, col, color }]) => (
              <RemoteCursor
                key={id}
                id={id}
                line={line}
                col={col}
                color={color}
              />
            ))}
          </div>

          <textarea
            ref={taRef}
            className="ed-textarea"
            onInput={handleInput}
            onKeyUp={scheduleCursor}
            onClick={scheduleCursor}
            onSelect={scheduleCursor}
            spellCheck={false}
            autoCorrect="off"
            autoCapitalize="off"
          />
        </div>

        {/* Bottom minimal pill bar */}
        <div className="ed-bar">
          <div className="ed-bar-inner">
            <div className="ed-status-dot" />
            <div className="ed-session">
              session <span>{status}</span>
            </div>
            {allUsers.length > 0 && <div className="ed-sep" />}
            <div className="ed-users">
              {allUsers.map(({ id, color }) => (
                <UserDot key={id} id={id} color={color} />
              ))}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
