package openfga

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	openfga "github.com/openfga/go-sdk"
	ofgaClient "github.com/openfga/go-sdk/client"
	"github.com/openfga/go-sdk/credentials"
	"github.com/openfga/language/pkg/go/transformer"
	extensionsv1 "openfga-controller/api/v1"
	"time"
)

type PermissionServiceFactory interface {
	GetService(config Config) (PermissionService, error)
}

type PermissionService interface {
	SetStoreId(storeId string)
	SetAuthorizationModelId(authorizationModelId string) error
	CreateAuthorizationModel(ctx context.Context, authorizationModelRequest *extensionsv1.AuthorizationModelRequest, log *logr.Logger) (string, error)
	CheckExistingStores(ctx context.Context, storeName string) (*Store, error)
	CreateStore(ctx context.Context, storeName string, log *logr.Logger) (*Store, error)
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

func (s *OpenFgaService) CheckExistingStores(ctx context.Context, storeName string) (*Store, error) {
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
			if oldStore.Name == storeName {
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

func (s *OpenFgaService) SetAuthorizationModelId(authorizationModelId string) error {
	return s.client.SetAuthorizationModelId(authorizationModelId)
}

func (s *OpenFgaService) CreateAuthorizationModel(
	ctx context.Context,
	authorizationModelRequest *extensionsv1.AuthorizationModelRequest,
	log *logr.Logger) (string, error) {

	generatedJsonString, err := transformer.TransformDSLToJSON(authorizationModelRequest.Spec.AuthorizationModel)
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
