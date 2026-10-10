import { test, expect, request as pwRequest, type APIRequestContext, type Page } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

// PRODUCTION stock-flow audit (mutating, isolated).
// Fixtures: a dedicated product + two dedicated warehouses with their own
// INTERNAL racks, all suffixed AUD-<ts>. No existing product/warehouse stock
// is touched. Request shapes are taken from the Go structs:
//   receipt  usecase/wms/receipt_usecase.go:18-41
//   putaway  usecase/wms/putaway_usecase.go:14
//   opname   usecase/wms/wms_usecase.go:1281-1291
//   scrap    usecase/wms/wms_usecase.go:1552
//   transfer usecase/wms/wms_usecase.go:456-470
//   DO       usecase/wms/wms_usecase.go:925-946
//   POS      usecase/pos/pos.go:16-40
// Two admins are needed for segregation-of-duties steps (opname approval,
// transfer approval). approver@test.com is a real second admin in the tenant.

const BASE = process.env.AUDIT_BASE_URL || 'https://tayooli.my.id';
const PASS = process.env.TAYOOLI_PASSWORD || 'password123';
const ADMIN = process.env.TAYOOLI_EMAIL || 'admin@test.com';
const APPROVER = process.env.AUDIT_APPROVER_EMAIL || 'approver@test.com';
const OUT = path.resolve(__dirname, '../../test-results/audit');
const TS = Date.now().toString().slice(-8);
const SFX = `AUD-${TS}`;

type J = any;
const log: { step: string; verdict: 'PASS' | 'FAIL' | 'BLOCKED'; evidence: string }[] = [];
const artifacts: Record<string, string> = {};
function record(step: string, verdict: 'PASS' | 'FAIL' | 'BLOCKED', evidence: string) {
  log.push({ step, verdict, evidence });
  fs.mkdirSync(OUT, { recursive: true });
  fs.writeFileSync(path.join(OUT, 'stock-flows.json'), JSON.stringify({ suffix: SFX, log, artifacts }, null, 2));
}

type UserSession = { email: string; token: string; ctx: APIRequestContext };

async function loginUser(email: string): Promise<UserSession> {
  const ctx = await pwRequest.newContext({ baseURL: BASE });
  const r = await ctx.post('/api/v1/auth/login', { data: { email, password: PASS } });
  if (!r.ok()) throw new Error(`login ${email} -> ${r.status()} ${await r.text()}`);
  const data = await r.json();
  const token = data.token;
  if (!token) throw new Error(`login ${email}: no token in response`);
  return { email, token, ctx };
}

async function call(u: UserSession, method: string, url: string, body?: J): Promise<[number, J]> {
  const r = await u.ctx.fetch('/api/v1' + url, {
    method,
    headers: {
      'Authorization': `Bearer ${u.token}`,
      ...(body ? { 'Content-Type': 'application/json' } : {}),
    },
    data: body,
  });
  const t = await r.text();
  let j: J = t;
  try { j = JSON.parse(t); } catch { /* text */ }
  return [r.status(), j?.data !== undefined ? j.data : j];
}
const num = (v: J) => Number(v ?? 0);

let A: UserSession; // admin@test.com (creator / operator)
let B: UserSession; // approver@test.com (second admin)
let productId = '', wh1 = '', wh2 = '', rack1 = '', rack2 = '', stg1 = '';

