package v1

import (
	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
	"time"
)

func TestSortAuthorizationModelInstancesByVersionAndCreatedAtDesc(t *testing.T) {
	tests := []struct {
		name     string
		input    []AuthorizationModelInstance
		expected []AuthorizationModelInstance
	}{
		{
			name: "Sorted by version and then by created at",
			input: []AuthorizationModelInstance{
				{Id: "1", Version: ModelVersion{Major: 1, Minor: 0, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "2", Version: ModelVersion{Major: 1, Minor: 0, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", Version: ModelVersion{Major: 2, Minor: 0, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "4", Version: ModelVersion{Major: 1, Minor: 0, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "5", Version: ModelVersion{Major: 2, Minor: 1, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2019, time.January, 1, 0, 0, 0, 0, time.UTC)}},
			},
			expected: []AuthorizationModelInstance{
				{Id: "5", Version: ModelVersion{Major: 2, Minor: 1, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2019, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", Version: ModelVersion{Major: 2, Minor: 0, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "2", Version: ModelVersion{Major: 1, Minor: 0, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "4", Version: ModelVersion{Major: 1, Minor: 0, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "1", Version: ModelVersion{Major: 1, Minor: 0, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
			},
		},
		{
			name: "Different Major and Minor versions",
			input: []AuthorizationModelInstance{
				{Id: "1", Version: ModelVersion{Major: 1, Minor: 1, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2020, time.February, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "2", Version: ModelVersion{Major: 2, Minor: 0, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2021, time.March, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", Version: ModelVersion{Major: 1, Minor: 2, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2022, time.April, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "4", Version: ModelVersion{Major: 1, Minor: 1, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2023, time.May, 1, 0, 0, 0, 0, time.UTC)}},
			},
			expected: []AuthorizationModelInstance{
				{Id: "2", Version: ModelVersion{Major: 2, Minor: 0, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2021, time.March, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", Version: ModelVersion{Major: 1, Minor: 2, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2022, time.April, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "1", Version: ModelVersion{Major: 1, Minor: 1, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2020, time.February, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "4", Version: ModelVersion{Major: 1, Minor: 1, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2023, time.May, 1, 0, 0, 0, 0, time.UTC)}},
			},
		},
		{
			name: "Reverse sorted by version and created at",
			input: []AuthorizationModelInstance{
				{Id: "1", Version: ModelVersion{Major: 1, Minor: 0, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "2", Version: ModelVersion{Major: 1, Minor: 0, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", Version: ModelVersion{Major: 1, Minor: 0, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "4", Version: ModelVersion{Major: 2, Minor: 0, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
			},
			expected: []AuthorizationModelInstance{
				{Id: "4", Version: ModelVersion{Major: 2, Minor: 0, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", Version: ModelVersion{Major: 1, Minor: 0, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "2", Version: ModelVersion{Major: 1, Minor: 0, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "1", Version: ModelVersion{Major: 1, Minor: 0, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
			},
		},
		{
			name: "Complex case with mixed versions and dates",
			input: []AuthorizationModelInstance{
				{Id: "1", Version: ModelVersion{Major: 1, Minor: 2, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "2", Version: ModelVersion{Major: 2, Minor: 1, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", Version: ModelVersion{Major: 1, Minor: 2, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "4", Version: ModelVersion{Major: 2, Minor: 1, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "5", Version: ModelVersion{Major: 1, Minor: 2, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2022, time.February, 1, 0, 0, 0, 0, time.UTC)}},
			},
			expected: []AuthorizationModelInstance{
				{Id: "4", Version: ModelVersion{Major: 2, Minor: 1, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "2", Version: ModelVersion{Major: 2, Minor: 1, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "1", Version: ModelVersion{Major: 1, Minor: 2, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", Version: ModelVersion{Major: 1, Minor: 2, Patch: 1}, CreatedAt: &metav1.Time{Time: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "5", Version: ModelVersion{Major: 1, Minor: 2, Patch: 0}, CreatedAt: &metav1.Time{Time: time.Date(2022, time.February, 1, 0, 0, 0, 0, time.UTC)}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SortAuthorizationModelInstancesByVersionAndCreatedAtDesc(tt.input)
			if !reflect.DeepEqual(tt.input, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, tt.input)
			}
		})
	}
}

func TestWhenNoVersionMatchThenReturnError(t *testing.T) {
	// Arrange
	currentTime := metaTime(time.Now())
	version := "1.2.1"
	authModel := AuthorizationModel{
		Status: AuthorizationModelStatus{},
		Spec: AuthorizationModelSpec{
			Instances: []AuthorizationModelInstance{
				{
					Id:                 uuid.NewString(),
					Version:            ModelVersion{Major: 2, Minor: 0, Patch: 0},
					AuthorizationModel: "some",
					CreatedAt:          metaTime(currentTime.Add(-time.Second * 7)),
				},
				{
					Id:                 uuid.NewString(),
					Version:            ModelVersion{Major: 2, Minor: 0, Patch: 1},
					AuthorizationModel: "some",
					CreatedAt:          metaTime(currentTime.Add(-time.Second * 5)),
				},
				{
					Id:                 uuid.NewString(),
					Version:            ModelVersion{Major: 1, Minor: 2, Patch: 0},
					AuthorizationModel: "some",
					CreatedAt:          metaTime(currentTime.Add(-time.Second * 6)),
				},
			},
		},
	}
	deployment := createDeployment()
	deployment.Labels[FgaAuthModelVersionLabel] = version

	// Act
	_, err := authModel.GetVersionFromDeployment(deployment)

	// Assert
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestAuthorizationModelGetVersionWithLatest(t *testing.T) {
	currentTime := metaTime(time.Now())
	version := ModelVersion{1, 1, 1}
	id := uuid.NewString()

	tests := []struct {
		name                 string
		currentId            string
		currentSchemaVersion ModelVersion
		firstLatestId        string
		firstLatestVersion   ModelVersion
		secondLatestId       string
		secondLatestVersion  ModelVersion
		thirdLatestId        string
		thirdLatestVersion   ModelVersion
	}{
		{
			name:                 "Given current and latest has same versions then return current",
			currentId:            id,
			currentSchemaVersion: version,
			firstLatestId:        uuid.NewString(),
			firstLatestVersion:   ModelVersion{0, 0, 0},
			secondLatestId:       uuid.NewString(),
			secondLatestVersion:  version,
			thirdLatestId:        uuid.NewString(),
			thirdLatestVersion:   version,
		},
		{
			name:                 "Given multiple with same version then return latest",
			currentId:            uuid.NewString(),
			currentSchemaVersion: ModelVersion{0, 0, 0},
			firstLatestId:        uuid.NewString(),
			firstLatestVersion:   version,
			secondLatestId:       uuid.NewString(),
			secondLatestVersion:  ModelVersion{0, 0, 1},
			thirdLatestId:        id,
			thirdLatestVersion:   version,
		},
		{
			name:                 "Given latest instances when version is in latest then return from latest",
			currentId:            uuid.NewString(),
			currentSchemaVersion: ModelVersion{0, 0, 3},
			firstLatestId:        uuid.NewString(),
			firstLatestVersion:   ModelVersion{0, 0, 2},
			secondLatestId:       id,
			secondLatestVersion:  version,
			thirdLatestId:        uuid.NewString(),
			thirdLatestVersion:   ModelVersion{0, 0, 1},
		},
		{
			name:                 "Given latest instances when version is same as current instance then return current instance",
			currentId:            id,
			currentSchemaVersion: version,
			firstLatestId:        uuid.NewString(),
			firstLatestVersion:   ModelVersion{0, 0, 2},
			secondLatestId:       uuid.NewString(),
			secondLatestVersion:  version,
			thirdLatestId:        uuid.NewString(),
			thirdLatestVersion:   ModelVersion{0, 0, 1},
		},
		{
			name:                 "Given latest instances when version is same as current instance then return current instance",
			currentId:            id,
			currentSchemaVersion: version,
			firstLatestId:        uuid.NewString(),
			firstLatestVersion:   ModelVersion{0, 0, 2},
			secondLatestId:       uuid.NewString(),
			secondLatestVersion:  version,
			thirdLatestId:        uuid.NewString(),
			thirdLatestVersion:   ModelVersion{0, 0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			instance := AuthorizationModelInstance{
				Id:        tt.currentId,
				Version:   tt.currentSchemaVersion,
				CreatedAt: currentTime,
			}
			authModel := AuthorizationModel{
				Status: AuthorizationModelStatus{},
				Spec: AuthorizationModelSpec{
					Instances: []AuthorizationModelInstance{
						{
							Id:                 tt.currentId,
							AuthorizationModel: "AuthorizationModel",
							Version:            tt.currentSchemaVersion,
							CreatedAt:          currentTime,
						},
						{
							Id:                 tt.firstLatestId,
							AuthorizationModel: "AuthorizationModel",
							Version:            tt.firstLatestVersion,
							CreatedAt:          metaTime(currentTime.Add(-time.Second * 7)),
						},
						{
							Id:                 tt.secondLatestId,
							AuthorizationModel: "AuthorizationModel",
							Version:            tt.secondLatestVersion,
							CreatedAt:          metaTime(currentTime.Add(-time.Second * 5)),
						},
						{
							Id:                 tt.thirdLatestId,
							AuthorizationModel: "AuthorizationModel",
							Version:            tt.thirdLatestVersion,
							CreatedAt:          metaTime(currentTime.Add(-time.Second * 6)),
						},
					},
				},
			}
			deployment := createDeployment()
			deployment.Labels[FgaAuthModelVersionLabel] = version.String()

			// Act
			actualInstance, err := authModel.GetVersionFromDeployment(deployment)

			// Assert
			if err != nil {
				t.Fatalf("Error getting version: %v", err)
			}
			if id != actualInstance.Id {
				t.Errorf("Unexpected version. Expected %v, got %v", instance.Id, actualInstance.Id)
			}
		})
	}
}

func TestWhenVersionIsSameAsCurrentInstanceThenReturnCurrentInstance(t *testing.T) {
	// Arrange
	currentTime := metaTime(time.Now())
	version := ModelVersion{1, 2, 0}
	instance := AuthorizationModelInstance{
		Id:                 uuid.NewString(),
		AuthorizationModel: "AuthorizationModel",
		Version:            version,
		CreatedAt:          currentTime,
	}
	authModel := AuthorizationModel{
		Status: AuthorizationModelStatus{},
		Spec: AuthorizationModelSpec{
			Instances: []AuthorizationModelInstance{instance},
		},
	}
	deployment := createDeployment()
	deployment.Labels[FgaAuthModelVersionLabel] = version.String()

	// Act
	actualInstance, err := authModel.GetVersionFromDeployment(deployment)

	// Assert
	if err != nil {
		t.Fatalf("Error getting version: %v", err)
	}
	if instance.Id != actualInstance.Id {
		t.Errorf("Unexpected version. Expected %v, got %v", instance.Id, actualInstance.Id)
	}
}

func TestWhenNoVersionIsPresentThenAddReturnLatest(t *testing.T) {
	// Arrange
	currentTime := metaTime(time.Now())
	version := ModelVersion{1, 2, 0}
	instance := AuthorizationModelInstance{
		Id:                 uuid.NewString(),
		AuthorizationModel: "AuthorizationModel",
		Version:            version,
		CreatedAt:          currentTime,
	}
	authModel := AuthorizationModel{
		Status: AuthorizationModelStatus{},
		Spec: AuthorizationModelSpec{
			Instances: []AuthorizationModelInstance{instance},
		},
	}
	deployment := createDeployment()

	// Act
	actualInstance, err := authModel.GetVersionFromDeployment(deployment)

	// Assert
	if err != nil {
		t.Fatalf("Error getting version: %v", err)
	}
	if instance.Id != actualInstance.Id {
		t.Errorf("Unexpected version. Expected %v, got %v", instance.Id, actualInstance.Id)
	}
}

func metaTime(t time.Time) *metav1.Time {
	return &metav1.Time{Time: t}
}

func createDeployment() appsv1.Deployment {
	name := "name"
	namespace := "namespace"
	objectMeta := metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels: map[string]string{
			"webserver": name,
		},
	}
	replicas := int32(1)
	deployment := appsv1.Deployment{
		ObjectMeta: objectMeta,
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"webserver": name,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: objectMeta,
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "main",
							Image: "nginx:alpine",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "html",
									MountPath: "/usr/share/nginx/html",
									ReadOnly:  true,
								},
								{
									Name:      "config",
									MountPath: "/etc/nginx/nginx.conf",
									ReadOnly:  true,
								},
							},
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 8080,
									Protocol:      "TCP",
								},
							},
						},
						{
							Name:  "git-sync",
							Image: "registry.k8s.io/git-sync/git-sync:v4.2.2",
							Env: []v1.EnvVar{
								{
									Name:  "GITSYNC_REF",
									Value: "master",
								},
								{
									Name:  "GITSYNC_ROOT",
									Value: "/tmp/git",
								},
								{
									Name:  "GITSYNC_PERIOD",
									Value: "30s",
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "html",
									MountPath: "/tmp/git",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "html",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{
									Medium: v1.StorageMediumDefault,
								},
							},
						},
						{
							Name: "config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: name,
									},
									Items: []v1.KeyToPath{
										{
											Key:  "nginx-config.conf",
											Path: "nginx-config.conf",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return deployment
}
