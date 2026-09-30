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