// Stock of the test product per INTERNAL rack, from the WMS ledger.
async function rackQty(whId: string, locId: string): Promise<{ qty: number; avail: number; alloc: number }> {
  const [, rows] = await call(A, 'GET', `/wms/stock?warehouse_id=${whId}`);
  const row = (rows || []).find((x: J) => x.product_id === productId && x.location_id === locId);
  return { qty: num(row?.quantity), avail: num(row?.available_qty), alloc: num(row?.allocated_qty) };
}
async function productTotals(): Promise<{ wms: number; inventory: number }> {
  const [, rows] = await call(A, 'GET', '/wms/stock');
  const wms = (rows || []).filter((x: J) => x.product_id === productId).reduce((s: number, x: J) => s + num(x.quantity), 0);
  const [, inv] = await call(A, 'GET', '/inventory');
  const list = Array.isArray(inv) ? inv : inv?.items || [];
  const it = list.find((x: J) => x.product_id === productId);
  return { wms, inventory: num(it?.total_quantity ?? it?.quantity) };
}
async function movements(): Promise<J[]> {
  const [, rows] = await call(A, 'GET', `/wms/movements?product_id=${productId}&limit=500`);
  return rows || [];
}

test.describe.configure({ mode: 'serial' });
test.setTimeout(120_000);

test.beforeAll(async () => {
  A = await loginUser(ADMIN);
  B = await loginUser(APPROVER);
});
test.afterAll(async () => { await A?.ctx?.dispose(); await B?.ctx?.dispose(); });

test('0. fixtures: product + 2 isolated warehouses + INTERNAL racks', async () => {
  let [s, p] = await call(A, 'POST', '/products', { name: `Produk Audit ${SFX}`, sku: SFX, price: 1000 });
  expect.soft(s, JSON.stringify(p)).toBe(201);
  productId = p.id; artifacts.product = productId;

  [s, p] = await call(A, 'POST', '/wms/warehouses', { code: `WA-${TS}`, name: `AUDIT-${TS}-A` });
  expect.soft(s, JSON.stringify(p)).toBeLessThan(300); wh1 = p.id; artifacts.warehouse1 = wh1;
  [s, p] = await call(A, 'POST', '/wms/warehouses', { code: `WB-${TS}`, name: `AUDIT-${TS}-B` });
  expect.soft(s, JSON.stringify(p)).toBeLessThan(300); wh2 = p.id; artifacts.warehouse2 = wh2;

  [s, p] = await call(A, 'POST', '/wms/locations', { warehouse_id: wh1, code: `RA-${TS}`, name: 'Rak Audit A', type: 'INTERNAL' });
  expect.soft(s, JSON.stringify(p)).toBeLessThan(300); rack1 = p.id;
  [s, p] = await call(A, 'POST', '/wms/locations', { warehouse_id: wh2, code: `RB-${TS}`, name: 'Rak Audit B', type: 'INTERNAL' });
  expect.soft(s, JSON.stringify(p)).toBeLessThan(300); rack2 = p.id;
  record('0 fixtures', 'PASS', `product=${productId} wh1=${wh1} wh2=${wh2} rack1=${rack1} rack2=${rack2}`);
});

