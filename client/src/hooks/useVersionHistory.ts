import { useState, useCallback } from "react";
import { versionsAPI } from "@/lib/api";
import { useToastStore } from "@/store/toastStore";
import type { DocumentVersion } from "@/types";

interface UseVersionHistoryOptions {
  documentId: string;
  onRestoreStart?: () => void;
  onRestoreComplete?: () => void;
}

export function useVersionHistory({
  documentId,
  onRestoreStart,
  onRestoreComplete,
}: UseVersionHistoryOptions) {
  const toast = useToastStore();

  const [versions, setVersions] = useState<DocumentVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [restoring, setRestoring] = useState(false);

  const fetchVersions = useCallback(async () => {
    setLoading(true);
    try {
      const res = await versionsAPI.list(documentId);
      const all: DocumentVersion[] = res.data.data ?? [];
      setVersions(all.slice(0, 20));
    } catch {
      toast.error("Failed to load version history");
    } finally {
      setLoading(false);
    }
  }, [documentId, toast]);

  const restoreVersion = useCallback(
    async (versionNumber: number) => {
      setRestoring(true);
      onRestoreStart?.();
      try {
        await versionsAPI.restore(documentId, versionNumber);
        toast.success(`Restored to version ${versionNumber}`);
        await fetchVersions();
        onRestoreComplete?.();
      } catch {
        toast.error("Failed to restore version");
      } finally {
        setRestoring(false);
      }
    },
    [documentId, fetchVersions, onRestoreStart, onRestoreComplete, toast],
  );

  return { versions, loading, restoring, fetchVersions, restoreVersion };
}
