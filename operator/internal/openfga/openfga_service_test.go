package openfga

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

var (
	service OpenFgaService
	ctx     context.Context
	logger  logr.Logger
)

func setupIntegrationTest(t *testing.T) {
	var err error
	service, err = newOpenFgaService(Config{
		ApiUrl:   "http://localhost:8080",
		ApiToken: "foobar",
	})
	if err != nil {
		t.Fatalf("failed to initialize OpenFGA service: %v", err)
	}
	ctx = context.TODO()
	logger = log.FromContext(context.Background())
}

func TestPositiveStoreIntegration(t *testing.T) {
	setupIntegrationTest(t)
	testStoreName := uuid.NewString()

	// ACT: Create a test store
	createdStore, err := service.CreateStore(ctx, testStoreName, &logger)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	// ASSERT: Check if the test store exists
	existingStore, err := service.CheckExistingStores(ctx, testStoreName)
	if err != nil {
		t.Fatalf("failed to check existing stores: %v", err)
	}

	// Validate that the store returned by CheckExistingStores matches the one created
	if existingStore == nil {
		t.Fatalf("expected test store %q to exist, but it doesn't", testStoreName)
	}
	if existingStore.Name != createdStore.Name || existingStore.Id != createdStore.Id {
		t.Fatalf("created store %q does not match the store returned by CheckExistingStores", testStoreName)
	}
}

func TestNegativeStoreIntegration(t *testing.T) {
	setupIntegrationTest(t)
	nonExistingStoreName := "non-existing-store"

	// ASSERT: Check behavior when store doesn't exist
	nonExistingStore, err := service.CheckExistingStores(ctx, nonExistingStoreName)
	if err != nil {
		t.Fatalf("failed to check existing stores: %v", err)
	}
	if nonExistingStore != nil {
		t.Fatalf("expected store %q to be non-existent, but it exists", nonExistingStoreName)
	}
}
