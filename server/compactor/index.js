"use strict";

const express = require("express");

const PORT = process.env.PORT || 3001;

let mergeUpdates = null;

async function main() {
  const yjs = await import("yjs");
  mergeUpdates = yjs.mergeUpdates;

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

      const merged = mergeUpdates(uint8Arrays);

      const snapshot = Buffer.from(merged).toString("base64");

      res.json({ snapshot });
    } catch (err) {
      console.error("[compactor] error during merge:", err.message);
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
