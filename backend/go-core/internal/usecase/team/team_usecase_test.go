package team

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type fakeRepo struct {
	users         map[uuid.UUID]*domain.User
	warehouses    map[uuid.UUID][]uuid.UUID
	failWarehouse bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: map[uuid.UUID]*domain.User{}, warehouses: map[uuid.UUID][]uuid.UUID{}}
}

func (f *fakeRepo) ListByTenant(_ context.Context, _ uuid.UUID) ([]domain.User, error) {
	var out []domain.User
	for _, u := range f.users {
		out = append(out, *u)
	}
	return out, nil
}
func (f *fakeRepo) GetByID(_ context.Context, id, _ uuid.UUID) (*domain.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}
func (f *fakeRepo) GetByEmail(_ context.Context, _ uuid.UUID, email string) (*domain.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (f *fakeRepo) CreateUser(_ context.Context, u *domain.User) error {
	f.users[u.ID] = u
	return nil
}
func (f *fakeRepo) UpdateUserRole(_ context.Context, id, _ uuid.UUID, role string) error {
	u, ok := f.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	u.Role = role
	return nil
}
func (f *fakeRepo) DeleteUser(_ context.Context, id, _ uuid.UUID) error {
	delete(f.users, id)
	return nil
}
func (f *fakeRepo) CreateUserWithWarehouses(_ context.Context, u *domain.User, ids []uuid.UUID) error {
	if f.failWarehouse {
		return domain.ErrInvalidInput // simulasi FK gagal: tx di-rollback, user tidak tersimpan
	}
	f.users[u.ID] = u
	f.warehouses[u.ID] = append([]uuid.UUID{}, ids...)
	return nil
}
func (f *fakeRepo) UpdateUserRoleAndWarehouses(_ context.Context, id, _ uuid.UUID, role string, ids []uuid.UUID) error {
	u, ok := f.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	if f.failWarehouse {
		return domain.ErrInvalidInput
	}
	u.Role = role
	f.warehouses[id] = append([]uuid.UUID{}, ids...)
	return nil
}
func (f *fakeRepo) GetUserWarehouses(_ context.Context, _, userID uuid.UUID) ([]domain.AssignedWarehouse, error) {
	out := []domain.AssignedWarehouse{}
	for _, id := range f.warehouses[userID] {
		out = append(out, domain.AssignedWarehouse{ID: id, Code: "WH", Name: "Gudang"})
	}
	return out, nil
}

func TestInviteMember_AllNewRolesAccepted(t *testing.T) {
	ctx := context.Background()
	tenant, requester := uuid.New(), uuid.New()
	wh := []uuid.UUID{uuid.New()}
	cases := map[string][]uuid.UUID{
		"admin": nil, "warehouse_manager": wh, "warehouse": wh, "cashier": nil, "auditor": nil, "member": nil,
	}
	for role, whs := range cases {
		uc := New(newFakeRepo())
		u, err := uc.InviteMember(ctx, tenant, role+"@x.com", role, whs, requester)
		if err != nil {
			t.Fatalf("role %s: unexpected error %v", role, err)
		}
		if u.Role != role {
			t.Fatalf("role %s: got %s", role, u.Role)
		}
		if len(u.AssignedWarehouses) != len(whs) {
			t.Fatalf("role %s: expected %d warehouses, got %d", role, len(whs), len(u.AssignedWarehouses))
		}
	}
}

func TestInviteMember_RejectsOwnerAndUnknownRole(t *testing.T) {
	uc := New(newFakeRepo())
	for _, role := range []string{"owner", "superadmin", "", "cfo"} {
		if _, err := uc.InviteMember(context.Background(), uuid.New(), "a@x.com", role, nil, uuid.New()); err == nil {
			t.Fatalf("role %q must be rejected", role)
		}
	}
}

func TestInviteMember_WarehouseRolesRequireWarehouse(t *testing.T) {
	uc := New(newFakeRepo())
	for _, role := range []string{"warehouse", "warehouse_manager"} {
		if _, err := uc.InviteMember(context.Background(), uuid.New(), "a@x.com", role, nil, uuid.New()); err == nil {
			t.Fatalf("role %s without warehouse must be rejected", role)
		}
	}
}

func TestChangeRole_Invariants(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	uc := New(repo)
	tenant := uuid.New()
	owner := &domain.User{ID: uuid.New(), Role: "owner", Email: "o@x.com"}
	admin := &domain.User{ID: uuid.New(), Role: "admin", Email: "a@x.com"}
	staff := &domain.User{ID: uuid.New(), Role: "warehouse", Email: "s@x.com"}
	repo.users[owner.ID], repo.users[admin.ID], repo.users[staff.ID] = owner, admin, staff
	repo.warehouses[staff.ID] = []uuid.UUID{uuid.New()}

	if err := uc.ChangeRole(ctx, tenant, owner.ID, "member", nil, admin.ID); err == nil {
		t.Fatal("owner role must not be changeable")
	}
	if err := uc.ChangeRole(ctx, tenant, admin.ID, "member", nil, admin.ID); err == nil {
		t.Fatal("self role change must be rejected")
	}
	if err := uc.ChangeRole(ctx, tenant, staff.ID, "warehouse_manager", nil, admin.ID); err == nil {
		t.Fatal("warehouse_manager without warehouse must be rejected")
	}
	if err := uc.ChangeRole(ctx, tenant, staff.ID, "owner", nil, admin.ID); err == nil {
		t.Fatal("promotion to owner must be rejected")
	}
	// Staff becomes cashier: warehouse assignments must be cleared.
	if err := uc.ChangeRole(ctx, tenant, staff.ID, "cashier", nil, admin.ID); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if staff.Role != "cashier" || len(repo.warehouses[staff.ID]) != 0 {
		t.Fatalf("expected cashier with 0 warehouses, got %s/%d", staff.Role, len(repo.warehouses[staff.ID]))
	}
}

func TestRemoveMember_OwnerProtected(t *testing.T) {
	repo := newFakeRepo()
	uc := New(repo)
	owner := &domain.User{ID: uuid.New(), Role: "owner"}
	repo.users[owner.ID] = owner
	if err := uc.RemoveMember(context.Background(), uuid.New(), owner.ID, uuid.New()); err == nil {
		t.Fatal("owner must not be removable")
	}
}

func TestInviteMember_AtomicOnWarehouseFailure(t *testing.T) {
	repo := newFakeRepo()
	repo.failWarehouse = true
	uc := New(repo)
	_, err := uc.InviteMember(context.Background(), uuid.New(), "s@x.com", "warehouse", []uuid.UUID{uuid.New()}, uuid.New())
	if err == nil {
		t.Fatal("expected error when warehouse assignment fails")
	}
	if len(repo.users) != 0 {
		t.Fatalf("no user may be persisted on failure, got %d", len(repo.users))
	}
}

func TestChangeRole_AtomicOnWarehouseFailure(t *testing.T) {
	repo := newFakeRepo()
	uc := New(repo)
	staff := &domain.User{ID: uuid.New(), Role: "cashier"}
	repo.users[staff.ID] = staff
	repo.failWarehouse = true
	if err := uc.ChangeRole(context.Background(), uuid.New(), staff.ID, "warehouse", []uuid.UUID{uuid.New()}, uuid.New()); err == nil {
		t.Fatal("expected error")
	}
	if staff.Role != "cashier" {
		t.Fatalf("role must remain cashier on failure, got %s", staff.Role)
	}
}

func TestInviteMember_NonWarehouseRoleDropsWarehouses(t *testing.T) {
	repo := newFakeRepo()
	uc := New(repo)
	u, err := uc.InviteMember(context.Background(), uuid.New(), "k@x.com", "cashier", []uuid.UUID{uuid.New()}, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.warehouses[u.ID]) != 0 {
		t.Fatal("cashier must not keep warehouse assignments")
	}
}