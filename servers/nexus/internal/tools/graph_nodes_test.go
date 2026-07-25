package tools

import (
	"testing"

	"adversarychef/nexus/internal/models"
	"adversarychef/nexus/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostCRUD(t *testing.T) {
	st, err := store.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	defer st.Close()

	// Create
	h := &models.HostNode{
		ProjectID: "proj_001",
		IPs:       []string{"10.0.0.1"},
		Hostname:  "web01",
	}
	err = st.CreateHost(h)
	require.NoError(t, err)
	assert.NotEmpty(t, h.ID, "host ID should be auto-generated")
	assert.Equal(t, "proj_001", h.ProjectID)

	// List
	hosts, err := st.ListHosts("proj_001")
	require.NoError(t, err)
	assert.Len(t, hosts, 1)
	assert.Equal(t, "web01", hosts[0].Hostname)
	assert.Equal(t, []string{"10.0.0.1"}, hosts[0].IPs)

	// Get by ID
	got, err := st.GetHost(h.ID)
	require.NoError(t, err)
	assert.Equal(t, h.ID, got.ID)
	assert.Equal(t, "web01", got.Hostname)
	assert.Equal(t, []string{"10.0.0.1"}, got.IPs)
	assert.Equal(t, "proj_001", got.ProjectID)
	assert.False(t, got.CreatedAt.IsZero(), "created_at should be set")
}

func TestHostCRUD_MultipleProjects(t *testing.T) {
	st, err := store.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	defer st.Close()

	hostA := &models.HostNode{ProjectID: "proj_A", IPs: []string{"10.0.0.1"}, Hostname: "host-a"}
	hostB := &models.HostNode{ProjectID: "proj_B", IPs: []string{"10.0.0.2"}, Hostname: "host-b"}

	require.NoError(t, st.CreateHost(hostA))
	require.NoError(t, st.CreateHost(hostB))

	// List per project
	projAHosts, err := st.ListHosts("proj_A")
	require.NoError(t, err)
	assert.Len(t, projAHosts, 1)
	assert.Equal(t, "host-a", projAHosts[0].Hostname)

	projBHosts, err := st.ListHosts("proj_B")
	require.NoError(t, err)
	assert.Len(t, projBHosts, 1)
	assert.Equal(t, "host-b", projBHosts[0].Hostname)
}

func TestHostCRUD_EmptyFields(t *testing.T) {
	st, err := store.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	defer st.Close()

	// Create host with only required fields
	h := &models.HostNode{
		ProjectID: "proj_001",
		IPs:       []string{},
	}
	err = st.CreateHost(h)
	require.NoError(t, err)
	assert.NotEmpty(t, h.ID)

	got, err := st.GetHost(h.ID)
	require.NoError(t, err)
	assert.Equal(t, h.ID, got.ID)
	assert.Empty(t, got.Hostname)
	assert.Empty(t, got.OS)
	assert.Empty(t, got.IPs)
}
