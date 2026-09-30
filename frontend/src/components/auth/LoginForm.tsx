"use client";

import { useForm } from "@tanstack/react-form";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { FormFields, TextField } from "@/components/forms/fields";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { mapLoginError } from "@/lib/errors";
import { useLogin, useSession } from "@/lib/queries/useSession";
import { loginSchema } from "@/lib/schemas";

export function LoginForm() {
  const login = useLogin();
  const session = useSession();
  const router = useRouter();

  useEffect(() => {
    if (session.data) router.replace("/dashboard");
  }, [router, session.data]);

  const form = useForm({
    defaultValues: { email: "", password: "" },
    onSubmit: async ({ value }) => {
      const parsed = loginSchema.safeParse(value);
      if (!parsed.success) return;
      await login.mutateAsync(parsed.data);
      router.replace("/dashboard");
    },
  });

  return (
    <main className="flex min-h-screen items-center justify-center bg-canvas px-6">
      <Card className="w-full max-w-sm rounded-lg border border-border shadow-card">
        <CardHeader>
          <CardTitle className="text-2xl font-semibold text-ink">Masuk ke Aegis</CardTitle>
          <p className="text-sm text-subtle">Gunakan akun perusahaan untuk melaporkan dan meninjau insiden K3.</p>
        </CardHeader>
        <CardContent>
          <form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void form.handleSubmit();
            }}
          >
            <FormFields>
              <TextField form={form} name="email" label="Email" schema={loginSchema.shape.email} />
              <TextField
                form={form}
                name="password"
                label="Password"
                type="password"
                schema={loginSchema.shape.password}
              />
            </FormFields>
            {login.isError ? (
              <p role="alert" className="text-xs text-danger-500">
                {mapLoginError(login.error)}
              </p>
            ) : null}
            <Button type="submit" size="touch" disabled={login.isPending}>
              {login.isPending ? "Memproses" : "Masuk"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
