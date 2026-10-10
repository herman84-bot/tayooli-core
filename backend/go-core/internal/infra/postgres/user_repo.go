package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error) {
	const q = `SELECT id, tenant_id, email, password_hash, role, created_at, updated_at
	           FROM users WHERE tenant_id = $1 AND email = $2`
	var u domain.User
	err := r.db.QueryRowContext(ctx, q, tenantID, email).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &u, err
}

// GetByEmailGlobal looks up a user by email across all tenants.
// Used only by the login flow where no tenant context exists yet.
// Uses SECURITY DEFINER function to bypass RLS safely.
func (r *UserRepo) GetByEmailGlobal(ctx context.Context, email string) (*domain.User, error) {
	const q = `SELECT id, tenant_id, email, password_hash, role, created_at, updated_at
	           FROM get_user_by_email($1)`
	var u domain.User
	err := r.db.QueryRowContext(ctx, q, email).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &u, err
}

func (r *UserRepo) GetByID(ctx context.Context, userID, tenantID uuid.UUID) (*domain.User, error) {
	const q = `SELECT id, tenant_id, email, password_hash, role, created_at, updated_at
	           FROM get_user_by_id($1, $2)`
	var u domain.User
	err := r.db.QueryRowContext(ctx, q, userID, tenantID).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &u, err
}

func (r *UserRepo) CountUsersByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	const q = `SELECT COUNT(*) FROM users WHERE tenant_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, q, tenantID).Scan(&count)
	return count, err
}

func VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (r *UserRepo) CreateUser(ctx context.Context, user *domain.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, user.TenantID); err != nil {
		return err
	}

	const q = `INSERT INTO users (id, tenant_id, email, password_hash, role, created_at, updated_at)
	           VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx.ExecContext(ctx, q,
		user.ID, user.TenantID, user.Email, user.PasswordHash, user.Role,
		user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// CreateTenantAndUser provisions a new tenant and owner user in a single transaction,
// setting RLS context and creating default AI permissions and trial subscription records.
func (r *UserRepo) CreateTenantAndUser(ctx context.Context, tenant *domain.Tenant, user *domain.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Set RLS context for tenant and subsequent table insertions
	if err := setTenantLocally(ctx, tx, tenant.ID); err != nil {
		return err
	}

	// 2. Insert tenant
	const qTenant = `INSERT INTO tenants (id, name, plan, created_at, updated_at)
	                 VALUES ($1, $2, $3, $4, $5)`
	if _, err := tx.ExecContext(ctx, qTenant, tenant.ID, tenant.Name, tenant.Plan, tenant.CreatedAt, tenant.UpdatedAt); err != nil {
		return err
	}

	// 3. Insert user
	const qUser = `INSERT INTO users (id, tenant_id, email, full_name, password_hash, role, verification_token, verification_expires_at, email_verified_at, created_at, updated_at)
	               VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	if _, err := tx.ExecContext(ctx, qUser,
		user.ID, user.TenantID, user.Email, user.FullName, user.PasswordHash, user.Role,
		user.VerificationToken, user.VerificationExpiresAt, user.EmailVerifiedAt,
		user.CreatedAt, user.UpdatedAt); err != nil {
		return err
	}

	// 4. Default AI permissions
	const qAI = `INSERT INTO tenant_ai_permissions (tenant_id, autonomy_level, allowed_scopes, emergency_stop, created_at, updated_at)
	             VALUES ($1, 'assisted', '["workspace.read", "workspace.profile_write", "workspace.master_write"]'::jsonb, false, NOW(), NOW())
	             ON CONFLICT (tenant_id) DO NOTHING`
	if _, err := tx.ExecContext(ctx, qAI, tenant.ID); err != nil {
		return err
	}

	// 5. Default subscription
	const qSub = `INSERT INTO tenant_subscriptions (tenant_id, plan, status, trial_started_at, trial_ends_at, billing_period)
	              VALUES ($1, 'trial', 'trialing', NOW(), NOW() + INTERVAL '14 days', 'monthly')
	              ON CONFLICT (tenant_id) DO NOTHING`
	if _, err := tx.ExecContext(ctx, qSub, tenant.ID); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateTenantName updates tenant company name and records setup completion time.
func (r *UserRepo) UpdateTenantName(ctx context.Context, tenantID uuid.UUID, name string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return err
	}

	const q = `UPDATE tenants SET name = $1, setup_completed_at = NOW(), updated_at = NOW() WHERE id = $2`
	result, err := tx.ExecContext(ctx, q, name, tenantID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit()
}

// GetTenantProfile retrieves the tenant's company branding and profile.
func (r *UserRepo) GetTenantProfile(ctx context.Context, tenantID uuid.UUID) (*domain.TenantProfile, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	const q = `
SELECT id, name, COALESCE(address, ''), COALESCE(phone, ''), COALESCE(email, ''),
       COALESCE(tax_id, ''), COALESCE(website, ''), COALESCE(logo_url, ''),
       COALESCE(tagline, ''), COALESCE(division, '')
FROM tenants
WHERE id = $1`

	var p domain.TenantProfile
	err = tx.QueryRowContext(ctx, q, tenantID).Scan(
		&p.TenantID, &p.Name, &p.Address, &p.Phone, &p.Email,
		&p.TaxID, &p.Website, &p.LogoURL, &p.Tagline, &p.Division,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	_ = tx.Commit()
	return &p, nil
}

// UpdateTenantProfile updates the tenant's full company profile.
func (r *UserRepo) UpdateTenantProfile(ctx context.Context, tenantID uuid.UUID, p domain.TenantProfile) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return err
	}

	const q = `
UPDATE tenants
SET name = $1,
    address = $2,
    phone = $3,
    email = $4,
    tax_id = $5,
    website = $6,
    logo_url = $7,
    tagline = $8,
    division = $9,
    updated_at = NOW()
WHERE id = $10`

	result, err := tx.ExecContext(ctx, q,
		p.Name, p.Address, p.Phone, p.Email,
		p.TaxID, p.Website, p.LogoURL, p.Tagline, p.Division,
		tenantID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit()
}

func (r *UserRepo) GetUserByVerificationToken(ctx context.Context, token string) (*domain.User, error) {
	const q = `SELECT id, tenant_id, email, password_hash, role, created_at, updated_at
	           FROM users WHERE verification_token = $1 AND verification_expires_at > NOW()`
	var u domain.User
	err := r.db.QueryRowContext(ctx, q, token).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &u, err
}

func (r *UserRepo) UpdateUserVerification(ctx context.Context, userID, tenantID uuid.UUID, verifiedAt time.Time) error {
	const q = `UPDATE users SET email_verified_at = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`
	_, err := r.db.ExecContext(ctx, q, verifiedAt, userID, tenantID)
	return err
}

func (r *UserRepo) ClearVerificationToken(ctx context.Context, userID, tenantID uuid.UUID) error {
	const q = `UPDATE users SET verification_token = NULL, verification_expires_at = NULL, updated_at = NOW() WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, q, userID, tenantID)
	return err
}

