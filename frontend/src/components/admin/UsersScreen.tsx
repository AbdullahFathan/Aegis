"use client";

import { useForm } from "@tanstack/react-form";
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
} from "@tanstack/react-table";
import { useEffect, useMemo, useState } from "react";

import { FormFields, SelectField, TextField } from "@/components/forms/fields";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { LoadingBlock, PageHeader, QueryError } from "@/components/shared/EmptyState";
import { ApiError } from "@/lib/api/client";
import { mapApiError } from "@/lib/errors";
import { useCreateUser, usePatchUser, useUsers } from "@/lib/queries/useAdmin";
import { isAdminRole } from "@/lib/rbac";
import { useSession } from "@/lib/queries/useSession";
import {
  ROLES,
  USER_STATUSES,
  roleLabels,
  statusLabels,
  userFormSchema,
  userPatchSchema,
  type Role,
  type UserStatus,
} from "@/lib/schemas";
import type { User, UserFilters } from "@/lib/types";

const roleOptions = ROLES.map((role) => ({ value: role, label: roleLabels[role] }));
const statusOptions = USER_STATUSES.map((status) => ({
  value: status,
  label: statusLabels[status],
}));

function RoleBadge({ role }: { role: string }) {
  const label = role in roleLabels ? roleLabels[role as Role] : role;
  return (
    <span className="inline-flex rounded-md border border-border-strong bg-canvas px-2 py-0.5 text-xs font-medium text-ink">
      {label}
    </span>
  );
}

