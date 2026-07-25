package scheduler

import (
	"fmt"
	"testing"

	"adversarychef/acasched/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatcherTick(t *testing.T) {
	s, err := store.NewStore(":memory:")
	require.NoError(t, err)
	defer s.Close()

	err = s.CreateProject(&store.Project{ID: "proj_001", Name: "test"})
	require.NoError(t, err)

	err = s.CreateTask(&store.Task{
		ID:          "t1",
		ProjectID:   "proj_001",
		Agent:       "echo",
		Title:       "test",
		Description: "do recon",
		Status:      "pending",
		CreatedBy:   "human",
	})
	require.NoError(t, err)

	tasks, err := s.ListPending("proj_001")
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "pending", tasks[0].Status)
	assert.Equal(t, "t1", tasks[0].ID)

	// Mark as dispatched via the lifecycle helper
	err = TransitionToDispatched(s, "t1")
	require.NoError(t, err)

	// Verify it is no longer pending
	remaining, err := s.ListPending("proj_001")
	require.NoError(t, err)
	assert.Len(t, remaining, 0, "dispatched task should not appear in pending list")

	// Verify dispatched status directly
	task, err := s.GetTask("t1")
	require.NoError(t, err)
	assert.Equal(t, "dispatched", task.Status)
}

func TestDispatcherTick_MultipleTasks(t *testing.T) {
	s, err := store.NewStore(":memory:")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, s.CreateProject(&store.Project{ID: "proj_001", Name: "test"}))

	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("t%d", i+1)
		require.NoError(t, s.CreateTask(&store.Task{
			ID:          id,
			ProjectID:   "proj_001",
			Agent:       "echo",
			Title:       "task " + id,
			Description: "desc",
			Status:      "pending",
			CreatedBy:   "human",
		}))
	}

	// All three should be pending
	tasks, err := s.ListPending("proj_001")
	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	// Dispatch one
	require.NoError(t, TransitionToDispatched(s, "t1"))

	// Now only two remain pending
	remaining, err := s.ListPending("proj_001")
	require.NoError(t, err)
	assert.Len(t, remaining, 2)
}

func TestDispatcherTick_NoTasks(t *testing.T) {
	s, err := store.NewStore(":memory:")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, s.CreateProject(&store.Project{ID: "proj_001", Name: "test"}))

	tasks, err := s.ListPending("proj_001")
	require.NoError(t, err)
	assert.Len(t, tasks, 0)
}
