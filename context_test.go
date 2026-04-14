package spi_test

import (
	"context"
	"testing"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

func TestTenantIDIsNamedType(t *testing.T) {
	var tid spi.TenantID = "test-tenant"
	if tid == "" {
		t.Fatal("expected non-empty tenant ID")
	}
}

func TestSystemTenantIDConstant(t *testing.T) {
	if spi.SystemTenantID == "" {
		t.Fatal("expected SystemTenantID to be defined")
	}
	if spi.SystemTenantID != "SYSTEM" {
		t.Errorf("expected SYSTEM, got %s", spi.SystemTenantID)
	}
}

func TestUserContextCarriesTenant(t *testing.T) {
	tenant := spi.Tenant{ID: "tenant-A", Name: "Tenant A"}
	uc := &spi.UserContext{
		UserID: "user-1",
		Tenant: tenant,
		Roles:  []string{"USER"},
	}
	ctx := spi.WithUserContext(context.Background(), uc)
	got := spi.MustGetUserContext(ctx)
	if got.Tenant.ID != "tenant-A" {
		t.Errorf("expected tenant-A, got %s", got.Tenant.ID)
	}
	if got.Tenant.Name != "Tenant A" {
		t.Errorf("expected Tenant A, got %s", got.Tenant.Name)
	}
}
