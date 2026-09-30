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

export type FiveWhy = {
  why: string;
  answer: string;
};

export type Fishbone = {
  man: string;
  machine: string;
  method: string;
  material: string;
  environment: string;
  measurement: string;
};

export type Rca = {
  id: string;
  incidentId: string;
  timeline: string;
  humanFactor: string;
  environmentFactor: string;
  equipmentFactor: string;
  fiveWhys: FiveWhy[] | null;
  fishbone: Fishbone | null;
  investigatorId: string;
  completedAt: string | null;
  updatedAt: string;
};

export type RcaTemplatePayload = {
  timeline: string;
  humanFactor: string;
  environmentFactor: string;
  equipmentFactor: string;
  fiveWhys: FiveWhy[];
  fishbone: Fishbone;
};

export type RcaTemplate = {
  id: string;
  category: IncidentCategory;
  payload: RcaTemplatePayload;
};

export type CaActionType = "IMMEDIATE" | "SHORT_TERM" | "LONG_TERM";
export type CaPriority = "LOW" | "MEDIUM" | "HIGH";
export type CaStatus = "OPEN" | "IN_PROGRESS" | "DONE" | "OVERDUE" | "VERIFIED";

export type CorrectiveAction = {
  id: string;
  incidentId: string;
  description: string;
  actionType: CaActionType;
  priority: CaPriority;
  status: CaStatus;
  assigneeId: string;
  dueDate: string;
  completionNotes: string | null;
  completedAt: string | null;
  verifiedById: string | null;
  verifiedAt: string | null;
};

export type CaTrackerFilters = {
  status?: CaStatus;
  assigneeId?: string;
  priority?: CaPriority;
  locationId?: string;
  dueFrom?: string;
  dueTo?: string;
  view?: "table" | "kanban";
};

export function caTrackerQueryKey(filters: CaTrackerFilters) {
  return ["corrective-actions", filters] as const;
}

export type NotificationType =
  | "INCIDENT_SUBMITTED"
  | "INCIDENT_ESCALATED"
  | "INCIDENT_VERIFIED"
  | "INCIDENT_REJECTED"
  | "INCIDENT_CLOSED"
  | "SLA_WARNING"
  | "SLA_OVERDUE"
  | "CA_ASSIGNED"
  | "CA_DUE_SOON"
  | "CA_OVERDUE"
  | "REPORT_READY";

export type NotificationPriority = "INFO" | "MEDIUM" | "HIGH" | "CRITICAL";

export type AppNotification = {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  priority: NotificationPriority;
  isRead: boolean;
  referenceType: string;
  referenceId: string;
  createdAt: string;
};

export type DashboardFilters = {
  category?: IncidentCategory;
  locationId?: string;
  from?: string;
  to?: string;
};

export type DashboardPipeline = {
  PENDING_REVIEW: number;
  UNDER_INVESTIGATION: number;
  CORRECTIVE_ACTION: number;
  CLOSED: number;
};

export type DashboardSummary = {
  thisMonthCount: number;
  lastMonthCount: number;
  delta: number;
  trend: "up" | "down" | "flat" | string;
  caOverdueCount: number;
  pipeline: DashboardPipeline;
  ltifr?: number | null;
  trifr?: number | null;
  workHours?: number | null;
};

export type DashboardMonthBucket = {
  year: number;
  month: number;
  count: number;
};

export type DashboardHeatCell = {
  locationId: string;
  locationCode: string;
  category: IncidentCategory | string;
  count: number;
};

export function dashboardQueryKey(filters: DashboardFilters) {
  return ["dashboard", filters] as const;
}

export type ReportJob = {
  jobId?: string;
  id?: string;
  status: string;
  type?: string;
  storedKey?: string | null;
  downloadUrl?: string;
  error?: string | null;
};

export type ArchiveReport = {
  id: string;
  type: string;
  status: string;
  storedKey: string | null;
  createdAt: string;
  errorMessage: string | null;
  downloadUrl?: string;
};

export type AuditLog = {
  id: string;
  userId: string;
  userRole: string;
  ipAddress: string;
  entityType: string;
  entityId: string;
  action: string;
  createdAt: string;
};

export type AuditLogList = {
  items: AuditLog[];
  total: number;
  page: number;
  pageSize?: number;
};

export type AuditFilters = {
  from?: string;
  to?: string;
  userId?: string;
  entityType?: string;
  action?: string;
  page: number;
  pageSize: number;
};

export function auditLogsQueryKey(filters: AuditFilters) {
  return ["audit-logs", filters] as const;
}
