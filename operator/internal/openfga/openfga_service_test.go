package openfga

import (
	"context"
	v1 "fga-controller/api/v1"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

const model = `
model
  schema 1.1

type user

type document
  relations
    define foo: [user]
    define reader: [user]
    define writer: [user]
    define owner: [user]
`

var (
	service PermissionService
	ctx     context.Context
	logger  logr.Logger
	version = v1.ModelVersion{
		Major: 1,
		Minor: 1,
		Patch: 1,
	}
)

func setupIntegrationTest(t *testing.T) {
	var err error
	service, err = newOpenFgaService(Config{
		ApiUrl:   "http://localhost:8089",
		ApiToken: "foobar",
	})
	if err != nil {
		t.Fatalf("failed to initialize OpenFGA service: %v", err)
	}
	ctx = context.TODO()
	logger = log.FromContext(context.Background())
}

func TestPositiveStoreIntegration(t *testing.T) {
	// Arrange
	setupIntegrationTest(t)
	testStoreName := uuid.NewString()

	// Act
	createdStore, err := service.CreateStore(ctx, testStoreName, &logger)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	// Assert
	existingStore, err := service.CheckExistingStores(ctx, testStoreName)
	if err != nil {
		t.Fatalf("failed to check existing stores: %v", err)
	}

	if existingStore == nil {
		t.Fatalf("expected test store %q to exist, but it doesn't", testStoreName)
	}
	if existingStore.Name != createdStore.Name || existingStore.Id != createdStore.Id {
		t.Fatalf("created store %q does not match the store returned by CheckExistingStores", testStoreName)
	}
}

func TestNegativeStoreIntegration(t *testing.T) {
	// Arrange
	setupIntegrationTest(t)
	nonExistingStoreName := "non-existing-store"

	// Assert
	nonExistingStore, err := service.CheckExistingStores(ctx, nonExistingStoreName)
	if err != nil {
		t.Fatalf("failed to check existing stores: %v", err)
	}
	if nonExistingStore != nil {
		t.Fatalf("expected store %q to be non-existent, but it exists", nonExistingStoreName)
	}
}

func TestCreateAuthorizationModelIntegration(t *testing.T) {
	setupIntegrationTest(t)

	// Arrange
	storeName := uuid.NewString()
	store, err := service.CreateStore(ctx, storeName, &logger)
	if err != nil {
		t.Fatalf("failed to seed store: %v", err)
	}
	service.SetStoreId(store.Id)

	// Act
	modelID, err := service.CreateAuthorizationModel(ctx, model, &logger)
	if err != nil {
		t.Fatalf("failed to create authorization model: %v", err)
	}

	// Assert
	if modelID == "" {
		t.Fatal("authorization model ID is empty")
	}

	// Act & Assert
	if err := service.SetAuthorizationModelId(modelID); err != nil {
		t.Fatalf("failed to set authorization model id: %v", err)
	}
}

func TestCreateAuthorizationModelIntegration_BadModel(t *testing.T) {
	setupIntegrationTest(t)

	// Arrange
	storeName := uuid.NewString()
	store, err := service.CreateStore(ctx, storeName, &logger)
	if err != nil {
		t.Fatalf("failed to seed store: %v", err)
	}
	service.SetStoreId(store.Id)
	authorizationModel := `{"bad": "authorization model"}`

	// Act
	_, err = service.CreateAuthorizationModel(ctx, authorizationModel, &logger)

	// Assert
	if err == nil {
		t.Fatal("expected error when creating authorization model with bad model, but got nil")
	}
}
