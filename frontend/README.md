# Aegis frontend

UI Next.js (App Router) untuk pelaporan insiden K3. **Semua data bisnis berasal dari REST API Go**, bukan Route Handlers Next.js.

## Prasyarat

- [Bun](https://bun.sh) 1.4+
- Backend Go di `http://localhost:8080` (lihat README backend)
- Salin [`.env.example`](./.env.example) ke `.env.local`

## Perintah

```bash
bun install
bun dev
```

Buka [http://localhost:3000](http://localhost:3000). Proxy `/api/*` menuju `API_PROXY_TARGET` (default `http://localhost:8080`) agar cookie refresh HttpOnly tetap di origin Next.js.

```bash
bun run lint
bun run typecheck
bun run test
```

## E2E (Playwright)

Sebelum menjalankan tes, siapkan kredensial login untuk seluruh peran dan siapkan data yang dipakai alur (pengguna, lokasi, area, dan data terkait). API Go harus hidup (`GET http://localhost:8080/health`).

Spec `e2e/incident-draft.spec.ts` menjalankan siklus pelaporan dari draft sampai tutup, termasuk unggah bukti. Viewport tes 375×667.

```bash
bunx playwright install chromium
bun run e2e
```

Variabel:

| Env | Default | Keterangan |
| --- | --- | --- |
| `E2E_BASE_URL` | `http://localhost:3000` | Origin Next.js |
| `E2E_API_URL` | `http://localhost:8080` | Health check Go |
| `E2E_EMAIL` / `E2E_PASSWORD` | `reporter` / `REPORTER` | Seed, bukan akun produksi |
| `E2E_SKIP_WEBSERVER=1` | — | Jangan menjalankan `bun run dev`; pakai server yang sudah jalan |

## Aksesibilitas (AA)

- Label eksplisit pada field; placeholder bukan pengganti label.
- Focus ring orange (`--ring` / `orange-500`).
- Teks tubuh `ink` `#1E293B` di `canvas` `#F8FAFC`. Caption `subtle` `#475569` (gelap dari `#64748B` PRD) agar kontras AA pada kanvas dan kartu putih.
- Tombol aksi utama `min-h-11` (44px). Checklist axe tidak diwajibkan di CI.

## Empty / error / loading (halaman list)

| Route | Loading | Error | Empty |
| --- | --- | --- | --- |
| `/incidents` | skeleton | pesan API / 403 | + CTA buat laporan |
| `/corrective-actions` | skeleton | pesan API | “Tidak ada CA sesuai filter” |
| `/notifications` | skeleton | pesan API | EmptyState + filter |
| `/dashboard` | skeleton | QueryError | widget kosong |
| `/reports` | skeleton | gagal generate spesifik | arsip kosong |
| `/locations` | skeleton | 403/API | CTA daftarkan lokasi |
| `/admin/users` | skeleton | 403/API | EmptyState + tambah user |
| `/admin/audit-logs` | skeleton | 403/API | EmptyState filter |

## Design QA (halaman utama)

- Palet Warm Safety: CTA `#E85D04`, sidebar `#1A237E`, canvas `#F8FAFC`.
- Lifecycle insiden: `DRAFT` … `CLOSED` / `REJECTED` (bukan `OPEN` / `IN_REVIEW`).
- Radius `rounded-md` / `rounded-lg`; tanpa FAB `rounded-full`.
- Bukan Vite; data dari Go API.

## Docker frontend

Image Compose production untuk service `frontend` **belum** termasuk phase ini (tiket FE-P5-06).
