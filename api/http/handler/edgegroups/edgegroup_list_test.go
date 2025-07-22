package edgegroups

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/datastore"
	"github.com/portainer/portainer/api/internal/testhelpers"
	"github.com/portainer/portainer/api/roar"

	"github.com/segmentio/encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getEndpointTypes(t *testing.T) {
	endpoints := []portainer.Endpoint{
		{ID: 1, Type: portainer.DockerEnvironment},
		{ID: 2, Type: portainer.AgentOnDockerEnvironment},
		{ID: 3, Type: portainer.AzureEnvironment},
		{ID: 4, Type: portainer.EdgeAgentOnDockerEnvironment},
		{ID: 5, Type: portainer.KubernetesLocalEnvironment},
		{ID: 6, Type: portainer.AgentOnKubernetesEnvironment},
		{ID: 7, Type: portainer.EdgeAgentOnKubernetesEnvironment},
	}

	datastore := testhelpers.NewDatastore(testhelpers.WithEndpoints(endpoints))

	tests := []struct {
		endpointIds []portainer.EndpointID
		expected    []portainer.EndpointType
	}{
		{endpointIds: []portainer.EndpointID{1}, expected: []portainer.EndpointType{portainer.DockerEnvironment}},
		{endpointIds: []portainer.EndpointID{2}, expected: []portainer.EndpointType{portainer.AgentOnDockerEnvironment}},
		{endpointIds: []portainer.EndpointID{3}, expected: []portainer.EndpointType{portainer.AzureEnvironment}},
		{endpointIds: []portainer.EndpointID{4}, expected: []portainer.EndpointType{portainer.EdgeAgentOnDockerEnvironment}},
		{endpointIds: []portainer.EndpointID{5}, expected: []portainer.EndpointType{portainer.KubernetesLocalEnvironment}},
		{endpointIds: []portainer.EndpointID{6}, expected: []portainer.EndpointType{portainer.AgentOnKubernetesEnvironment}},
		{endpointIds: []portainer.EndpointID{7}, expected: []portainer.EndpointType{portainer.EdgeAgentOnKubernetesEnvironment}},
		{endpointIds: []portainer.EndpointID{7, 2}, expected: []portainer.EndpointType{portainer.EdgeAgentOnKubernetesEnvironment, portainer.AgentOnDockerEnvironment}},
		{endpointIds: []portainer.EndpointID{6, 4, 1}, expected: []portainer.EndpointType{portainer.AgentOnKubernetesEnvironment, portainer.EdgeAgentOnDockerEnvironment, portainer.DockerEnvironment}},
		{endpointIds: []portainer.EndpointID{1, 2, 3}, expected: []portainer.EndpointType{portainer.DockerEnvironment, portainer.AgentOnDockerEnvironment, portainer.AzureEnvironment}},
	}

	for _, test := range tests {
		ans, err := getEndpointTypes(datastore, roar.FromSlice(test.endpointIds))
		assert.NoError(t, err, "getEndpointTypes shouldn't fail")

		assert.ElementsMatch(t, test.expected, ans, "getEndpointTypes expected to return %b for %v, but returned %b", test.expected, test.endpointIds, ans)
	}
}

func Test_getEndpointTypes_failWhenEndpointDontExist(t *testing.T) {
	datastore := testhelpers.NewDatastore(testhelpers.WithEndpoints([]portainer.Endpoint{}))

	_, err := getEndpointTypes(datastore, roar.FromSlice([]portainer.EndpointID{1}))
	assert.Error(t, err, "getEndpointTypes should fail")
}

func TestEdgeGroupListHandler(t *testing.T) {
	_, store := datastore.MustNewTestStore(t, true, true)

	handler := NewHandler(testhelpers.NewTestRequestBouncer())
	handler.DataStore = store

	err := store.EndpointGroup().Create(&portainer.EndpointGroup{
		ID:   1,
		Name: "Test Group",
	})
	require.NoError(t, err)

	for i := range 3 {
		err = store.Endpoint().Create(&portainer.Endpoint{
			ID:      portainer.EndpointID(i + 1),
			Name:    "Test Endpoint " + strconv.Itoa(i+1),
			Type:    portainer.EdgeAgentOnDockerEnvironment,
			GroupID: 1,
		})
		require.NoError(t, err)

		err = store.EndpointRelation().Create(&portainer.EndpointRelation{
			EndpointID: portainer.EndpointID(i + 1),
			EdgeStacks: map[portainer.EdgeStackID]bool{},
		})
		require.NoError(t, err)
	}

	err = store.EdgeGroup().Create(&portainer.EdgeGroup{
		ID:          1,
		Name:        "Test Edge Group",
		EndpointIDs: roar.FromSlice([]portainer.EndpointID{1, 2, 3}),
	})
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	req := httptest.NewRequest(
		http.MethodGet,
		"/edge_groups",
		nil,
	)

	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)

	var responseGroups []decoratedEdgeGroup
	err = json.NewDecoder(rr.Body).Decode(&responseGroups)
	require.NoError(t, err)

	require.Len(t, responseGroups, 1)
	require.ElementsMatch(t, []portainer.EndpointID{1, 2, 3}, responseGroups[0].Endpoints)
	require.Len(t, responseGroups[0].TrustedEndpoints, 0)
}
