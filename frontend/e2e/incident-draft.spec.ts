import { expect, test } from "@playwright/test";

import {
  TINY_JPEG,
  USERS,
  confirmDialog,
  fillIncidentBasics,
  loginAs,
  advanceIncidentWizard,
} from "./helpers";

const apiBase = process.env.E2E_API_URL ?? "http://localhost:8080";

test.describe.configure({ mode: "serial", timeout: 180_000 });

test.describe("siklus pelaporan insiden", () => {
  test("submit, unggah, reject, verify, CA mobile, close", async ({ page, request }) => {
    const health = await request.get(`${apiBase}/health`);
    expect(
      health.ok(),
      `API ${apiBase}/health harus 200. Jalankan backend Go sebelum E2E.`,
    ).toBeTruthy();

    const title = `E2E siklus ${Date.now()}`;

    await page.goto("/login");
    await loginAs(page, USERS.reporter.email, USERS.reporter.password);
    await page.goto("/incidents/new");
    await expect(page.getByRole("heading", { name: "Laporan insiden baru" })).toBeVisible();

    const saveDraft = page.getByTestId("save-draft");
    const box = await saveDraft.boundingBox();
    expect(box?.height ?? 0, "Touch target aksi utama minimal 44px").toBeGreaterThanOrEqual(44);

    await fillIncidentBasics(page, title);
    await saveDraft.scrollIntoViewIfNeeded();
    await saveDraft.click();
    await expect(page).toHaveURL(/\/incidents\/new\?id=/);
    await expect(page.getByLabel("Judul singkat")).toHaveValue(title);

    await page.getByRole("button", { name: "Lanjut" }).click();
    await expect(page.getByRole("checkbox", { name: "Ada korban" })).toBeVisible();
    await page.getByRole("button", { name: "Lanjut" }).click();
    await expect(page.getByTestId("evidence-dropzone")).toBeVisible();

    await page.getByTestId("evidence-file").setInputFiles({
      name: "tiny.jpg",
      mimeType: "image/jpeg",
      buffer: TINY_JPEG,
    });
    await expect(page.getByText("tiny.jpg")).toBeVisible({ timeout: 20_000 });

    await page.getByRole("button", { name: "Lanjut" }).click();
    await page.getByRole("button", { name: "Kirim laporan" }).click();
    await page.waitForURL(/\/incidents\/[0-9a-f-]{36}/);
    const incidentPath = new URL(page.url()).pathname;
    expect(incidentPath).toMatch(/^\/incidents\/[0-9a-f-]{36}$/);
    await expect(page.getByText(/^INC-\d{4}-\d{2}-\d{4}$/)).toBeVisible();
    await expect(page.getByText("Menunggu review")).toBeVisible();

    await page.getByRole("tab", { name: "Files" }).click();
    await expect(page.getByText("tiny.jpg")).toBeVisible();

    await loginAs(page, USERS.supervisor.email, USERS.supervisor.password);
    await page.goto(incidentPath);
    await page.getByRole("button", { name: "Kembalikan" }).click();
    await page.getByLabel("Alasan pengembalian").fill("Lengkapi kronologi shift dan saksi.");
    await page.getByRole("dialog").getByRole("button", { name: "Kembalikan" }).click();
    await expect(page.getByText("Dikembalikan")).toBeVisible();

    await loginAs(page, USERS.reporter.email, USERS.reporter.password);
    await page.goto(incidentPath);
    await page.getByRole("link", { name: "Lanjutkan pelaporan" }).click();
    await fillIncidentBasics(page, title);
    await advanceIncidentWizard(page);
    await page.getByRole("button", { name: "Kirim laporan" }).click();
    await page.waitForURL(/\/incidents\/[0-9a-f-]{36}/);
    await expect(page.getByText("Menunggu review")).toBeVisible();
    await expect(page.getByText(/^INC-\d{4}-\d{2}-\d{4}$/)).toBeVisible();

    await loginAs(page, USERS.supervisor.email, USERS.supervisor.password);
    await page.goto(incidentPath);
    await page.getByRole("button", { name: "Verifikasi" }).click();
    await confirmDialog(page);
    await expect(page.getByText("Investigasi", { exact: true })).toBeVisible();

    await loginAs(page, USERS.officer.email, USERS.officer.password);
    await page.goto(incidentPath);
    await page.getByRole("tab", { name: "RCA" }).click();
    await page.getByRole("link", { name: "Isi investigasi" }).click();
    await page.getByLabel("Kronologi kejadian").fill("Rekonstruksi singkat untuk E2E.");
    await page.getByRole("button", { name: "Lanjut" }).click();
    await page.getByRole("button", { name: "Lanjut" }).click();
    await page.getByRole("button", { name: "Lanjut" }).click();
    await page.getByRole("button", { name: "Simpan RCA" }).click();
    await expect(page.getByRole("link", { name: "Isi investigasi" })).toBeVisible();

    await page.goto(incidentPath);
    await page.getByRole("button", { name: "Mulai tindakan perbaikan" }).click();
    await confirmDialog(page);
    await expect(page.getByText("Tindakan perbaikan", { exact: true })).toBeVisible();

    await page.getByRole("tab", { name: "Corrective actions" }).click();
    await page.getByLabel("Deskripsi tindakan").fill("Bersihkan tumpahan dan pasang rambu basah.");
    const due = new Date();
    due.setDate(due.getDate() + 7);
    await page.getByLabel("Tenggat").fill(due.toISOString().slice(0, 10));
    await page.getByTestId("select-assigneeId").click();
    await page.getByRole("option", { name: "Pelapor" }).click();
    await page.keyboard.press("Escape");
    const addCa = page.getByRole("button", { name: "Tambah tindakan" });
    await addCa.scrollIntoViewIfNeeded();
    await addCa.click({ force: true });
    await expect(page.getByText("Bersihkan tumpahan dan pasang rambu basah.")).toBeVisible();

    await loginAs(page, USERS.reporter.email, USERS.reporter.password);
    await page.goto(`${incidentPath}?tab=ca`);
    await page.getByRole("tab", { name: "Corrective actions" }).click();
    await page.getByRole("button", { name: "Mulai dikerjakan" }).click();
    await expect(page.getByText("Sedang dikerjakan")).toBeVisible();
    await page.getByLabel("Catatan penyelesaian").fill("Area sudah dikeringkan dan rambu terpasang.");
    await page.getByRole("button", { name: "Tandai selesai" }).click();
    await expect(page.getByText("Selesai", { exact: true }).first()).toBeVisible();

    await loginAs(page, USERS.officer.email, USERS.officer.password);
    await page.goto(`${incidentPath}?tab=ca`);
    await page.getByRole("tab", { name: "Corrective actions" }).click();
    await page.getByRole("button", { name: "Verifikasi" }).click();
    await expect(page.getByText("Terverifikasi")).toBeVisible();

    await loginAs(page, USERS.manager.email, USERS.manager.password);
    await page.goto(incidentPath);
    await page.getByRole("button", { name: "Tutup laporan" }).click();
    await confirmDialog(page);
    await expect(page.getByText("Ditutup")).toBeVisible();
    await expect(page.getByText("Laporan Final — Tidak Dapat Diubah")).toBeVisible();
  });
});
