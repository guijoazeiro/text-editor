"use strict";

const express = require("express");

const PORT = process.env.PORT || 3001;

let yjs = null;

async function main() {
  yjs = await import("yjs");

  const app = express();

  app.use(express.json({ limit: "10mb" }));

  app.post("/compact", (req, res) => {
    const { updates } = req.body;

    if (!Array.isArray(updates) || updates.length === 0) {
      return res
        .status(400)
        .json({ error: "updates must be a non-empty array" });
    }

    try {
      const uint8Arrays = updates.map((u, idx) => {
        if (!Array.isArray(u)) {
          throw new Error(`updates[${idx}] expected array, got ${typeof u}`);
        }
        return new Uint8Array(u);
      });

      const merged = yjs.mergeUpdates(uint8Arrays);

      const snapshot = Buffer.from(merged).toString("base64");

      res.json({ snapshot });
    } catch (err) {
      console.error("[compactor] error during merge:", err.message);
      res.status(500).json({ error: err.message });
    }
  });

  app.post("/state-vector", (req, res) => {
    const { snapshot } = req.body;

    if (!snapshot || typeof snapshot !== "string") {
      return res
        .status(400)
        .json({ error: "snapshot must be a base64 string" });
    }

    try {
      const snapshotBytes = new Uint8Array(Buffer.from(snapshot, "base64"));

      const stateVector = yjs.encodeStateVectorFromUpdate(snapshotBytes);
      res.json({ stateVector: Buffer.from(stateVector).toString("base64") });
    } catch (err) {
      console.error("[compactor] error computing state vector:", err.message);
      res.status(500).json({ error: err.message });
    }
  });

  app.get("/health", (_req, res) => {
    res.json({ status: "ok" });
  });

  app.listen(PORT, () => {
    console.log(`[compactor] listening on port ${PORT}`);
  });
}

main().catch((err) => {
  console.error("[compactor] startup failed:", err);
  process.exit(1);
});
