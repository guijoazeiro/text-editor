import * as Y from "yjs";
import * as awarenessProtocol from "y-protocols/awareness";
import * as syncProtocol from "y-protocols/sync";
import { Observable } from "lib0/observable";
import * as encoding from "lib0/encoding";
import * as decoding from "lib0/decoding";
import { WebSocketClient, WSMessage } from "./websocket";

export class YjsWebSocketProvider extends Observable<string> {
  private _synced = false;

  private _initReceived = false;
  private _initTimeout: ReturnType<typeof setTimeout> | null = null;

  private _syncTimeout: ReturnType<typeof setTimeout> | null = null;
  private _peerSyncStarted = false;

  constructor(
    private documentId: string,
    private doc: Y.Doc,
    private ws: WebSocketClient,
    public awareness: awarenessProtocol.Awareness,
  ) {
    super();

    this._setupDocumentListeners();
    this._setupWebSocketListeners();
    this._setupAwarenessListeners();

    if (this.ws.isConnected()) {
      this._onConnected();
    }
  }

  private _onConnected() {
    this._synced = false;
    this._initReceived = false;
    this._peerSyncStarted = false;

    if (this._initTimeout) clearTimeout(this._initTimeout);
    this._initTimeout = setTimeout(() => {
      this._initTimeout = null;
      if (!this._initReceived) {
        console.log("[Yjs] No init snapshot → doc is new, starting sync");
        this._startPeerSync();
      }
    }, 400);
  }

  private _handleYjsInit(data: unknown) {
    if (!data || typeof data !== "object") return;
    const payload = data as Record<string, unknown>;

    this._initReceived = true;
    if (this._initTimeout) {
      clearTimeout(this._initTimeout);
      this._initTimeout = null;
    }

    const rawBatch = payload["updates"];
    const serverDoc = new Y.Doc();

    if (rawBatch && Array.isArray(rawBatch) && rawBatch.length > 0) {
      try {
        const arrays = (rawBatch as unknown[])
          .map((item) => {
            if (typeof item === "string") {
              const binaryString = atob(item);
              const bytes = new Uint8Array(binaryString.length);
              for (let i = 0; i < binaryString.length; i++) {
                bytes[i] = binaryString.charCodeAt(i);
              }
              return bytes;
            }
            return Array.isArray(item)
              ? new Uint8Array(item as number[])
              : (item as Uint8Array);
          })
          .filter((u) => u.length > 0);

        if (arrays.length === 0) {
          console.log("[Yjs] All init updates were invalid (old format?)");
        } else {
          const merged = Y.mergeUpdates(arrays);

          Y.applyUpdate(serverDoc, merged);

          Y.applyUpdate(this.doc, merged, this);
          console.log(
            `[Yjs] Applied init snapshot (${arrays.length} updates merged)`,
          );
        }
      } catch (err) {
        console.error("[Yjs] Error applying init snapshot:", err);
      }
    }

    const serverStateVector = Y.encodeStateVector(serverDoc);
    const missingUpdates = Y.encodeStateAsUpdate(this.doc, serverStateVector);

    if (missingUpdates.byteLength > 2) {
      console.log(
        `[Yjs] Found offline edits (${missingUpdates.byteLength} bytes). Syncing to server...`,
      );
      this._sendUpdate(missingUpdates);
    }

    this._startPeerSync();
  }

  private _startPeerSync() {
    if (this._peerSyncStarted) return;
    this._peerSyncStarted = true;

    this._sendSyncStep1();

    if (this._syncTimeout) clearTimeout(this._syncTimeout);
    this._syncTimeout = setTimeout(() => {
      this._syncTimeout = null;
      if (!this._synced) {
        console.log("[Yjs] No SyncStep2 received — marking synced (no peers)");
        this._synced = true;
        this.emit("synced", [true]);
      }
    }, 2000);
  }

  private _setupDocumentListeners() {
    this.doc.on("update", (update: Uint8Array, origin: unknown) => {
      if (origin === this) return;
      this._sendUpdate(update);
    });
  }

  private _setupWebSocketListeners() {
    this.ws.onConnect(() => {
      console.log("[Yjs] WebSocket connected");
      this._onConnected();
    });

    this.ws.on("yjs-init", (message: WSMessage) => {
      this._handleYjsInit(message.data);
    });

    this.ws.on("yjs-sync", (message: WSMessage) => {
      this._handleSyncMessage(message.data);
    });

    this.ws.on("yjs-awareness", (message: WSMessage) => {
      this._handleAwarenessUpdate(message.data);
    });

    this.ws.on("yjs-awareness-off", (message: WSMessage) => {
      this._handleAwarenessOff(message);
    });
  }

  private _setupAwarenessListeners() {
    this.awareness.on(
      "update",
      ({
        added,
        updated,
        removed,
      }: {
        added: number[];
        updated: number[];
        removed: number[];
      }) => {
        this._sendAwarenessUpdate([...added, ...updated, ...removed]);
      },
    );
  }

