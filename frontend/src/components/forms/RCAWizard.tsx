"use client";

import { useForm } from "@tanstack/react-form";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { FormFields, TextareaField } from "@/components/forms/fields";
import { RcaReadView } from "@/components/rca/RcaReadView";
import { Button, buttonVariants } from "@/components/ui/button";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { QueryError } from "@/components/shared/EmptyState";
import { mapApiError } from "@/lib/errors";
import { useIncident } from "@/lib/queries/useIncidents";
import { useRca, useRcaTemplate, useSaveRcaTemplate, useUpsertRca } from "@/lib/queries/useRca";
import { useSession } from "@/lib/queries/useSession";
import { canWriteRca } from "@/lib/rbac";
import {
  emptyFishbone,
  emptyRcaValues,
  FISHBONE_KEYS,
  fishboneLabels,
  rcaSchema,
  type RcaFormValues,
} from "@/lib/schemas";
import { cn } from "@/lib/utils";
import type { IncidentStatus, Rca } from "@/lib/types";

const STEPS = ["Kronologi", "5-Why", "Fishbone", "Tinjau"] as const;

function fromRca(rca: Rca | null | undefined): RcaFormValues {
  if (!rca) return emptyRcaValues();
  const fiveWhys =
    rca.fiveWhys && rca.fiveWhys.length > 0 ? rca.fiveWhys.slice(0, 5) : [{ why: "", answer: "" }];
  return {
    timeline: rca.timeline ?? "",
    humanFactor: rca.humanFactor ?? "",
    environmentFactor: rca.environmentFactor ?? "",
    equipmentFactor: rca.equipmentFactor ?? "",
    fiveWhys,
    fishbone: { ...emptyFishbone, ...(rca.fishbone ?? {}) },
  };
}

