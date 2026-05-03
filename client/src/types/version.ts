export interface DocumentVersion {
  id: string;
  document_id: string;
  version_number: number;
  title: string;
  content: string;
  has_snapshot: boolean;
  created_by: string;
  created_at: string;
  created_by_user?: {
    id: string;
    name: string;
    email: string;
  };
}

export interface VersionDiff {
  version1: number;
  version2: number;
  title_changed: boolean;
  content_diff: string;
}