export function UsersScreen() {
  const session = useSession();
  const admin = isAdminRole(session.data?.role ?? "");
  const [filters, setFilters] = useState<UserFilters>({ page: 1, pageSize: 20 });
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const users = useUsers(filters);
  const createUser = useCreateUser();
  const patchUser = usePatchUser();

  const columns = useMemo<ColumnDef<User>[]>(
    () => [
      { accessorKey: "name", header: "Nama" },
      { accessorKey: "email", header: "Email" },
      {
        accessorKey: "role",
        header: "Role",
        cell: ({ row }) => <RoleBadge role={row.original.role} />,
      },
      {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) =>
          row.original.status in statusLabels
            ? statusLabels[row.original.status as UserStatus]
            : row.original.status,
      },
      {
        id: "actions",
        header: () => <span className="block text-right">Aksi</span>,
        cell: ({ row }) => (
          <div className="text-right">
            <Button variant="ghost" size="sm" onClick={() => setEditing(row.original)}>
              Ubah
            </Button>
          </div>
        ),
      },
    ],
    [],
  );

  // TanStack Table returns unstable function identities; the React Compiler skips this call.
  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({
    data: users.data?.items ?? [],
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const createForm = useForm({
    defaultValues: {
      name: "",
      email: "",
      password: "",
      role: "REPORTER" as Role,
      status: "ACTIVE" as UserStatus,
    },
    onSubmit: async ({ value }) => {
      const parsed = userFormSchema.safeParse(value);
      if (!parsed.success) return;
      await createUser.mutateAsync(parsed.data);
      setCreateOpen(false);
      createForm.reset();
    },
  });

  const editForm = useForm({
    defaultValues: {
      name: "",
      role: "REPORTER" as Role,
      status: "ACTIVE" as UserStatus,
    },
    onSubmit: async ({ value }) => {
      if (!editing) return;
      const parsed = userPatchSchema.safeParse(value);
      if (!parsed.success) return;
      await patchUser.mutateAsync({ id: editing.id, ...parsed.data });
      setEditing(null);
    },
  });

  useEffect(() => {
    if (!editing) return;
    editForm.reset({
      name: editing.name,
      role: editing.role as Role,
      status: editing.status as UserStatus,
    });
  }, [editForm, editing]);

  const forbidden = users.error instanceof ApiError && users.error.status === 403;
  const total = users.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / filters.pageSize));

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Users"
        actions={
          admin ? (
            <Button onClick={() => setCreateOpen(true)}>Tambah user</Button>
          ) : null
        }
      />
      {users.isPending ? <LoadingBlock /> : null}
      {users.isError ? (
        <QueryError
          message={
            forbidden
              ? "Anda tidak memiliki akses ke halaman ini."
              : mapApiError(users.error, "Gagal memuat daftar user.")
          }
        />
      ) : null}
      {users.data ? (
        users.data.items.length === 0 ? (
          <p className="text-sm text-subtle">Belum ada user pada filter ini.</p>
        ) : (
          <div className="overflow-hidden rounded-lg border border-border bg-white">
            <Table>
              <TableHeader className="bg-canvas">
                {table.getHeaderGroups().map((group) => (
                  <TableRow key={group.id} className="hover:bg-transparent">
                    {group.headers.map((header) => (
                      <TableHead
                        key={header.id}
                        className="h-10 px-4 text-xs font-medium tracking-wide text-subtle uppercase"
                      >
                        {header.isPlaceholder
                          ? null
                          : flexRender(header.column.columnDef.header, header.getContext())}
                      </TableHead>
                    ))}
                  </TableRow>
                ))}
              </TableHeader>
              <TableBody>
                {table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id} className="h-12 hover:bg-canvas">
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id} className="px-4 text-sm text-ink">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )
      ) : null}
      {users.data ? (
        <div className="flex items-center justify-between text-sm text-subtle">
          <span>
            Halaman {filters.page} dari {pageCount}
          </span>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={filters.page <= 1}
              onClick={() => setFilters((current) => ({ ...current, page: current.page - 1 }))}
            >
              Sebelumnya
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={filters.page >= pageCount}
              onClick={() => setFilters((current) => ({ ...current, page: current.page + 1 }))}
            >
              Berikutnya
            </Button>
          </div>
        </div>
      ) : null}

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="rounded-lg border border-border shadow-modal sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Tambah user</DialogTitle>
            <DialogDescription>Akun baru dapat masuk setelah dibuat.</DialogDescription>
          </DialogHeader>
          <form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              void createForm.handleSubmit();
            }}
          >
            <FormFields>
              <TextField form={createForm} name="name" label="Nama" schema={userFormSchema.shape.name} />
              <TextField form={createForm} name="email" label="Email" schema={userFormSchema.shape.email} />
              <TextField
                form={createForm}
                name="password"
                label="Password"
                type="password"
                schema={userFormSchema.shape.password}
              />
              <SelectField
                form={createForm}
                name="role"
                label="Role"
                schema={userFormSchema.shape.role}
                placeholder="Pilih role"
                options={roleOptions}
              />
              <SelectField
                form={createForm}
                name="status"
                label="Status"
                schema={userFormSchema.shape.status}
                placeholder="Pilih status"
                options={statusOptions}
              />
            </FormFields>
            {createUser.isError ? (
              <p role="alert" className="text-xs text-danger-500">
                {mapApiError(createUser.error, "Gagal membuat user.")}
              </p>
            ) : null}
            <DialogFooter className="border-border bg-white">
              <Button type="submit" disabled={createUser.isPending}>
                Simpan
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog
        open={editing !== null}
        onOpenChange={(open) => {
          if (!open) setEditing(null);
        }}
      >
        <DialogContent className="rounded-lg border border-border shadow-modal sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Ubah user</DialogTitle>
            <DialogDescription>Perbarui role atau status akun.</DialogDescription>
          </DialogHeader>
          {editing ? (
            <form
              className="flex flex-col gap-4"
              onSubmit={(event) => {
                event.preventDefault();
                void editForm.handleSubmit();
              }}
            >
              <FormFields>
                <TextField form={editForm} name="name" label="Nama" schema={userPatchSchema.shape.name} />
                <SelectField
                  form={editForm}
                  name="role"
                  label="Role"
                  schema={userPatchSchema.shape.role}
                  placeholder="Pilih role"
                  options={roleOptions}
                />
                <SelectField
                  form={editForm}
                  name="status"
                  label="Status"
                  schema={userPatchSchema.shape.status}
                  placeholder="Pilih status"
                  options={statusOptions}
                />
              </FormFields>
              {patchUser.isError ? (
                <p role="alert" className="text-xs text-danger-500">
                  {mapApiError(patchUser.error, "Gagal memperbarui user.")}
                </p>
              ) : null}
              <DialogFooter className="border-border bg-white">
                <Button type="submit" disabled={patchUser.isPending}>
                  Simpan
                </Button>
              </DialogFooter>
            </form>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