  private _sendSyncStep1() {
    const encoder = encoding.createEncoder();
    encoding.writeVarUint(encoder, syncProtocol.messageYjsSyncStep1);
    syncProtocol.writeSyncStep1(encoder, this.doc);
    this._sendBytes(encoding.toUint8Array(encoder));
  }

  private _sendUpdate(update: Uint8Array) {
    if (!this.ws.isConnected()) return;
    const encoder = encoding.createEncoder();
    encoding.writeVarUint(encoder, syncProtocol.messageYjsUpdate);
    encoding.writeVarUint8Array(encoder, update);
    this._sendBytes(encoding.toUint8Array(encoder));
  }

  private _sendBytes(bytes: Uint8Array) {
    this.ws.send({
      type: "yjs-sync",
      data: { update: Array.from(bytes) },
    });
  }

  private _sendAwarenessUpdate(changedClients: number[]) {
    if (!this.ws.isConnected()) return;
    const update = awarenessProtocol.encodeAwarenessUpdate(
      this.awareness,
      changedClients,
    );
    this.ws.send({
      type: "yjs-awareness",
      data: { update: Array.from(update) },
    });
  }

  private _handleSyncMessage(data: unknown) {
    if (!data || typeof data !== "object") return;
    const payload = data as Record<string, unknown>;

    const raw = payload["update"];
    if (!raw) return;

    let bytes: Uint8Array;
    if (raw instanceof Uint8Array) {
      bytes = raw;
    } else if (Array.isArray(raw)) {
      bytes = new Uint8Array(raw as number[]);
    } else {
      console.warn("[Yjs] unexpected update format:", typeof raw);
      return;
    }

    try {
      const decoder = decoding.createDecoder(bytes);
      const msgType = decoding.readVarUint(decoder);

      switch (msgType) {
        case syncProtocol.messageYjsSyncStep1: {
          const encoder = encoding.createEncoder();
          encoding.writeVarUint(encoder, syncProtocol.messageYjsSyncStep2);
          // @ts-ignore — @types/y-protocols declares 3rd arg as Uint8Array but runtime expects Decoder
          syncProtocol.writeSyncStep2(encoder, this.doc, decoder);
          this._sendBytes(encoding.toUint8Array(encoder));
          break;
        }
        case syncProtocol.messageYjsSyncStep2: {
          syncProtocol.readSyncStep2(decoder, this.doc, this);
          if (this._syncTimeout) {
            clearTimeout(this._syncTimeout);
            this._syncTimeout = null;
          }
          if (!this._synced) {
            this._synced = true;
            this.emit("synced", [true]);
            console.log("[Yjs] Synced via SyncStep2");
          }
          break;
        }
        case syncProtocol.messageYjsUpdate: {
          const update = decoding.readVarUint8Array(decoder);
          Y.applyUpdate(this.doc, update, this);
          if (this._syncTimeout) {
            clearTimeout(this._syncTimeout);
            this._syncTimeout = null;
          }
          if (!this._synced) {
            this._synced = true;
            this.emit("synced", [true]);
            console.log("[Yjs] Synced via incremental update");
          }
          break;
        }
        default:
          console.warn("[Yjs] unknown sync message type:", msgType);
      }
    } catch (err) {
      console.error("[Yjs] Error handling sync message:", err);
    }
  }

  private _handleAwarenessUpdate(data: unknown) {
    if (!data || typeof data !== "object") return;
    const raw = (data as Record<string, unknown>)["update"];
    if (!raw) return;

    try {
      const bytes = Array.isArray(raw)
        ? new Uint8Array(raw as number[])
        : (raw as Uint8Array);
      awarenessProtocol.applyAwarenessUpdate(this.awareness, bytes, this);
    } catch (err) {
      console.error("[Yjs] Error applying awareness update:", err);
    }
  }

  private _handleAwarenessOff(message: WSMessage) {
    const userId = message.user_id;
    if (!userId) return;

    const clientIdsToRemove: number[] = [];
    this.awareness.getStates().forEach((state, clientId) => {
      if (state?.user?.id === userId) {
        clientIdsToRemove.push(clientId);
      }
    });

    if (clientIdsToRemove.length > 0) {
      awarenessProtocol.removeAwarenessStates(
        this.awareness,
        clientIdsToRemove,
        "server-disconnect",
      );
      console.log(
        `[Yjs] Removed awareness for disconnected user=${userId} (clientIds=${clientIdsToRemove.join(",")})`,
      );
    }
  }

  public setAwarenessField(field: string, value: unknown) {
    this.awareness.setLocalState({
      ...(this.awareness.getLocalState() ?? {}),
      [field]: value,
    });
  }

  public isSynced(): boolean {
    return this._synced;
  }

  public destroy() {
    if (this._initTimeout) clearTimeout(this._initTimeout);
    if (this._syncTimeout) clearTimeout(this._syncTimeout);
    this.awareness.destroy();
    super.destroy();
  }
}
