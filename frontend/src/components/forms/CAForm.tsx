"use client";

import { useForm } from "@tanstack/react-form";
import { useState } from "react";

import { FormFields, SelectField, TextareaField, TextField } from "@/components/forms/fields";
import { Button } from "@/components/ui/button";
import { mapApiError } from "@/lib/errors";
import { useCreateCorrectiveAction } from "@/lib/queries/useCorrectiveActions";
import {
  CA_ACTION_TYPES,
  CA_PRIORITIES,
  caActionTypeLabels,
  caCreateSchema,
  caPriorityLabels,
  type CaCreateValues,
} from "@/lib/schemas";
import type { AssigneeOption } from "@/lib/assignees";

export function CAForm({
  incidentId,
  assignees,
}: {
  incidentId: string;
  assignees: AssigneeOption[];
}) {
  const create = useCreateCorrectiveAction(incidentId);
  const [error, setError] = useState<string | null>(null);

  const form = useForm({
    defaultValues: {
      description: "",
      actionType: "SHORT_TERM",
      priority: "MEDIUM",
      assigneeId: assignees[0]?.id ?? "",
      dueDate: "",
    } as CaCreateValues,
    onSubmit: async ({ value }) => {
      const parsed = caCreateSchema.safeParse(value);
      if (!parsed.success) return;
      setError(null);
      try {
        await create.mutateAsync(parsed.data);
        form.reset();
      } catch (cause) {
        setError(mapApiError(cause, "Gagal membuat corrective action."));
      }
    },
  });

  return (
    <form
      className="rounded-lg border border-border bg-white p-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <h2 className="text-base font-medium text-ink">Buat corrective action</h2>
      <div className="mt-3 grid gap-4 md:grid-cols-2">
        <div className="md:col-span-2">
          <FormFields>
            <TextareaField
              form={form}
              name="description"
              label="Deskripsi tindakan"
              schema={caCreateSchema.shape.description}
            />
          </FormFields>
        </div>
        <SelectField
          form={form}
          name="actionType"
          label="Tipe"
          placeholder="Pilih tipe"
          schema={caCreateSchema.shape.actionType}
          options={CA_ACTION_TYPES.map((value) => ({ value, label: caActionTypeLabels[value] }))}
        />
        <SelectField
          form={form}
          name="priority"
          label="Prioritas"
          placeholder="Pilih prioritas"
          schema={caCreateSchema.shape.priority}
          options={CA_PRIORITIES.map((value) => ({ value, label: caPriorityLabels[value] }))}
        />
        <SelectField
          form={form}
          name="assigneeId"
          label="Penanggung jawab"
          placeholder="Pilih user"
          schema={caCreateSchema.shape.assigneeId}
          options={assignees.map((item) => ({ value: item.id, label: item.label }))}
        />
        <TextField form={form} name="dueDate" label="Tenggat" type="date" schema={caCreateSchema.shape.dueDate} />
      </div>
      {error ? (
        <p role="alert" className="mt-2 text-xs text-danger-500">
          {error}
        </p>
      ) : null}
      <Button type="submit" className="mt-4 min-h-11" disabled={create.isPending || assignees.length === 0}>
        Tambah tindakan
      </Button>
      {assignees.length === 0 ? (
        <p className="mt-2 text-xs text-subtle">Tidak ada penanggung jawab yang dapat dipilih.</p>
      ) : null}
    </form>
  );
}
