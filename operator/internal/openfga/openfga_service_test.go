package openfga

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	v1 "openfga-controller/api/v1"
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
const version = "1.1.1"

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

func TestCreateAuthorizationModelIntegration(t *testing.T) {
	setupIntegrationTest(t)

	// Seed a store first
	storeName := uuid.NewString()
	store, err := service.CreateStore(ctx, storeName, &logger)
	if err != nil {
		t.Fatalf("failed to seed store: %v", err)
	}
	service.SetStoreId(store.Id)

	// Create an authorization model request
	authorizationModelRequest := &v1.AuthorizationModelRequest{
		Spec: v1.AuthorizationModelRequestSpec{
			AuthorizationModel: model,
			Version:            version,
		},
	}

	// ACT: Create authorization model
	modelID, err := service.CreateAuthorizationModel(ctx, authorizationModelRequest, &logger)
	if err != nil {
		t.Fatalf("failed to create authorization model: %v", err)
	}

	// ASSERT: Check if model ID is not empty
	if modelID == "" {
		t.Fatal("authorization model ID is empty")
	}

	// ACT and ASSERT: Check possible to set auth id
	if err := service.SetAuthorizationModelId(modelID); err != nil {
		t.Fatalf("failed to set authorization model id: %v", err)
	}
}

func TestCreateAuthorizationModelIntegration_BadModel(t *testing.T) {
	setupIntegrationTest(t)

	// Seed a store first
	storeName := uuid.NewString()
	store, err := service.CreateStore(ctx, storeName, &logger)
	if err != nil {
		t.Fatalf("failed to seed store: %v", err)
	}
	service.SetStoreId(store.Id)

	// Create an authorization model request with a bad model
	authorizationModelRequest := &v1.AuthorizationModelRequest{
		Spec: v1.AuthorizationModelRequestSpec{
			AuthorizationModel: `{"bad": "authorization model"}`, // Invalid JSON
			Version:            "v1",
		},
	}

	// ACT: Attempt to create authorization model
	_, err = service.CreateAuthorizationModel(ctx, authorizationModelRequest, &logger)

	// ASSERT: Check if error is not nil
	if err == nil {
		t.Fatal("expected error when creating authorization model with bad model, but got nil")
	}
}
