"use client";

import { useForm } from "@tanstack/react-form";
import { CircleHelp } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";

import { FormFields, SelectField, TextareaField, TextField } from "@/components/forms/fields";
import { FileUploader } from "@/components/incidents/FileUploader";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@/components/ui/popover";
import { mapIncidentError } from "@/lib/errors";
import { toDatetimeLocal } from "@/lib/datetime";
import {
  useCreateIncident,
  useIncident,
  useIncidentFiles,
  usePatchIncident,
  useSubmitIncident,
  useUploadIncidentFiles,
} from "@/lib/queries/useIncidents";
import { useLocations } from "@/lib/queries/useAdmin";
import {
  INCIDENT_CATEGORIES,
  INCIDENT_SEVERITIES,
  categoryGuides,
  categoryLabels,
  incidentFormFields,
  incidentFormSchema,
  severityLabels,
  type IncidentFormValues,
} from "@/lib/schemas";
import type { Incident } from "@/lib/types";

const STEPS = ["Info dasar", "Korban & saksi", "Unggah bukti", "Tinjau & kirim"] as const;

function emptyValues(): IncidentFormValues {
  return {
    title: "",
    description: "",
    category: "NEAR_MISS",
    severity: "LOW",
    incidentDatetime: toDatetimeLocal(new Date().toISOString()),
    locationId: "",
    areaId: "",
    hasVictim: false,
    victimName: "",
    victimPosition: "",
    injuryDescription: "",
    initialTreatment: "",
    witnesses: [{ name: "", position: "" }],
  };
}

function fromIncident(incident: Incident): IncidentFormValues {
  return {
    title: incident.title,
    description: incident.description,
    category: incident.category,
    severity: incident.severity,
    incidentDatetime: toDatetimeLocal(incident.incidentDatetime),
    locationId: incident.locationId,
    areaId: incident.areaId ?? "",
    hasVictim: incident.hasVictim,
    victimName: incident.victimName ?? "",
    victimPosition: incident.victimPosition ?? "",
    injuryDescription: incident.injuryDescription ?? "",
    initialTreatment: incident.initialTreatment ?? "",
    witnesses:
      incident.witnesses && incident.witnesses.length > 0
        ? incident.witnesses
        : [{ name: "", position: "" }],
  };
}

function validateStep(step: number, values: IncidentFormValues) {
  if (step === 0) {
    return incidentFormFields
      .pick({
        title: true,
        description: true,
        category: true,
        severity: true,
        incidentDatetime: true,
        locationId: true,
      })
      .safeParse(values);
  }
  if (step === 1) {
    return incidentFormSchema.safeParse(values);
  }
  return incidentFormSchema.safeParse(values);
}

