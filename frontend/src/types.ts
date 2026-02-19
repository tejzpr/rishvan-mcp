export interface Request {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string | null;
  source_name: string;
  app_name: string;
  question: string;
  response: string;
  status: string;
  responded_at: string | null;
}
