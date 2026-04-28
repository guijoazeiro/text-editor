export interface User {
  id: string;
  name: string;
  email: string;
}

export interface RemoteUser {
  clientId: number;
  user: {
    id: string;
    name: string;
    color: string;
  };
  cursor?: unknown;
}
