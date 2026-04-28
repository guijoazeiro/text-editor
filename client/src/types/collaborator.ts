import type { User } from "./user";

export interface Collaborator {
  user_id: string;
  document_id: string;
  permission: "editor" | "viewer";
  user: User;
  created_at: string;
}

export interface AddCollaboratorPayload {
  email: string;
  permission: "editor" | "viewer";
}

export interface UpdateCollaboratorPayload {
  permission: "editor" | "viewer";
}