test('a. barang masuk: receipt 10 -> post -> putaway -> 10 di rak INTERNAL (+ double post)', async () => {
  const body = {
    receipt_type: 'VENDOR', warehouse_id: wh1, dest_location_id: rack1,
    supplier_name: `Supplier ${SFX}`, from_name: `Supplier ${SFX}`, supplier_ref: `PO-${SFX}`,
    items: [{ product_id: productId, ordered_qty: '10', accepted_qty: '10', rejected_qty: '0', batch_number: `LOT-${TS}` }],
  };
  // adversarial: qty 0 / negative must be rejected
  const [sz] = await call(A, 'POST', '/wms/receipts', { ...body, items: [{ ...body.items[0], accepted_qty: '0' }] });
  const [sn] = await call(A, 'POST', '/wms/receipts', { ...body, items: [{ ...body.items[0], accepted_qty: '-5' }] });

  let [s, rc] = await call(A, 'POST', '/wms/receipts', body);
  expect.soft(s, JSON.stringify(rc)).toBeLessThan(300);
  const rcId = (rc.receipt || rc).id; artifacts.receipt = rcId;
  const [s1, p1] = await call(A, 'POST', `/wms/receipts/${rcId}/post`);
  const [s2] = await call(A, 'POST', `/wms/receipts/${rcId}/post`); // double-submit

  const [, pend] = await call(A, 'GET', `/wms/putaway/pending?warehouse_id=${wh1}`);
  const line = (pend || []).find((x: J) => x.product_id === productId);
  expect.soft(line, 'putaway pending line for test product').toBeTruthy();
  stg1 = line.staging_location_id;
  const stagedQty = num(line.quantity);
  const [s3, pa] = await call(A, 'POST', '/wms/putaway/confirm', {
    warehouse_id: wh1, product_id: productId, batch_id: line.batch_id, quantity: String(stagedQty), dest_location_id: rack1,
    reason: 'audit putaway ke rak audit',
  });
  const [s4] = await call(A, 'POST', '/wms/putaway/confirm', {
    warehouse_id: wh1, product_id: productId, batch_id: line.batch_id, quantity: '1', dest_location_id: rack1, reason: 'audit over-putaway',
  });
  const st = await rackQty(wh1, rack1);
  const ok = s1 < 300 && s2 === 409 && stagedQty === 10 && s3 < 300 && s4 >= 400 && st.qty === 10 && sz >= 400 && sn >= 400;
  record('a barang masuk', ok ? 'PASS' : 'FAIL',
    `qty0=${sz} qtyNeg=${sn} post=${s1} postLagi=${s2} staging=${stagedQty} putaway=${s3}${s3 >= 300 ? ' ' + JSON.stringify(pa) : ''} overPutaway=${s4} rak=${st.qty} batch=${line.batch_id}`);
  expect.soft(ok).toBe(true);
  void p1;
});

test('b. opname: hitung 9 -> complete (admin kedua) -> stok 9', async () => {
  let [s, op] = await call(A, 'POST', '/wms/opnames', { warehouse_id: wh1, notes: `Audit ${SFX}` });
  expect.soft(s, JSON.stringify(op)).toBeLessThan(300);
  const opId = op.id; artifacts.opname = opId;
  const [si, it] = await call(A, 'POST', `/wms/opnames/${opId}/items`, { product_id: productId, location_id: rack1, physical_qty: '9', notes: 'audit selisih -1' });
  const [sSelf, self] = await call(A, 'POST', `/wms/opnames/${opId}/complete`); // self-approval must fail
  const [sc, c] = await call(B, 'POST', `/wms/opnames/${opId}/complete`);
  const st = await rackQty(wh1, rack1);
  const ok = si < 300 && sSelf === 403 && sc < 300 && st.qty === 9;
  record('b opname', ok ? 'PASS' : 'FAIL',
    `addItem=${si} selfApprove=${sSelf} approveAdmin2=${sc}${sc >= 300 ? ' ' + JSON.stringify(c) : ''} rak=${st.qty}`);
  expect.soft(ok, `opname: ${sc} ${JSON.stringify(c)} / item ${JSON.stringify(it)} / self ${JSON.stringify(self)}`).toBe(true);
});

test('c. scrap 1 -> stok 8; tolak qty<=0 dan qty>tersedia', async () => {
  const base = { warehouse_id: wh1, product_id: productId, source_location_id: rack1, reason: `Audit scrap ${SFX}` };
  const before = (await rackQty(wh1, rack1)).qty;
  const [s0] = await call(A, 'POST', '/wms/scraps', { ...base, quantity: '0' });
  const [sn] = await call(A, 'POST', '/wms/scraps', { ...base, quantity: '-1' });
  const [sOver, over] = await call(A, 'POST', '/wms/scraps', { ...base, quantity: String(before + 1) });
  const [s1, sc] = await call(A, 'POST', '/wms/scraps', { ...base, quantity: '1' });
  if (s1 < 300) artifacts.scrap1 = sc.id;
  const st = await rackQty(wh1, rack1);
  const ok = s0 >= 400 && sn >= 400 && sOver >= 400 && s1 < 300 && st.qty === before - 1;
  record('c scrap', ok ? 'PASS' : 'FAIL',
    `sebelum=${before} qty0=${s0} qtyNeg=${sn} qtyLebih=${sOver} scrap1=${s1}${s1 >= 300 ? ' ' + JSON.stringify(sc) : ''} sesudah=${st.qty}`);
  expect.soft(ok, JSON.stringify(over)).toBe(true);
});