func (r *UserRepo) UpdateVerificationToken(ctx context.Context, userID, tenantID uuid.UUID, token string, expiresAt time.Time) error {
	const q = `UPDATE users SET verification_token = $1, verification_expires_at = $2, updated_at = NOW() WHERE id = $3 AND tenant_id = $4`
	_, err := r.db.ExecContext(ctx, q, token, expiresAt, userID, tenantID)
	return err
}

// ListByTenant returns all users belonging to a tenant.
func (r *UserRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	const q = `SELECT id, tenant_id, email, email AS full_name, role, created_at, updated_at
	           FROM users WHERE tenant_id = $1 ORDER BY created_at ASC`
	rows, err := tx.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.FullName, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.AssignedWarehouses = []domain.AssignedWarehouse{}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch assigned warehouses for all users in this tenant
	const whQuery = `
		SELECT uw.user_id, w.id, w.code, w.name
		FROM user_warehouses uw
		JOIN warehouses w ON w.id = uw.warehouse_id AND w.tenant_id = uw.tenant_id
		WHERE uw.tenant_id = $1
		ORDER BY w.name ASC`
	whRows, err := tx.QueryContext(ctx, whQuery, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list user warehouses: %w", err)
	}
	defer whRows.Close()
	userWhMap := make(map[uuid.UUID][]domain.AssignedWarehouse)
	for whRows.Next() {
		var uid uuid.UUID
		var aw domain.AssignedWarehouse
		if err := whRows.Scan(&uid, &aw.ID, &aw.Code, &aw.Name); err != nil {
			return nil, fmt.Errorf("scan user warehouse: %w", err)
		}
		userWhMap[uid] = append(userWhMap[uid], aw)
	}
	if err := whRows.Err(); err != nil {
		return nil, fmt.Errorf("user warehouses rows: %w", err)
	}
	for i := range users {
		if whs, ok := userWhMap[users[i].ID]; ok {
			users[i].AssignedWarehouses = whs
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUserRole changes a user's role within a tenant.
func (r *UserRepo) UpdateUserRole(ctx context.Context, userID, tenantID uuid.UUID, role string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return err
	}

	const q = `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`
	result, err := tx.ExecContext(ctx, q, role, userID, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit()
}

// DeleteUser removes a user from a tenant.
func (r *UserRepo) DeleteUser(ctx context.Context, userID, tenantID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return err
	}

	const q = `DELETE FROM users WHERE id = $1 AND tenant_id = $2`
	result, err := tx.ExecContext(ctx, q, userID, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit()
}

// replaceUserWarehousesTx swaps a user's warehouse assignments inside an
// existing transaction. A warehouse that does not belong to the tenant fails
// the composite FK (warehouse_id, tenant_id) and aborts the whole tx.
func replaceUserWarehousesTx(ctx context.Context, tx *sql.Tx, tenantID, userID uuid.UUID, warehouseIDs []uuid.UUID) error {
	const delQ = `DELETE FROM user_warehouses WHERE tenant_id = $1 AND user_id = $2`
	if _, err := tx.ExecContext(ctx, delQ, tenantID, userID); err != nil {
		return fmt.Errorf("delete user_warehouses: %w", err)
	}
	const insQ = `INSERT INTO user_warehouses (user_id, warehouse_id, tenant_id, assigned_at)
	              VALUES ($1, $2, $3, NOW())
	              ON CONFLICT (user_id, warehouse_id) DO NOTHING`
	for _, whID := range warehouseIDs {
		if _, err := tx.ExecContext(ctx, insQ, userID, whID, tenantID); err != nil {
			return fmt.Errorf("%w: gudang %s tidak valid", domain.ErrInvalidInput, whID)
		}
	}
	return nil
}

// CreateUserWithWarehouses inserts a user and its warehouse assignments
// atomically: either both persist or neither does.
func (r *UserRepo) CreateUserWithWarehouses(ctx context.Context, user *domain.User, warehouseIDs []uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, user.TenantID); err != nil {
		return err
	}
	const q = `INSERT INTO users (id, tenant_id, email, password_hash, role, created_at, updated_at)
	           VALUES ($1, $2, $3, $4, $5, $6, $7)`
	if _, err := tx.ExecContext(ctx, q,
		user.ID, user.TenantID, user.Email, user.PasswordHash, user.Role,
		user.CreatedAt, user.UpdatedAt); err != nil {
		return err
	}
	if err := replaceUserWarehousesTx(ctx, tx, user.TenantID, user.ID, warehouseIDs); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateUserRoleAndWarehouses changes a user's role and replaces its
// warehouse assignments in one transaction (no partial state).
func (r *UserRepo) UpdateUserRoleAndWarehouses(ctx context.Context, userID, tenantID uuid.UUID, role string, warehouseIDs []uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return err
	}
	const q = `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`
	result, err := tx.ExecContext(ctx, q, role, userID, tenantID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	if err := replaceUserWarehousesTx(ctx, tx, tenantID, userID, warehouseIDs); err != nil {
		return err
	}
	return tx.Commit()
}

// GetUserWarehouses returns the list of warehouses assigned to a specific user.
func (r *UserRepo) GetUserWarehouses(ctx context.Context, tenantID, userID uuid.UUID) ([]domain.AssignedWarehouse, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	const q = `
		SELECT w.id, w.code, w.name
		FROM user_warehouses uw
		JOIN warehouses w ON w.id = uw.warehouse_id AND w.tenant_id = uw.tenant_id
		WHERE uw.tenant_id = $1 AND uw.user_id = $2
		ORDER BY w.name ASC`
	rows, err := tx.QueryContext(ctx, q, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.AssignedWarehouse
	for rows.Next() {
		var aw domain.AssignedWarehouse
		if err := rows.Scan(&aw.ID, &aw.Code, &aw.Name); err != nil {
			return nil, err
		}
		result = append(result, aw)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if result == nil {
		result = []domain.AssignedWarehouse{}
	}
	return result, nil
}

// UpdatePasswordResetToken stores a reset token and its expiry for a user.
// Runs through a SECURITY DEFINER function because forgot-password is a public
// route without tenant context — a plain UPDATE would be silently filtered by RLS.
func (r *UserRepo) UpdatePasswordResetToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	const q = `SELECT set_password_reset_token($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, q, userID, token, expiresAt)
	return err
}

// GetUserByResetToken looks up a user by their password reset token.
// Uses a SECURITY DEFINER function for the same RLS reason as above.
func (r *UserRepo) GetUserByResetToken(ctx context.Context, token string) (*domain.User, error) {
	const q = `SELECT id, tenant_id, email, password_hash, role, created_at, updated_at
	           FROM get_user_by_password_reset_token($1)`
	var u domain.User
	err := r.db.QueryRowContext(ctx, q, token).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return &u, err
}

// UpdatePassword sets a new password hash and clears the reset token.
// Uses a SECURITY DEFINER function for the same RLS reason as above.
func (r *UserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	const q = `SELECT update_user_password($1, $2)`
	_, err := r.db.ExecContext(ctx, q, userID, passwordHash)
	return err
}
