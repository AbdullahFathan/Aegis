"use client";

import type { ReactNode } from "react";
import type { ZodType } from "zod";

import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export const fieldClass = "h-9 rounded-md text-sm";

type BoundField = {
  name: string;
  state: { value: unknown; meta: { errors: Array<unknown> } };
  handleBlur: () => void;
  handleChange: (value: unknown) => void;
};

type FieldForm = {
  Field: (props: {
    name: string;
    validators?: {
      onSubmit?: (props: { value: unknown }) => string | undefined;
    };
    children: (field: BoundField) => ReactNode;
  }) => ReactNode;
};

export function issueMessage(schema: ZodType, value: unknown) {
  const parsed = schema.safeParse(value);
  if (parsed.success) return undefined;
  return parsed.error.issues[0]?.message ?? "Nilai tidak valid";
}

export function TextField({
  form,
  name,
  label,
  schema,
  type = "text",
}: {
  form: object;
  name: string;
  label: string;
  schema: ZodType;
  type?: string;
}) {
  const bound = form as FieldForm;
  return (
    <bound.Field
      name={name}
      validators={{
        onSubmit: ({ value }) => issueMessage(schema, value),
      }}
    >
      {(field) => {
        const invalid = field.state.meta.errors.length > 0;
        return (
          <Field data-invalid={invalid || undefined}>
            <FieldLabel htmlFor={field.name}>{label}</FieldLabel>
            <Input
              id={field.name}
              name={field.name}
              type={type}
              className={fieldClass}
              value={String(field.state.value ?? "")}
              onBlur={field.handleBlur}
              onChange={(event) => field.handleChange(event.target.value)}
              aria-invalid={invalid}
              spellCheck={false}
            />
            <FieldError
              className="text-xs text-danger-500"
              errors={field.state.meta.errors.map((error) => ({
                message: typeof error === "string" ? error : "Nilai tidak valid",
              }))}
            />
          </Field>
        );
      }}
    </bound.Field>
  );
}

export function SelectField({
  form,
  name,
  label,
  schema,
  placeholder,
  options,
}: {
  form: object;
  name: string;
  label: string;
  schema: ZodType;
  placeholder: string;
  options: Array<{ value: string; label: string }>;
}) {
  const bound = form as FieldForm;
  return (
    <bound.Field
      name={name}
      validators={{
        onSubmit: ({ value }) => issueMessage(schema, value),
      }}
    >
      {(field) => {
        const invalid = field.state.meta.errors.length > 0;
        const current = String(field.state.value ?? "");
        return (
          <Field data-invalid={invalid || undefined}>
            <FieldLabel htmlFor={field.name}>{label}</FieldLabel>
            <Select
              value={current || null}
              onValueChange={(value) => field.handleChange(value ?? "")}
            >
              <SelectTrigger id={field.name} className="h-9 w-full rounded-md" aria-invalid={invalid}>
                <SelectValue placeholder={placeholder} />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {options.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldError
              className="text-xs text-danger-500"
              errors={field.state.meta.errors.map((error) => ({
                message: typeof error === "string" ? error : "Nilai tidak valid",
              }))}
            />
          </Field>
        );
      }}
    </bound.Field>
  );
}

export function TextareaField({
  form,
  name,
  label,
  schema,
  hint,
}: {
  form: object;
  name: string;
  label: string;
  schema: ZodType;
  hint?: string;
}) {
  const bound = form as FieldForm;
  return (
    <bound.Field
      name={name}
      validators={{
        onSubmit: ({ value }) => issueMessage(schema, value),
      }}
    >
      {(field) => {
        const invalid = field.state.meta.errors.length > 0;
        return (
          <Field data-invalid={invalid || undefined}>
            <FieldLabel htmlFor={field.name}>{label}</FieldLabel>
            <Textarea
              id={field.name}
              name={field.name}
              className="min-h-24 rounded-md text-sm"
              value={String(field.state.value ?? "")}
              onBlur={field.handleBlur}
              onChange={(event) => field.handleChange(event.target.value)}
              aria-invalid={invalid}
            />
            {hint ? <p className="text-xs text-subtle">{hint}</p> : null}
            <FieldError
              className="text-xs text-danger-500"
              errors={field.state.meta.errors.map((error) => ({
                message: typeof error === "string" ? error : "Nilai tidak valid",
              }))}
            />
          </Field>
        );
      }}
    </bound.Field>
  );
}

export function FormFields({ children }: { children: React.ReactNode }) {
  return <FieldGroup>{children}</FieldGroup>;
}