test('d. transfer 2 unit wh1 -> wh2 (submit/approve/dispatch/receive) + guard', async () => {
  const before = await productTotals();
  const r1 = (await rackQty(wh1, rack1)).qty;
  // self-loop guard (same warehouse)
  const [sLoop] = await call(A, 'POST', '/wms/transfers', {
    from_warehouse_id: wh1, to_warehouse_id: wh1, transfer_number: `TRL-${TS}`,
    items: [{ product_id: productId, requested_qty: '1', source_location_id: rack1, dest_location_id: rack1 }],
  });
  let [s, t] = await call(A, 'POST', '/wms/transfers', {
    from_warehouse_id: wh1, to_warehouse_id: wh2, transfer_number: `TR-${TS}`, notes: `Audit ${SFX}`,
    items: [{ product_id: productId, requested_qty: '2', source_location_id: rack1, dest_location_id: rack2 }],
  });
  expect.soft(s, JSON.stringify(t)).toBeLessThan(300);
  const tId = t.id; artifacts.transfer = tId;
  const [sDispEarly] = await call(A, 'POST', `/wms/transfers/${tId}/dispatch`); // not approved yet
  const [sSub] = await call(A, 'POST', `/wms/transfers/${tId}/submit`);
  const [sSelf] = await call(A, 'POST', `/wms/transfers/${tId}/approve`);
  const [sApp, app] = await call(B, 'POST', `/wms/transfers/${tId}/approve`);
  const [sDisp, disp] = await call(A, 'POST', `/wms/transfers/${tId}/dispatch`);
  const mid = await productTotals();
  const [sRcv, rcv] = await call(A, 'POST', `/wms/transfers/${tId}/receive`);
  const [sRcv2] = await call(A, 'POST', `/wms/transfers/${tId}/receive`); // double receive
  const after1 = (await rackQty(wh1, rack1)).qty;
  const after2 = (await rackQty(wh2, rack2)).qty;
  const end = await productTotals();
  const ok = sLoop >= 400 && sDispEarly >= 400 && sSub < 300 && sSelf === 403 && sApp < 300 && sDisp < 300 && sRcv < 300 && sRcv2 >= 400
    && after1 === r1 - 2 && after2 === 2 && end.wms === before.wms;
  record('d transfer', ok ? 'PASS' : 'FAIL',
    `selfLoop=${sLoop} dispatchSebelumApprove=${sDispEarly} submit=${sSub} selfApprove=${sSelf} approveAdmin2=${sApp} dispatch=${sDisp} receive=${sRcv} receiveLagi=${sRcv2} ` +
    `rakA ${r1}->${after1} rakB=${after2} totalINTERNAL ${before.wms}->(transit ${mid.wms})->${end.wms}` +
    (sApp >= 300 ? ` approveErr=${JSON.stringify(app)}` : '') + (sDisp >= 300 ? ` dispErr=${JSON.stringify(disp)}` : '') + (sRcv >= 300 ? ` rcvErr=${JSON.stringify(rcv)}` : ''));
  expect.soft(ok).toBe(true);
});

