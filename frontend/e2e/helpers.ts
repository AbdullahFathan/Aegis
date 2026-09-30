import { expect, type Page } from "@playwright/test";

export const USERS = {
  reporter: { email: process.env.E2E_EMAIL ?? "reporter", password: process.env.E2E_PASSWORD ?? "REPORTER" },
  supervisor: { email: "supervisor", password: "SUPERVISOR" },
  officer: { email: "hse_officer", password: "HSE_OFFICER" },
  manager: { email: "hse_manager", password: "HSE_MANAGER" },
} as const;

/** 1×1 JPEG — unggah evidence tanpa file di disk. */
export const TINY_JPEG = Buffer.from(
  "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAn/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIQAxAAAAGP/8QAFBEBAAAAAAAAAAAAAAAAAAAAAP/aAAgBAQABPwB//9k=",
  "base64",
);

export async function loginAs(page: Page, email: string, password: string) {
  const keluar = page.getByRole("button", { name: "Keluar" });
  if ((await keluar.count()) > 0) {
    await keluar.click();
    await page.waitForURL("**/login");
  } else if (!page.url().includes("/login")) {
    await page.goto("/login");
  }

  for (let attempt = 0; attempt < 12; attempt++) {
    await page.getByLabel("Email").fill(email);
    await page.getByLabel("Password").fill(password);
    await page.getByRole("button", { name: "Masuk" }).click();

    for (let i = 0; i < 40; i++) {
      if (page.url().includes("/dashboard")) return;
      const alert = page.getByRole("alert");
      if ((await alert.count()) > 0) {
        const text = (await alert.first().textContent()) ?? "";
        if (text.includes("Terlalu banyak")) {
          await page.waitForTimeout(16_000);
          break;
        }
        if (text.includes("Kredensial salah")) {
          throw new Error(`Kredensial salah untuk ${email}`);
        }
      }
      await page.waitForTimeout(250);
    }
    if (page.url().includes("/dashboard")) return;
  }
  throw new Error(`Gagal masuk sebagai ${email}. Cek rate limit login atau seed user.`);
}

export async function pickLocation(page: Page) {
  await page.getByTestId("select-locationId").click();
  const preferred = page.getByRole("option", { name: /E2E-TMB/ });
  const fallback = page.getByRole("option").first();
  const option = preferred.or(fallback).first();
  await expect(
    option,
    "Tidak ada lokasi di formulir. Seed E2E-TMB  atau daftarkan lokasi.",
  ).toBeVisible();
  if ((await preferred.count()) > 0) {
    await preferred.first().click();
    return;
  }
  await fallback.click();
}

export async function fillIncidentBasics(page: Page, title: string) {
  await page.getByLabel("Judul singkat").fill(title);
  await page.getByLabel("Deskripsi kejadian").fill(
    "Operator melihat tumpahan oli di area crusher saat shift pagi dan segera mengamankan area.",
  );
  await pickLocation(page);
  await expect(page.getByRole("combobox", { name: "Lokasi" })).not.toContainText("Pilih lokasi");
}

export async function advanceIncidentWizard(page: Page) {
  await page.getByRole("button", { name: "Lanjut" }).click();
  await expect(page.getByRole("checkbox", { name: "Ada korban" })).toBeVisible();
  await page.getByRole("button", { name: "Lanjut" }).click();
  await expect(page.getByTestId("evidence-dropzone")).toBeVisible();
  await page.getByRole("button", { name: "Lanjut" }).click();
  await expect(page.getByRole("button", { name: "Kirim laporan" })).toBeVisible();
}

export async function confirmDialog(page: Page) {
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  await dialog.getByRole("button", { name: "Konfirmasi" }).click();
  await expect(dialog).toBeHidden();
}
