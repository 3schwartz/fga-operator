package openfga

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	openfga "github.com/openfga/go-sdk"
	ofgaClient "github.com/openfga/go-sdk/client"
	"github.com/openfga/go-sdk/credentials"
	"github.com/openfga/language/pkg/go/transformer"
	"time"
)

type PermissionServiceFactory interface {
	GetService(config Config) (PermissionService, error)
}

type PermissionService interface {
	SetStoreId(storeId string)
	CreateAuthorizationModel(ctx context.Context, authorizationModel string, log *logr.Logger) (string, error)
	CheckExistingStoresByName(ctx context.Context, storeName string) (*Store, error)
	CheckExistingStoresById(ctx context.Context, storeId string) (*Store, error)
	CreateStore(ctx context.Context, storeName string, log *logr.Logger) (*Store, error)
	CheckAuthorizationModelExists(ctx context.Context, authorizationModelId string) (bool, error)
}

type Store struct {
	Id        string
	Name      string
	CreatedAt time.Time
}

type OpenFgaServiceFactory struct{}

func (_ OpenFgaServiceFactory) GetService(config Config) (PermissionService, error) {
	return newOpenFgaService(config)
}

type OpenFgaService struct {
	client ofgaClient.OpenFgaClient
}

func newOpenFgaService(config Config) (PermissionService, error) {
	client, err := ofgaClient.NewSdkClient(&ofgaClient.ClientConfiguration{
		ApiUrl: config.ApiUrl,
		Credentials: &credentials.Credentials{
			Method: credentials.CredentialsMethodApiToken,
			Config: &credentials.Config{
				ApiToken: config.ApiToken,
			},
		},
	})
	if err != nil {
		return &OpenFgaService{}, err
	}
	return &OpenFgaService{
		*client,
	}, nil
}

func (s *OpenFgaService) SetStoreId(storeId string) {
	s.client.SetStoreId(storeId)
}

func (s *OpenFgaService) CheckExistingStoresByName(ctx context.Context, storeName string) (*Store, error) {
	return s.checkExistingStores(ctx, storeName, "")
}

func (s *OpenFgaService) CheckExistingStoresById(ctx context.Context, storeId string) (*Store, error) {
	return s.checkExistingStores(ctx, "", storeId)
}

func (s *OpenFgaService) checkExistingStores(ctx context.Context, storeName, storeId string) (*Store, error) {
	pageSize := openfga.PtrInt32(10)
	options := ofgaClient.ClientListStoresOptions{
		PageSize: pageSize,
	}
	for {
		stores, err := s.client.ListStores(ctx).Options(options).Execute()
		if err != nil {
			return nil, err
		}
		for _, oldStore := range stores.Stores {
			if storeName != "" && oldStore.Name == storeName || storeId != "" && oldStore.Id == storeId {
				return &Store{
					Id:        oldStore.Id,
					Name:      oldStore.Name,
					CreatedAt: oldStore.CreatedAt,
				}, nil
			}
		}
		if stores.ContinuationToken == "" {
			break
		}
		options = ofgaClient.ClientListStoresOptions{
			PageSize:          pageSize,
			ContinuationToken: openfga.PtrString(stores.ContinuationToken),
		}
	}
	return nil, nil
}

func (s *OpenFgaService) CheckAuthorizationModelExists(ctx context.Context, authorizationModelId string) (bool, error) {
	pageSize := openfga.PtrInt32(10)
	options := ofgaClient.ClientReadAuthorizationModelsOptions{
		PageSize: pageSize,
	}
	for {
		authModels, err := s.client.ReadAuthorizationModels(ctx).Options(options).Execute()
		if err != nil {
			return false, err
		}
		for _, authModel := range authModels.AuthorizationModels {
			if authModel.Id == authorizationModelId {
				return true, nil
			}
		}
		if authModels.ContinuationToken == nil || *authModels.ContinuationToken == "" {
			break
		}
		options = ofgaClient.ClientReadAuthorizationModelsOptions{
			PageSize:          pageSize,
			ContinuationToken: authModels.ContinuationToken,
		}
	}
	return false, nil
}

func (s *OpenFgaService) CreateStore(ctx context.Context, storeName string, log *logr.Logger) (*Store, error) {
	body := ofgaClient.ClientCreateStoreRequest{Name: storeName}
	store, err := s.client.CreateStore(ctx).Body(body).Execute()
	if err != nil {
		return nil, err
	}
	log.V(0).Info("Created store in OpenFGA", "storeOpenFGA", store)
	return &Store{
		Id:        store.Id,
		Name:      store.Name,
		CreatedAt: store.CreatedAt,
	}, nil
}

func (s *OpenFgaService) CreateAuthorizationModel(
	ctx context.Context,
	authorizationModel string,
	log *logr.Logger) (string, error) {

	generatedJsonString, err := transformer.TransformDSLToJSON(authorizationModel)
	if err != nil {
		return "", err
	}
	var body ofgaClient.ClientWriteAuthorizationModelRequest
	if err := json.Unmarshal([]byte(generatedJsonString), &body); err != nil {
	}
	data, err := s.client.WriteAuthorizationModel(ctx).Body(body).Execute()
	if err != nil {
		return "", err
	}
	log.V(0).Info("Created authorization model in OpenFGA", "authorizationModelBody", body, "authorizationModelData", data)

	return data.AuthorizationModelId, nil
}