test('e. surat jalan: over-alokasi ditolak, DRAFT tidak bisa dispatch, confirm -> dispatch -> stok turun', async () => {
  const r1 = await rackQty(wh1, rack1);
  const [, pend] = await call(A, 'GET', `/wms/stock?warehouse_id=${wh1}`);
  void pend;
  const [sOver, over] = await call(A, 'POST', '/wms/delivery-orders', {
    warehouse_id: wh1, do_number: `DOX-${TS}`, recipient_name: 'Penerima Audit', items: [{ product_id: productId, quantity: String(r1.avail + 1), location_id: rack1 }],
  });
  const [sZero] = await call(A, 'POST', '/wms/delivery-orders', {
    warehouse_id: wh1, do_number: `DOZ-${TS}`, recipient_name: 'Penerima Audit', items: [{ product_id: productId, quantity: '0', location_id: rack1 }],
  });
  let [s, d] = await call(A, 'POST', '/wms/delivery-orders', {
    warehouse_id: wh1, do_number: `DO-${TS}`, expedition_name: 'Audit Ekspedisi', driver_name: 'Sopir Audit', vehicle_plate: 'B 1234 AUD',
    recipient_name: 'Penerima Audit', items: [{ product_id: productId, quantity: '2', location_id: rack1 }],
  });
  expect.soft(s, JSON.stringify(d)).toBeLessThan(300);
  const doId = d.id; artifacts.delivery_order = doId;
  const allocAfterCreate = (await rackQty(wh1, rack1)).alloc;
  const [sDraft] = await call(A, 'POST', `/wms/delivery-orders/${doId}/dispatch`);
  const [sConf] = await call(A, 'POST', `/wms/delivery-orders/${doId}/confirm`);
  const [sConf2] = await call(A, 'POST', `/wms/delivery-orders/${doId}/confirm`);
  const [sCancelConf] = await call(A, 'DELETE', `/wms/delivery-orders/${doId}`); // cancelling confirmed DO must be rejected (409)
  // Test cancelling a DRAFT DO
  const [, canDO] = await call(A, 'POST', '/wms/delivery-orders', {
    warehouse_id: wh1, do_number: `DOCAN-${TS}`, recipient_name: 'Penerima Cancel', items: [{ product_id: productId, quantity: '1', location_id: rack1 }],
  });
  const canId = (canDO.delivery_order || canDO).id;
  const [sCancelDraft] = await call(A, 'DELETE', `/wms/delivery-orders/${canId}`); // cancelling draft DO succeeds (200)

  const [sDisp, disp] = await call(A, 'POST', `/wms/delivery-orders/${doId}/dispatch`);
  const [sDisp2] = await call(A, 'POST', `/wms/delivery-orders/${doId}/dispatch`);
  const after = await rackQty(wh1, rack1);
  const ok = sOver === 422 && sZero >= 400 && sDraft >= 400 && sConf < 300 && sConf2 >= 400 && sCancelConf === 409 && sCancelDraft < 300 && sDisp < 300 && sDisp2 >= 400 && after.qty === r1.qty - 2;
  record('e surat jalan', ok ? 'PASS' : 'FAIL',
    `overAlokasi=${sOver} qty0=${sZero} alokasiSetelahCreate=${allocAfterCreate} dispatchDraft=${sDraft} confirm=${sConf} confirmLagi=${sConf2} ` +
    `batalDraft=${sCancelDraft} batalConfirmed=${sCancelConf} dispatch=${sDisp}${sDisp >= 300 ? ' ' + JSON.stringify(disp) : ''} dispatchLagi=${sDisp2} rak ${r1.qty}->${after.qty}`);
  expect.soft(ok, JSON.stringify(over)).toBe(true);
});

test('f. POS jual 1 unit dari gudang audit (UI tidak dipakai: pilihan gudang default = gudang pertama)', async () => {
  const before = (await rackQty(wh1, rack1)).qty;
  const [sZero] = await call(A, 'POST', '/pos/checkout', { items: [{ product_id: productId, qty: 0, price: 1000 }], warehouse_id: wh1, payments: [{ method: 'CASH', amount: 1000 }] });
  const [sOver] = await call(A, 'POST', '/pos/checkout', { items: [{ product_id: productId, qty: before + 5, price: 1000 }], warehouse_id: wh1, payments: [{ method: 'CASH', amount: 999999 }] });
  const [s, o] = await call(A, 'POST', '/pos/checkout', { items: [{ product_id: productId, qty: 1, price: 1000 }], warehouse_id: wh1, payments: [{ method: 'CASH', amount: 1000 }] });
  if (s < 300) artifacts.pos_order = o.order_number;
  const after = (await rackQty(wh1, rack1)).qty;
  const ok = sZero >= 400 && sOver >= 400 && s < 300 && after === before - 1;
  record('f POS', ok ? 'PASS' : 'FAIL', `qty0=${sZero} qtyLebih=${sOver} jual1=${s}${s >= 300 ? ' ' + JSON.stringify(o) : ''} rak ${before}->${after}`);
  expect.soft(ok).toBe(true);
});