export function RCAWizard({ incidentId }: { incidentId: string }) {
  const router = useRouter();
  const session = useSession();
  const incident = useIncident(incidentId);
  const rca = useRca(incidentId);
  const template = useRcaTemplate(incident.data?.category);
  const upsert = useUpsertRca(incidentId);
  const saveTemplate = useSaveRcaTemplate();
  const [step, setStep] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const writable = Boolean(
    session.data &&
      incident.data &&
      canWriteRca(session.data.role, incident.data.status as IncidentStatus),
  );

  const form = useForm({
    defaultValues: emptyRcaValues(),
    onSubmit: async ({ value }) => {
      setError(null);
      try {
        await upsert.mutateAsync({ ...value, completed: true });
        router.push(`/incidents/${incidentId}?tab=rca`);
      } catch (cause) {
        setError(mapApiError(cause, "Gagal menyimpan RCA."));
      }
    },
  });

  useEffect(() => {
    if (rca.data === undefined) return;
    form.reset(fromRca(rca.data));
  }, [rca.data, form]);

  if (rca.isPending || incident.isPending) {
    return <p className="text-sm text-subtle">Memuat investigasi…</p>;
  }

  if (rca.isError) {
    return <QueryError message={mapApiError(rca.error, "Gagal memuat RCA.")} />;
  }

  if (!writable) {
    return (
      <div className="flex flex-col gap-4">
        <h1 className="text-xl font-semibold text-ink">Investigasi RCA</h1>
        {rca.data ? (
          <RcaReadView rca={rca.data} />
        ) : (
          <p className="text-sm text-subtle">Belum diisi.</p>
        )}
        <Link href={`/incidents/${incidentId}?tab=rca`} className={cn(buttonVariants({ variant: "secondary" }), "w-fit")}>
          Kembali ke laporan
        </Link>
      </div>
    );
  }

  return (
    <form
      className="flex flex-col gap-6"
      onSubmit={(event) => {
        event.preventDefault();
        if (step < STEPS.length - 1) {
          setStep(step + 1);
          return;
        }
        void form.handleSubmit();
      }}
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-xl font-semibold text-ink">Investigasi RCA</h1>
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="ghost"
            className="min-h-11"
            disabled={!template.data?.payload}
            onClick={() => {
              const payload = template.data?.payload;
              if (!payload) return;
              form.setFieldValue("timeline", payload.timeline ?? "");
              form.setFieldValue("humanFactor", payload.humanFactor ?? "");
              form.setFieldValue("environmentFactor", payload.environmentFactor ?? "");
              form.setFieldValue("equipmentFactor", payload.equipmentFactor ?? "");
              form.setFieldValue(
                "fiveWhys",
                payload.fiveWhys?.length ? payload.fiveWhys.slice(0, 5) : [{ why: "", answer: "" }],
              );
              form.setFieldValue("fishbone", { ...emptyFishbone, ...payload.fishbone });
              setNotice("Template kategori diterapkan.");
            }}
          >
            Pakai template
          </Button>
          <Button
            type="button"
            variant="secondary"
            className="min-h-11"
            disabled={!incident.data || saveTemplate.isPending}
            onClick={async () => {
              if (!incident.data) return;
              setError(null);
              try {
                await saveTemplate.mutateAsync({
                  category: incident.data.category,
                  payload: form.state.values,
                });
                setNotice("Template kategori disimpan.");
              } catch (cause) {
                setError(mapApiError(cause, "Gagal menyimpan template."));
              }
            }}
          >
            Simpan sebagai template
          </Button>
        </div>
      </div>
      <p className="text-xs text-subtle">
        Langkah {step + 1} dari {STEPS.length}: {STEPS[step]}
      </p>
      {step === 0 ? (
        <FormFields>
          <TextareaField form={form} name="timeline" label="Kronologi kejadian" schema={rcaSchema.shape.timeline} />
          <TextareaField form={form} name="humanFactor" label="Faktor manusia" schema={rcaSchema.shape.humanFactor} />
          <TextareaField
            form={form}
            name="environmentFactor"
            label="Faktor lingkungan"
            schema={rcaSchema.shape.environmentFactor}
          />
          <TextareaField
            form={form}
            name="equipmentFactor"
            label="Faktor peralatan"
            schema={rcaSchema.shape.equipmentFactor}
          />
        </FormFields>
      ) : null}
      {step === 1 ? (
        <form.Field name="fiveWhys">
          {(field) => (
            <div className="flex flex-col gap-4">
              {(field.state.value as RcaFormValues["fiveWhys"]).map((_, index) => (
                <div key={index} className="grid gap-3 rounded-lg border border-border bg-white p-4 md:grid-cols-2">
                  <Field>
                    <FieldLabel htmlFor={`why-${index}`}>Mengapa {index + 1}</FieldLabel>
                    <Input
                      id={`why-${index}`}
                      className="h-11 min-h-11 rounded-md"
                      value={field.state.value[index]?.why ?? ""}
                      onChange={(event) => {
                        const next = [...field.state.value];
                        next[index] = { ...next[index], why: event.target.value, answer: next[index]?.answer ?? "" };
                        field.handleChange(next);
                      }}
                    />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor={`answer-${index}`}>Jawaban</FieldLabel>
                    <Input
                      id={`answer-${index}`}
                      className="h-11 min-h-11 rounded-md"
                      value={field.state.value[index]?.answer ?? ""}
                      onChange={(event) => {
                        const next = [...field.state.value];
                        next[index] = { ...next[index], answer: event.target.value, why: next[index]?.why ?? "" };
                        field.handleChange(next);
                      }}
                    />
                  </Field>
                </div>
              ))}
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="secondary"
                  className="min-h-11"
                  disabled={field.state.value.length >= 5}
                  onClick={() => field.handleChange([...field.state.value, { why: "", answer: "" }])}
                >
                  Tambah mengapa
                </Button>
                {field.state.value.length > 1 ? (
                  <Button
                    type="button"
                    variant="ghost"
                    className="min-h-11"
                    onClick={() => field.handleChange(field.state.value.slice(0, -1))}
                  >
                    Hapus terakhir
                  </Button>
                ) : null}
              </div>
            </div>
          )}
        </form.Field>
      ) : null}
      {step === 2 ? (
        <FormFields>
          {FISHBONE_KEYS.map((key) => (
            <TextareaField
              key={key}
              form={form}
              name={`fishbone.${key}`}
              label={fishboneLabels[key]}
              schema={rcaSchema.shape.fishbone.shape[key]}
            />
          ))}
        </FormFields>
      ) : null}
      {step === 3 ? (
        <div className="rounded-lg border border-border bg-white p-4">
          <RcaReadView
            rca={{
              id: rca.data?.id ?? "",
              incidentId,
              ...form.state.values,
              investigatorId: rca.data?.investigatorId ?? "",
              completedAt: rca.data?.completedAt ?? null,
              updatedAt: rca.data?.updatedAt ?? "",
            }}
          />
        </div>
      ) : null}
      {notice ? <p className="text-xs text-subtle">{notice}</p> : null}
      {error ? (
        <p role="alert" className="text-xs text-danger-500">
          {error}
        </p>
      ) : null}
      <div className="flex flex-wrap gap-2">
        {step > 0 ? (
          <Button type="button" variant="secondary" className="min-h-11" onClick={() => setStep(step - 1)}>
            Kembali
          </Button>
        ) : (
          <Link href={`/incidents/${incidentId}?tab=rca`} className={cn(buttonVariants({ variant: "secondary" }), "min-h-11")}>
            Batal
          </Link>
        )}
        {step < STEPS.length - 1 ? (
          <Button type="submit" className="min-h-11">
            Lanjut
          </Button>
        ) : (
          <>
            <Button
              type="button"
              variant="ghost"
              className="min-h-11"
              disabled={upsert.isPending}
              onClick={async () => {
                setError(null);
                try {
                  await upsert.mutateAsync({ ...form.state.values, completed: false });
                  setNotice("Draf investigasi disimpan.");
                } catch (cause) {
                  setError(mapApiError(cause, "Gagal menyimpan draf."));
                }
              }}
            >
              Simpan draf
            </Button>
            <Button type="submit" className="min-h-11" disabled={upsert.isPending}>
              Simpan RCA
            </Button>
          </>
        )}
      </div>
    </form>
  );
}
