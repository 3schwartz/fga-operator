package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
	"time"
)

func TestSortAuthorizationModelInstancesByCreatedAtDesc(t *testing.T) {
	tests := []struct {
		name     string
		input    []AuthorizationModelInstance
		expected []AuthorizationModelInstance
	}{
		{
			name: "Sorted order",
			input: []AuthorizationModelInstance{
				{Id: "1", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "2", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
			},
			expected: []AuthorizationModelInstance{
				{Id: "2", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "1", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
			},
		},
		{
			name: "Already sorted",
			input: []AuthorizationModelInstance{
				{Id: "2", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "1", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
			},
			expected: []AuthorizationModelInstance{
				{Id: "2", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "1", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
			},
		},
		{
			name: "Reverse sorted",
			input: []AuthorizationModelInstance{
				{Id: "3", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "1", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "2", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
			},
			expected: []AuthorizationModelInstance{
				{Id: "2", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "1", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)}},
				{Id: "3", SchemaVersion: "v1", CreatedAt: &metav1.Time{Time: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SortAuthorizationModelInstancesByCreatedAtDesc(tt.input)
			if !reflect.DeepEqual(tt.input, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, tt.input)
			}
		})
	}
}

func TestFoo(t *testing.T) {
	// Arrange
	time := &metav1.Time{Time: time.Now()}
	authModel := AuthorizationModel{
		Status: AuthorizationModelStatus{},
		Spec: AuthorizationModelSpec{
			Instance: AuthorizationModelInstance{
				Id:            "asd",
				SchemaVersion: "1.2",
				CreatedAt:     time,
			},
			AuthorizationModel: "AuthorizationModel",
			LatestModels:       make([]AuthorizationModelInstance, 0),
		},
	}
	deployment := createDeployment("name", "namespace")

	// Act

	// Assert

}

func createDeployment(name, namespace string) *appsv1.Deployment {
	objectMeta := metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels: map[string]string{
			"webserver": name,
		},
	}
	replicas := int32(1)
	deployment := &appsv1.Deployment{
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