test('g. invarian: stok tidak negatif, semua mutasi punya batch_id, /inventory == /wms/stock', async () => {
  const [, rows] = await call(A, 'GET', '/wms/stock');
  const mine = (rows || []).filter((x: J) => x.product_id === productId);
  const negative = mine.filter((x: J) => num(x.quantity) < 0);
  const movs = await movements();
  const noBatch = movs.filter((m: J) => !m.batch_id);
  const t = await productTotals();
  const ok = negative.length === 0 && noBatch.length === 0 && movs.length > 0 && t.wms === t.inventory;
  record('g invarian', ok ? 'PASS' : 'FAIL',
    `baris stok=${mine.length} negatif=${negative.length} mutasi=${movs.length} tanpaBatch=${noBatch.length} wmsStock=${t.wms} inventory=${t.inventory}`);
  expect.soft(ok).toBe(true);
});

test('h. UI: halaman Warehouse & Stock menampilkan artefak audit', async ({ page }: { page: Page }) => {
  await page.goto('/login');
  await page.fill('#email', ADMIN);
  await page.fill('#password', PASS);
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/dashboard/, { timeout: 30_000 });
  await page.goto('/wms');
  await page.waitForLoadState('networkidle').catch(() => {});
  const whVisible = await page.getByText(`AUDIT-${TS}-A`).first().isVisible({ timeout: 10_000 }).catch(() => false);
  const tabBtn = page.getByRole('button', { name: /Riwayat Mutasi Stok/ });
  if (await tabBtn.isVisible().catch(() => false)) {
    await tabBtn.click();
    await page.waitForTimeout(1000);
  }
  const movVisible = await page.getByText(SFX).first().isVisible({ timeout: 10_000 }).catch(() => false);
  await page.screenshot({ path: path.join(OUT, 'stock-flows-wms-ui.png'), fullPage: true });
  record('h UI /wms', (whVisible || movVisible) ? 'PASS' : 'FAIL', `Gudang AUDIT-${TS}-A: ${whVisible}, Mutasi SKU ${SFX}: ${movVisible}`);
  expect.soft(whVisible || movVisible).toBe(true);
});

test('z. cleanup: stok -> 0, hapus produk', async () => {
  const notes: string[] = [];
  for (const [wh, rack] of [[wh1, rack1], [wh2, rack2]]) {
    if (!wh || !rack) continue;
    const q = (await rackQty(wh, rack)).avail;
    if (q > 0) {
      const [s, r] = await call(A, 'POST', '/wms/scraps', { warehouse_id: wh, product_id: productId, source_location_id: rack, quantity: String(q), reason: `Audit cleanup ${SFX}` });
      notes.push(`scrap ${q} @${rack.slice(0, 8)} -> ${s}${s >= 300 ? ' ' + JSON.stringify(r) : ''}`);
    }
  }
  const t = await productTotals();
  const [sd, del] = await call(A, 'DELETE', `/products/${productId}`);
  notes.push(`sisa stok INTERNAL=${t.wms}; DELETE produk -> ${sd}${sd >= 300 ? ' ' + JSON.stringify(del) : ''}`);
  record('z cleanup', t.wms === 0 ? 'PASS' : 'FAIL', notes.join(' | '));
});