export function IncidentForm({ incidentId }: { incidentId?: string }) {
  const router = useRouter();
  const [step, setStep] = useState(0);
  const [savedId, setSavedId] = useState<string | undefined>(incidentId);
  const [formError, setFormError] = useState<string | null>(null);
  const skipServerHydrate = useRef(false);
  const locations = useLocations();
  const existing = useIncident(savedId);
  const files = useIncidentFiles(savedId);
  const createIncident = useCreateIncident();
  const patchIncident = usePatchIncident();
  const submitIncident = useSubmitIncident();
  const uploadFiles = useUploadIncidentFiles();

  const sites = useMemo(
    () => (locations.data ?? []).filter((site) => site.isActive),
    [locations.data],
  );

  const form = useForm({
    defaultValues: emptyValues(),
    onSubmit: async () => undefined,
  });

  useEffect(() => {
    if (!existing.data?.title) return;
    if (skipServerHydrate.current) return;
    form.reset(fromIncident(existing.data));
    skipServerHydrate.current = true;
  }, [existing.data, form]);

  const submittedLocked =
    existing.data &&
    existing.data.status !== "DRAFT" &&
    existing.data.status !== "REJECTED";

  async function persist(values: IncidentFormValues) {
    skipServerHydrate.current = true;
    if (savedId) {
      return patchIncident.mutateAsync({ id: savedId, values });
    }
    const created = await createIncident.mutateAsync(values);
    setSavedId(created.id);
    router.replace(`/incidents/new?id=${created.id}`);
    return created;
  }

  async function handleSaveDraft() {
    setFormError(null);
    const values = form.state.values;
    const parsed = incidentFormSchema.safeParse(values);
    if (!parsed.success) {
      setFormError(parsed.error.issues[0]?.message ?? "Lengkapi formulir sebelum menyimpan.");
      return;
    }
    try {
      await persist(parsed.data);
    } catch (error) {
      setFormError(mapIncidentError(error, "Gagal menyimpan draft."));
    }
  }

  async function goNext() {
    setFormError(null);
    const parsed = validateStep(step, form.state.values);
    if (!parsed.success) {
      setFormError(parsed.error.issues[0]?.message ?? "Lengkapi langkah ini.");
      return;
    }
    if (step === 1) {
      const full = incidentFormSchema.safeParse(form.state.values);
      if (!full.success) {
        setFormError(full.error.issues[0]?.message ?? "Lengkapi langkah ini.");
        return;
      }
      try {
        await persist(full.data);
      } catch (error) {
        setFormError(mapIncidentError(error, "Gagal menyimpan draft."));
        return;
      }
    }
    setStep((current) => Math.min(current + 1, STEPS.length - 1));
  }

  async function handleSubmitReport() {
    setFormError(null);
    const parsed = incidentFormSchema.safeParse(form.state.values);
    if (!parsed.success) {
      setFormError(parsed.error.issues[0]?.message ?? "Lengkapi formulir.");
      return;
    }
    try {
      const saved = await persist(parsed.data);
      await submitIncident.mutateAsync(saved.id);
      router.push(`/incidents/${saved.id}`);
    } catch (error) {
      setFormError(mapIncidentError(error, "Gagal mengirim laporan."));
    }
  }

  const categoryOptions = INCIDENT_CATEGORIES.map((value) => ({
    value,
    label: categoryLabels[value],
  }));
  const severityOptions = INCIDENT_SEVERITIES.map((value) => ({
    value,
    label: severityLabels[value],
  }));
  const locationOptions = sites.map((site) => ({
    value: site.id,
    label: `${site.code} — ${site.name}`,
  }));
  const selectedSite = sites.find((site) => site.id === form.state.values.locationId);
  const areaOptions = (selectedSite?.areas ?? []).map((area) => ({
    value: area.id,
    label: `${area.code} — ${area.name}`,
  }));

  const pending =
    createIncident.isPending ||
    patchIncident.isPending ||
    submitIncident.isPending ||
    uploadFiles.isPending;

  return (
    <div className="flex flex-col gap-6">
      <ol className="flex flex-wrap gap-2 text-xs text-subtle">
        {STEPS.map((label, index) => (
          <li
            key={label}
            className={
              index === step
                ? "rounded-md bg-orange-50 px-2 py-1 font-medium text-orange-700"
                : "rounded-md px-2 py-1"
            }
          >
            {index + 1}. {label}
          </li>
        ))}
      </ol>

      {submittedLocked ? (
        <p className="text-sm text-subtle">
          Laporan sudah dikirim dan tidak dapat diubah dari formulir ini.
        </p>
      ) : (
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault();
          }}
        >
          {step === 0 ? (
            <FormFields>
              <TextField form={form} name="title" label="Judul singkat" schema={incidentFormFields.shape.title} />
              <TextField
                form={form}
                name="incidentDatetime"
                label="Tanggal dan waktu kejadian"
                type="datetime-local"
                schema={incidentFormFields.shape.incidentDatetime}
              />
              <SelectField
                form={form}
                name="locationId"
                label="Lokasi"
                schema={incidentFormFields.shape.locationId}
                placeholder="Pilih lokasi"
                options={locationOptions}
              />
              {areaOptions.length > 0 ? (
                <SelectField
                  form={form}
                  name="areaId"
                  label="Area kerja"
                  schema={incidentFormFields.shape.areaId}
                  placeholder="Opsional"
                  options={areaOptions}
                />
              ) : null}
              <div className="flex items-end gap-2">
                <div className="min-w-0 flex-1">
                  <SelectField
                    form={form}
                    name="category"
                    label="Kategori"
                    schema={incidentFormFields.shape.category}
                    placeholder="Pilih kategori"
                    options={categoryOptions}
                  />
                </div>
                <Popover>
                  <PopoverTrigger
                    render={
                      <Button type="button" variant="ghost" size="icon" className="min-h-11 min-w-11" />
                    }
                  >
                    <CircleHelp />
                    <span className="sr-only">Panduan kategori</span>
                  </PopoverTrigger>
                  <PopoverContent className="w-80 rounded-md border border-border p-3 text-sm shadow-sm">
                    <PopoverHeader>
                      <PopoverTitle>Panduan kategori</PopoverTitle>
                      <PopoverDescription>
                        {categoryGuides[form.state.values.category]}
                      </PopoverDescription>
                    </PopoverHeader>
                    <ul className="mt-2 flex flex-col gap-2 text-sm text-ink">
                      {INCIDENT_CATEGORIES.map((category) => (
                        <li key={category}>
                          <span className="font-medium">{categoryLabels[category]}.</span>{" "}
                          {categoryGuides[category]}
                        </li>
                      ))}
                    </ul>
                  </PopoverContent>
                </Popover>
              </div>
              <SelectField
                form={form}
                name="severity"
                label="Tingkat keparahan"
                schema={incidentFormFields.shape.severity}
                placeholder="Pilih keparahan"
                options={severityOptions}
              />
              <TextareaField
                form={form}
                name="description"
                label="Deskripsi kejadian"
                hint="Minimal 50 karakter. Ceritakan apa yang terjadi secara ringkas."
                schema={incidentFormFields.shape.description}
              />
            </FormFields>
          ) : null}

          {step === 1 ? (
            <FormFields>
              <form.Field name="hasVictim">
                {(field) => (
                  <Field>
                    <div className="flex items-center gap-2">
                      <Checkbox
                        id="hasVictim"
                        checked={Boolean(field.state.value)}
                        onCheckedChange={(checked) => field.handleChange(Boolean(checked))}
                      />
                      <FieldLabel htmlFor="hasVictim">Ada korban</FieldLabel>
                    </div>
                  </Field>
                )}
              </form.Field>
              {form.state.values.hasVictim ? (
                <>
                  <TextField
                    form={form}
                    name="victimName"
                    label="Nama korban"
                    schema={incidentFormFields.shape.victimName}
                  />
                  <TextField
                    form={form}
                    name="victimPosition"
                    label="Jabatan / divisi"
                    schema={incidentFormFields.shape.victimPosition}
                  />
                  <TextareaField
                    form={form}
                    name="injuryDescription"
                    label="Jenis cedera / kondisi"
                    schema={incidentFormFields.shape.injuryDescription}
                  />
                  <TextareaField
                    form={form}
                    name="initialTreatment"
                    label="Penanganan awal"
                    schema={incidentFormFields.shape.initialTreatment}
                  />
                </>
              ) : null}
              <form.Field name="witnesses">
                {(field) => {
                  const rows = field.state.value as IncidentFormValues["witnesses"];
                  return (
                    <Field>
                      <FieldLabel>Saksi (opsional)</FieldLabel>
                      <div className="flex flex-col gap-3">
                        {rows.map((row, index) => (
                          <div key={index} className="grid gap-2 md:grid-cols-2">
                            <div className="flex flex-col gap-1">
                              <FieldLabel htmlFor={`witness-name-${index}`}>Nama saksi</FieldLabel>
                              <Input
                                id={`witness-name-${index}`}
                                className="h-11 min-h-11 rounded-md"
                                value={row.name}
                                onChange={(event) => {
                                  const next = rows.map((item, i) =>
                                    i === index ? { ...item, name: event.target.value } : item,
                                  );
                                  field.handleChange(next);
                                }}
                              />
                            </div>
                            <div className="flex flex-col gap-1">
                              <FieldLabel htmlFor={`witness-position-${index}`}>Jabatan saksi</FieldLabel>
                              <Input
                                id={`witness-position-${index}`}
                                className="h-11 min-h-11 rounded-md"
                                value={row.position}
                                onChange={(event) => {
                                  const next = rows.map((item, i) =>
                                    i === index ? { ...item, position: event.target.value } : item,
                                  );
                                  field.handleChange(next);
                                }}
                              />
                            </div>
                          </div>
                        ))}
                        <Button
                          type="button"
                          variant="ghost"
                          className="min-h-11 w-fit"
                          onClick={() => field.handleChange([...rows, { name: "", position: "" }])}
                        >
                          Tambah saksi
                        </Button>
                      </div>
                    </Field>
                  );
                }}
              </form.Field>
            </FormFields>
          ) : null}

          {step === 2 ? (
            <div className="flex flex-col gap-3">
              <p className="text-xs text-subtle">
                Simpan draft dulu agar file bisa dilampirkan ke nomor laporan.
              </p>
              {savedId ? (
                <FileUploader
                  files={files.data ?? []}
                  canUpload
                  uploading={uploadFiles.isPending}
                  onUpload={async (picked) => {
                    await uploadFiles.mutateAsync({ id: savedId, files: picked });
                  }}
                  onRefreshFiles={async () => (await files.refetch()).data}
                />
              ) : (
                <Button type="button" variant="secondary" className="min-h-11 w-fit" onClick={() => void handleSaveDraft()}>
                  Simpan draft untuk unggah
                </Button>
              )}
            </div>
          ) : null}

          {step === 3 ? (
            <div className="flex flex-col gap-3 rounded-lg border border-border bg-white p-4">
              <h2 className="text-xl font-semibold text-ink">Tinjau laporan</h2>
              <p className="text-sm text-ink">{form.state.values.title}</p>
              <p className="text-sm leading-relaxed text-ink">{form.state.values.description}</p>
              <p className="text-xs text-subtle">
                {categoryLabels[form.state.values.category]} · {severityLabels[form.state.values.severity]}
              </p>
              {form.state.values.hasVictim ? (
                <p className="text-sm text-ink">Korban: {form.state.values.victimName}</p>
              ) : (
                <p className="text-sm text-subtle">Tidak ada korban.</p>
              )}
            </div>
          ) : null}

          {formError ? (
            <p role="alert" className="text-xs text-danger-500">
              {formError}
            </p>
          ) : null}

          <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
            {step > 0 ? (
              <Button
                type="button"
                variant="secondary"
                className="min-h-11"
                onClick={() => setStep((current) => current - 1)}
              >
                Kembali
              </Button>
            ) : null}
            {step < 3 ? (
              <Button type="button" className="min-h-11" disabled={pending} onClick={() => void goNext()}>
                Lanjut
              </Button>
            ) : (
              <Button type="button" className="min-h-11" disabled={pending} onClick={() => void handleSubmitReport()}>
                Kirim laporan
              </Button>
            )}
            <Button
              type="button"
              variant="ghost"
              className="min-h-11"
              data-testid="save-draft"
              disabled={pending}
              onClick={() => void handleSaveDraft()}
            >
              Simpan draft
            </Button>
          </div>
        </form>
      )}
    </div>
  );
}
