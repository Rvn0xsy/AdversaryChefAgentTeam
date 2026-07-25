package scheduler

import "adversarychef/acasched/internal/store"

func TransitionToDispatched(s *store.Store, taskID string) error {
	return s.MarkDispatched(taskID)
}

func TransitionToRunning(s *store.Store, taskID string) error {
	return s.MarkRunning(taskID)
}

func TransitionToDone(s *store.Store, taskID, result string) error {
	return s.UpdateStatus(taskID, "done", result, "")
}

func TransitionToFailed(s *store.Store, taskID, errMsg string) error {
	return s.UpdateStatus(taskID, "failed", "", errMsg)
}

func TransitionToTimeout(s *store.Store, taskID string) error {
	return s.UpdateStatus(taskID, "timeout", "", "execution timed out")
}
