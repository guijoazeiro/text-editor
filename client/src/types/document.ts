export type ContentFormat = "text" | "tiptap";

export type Permission = "owner" | "editor" | "viewer";

export interface Document {
  id: string;
  title: string;
  content: string;
  content_format: ContentFormat;
  permission?: Permission;
  created_at: string;
  updated_at: string;
  user: {
    id: string;
    name: string;
    email: string;
  };
}

export interface DocumentMeta {
  title: string;
  permission: Permission;
  content_format: ContentFormat;
  content_plain?: string;
  updated_at?: string;
}

export interface ShareLink {
  id: string;
  document_id: string;
  token: string;
  permission: "viewer" | "editor";
  expires_at?: string;
  created_at: string;
}
