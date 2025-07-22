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

func TestEdgeGroupInspectHandler(t *testing.T) {
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
		"/edge_groups/1",
		nil,
	)

	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)

	var responseGroup portainer.EdgeGroup
	err = json.NewDecoder(rr.Body).Decode(&responseGroup)
	require.NoError(t, err)

	assert.ElementsMatch(t, []portainer.EndpointID{1, 2, 3}, responseGroup.Endpoints)
}

func TestDynamicEdgeGroupInspectHandler(t *testing.T) {
	_, store := datastore.MustNewTestStore(t, true, true)

	handler := NewHandler(testhelpers.NewTestRequestBouncer())
	handler.DataStore = store

	err := store.EndpointGroup().Create(&portainer.EndpointGroup{
		ID:   1,
		Name: "Test Group",
	})
	require.NoError(t, err)

	err = store.Tag().Create(&portainer.Tag{
		ID:   1,
		Name: "Test Tag",
		Endpoints: map[portainer.EndpointID]bool{
			1: true,
			2: true,
			3: true,
		},
	})
	require.NoError(t, err)

	for i := range 3 {
		err = store.Endpoint().Create(&portainer.Endpoint{
			ID:          portainer.EndpointID(i + 1),
			Name:        "Test Endpoint " + strconv.Itoa(i+1),
			Type:        portainer.EdgeAgentOnDockerEnvironment,
			GroupID:     1,
			TagIDs:      []portainer.TagID{1},
			UserTrusted: true,
		})
		require.NoError(t, err)

		err = store.EndpointRelation().Create(&portainer.EndpointRelation{
			EndpointID: portainer.EndpointID(i + 1),
			EdgeStacks: map[portainer.EdgeStackID]bool{},
		})
		require.NoError(t, err)
	}

	err = store.EdgeGroup().Create(&portainer.EdgeGroup{
		ID:      1,
		Name:    "Test Edge Group",
		Dynamic: true,
		TagIDs:  []portainer.TagID{1},
	})
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	req := httptest.NewRequest(
		http.MethodGet,
		"/edge_groups/1",
		nil,
	)

	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)

	var responseGroup portainer.EdgeGroup
	err = json.NewDecoder(rr.Body).Decode(&responseGroup)
	require.NoError(t, err)

	require.ElementsMatch(t, []portainer.EndpointID{1, 2, 3}, responseGroup.Endpoints)
}
