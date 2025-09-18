package cli

import (
	"context"
	"testing"

	portainer "github.com/portainer/portainer/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ktypes "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kfake "k8s.io/client-go/kubernetes/fake"
)

func Test_NamespaceAccessPoliciesDeleteNamespace_updatesPortainerConfig_whenConfigExists(t *testing.T) {
	testcases := []struct {
		name              string
		namespaceToDelete string
		expectedConfig    map[string]portainer.K8sNamespaceAccessPolicy
	}{
		{
			name:              "doesn't change config, when designated namespace absent",
			namespaceToDelete: "missing-namespace",
			expectedConfig: map[string]portainer.K8sNamespaceAccessPolicy{
				"ns1": {UserAccessPolicies: portainer.UserAccessPolicies{2: {RoleID: 0}}},
				"ns2": {UserAccessPolicies: portainer.UserAccessPolicies{2: {RoleID: 0}}},
			},
		},
		{
			name:              "removes designated namespace from config, when namespace is present",
			namespaceToDelete: "ns2",
			expectedConfig: map[string]portainer.K8sNamespaceAccessPolicy{
				"ns1": {UserAccessPolicies: portainer.UserAccessPolicies{2: {RoleID: 0}}},
			},
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			k := &KubeClient{
				cli:        kfake.NewSimpleClientset(),
				instanceID: "instance",
			}

			config := &ktypes.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      portainerConfigMapName,
					Namespace: portainerNamespace,
				},
				Data: map[string]string{
					"NamespaceAccessPolicies": `{"ns1":{"UserAccessPolicies":{"2":{"RoleId":0}}}, "ns2":{"UserAccessPolicies":{"2":{"RoleId":0}}}}`,
				},
			}

			_, err := k.cli.CoreV1().ConfigMaps(portainerNamespace).Create(context.Background(), config, metav1.CreateOptions{})
			require.NoError(t, err, "failed to create a portainer config")
			defer func() {
				k.cli.CoreV1().ConfigMaps(portainerNamespace).Delete(context.Background(), portainerConfigMapName, metav1.DeleteOptions{})
			}()

			err = k.NamespaceAccessPoliciesDeleteNamespace(test.namespaceToDelete)
			require.NoError(t, err, "failed to delete namespace")

			policies, err := k.GetNamespaceAccessPolicies()
			require.NoError(t, err, "failed to fetch policies")
			assert.Equal(t, test.expectedConfig, policies)
		})
	}
}

func TestKubeAdmin(t *testing.T) {
	kcl := &KubeClient{}
	require.False(t, kcl.GetIsKubeAdmin())

	kcl.SetIsKubeAdmin(true)
	require.True(t, kcl.GetIsKubeAdmin())

	kcl.SetIsKubeAdmin(false)
	require.False(t, kcl.GetIsKubeAdmin())
}

func TestClientNonAdminNamespaces(t *testing.T) {
	kcl := &KubeClient{}

	require.Empty(t, kcl.GetClientNonAdminNamespaces())

	nss := []string{"ns1", "ns2"}
	kcl.SetClientNonAdminNamespaces(nss)
	require.Equal(t, nss, kcl.GetClientNonAdminNamespaces())

	kcl.SetClientNonAdminNamespaces([]string{})
	require.Empty(t, kcl.GetClientNonAdminNamespaces())
}
