export type User = {
  id: string;
  email: string;
  name: string;
  role: string;
  status: string;
};

export type UserList = {
  items: User[];
  page: number;
  pageSize: number;
  total: number;
};

export type Region = {
  id: string;
  name: string;
  code: string;
};

export type Area = {
  id: string;
  name: string;
  code: string;
  locationId: string;
};

export type Site = {
  id: string;
  name: string;
  code: string;
  type: string;
  regionId: string | null;
  supervisorId: string;
  hseOfficerId: string;
  isActive: boolean;
  areas: Area[];
};

export type UserFilters = {
  page: number;
  pageSize: number;
  role?: string;
  status?: string;
};

export function usersQueryKey(filters: UserFilters) {
  return ["users", filters] as const;
}

export type IncidentCategory =
  | "NEAR_MISS"
  | "FIRST_AID"
  | "MEDICAL_TREATMENT"
  | "LTI"
  | "FATALITY"
  | "PROPERTY_DAMAGE"
  | "ENVIRONMENTAL";

export type IncidentSeverity = "LOW" | "MEDIUM" | "HIGH" | "CRITICAL";

export type IncidentStatus =
  | "DRAFT"
  | "PENDING_REVIEW"
  | "UNDER_INVESTIGATION"
  | "CORRECTIVE_ACTION"
  | "CLOSED"
  | "REJECTED";

export type Witness = {
  name: string;
  position: string;
};

export type Incident = {
  id: string;
  incidentNumber: string | null;
  title: string;
  description: string;
  category: IncidentCategory;
  severity: IncidentSeverity;
  escalationLevel: "L1" | "L2" | "L3";
  status: IncidentStatus;
  incidentDatetime: string;
  locationId: string;
  areaId: string | null;
  reporterId: string;
  hasVictim: boolean;
  victimName: string | null;
  victimPosition: string | null;
  injuryDescription: string | null;
  initialTreatment: string | null;
  witnesses: Witness[] | null;
  pendingReviewAt: string | null;
  closedAt: string | null;
  closedById: string | null;
  createdAt: string;
  updatedAt: string;
};

export type IncidentList = {
  items: Incident[];
  page: number;
  pageSize: number;
  total: number;
};

export type IncidentFilters = {
  page: number;
  pageSize: number;
  status?: IncidentStatus;
  category?: IncidentCategory;
  locationId?: string;
  from?: string;
  to?: string;
};

export type WorkflowLog = {
  id: string;
  fromStatus: IncidentStatus;
  toStatus: IncidentStatus;
  actorId: string;
  comment: string | null;
  createdAt: string;
};

export type IncidentFile = {
  id: string;
  originalName: string;
  storedKey: string;
  mimeType: string;
  sizeBytes: number;
  context: string;
  correctiveActionId: string | null;
  url?: string;
  expiresIn?: number;
  createdAt?: string;
};

export function incidentsQueryKey(filters: IncidentFilters) {
  return ["incidents", filters] as const;
}
