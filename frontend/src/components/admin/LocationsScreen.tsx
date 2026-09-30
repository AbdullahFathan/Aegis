"use client";

import { useForm } from "@tanstack/react-form";
import { useState } from "react";

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
import { EmptyState, LoadingBlock, PageHeader, QueryError } from "@/components/shared/EmptyState";
import { ApiError } from "@/lib/api/client";
import { mapApiError } from "@/lib/errors";
import {
  useCreateArea,
  useCreateLocation,
  useCreateRegion,
  useDeactivateLocation,
  useLocations,
  useRegions,
  useUsers,
} from "@/lib/queries/useAdmin";
import { isAdminRole } from "@/lib/rbac";
import { useSession } from "@/lib/queries/useSession";
import {
  LOCATION_TYPES,
  areaSchema,
  locationSchema,
  locationTypeLabels,
  regionSchema,
  roleLabels,
  type LocationType,
  type Role,
} from "@/lib/schemas";

export function LocationsScreen() {
  const session = useSession();
  const admin = isAdminRole(session.data?.role ?? "");
  const regions = useRegions();
  const locations = useLocations();
  const people = useUsers({ page: 1, pageSize: 100 });
  const createRegion = useCreateRegion();
  const createLocation = useCreateLocation();
  const createArea = useCreateArea();
  const deactivate = useDeactivateLocation();
  const [siteOpen, setSiteOpen] = useState(false);
  const [areaFor, setAreaFor] = useState<string | null>(null);
  const [confirmId, setConfirmId] = useState<string | null>(null);

  const userOptions = (people.data?.items ?? [])
    .filter((user) => user.status === "ACTIVE")
    .map((user) => ({
      value: user.id,
      label: `${user.name} · ${roleLabels[user.role as Role] ?? user.role}`,
    }));
  const regionOptions = [
    { value: "", label: "Tanpa wilayah" },
    ...(regions.data ?? []).map((region) => ({
      value: region.id,
      label: `${region.name} (${region.code})`,
    })),
  ];
  const names = new Map((people.data?.items ?? []).map((user) => [user.id, user.name]));
  const regionNames = new Map((regions.data ?? []).map((region) => [region.id, region.name]));

  const regionForm = useForm({
    defaultValues: { name: "", code: "" },
    onSubmit: async ({ value }) => {
      const parsed = regionSchema.safeParse(value);
      if (!parsed.success) return;
      await createRegion.mutateAsync(parsed.data);
      regionForm.reset();
    },
  });

  const siteForm = useForm({
    defaultValues: {
      name: "",
      code: "",
      type: "TAMBANG" as LocationType,
      regionId: "",
      supervisorId: "",
      hseOfficerId: "",
    },
    onSubmit: async ({ value }) => {
      const parsed = locationSchema.safeParse(value);
      if (!parsed.success) return;
      await createLocation.mutateAsync(parsed.data);
      setSiteOpen(false);
      siteForm.reset();
    },
  });

  const areaForm = useForm({
    defaultValues: { name: "", code: "" },
    onSubmit: async ({ value }) => {
      if (!areaFor) return;
      const parsed = areaSchema.safeParse(value);
      if (!parsed.success) return;
      await createArea.mutateAsync({ ...parsed.data, locationId: areaFor });
      setAreaFor(null);
      areaForm.reset();
    },
  });

  const error = regions.error ?? locations.error;
  const forbidden = error instanceof ApiError && error.status === 403;

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Lokasi"
        actions={
          admin ? (
            <Button onClick={() => setSiteOpen(true)}>Daftarkan lokasi</Button>
          ) : null
        }
      />
      {regions.isPending || locations.isPending ? <LoadingBlock /> : null}
      {error ? (
        <QueryError
          message={
            forbidden
              ? "Anda tidak memiliki akses ke halaman ini."
              : mapApiError(error, "Gagal memuat lokasi.")
          }
        />
      ) : null}

      {regions.data && locations.data && locations.data.length === 0 ? (
        <EmptyState
          title="Belum ada lokasi"
          description="Daftarkan wilayah, lalu site dan area kerja yang dipakai saat pelaporan insiden."
          action={
            admin ? (
              <Button onClick={() => setSiteOpen(true)}>Daftarkan lokasi</Button>
            ) : null
          }
        />
      ) : null}

      {admin && regions.data ? (
        <section className="rounded-lg border border-border bg-white p-4 shadow-card">
          <h2 className="mb-4 text-xl font-semibold text-ink">Wilayah</h2>
          <form
            className="mb-4 grid grid-cols-1 gap-4 md:grid-cols-2"
            onSubmit={(event) => {
              event.preventDefault();
              void regionForm.handleSubmit();
            }}
          >
            <FormFields>
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <TextField form={regionForm} name="name" label="Nama wilayah" schema={regionSchema.shape.name} />
                <TextField form={regionForm} name="code" label="Kode" schema={regionSchema.shape.code} />
              </div>
            </FormFields>
            {createRegion.isError ? (
              <p role="alert" className="text-xs text-danger-500 md:col-span-2">
                {mapApiError(createRegion.error, "Gagal menyimpan wilayah.")}
              </p>
            ) : null}
            <div className="md:col-span-2">
              <Button type="submit" variant="outline" disabled={createRegion.isPending}>
                Simpan wilayah
              </Button>
            </div>
          </form>
          {regions.data.length === 0 ? (
            <p className="text-sm text-subtle">Belum ada wilayah.</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {regions.data.map((region) => (
                <li key={region.id} className="text-sm text-ink">
                  <span className="font-mono text-xs text-subtle">{region.code}</span> {region.name}
                </li>
              ))}
            </ul>
          )}
        </section>
      ) : null}

      {locations.data && locations.data.length > 0 ? (
        <section className="flex flex-col gap-4">
          <h2 className="text-xl font-semibold text-ink">Site dan area kerja</h2>
          {locations.data.map((site) => (
            <article key={site.id} className="rounded-lg border border-border bg-white p-4 shadow-card">
              <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                <div className="flex flex-col gap-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="text-base font-medium text-ink">{site.name}</h3>
                    <span className="font-mono text-xs text-subtle">{site.code}</span>
                    <span className="rounded-md border border-border-strong bg-canvas px-2 py-0.5 text-xs text-ink">
                      {locationTypeLabels[site.type as LocationType] ?? site.type}
                    </span>
                    {!site.isActive ? (
                      <span className="rounded-md border border-border-strong px-2 py-0.5 text-xs text-subtle">
                        Nonaktif
                      </span>
                    ) : null}
                  </div>
                  <p className="text-xs text-subtle">
                    Wilayah: {site.regionId ? (regionNames.get(site.regionId) ?? site.regionId) : "—"} · Supervisor:{" "}
                    {names.get(site.supervisorId) ?? site.supervisorId} · HSE Officer:{" "}
                    {names.get(site.hseOfficerId) ?? site.hseOfficerId}
                  </p>
                </div>
                {admin && site.isActive ? (
                  <div className="flex gap-2">
                    <Button variant="ghost" size="sm" onClick={() => setAreaFor(site.id)}>
                      Tambah area
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => setConfirmId(site.id)}>
                      Nonaktifkan
                    </Button>
                  </div>
                ) : null}
              </div>
              <ul className="mt-3 flex flex-col gap-1">
                {site.areas.length === 0 ? (
                  <li className="text-sm text-subtle">Belum ada area kerja.</li>
                ) : (
                  site.areas.map((area) => (
                    <li key={area.id} className="text-sm text-ink">
                      <span className="font-mono text-xs text-subtle">{area.code}</span> {area.name}
                    </li>
                  ))
                )}
              </ul>
            </article>
          ))}
        </section>
      ) : null}

      <Dialog open={siteOpen} onOpenChange={setSiteOpen}>
        <DialogContent className="rounded-lg border border-border shadow-modal sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Daftarkan lokasi</DialogTitle>
            <DialogDescription>Site membutuhkan supervisor dan HSE Officer.</DialogDescription>
          </DialogHeader>
          <form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              void siteForm.handleSubmit();
            }}
          >
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <TextField form={siteForm} name="name" label="Nama" schema={locationSchema.shape.name} />
              <TextField form={siteForm} name="code" label="Kode" schema={locationSchema.shape.code} />
              <SelectField
                form={siteForm}
                name="type"
                label="Tipe"
                schema={locationSchema.shape.type}
                placeholder="Pilih tipe"
                options={LOCATION_TYPES.map((type) => ({
                  value: type,
                  label: locationTypeLabels[type],
                }))}
              />
              <SelectField
                form={siteForm}
                name="regionId"
                label="Wilayah"
                schema={locationSchema.shape.regionId}
                placeholder="Pilih wilayah"
                options={regionOptions.filter((option) => option.value !== "")}
              />
              <SelectField
                form={siteForm}
                name="supervisorId"
                label="Supervisor"
                schema={locationSchema.shape.supervisorId}
                placeholder="Pilih supervisor"
                options={userOptions}
              />
              <SelectField
                form={siteForm}
                name="hseOfficerId"
                label="HSE Officer"
                schema={locationSchema.shape.hseOfficerId}
                placeholder="Pilih HSE Officer"
                options={userOptions}
              />
            </div>
            {createLocation.isError ? (
              <p role="alert" className="text-xs text-danger-500">
                {mapApiError(createLocation.error, "Gagal menyimpan lokasi.")}
              </p>
            ) : null}
            <DialogFooter className="border-border bg-white">
              <Button type="submit" disabled={createLocation.isPending}>
                Simpan
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={areaFor !== null} onOpenChange={(open) => !open && setAreaFor(null)}>
        <DialogContent className="rounded-lg border border-border shadow-modal sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Tambah area kerja</DialogTitle>
            <DialogDescription>Area berada di bawah site yang dipilih.</DialogDescription>
          </DialogHeader>
          <form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              void areaForm.handleSubmit();
            }}
          >
            <TextField form={areaForm} name="name" label="Nama" schema={areaSchema.shape.name} />
            <TextField form={areaForm} name="code" label="Kode" schema={areaSchema.shape.code} />
            {createArea.isError ? (
              <p role="alert" className="text-xs text-danger-500">
                {mapApiError(createArea.error, "Gagal menyimpan area.")}
              </p>
            ) : null}
            <DialogFooter className="border-border bg-white">
              <Button type="submit" disabled={createArea.isPending}>
                Simpan
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={confirmId !== null} onOpenChange={(open) => !open && setConfirmId(null)}>
        <DialogContent className="rounded-lg border border-border shadow-modal sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Nonaktifkan lokasi</DialogTitle>
            <DialogDescription>
              Lokasi tidak dihapus permanen. Site yang nonaktif tidak dipilih untuk laporan baru.
            </DialogDescription>
          </DialogHeader>
          {deactivate.isError ? (
            <p role="alert" className="text-xs text-danger-500">
              {mapApiError(deactivate.error, "Gagal menonaktifkan lokasi.")}
            </p>
          ) : null}
          <DialogFooter className="border-border bg-white">
            <Button variant="outline" onClick={() => setConfirmId(null)}>
              Batal
            </Button>
            <Button
              variant="destructive"
              disabled={deactivate.isPending}
              onClick={() => {
                if (!confirmId) return;
                deactivate.mutate(confirmId, { onSuccess: () => setConfirmId(null) });
              }}
            >
              Nonaktifkan
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
