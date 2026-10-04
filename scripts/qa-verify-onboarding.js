// scripts/qa-verify-onboarding.js
// E2E QA Verification Script for Tayooli ERP User Registration & Onboarding Flow

const { execSync } = require('child_process');

const BASE_URL = process.env.API_URL || 'http://104.197.178.237:8081';
const ZONE = 'us-central1-c';
const INSTANCE = 'tayooli-server';

function runPsql(sql) {
  // Execute SQL command inside the PostgreSQL container or service on tayooli-server
  const cmd = `gcloud compute ssh ${INSTANCE} --zone=${ZONE} --command="sudo -u postgres psql -d tayooli_erp -t -A -F ',' -c \\"${sql.replace(/"/g, '\\"')}\\""`;
  try {
    const stdout = execSync(cmd, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
    return stdout.trim();
  } catch (err) {
    console.error('PSQL Error:', err.stderr || err.message);
    throw err;
  }
}

async function runVerification() {
  console.log('================================================================');
  console.log('🚀 TAYOOLI ERP - QA E2E ONBOARDING & REGISTRATION VERIFICATION');
  console.log(`🎯 Target API: ${BASE_URL}`);
  console.log('================================================================\n');

  const timestamp = Date.now();
  const testEmail = `qa_owner_${timestamp}@tayoolitest.internal`;
  const testFullName = `QA Owner ${timestamp}`;
  const testPassword = 'Password123!';
  const renamedCompany = `PT Verification Global ${timestamp}`;

  const results = {
    step1_postgres_tenant_user: false,
    step2_register_response_and_cookie: false,
    step3_rename_workspace: false,
    step4a_duplicate_email: false,
    step4b_weak_password: false,
    step4c_missing_fields: false,
    step4d_unauthorized_workspaces: false,
  };

  const evidence = {};
  let createdTenantId = null;
  let createdUserId = null;
  let authToken = null;
  let authCookie = null;

  try {
    // -------------------------------------------------------------------------
    // STEP 2 & 1: Register New User & Tenant
    // -------------------------------------------------------------------------
    console.log(`[TEST 1 & 2] POST /api/v1/auth/register for new user: ${testEmail}...`);
    const regRes = await fetch(`${BASE_URL}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        full_name: testFullName,
        email: testEmail,
        password: testPassword,
      }),
    });

    const regStatus = regRes.status;
    const regSetCookie = regRes.headers.get('set-cookie');
    const regBody = await regRes.json();

    console.log(`   -> HTTP Status: ${regStatus}`);
    console.log(`   -> Set-Cookie: ${regSetCookie}`);
    console.log(`   -> Response Body:`, JSON.stringify(regBody, null, 2));

    evidence.registrationResponse = {
      status: regStatus,
      setCookie: regSetCookie,
      body: regBody,
    };

    if (regStatus === 201 && regSetCookie && regSetCookie.includes('tayooli_auth=') && regBody.user && regBody.token) {
      results.step2_register_response_and_cookie = true;
      console.log('   ✅ STEP 2 PASSED: Status 201, tayooli_auth cookie set, user info returned.');
    } else {
      console.error('   ❌ STEP 2 FAILED');
    }

    createdTenantId = regBody.user?.tenant_id;
    createdUserId = regBody.user?.id;
    authToken = regBody.token;
    
    // Extract cookie value
    const match = regSetCookie ? regSetCookie.match(/tayooli_auth=([^;]+)/) : null;
    authCookie = match ? match[1] : authToken;

    // -------------------------------------------------------------------------
    // STEP 1: Verify in PostgreSQL
    // -------------------------------------------------------------------------
    console.log(`\n[TEST 1] Verifying PostgreSQL records for Tenant and User...`);
    console.log(`   Tenant ID: ${createdTenantId}`);
    console.log(`   User ID:   ${createdUserId}`);

    const dbUserRow = runPsql(`SELECT id, tenant_id, email, full_name, role FROM users WHERE id = '${createdUserId}';`);
    const dbTenantRow = runPsql(`SELECT id, name, plan FROM tenants WHERE id = '${createdTenantId}';`);

    console.log(`   -> DB User Row:   ${dbUserRow}`);
    console.log(`   -> DB Tenant Row: ${dbTenantRow}`);

    evidence.dbRecords = {
      userRow: dbUserRow,
      tenantRow: dbTenantRow,
    };

    const userParts = dbUserRow.split(',');
    const tenantParts = dbTenantRow.split(',');

    if (
      userParts[0] === createdUserId &&
      userParts[1] === createdTenantId &&
      tenantParts[0] === createdTenantId &&
      userParts[4] === 'owner'
    ) {
      results.step1_postgres_tenant_user = true;
      console.log('   ✅ STEP 1 PASSED: Tenant and User created in PostgreSQL with matching tenant_id and role="owner".');
    } else {
      console.error('   ❌ STEP 1 FAILED');
    }

    // -------------------------------------------------------------------------
    // STEP 3: Rename Company via POST /api/v1/workspaces
    // -------------------------------------------------------------------------
    console.log(`\n[TEST 3] POST /api/v1/workspaces to rename company to "${renamedCompany}"...`);
    const wsRes = await fetch(`${BASE_URL}/api/v1/workspaces`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Cookie': `tayooli_auth=${authCookie}`,
        'Authorization': `Bearer ${authToken}`,
      },
      body: JSON.stringify({
        company_name: renamedCompany,
      }),
    });

    const wsStatus = wsRes.status;
    const wsBody = await wsRes.json();

    console.log(`   -> HTTP Status: ${wsStatus}`);
    console.log(`   -> Response Body:`, JSON.stringify(wsBody, null, 2));

    const dbUpdatedTenant = runPsql(`SELECT id, name, setup_completed_at IS NOT NULL FROM tenants WHERE id = '${createdTenantId}';`);
    console.log(`   -> DB Updated Tenant: ${dbUpdatedTenant}`);

    evidence.workspaceRename = {
      status: wsStatus,
      body: wsBody,
      dbUpdatedTenant,
    };

    if (wsStatus === 200 && wsBody.success === true && wsBody.company_name === renamedCompany && dbUpdatedTenant.includes(renamedCompany)) {
      results.step3_rename_workspace = true;
      console.log('   ✅ STEP 3 PASSED: Workspace company renamed and returns 200 OK.');
    } else {
      console.error('   ❌ STEP 3 FAILED');
    }

    // -------------------------------------------------------------------------
    // STEP 4: Edge Cases
    // -------------------------------------------------------------------------
    console.log('\n[TEST 4] Edge Cases:');

    // 4a: Duplicate Email
    console.log(`   4a. Testing duplicate email registration (${testEmail})...`);
    const dupRes = await fetch(`${BASE_URL}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        full_name: 'Another User',
        email: testEmail,
        password: 'Password123!',
      }),
    });
    const dupStatus = dupRes.status;
    const dupBody = await dupRes.json();
    console.log(`       -> HTTP Status: ${dupStatus}`);
    console.log(`       -> Response Body:`, JSON.stringify(dupBody));
    evidence.edgeCaseDuplicateEmail = { status: dupStatus, body: dupBody };

    if (dupStatus === 409 && (dupBody.error?.message === 'email sudah terdaftar' || dupBody.error === 'email sudah terdaftar')) {
      results.step4a_duplicate_email = true;
      console.log('       ✅ 4a PASSED: 409 Conflict "email sudah terdaftar"');
    } else {
      console.error('       ❌ 4a FAILED');
    }

    // 4b: Weak Password (<8 chars)
    console.log(`   4b. Testing weak password registration (7 chars)...`);
    const weakRes = await fetch(`${BASE_URL}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        full_name: 'Weak Pass User',
        email: `weak_${timestamp}@tayoolitest.internal`,
        password: 'pass123',
      }),
    });
    const weakStatus = weakRes.status;
    const weakBody = await weakRes.json();
    console.log(`       -> HTTP Status: ${weakStatus}`);
    console.log(`       -> Response Body:`, JSON.stringify(weakBody));
    evidence.edgeCaseWeakPassword = { status: weakStatus, body: weakBody };

    if (weakStatus === 400 && (weakBody.error?.message?.includes('8') || weakBody.error?.includes('8'))) {
      results.step4b_weak_password = true;
      console.log('       ✅ 4b PASSED: 400 Bad Request for weak password');
    } else {
      console.error('       ❌ 4b FAILED');
    }

    // 4c: Missing Fields
    console.log(`   4c. Testing missing fields (empty object)...`);
    const missRes = await fetch(`${BASE_URL}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({}),
    });
    const missStatus = missRes.status;
    const missBody = await missRes.json();
    console.log(`       -> HTTP Status: ${missStatus}`);
    console.log(`       -> Response Body:`, JSON.stringify(missBody));
    evidence.edgeCaseMissingFields = { status: missStatus, body: missBody };

    if (missStatus === 400) {
      results.step4c_missing_fields = true;
      console.log('       ✅ 4c PASSED: 400 Bad Request for missing fields');
    } else {
      console.error('       ❌ 4c FAILED');
    }

    // 4d: Unauthorized access to /workspaces
    console.log(`   4d. Testing unauthorized access to POST /api/v1/workspaces...`);
    const unauthRes = await fetch(`${BASE_URL}/api/v1/workspaces`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        company_name: 'Hacker Company',
      }),
    });
    const unauthStatus = unauthRes.status;
    const unauthBody = await unauthRes.json();
    console.log(`       -> HTTP Status: ${unauthStatus}`);
    console.log(`       -> Response Body:`, JSON.stringify(unauthBody));
    evidence.edgeCaseUnauthorizedWorkspaces = { status: unauthStatus, body: unauthBody };

    if (unauthStatus === 401) {
      results.step4d_unauthorized_workspaces = true;
      console.log('       ✅ 4d PASSED: 401 Unauthorized for unauthenticated workspace access');
    } else {
      console.error('       ❌ 4d FAILED');
    }

  } finally {
    // -------------------------------------------------------------------------
    // Cleanup QA Test Data
    // -------------------------------------------------------------------------
    if (createdTenantId) {
      console.log(`\n[CLEANUP] Removing test tenant ${createdTenantId} from PostgreSQL...`);
      try {
        const delRes = runPsql(`DELETE FROM tenants WHERE id = '${createdTenantId}';`);
        console.log(`   -> Cleanup result: ${delRes}`);
      } catch (e) {
        console.warn(`   -> Warning: Cleanup failed:`, e.message);
      }
    }
  }

  console.log('\n================================================================');
  console.log('📋 VERIFICATION SUMMARY:');
  console.log('================================================================');
  console.log(`1. PostgreSQL Tenant & User created with matching tenant_id (role: owner): ${results.step1_postgres_tenant_user ? 'PASSED' : 'FAILED'}`);
  console.log(`2. POST /api/v1/auth/register (201, tayooli_auth cookie, user body):        ${results.step2_register_response_and_cookie ? 'PASSED' : 'FAILED'}`);
  console.log(`3. POST /api/v1/workspaces (company rename 200 OK):                         ${results.step3_rename_workspace ? 'PASSED' : 'FAILED'}`);
  console.log(`4a. Duplicate email -> 409 Conflict "email sudah terdaftar":                 ${results.step4a_duplicate_email ? 'PASSED' : 'FAILED'}`);
  console.log(`4b. Weak password (<8 chars) -> 400 Bad Request:                             ${results.step4b_weak_password ? 'PASSED' : 'FAILED'}`);
  console.log(`4c. Missing fields -> 400 Bad Request:                                       ${results.step4c_missing_fields ? 'PASSED' : 'FAILED'}`);
  console.log(`4d. Unauthorized access to /workspaces -> 401 Unauthorized:                   ${results.step4d_unauthorized_workspaces ? 'PASSED' : 'FAILED'}`);

  const allPassed = Object.values(results).every(Boolean);
  console.log('----------------------------------------------------------------');
  console.log(`OVERALL E2E VERDICT: ${allPassed ? 'PASSED' : 'FAILED'}`);
  console.log('================================================================\n');

  return { allPassed, results, evidence };
}

runVerification().then(
  ({ allPassed }) => {
    process.exit(allPassed ? 0 : 1);
  },
  (err) => {
    console.error('Fatal execution error:', err);
    process.exit(1);
  }
);
