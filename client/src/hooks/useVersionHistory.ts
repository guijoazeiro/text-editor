import { useState, useCallback } from "react";
import { versionsAPI } from "@/lib/api";
import { useToastStore } from "@/store/toastStore";
import type { DocumentVersion } from "@/types";

interface UseVersionHistoryOptions {
  documentId: string;
  onRestoreStart?: () => void;
  onRestoreComplete?: () => void;
}

interface PaginationMeta {
  total: number;
  page: number;
  limit: number;
  pages: number;
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
  const [pagination, setPagination] = useState<PaginationMeta>({
    total: 0,
    page: 1,
    limit: 20,
    pages: 1,
  });

  const fetchVersions = useCallback(
    async (page = 1, limit = 20) => {
      setLoading(true);
      try {
        const res = await versionsAPI.list(documentId, { page, limit });
        const data = res.data.data;

        if (data && typeof data === "object" && "versions" in data) {
          setVersions(data.versions ?? []);
          setPagination({
            total: data.total ?? 0,
            page: data.page ?? page,
            limit: data.limit ?? limit,
            pages: data.pages ?? 1,
          });
        } else {
          const all: DocumentVersion[] = data ?? [];
          setVersions(all.slice(0, limit));
        }
      } catch {
        toast.error("Failed to load version history");
      } finally {
        setLoading(false);
      }
    },
    [documentId, toast],
  );

  const restoreVersion = useCallback(
    async (versionNumber: number) => {
      setRestoring(true);
      onRestoreStart?.();
      try {
        await versionsAPI.restore(documentId, versionNumber);
        toast.success(`Restored to version ${versionNumber}`);
        await fetchVersions(pagination.page, pagination.limit);
        onRestoreComplete?.();
      } catch {
        toast.error("Failed to restore version");
      } finally {
        setRestoring(false);
      }
    },
    [
      documentId,
      fetchVersions,
      onRestoreStart,
      onRestoreComplete,
      pagination.page,
      pagination.limit,
      toast,
    ],
  );

  return {
    versions,
    loading,
    restoring,
    pagination,
    fetchVersions,
    restoreVersion,
  };
}
