// Package client provides types used by the Mythic MCP client.
package client

import "fmt"

// -------- Payload types --------

// CreatePayloadInput defines input for payload creation.
type CreatePayloadInput struct {
	PayloadType     string            `json:"payload_type"`
	Format          string            `json:"format,omitempty"`
	Description     string            `json:"description,omitempty"`
	Filename        string            `json:"filename,omitempty"`
	SelectedOS      string            `json:"selected_os,omitempty"`
	Commands        []string          `json:"commands,omitempty"`
	Parameters      map[string]string `json:"parameters,omitempty"`
	BuildParameters map[string]string `json:"build_parameters,omitempty"`
	C2Profiles      []C2ProfileInput  `json:"c2_profiles,omitempty"`
	Tag             string            `json:"tag,omitempty"`
}

// C2ProfileInput defines a C2 profile configuration.
type C2ProfileInput struct {
	Name       string            `json:"name"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// CreatePayloadOutput defines output for payload creation.
type CreatePayloadOutput struct {
	PayloadUUID string `json:"payload_uuid"`
	Status      string `json:"status"`
	DownloadURL string `json:"download_url,omitempty"`
}

// -------- Task types --------

// TaskAgentInput defines input for tasking an agent.
type TaskAgentInput struct {
	CallbackID      string   `json:"callback_id"`
	Command         string   `json:"command"`
	Parameters      string   `json:"parameters,omitempty"`
	PayloadType     string   `json:"payload_type,omitempty"`
	ParameterGroup  string   `json:"parameter_group,omitempty"`
	TaskingLocation string   `json:"tasking_location,omitempty"`
	FileIDs         []string `json:"file_ids,omitempty"`
}

// TaskAgentOutput defines output for tasking.
type TaskAgentOutput struct {
	TaskID   string `json:"task_id"`
	Status   string `json:"status"`
	Response string `json:"response,omitempty"`
	Error    string `json:"error,omitempty"`
}

// RunTaskInput defines input for issuing a task with wait policy.
type RunTaskInput struct {
	CallbackID      string   `json:"callback_id"`
	Command         string   `json:"command"`
	Parameters      string   `json:"parameters,omitempty"`
	Mode            string   `json:"mode,omitempty"`
	WaitTimeout     int      `json:"wait_timeout,omitempty"`
	PayloadType     string   `json:"payload_type,omitempty"`
	ParameterGroup  string   `json:"parameter_group,omitempty"`
	TaskingLocation string   `json:"tasking_location,omitempty"`
	FileIDs         []string `json:"file_ids,omitempty"`
}

// RunTaskOutput defines output for the high-level task workflow.
type RunTaskOutput struct {
	TaskID        string   `json:"task_id"`
	Status        string   `json:"status"`
	Completed     bool     `json:"completed"`
	Background    bool     `json:"background"`
	Mode          string   `json:"mode"`
	Command       string   `json:"command,omitempty"`
	DisplayParams string   `json:"display_params,omitempty"`
	Summary       string   `json:"summary,omitempty"`
	Outputs       []string `json:"outputs,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// WaitForTaskInput defines input for waiting on a task.
type WaitForTaskInput struct {
	TaskID  string `json:"task_id"`
	Timeout int    `json:"timeout,omitempty"`
}

// WaitForTaskOutput defines output for waiting on a task.
type WaitForTaskOutput struct {
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
	Completed bool   `json:"completed"`
	Command   string `json:"command,omitempty"`
	Summary   string `json:"summary,omitempty"`
}

// GetTaskStatusInput defines input for checking task status.
type GetTaskStatusInput struct {
	TaskID string `json:"task_id"`
}

// GetTaskStatusOutput defines output for a task status check.
type GetTaskStatusOutput struct {
	TaskID        string `json:"task_id"`
	Status        string `json:"status"`
	Completed     bool   `json:"completed"`
	Command       string `json:"command,omitempty"`
	DisplayParams string `json:"display_params,omitempty"`
	Stderr        string `json:"stderr,omitempty"`
}

// ListTasksInput defines input for listing tasks by callback.
type ListTasksInput struct {
	CallbackID string `json:"callback_id"`
	Limit      int    `json:"limit,omitempty"`
}

// ListTasksOutput defines output for listing tasks.
type ListTasksOutput struct {
	CallbackID string              `json:"callback_id"`
	Tasks      []TaskSummaryOutput `json:"tasks"`
	Count      int                 `json:"count"`
}

// TaskSummaryOutput holds a single task summary row.
type TaskSummaryOutput struct {
	TaskID        string `json:"task_id"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Completed     bool   `json:"completed"`
	DisplayParams string `json:"display_params,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
}

// GetTaskOutputInput defines input for retrieving task output.
type GetTaskOutputInput struct {
	TaskID string `json:"task_id"`
}

// GetTaskOutputOutput defines output for retrieving task responses.
type GetTaskOutputOutput struct {
	TaskID    string   `json:"task_id"`
	Status    string   `json:"status,omitempty"`
	Outputs   []string `json:"outputs"`
	Combined  string   `json:"combined,omitempty"`
	Completed bool     `json:"completed,omitempty"`
}

// -------- Callback types --------

// GetCallbacksInput defines input for listing callbacks.
type GetCallbacksInput struct {
	AgentID string `json:"agent_id,omitempty"`
}

// GetCallbacksOutput defines output for callback listing.
type GetCallbacksOutput struct {
	Callbacks []*Callback `json:"callbacks"`
	Count     int         `json:"count"`
}

// GetCallbackCommandsInput defines input for enumerating loaded commands.
type GetCallbackCommandsInput struct {
	CallbackID string `json:"callback_id"`
}

// CallbackCommandParameter describes a Mythic command parameter.
type CallbackCommandParameter struct {
	Name               string `json:"name"`
	CLIName            string `json:"cli_name,omitempty"`
	DisplayName        string `json:"display_name,omitempty"`
	Type               string `json:"type,omitempty"`
	ParameterGroupName string `json:"parameter_group_name,omitempty"`
	Required           bool   `json:"required,omitempty"`
	DefaultValue       string `json:"default_value,omitempty"`
	Description        string `json:"description,omitempty"`
}

// CallbackCommandInfo describes a loaded command available to a callback.
type CallbackCommandInfo struct {
	CommandName         string                     `json:"command_name"`
	PayloadType         string                     `json:"payload_type,omitempty"`
	SupportedUIFeatures []string                   `json:"supported_ui_features,omitempty"`
	ParameterGroups     []string                   `json:"parameter_groups,omitempty"`
	Parameters          []CallbackCommandParameter `json:"parameters,omitempty"`
}

// GetCallbackCommandsOutput defines output for enumerating loaded commands.
type GetCallbackCommandsOutput struct {
	CallbackID string                `json:"callback_id"`
	Commands   []CallbackCommandInfo `json:"commands"`
	Count      int                   `json:"count"`
}

// RemoveCallbackInput defines input for removing callbacks.
type RemoveCallbackInput struct {
	CallbackIDs []string `json:"callback_ids"`
}

// RemoveCallbackOutput defines output for removing callbacks.
type RemoveCallbackOutput struct {
	Status  string `json:"status"`
	Removed int    `json:"removed"`
}

// -------- File types --------

// FileMeta describes a file registered in Mythic.
type FileMeta struct {
	ID                  int    `json:"id"`
	AgentFileID         string `json:"agent_file_id"`
	TotalChunks         int    `json:"total_chunks"`
	ChunksReceived      int    `json:"chunks_received"`
	Complete            bool   `json:"complete"`
	Path                string `json:"path"`
	FullRemotePath      string `json:"full_remote_path"`
	Host                string `json:"host"`
	IsPayload           bool   `json:"is_payload"`
	IsScreenshot        bool   `json:"is_screenshot"`
	IsDownloadFromAgent bool   `json:"is_download_from_agent"`
	Filename            string `json:"filename"`
	MD5                 string `json:"md5"`
	SHA1                string `json:"sha1"`
	Size                int64  `json:"size"`
	Comment             string `json:"comment"`
	OperatorID          int    `json:"operator_id"`
	Timestamp           string `json:"timestamp"`
	Deleted             bool   `json:"deleted"`
	TaskDisplayID       string `json:"task_display_id,omitempty"`
}

// FormatSize returns a human-readable file size.
func (f *FileMeta) FormatSize() string {
	const unit = 1024
	if f.Size < unit {
		return fmt.Sprintf("%d B", f.Size)
	}
	div, exp := int64(unit), 0
	for n := f.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(f.Size)/float64(div), "KMGTPE"[exp])
}

// GetFilesInput defines input for listing files.
type GetFilesInput struct {
	Limit int `json:"limit,omitempty"`
}

// GetFilesOutput defines output for file listing.
type GetFilesOutput struct {
	Files []*FileMeta `json:"files"`
	Count int         `json:"count"`
}

// UploadFileInput defines input for file upload.
type UploadFileInput struct {
	CallbackID string `json:"callback_id"`
	FileName   string `json:"file_name"`
	FilePath   string `json:"file_path,omitempty"`
	Contents   string `json:"contents,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

// UploadFileOutput defines output for file upload.
type UploadFileOutput struct {
	FileID string `json:"file_id"`
	Status string `json:"status"`
}

// UpdateFileCommentInput defines input for setting a file comment.
type UpdateFileCommentInput struct {
	FileID  string `json:"file_id"`
	Comment string `json:"comment"`
}

// UpdateFileCommentOutput defines output for setting a file comment.
type UpdateFileCommentOutput struct {
	FileID  string `json:"file_id"`
	Comment string `json:"comment"`
	Status  string `json:"status"`
}

// DeleteFileInput defines input for deleting a file.
type DeleteFileInput struct {
	FileID string `json:"file_id"`
}

// DeleteFileOutput defines output for deleting a file.
type DeleteFileOutput struct {
	FileID string `json:"file_id"`
	Status string `json:"status"`
}

// DownloadFileInput defines input for file download.
type DownloadFileInput struct {
	FileID   string `json:"file_id"`
	SavePath string `json:"save_path,omitempty"`
}

// DownloadFileOutput defines output for file download.
type DownloadFileOutput struct {
	FileID    string `json:"file_id"`
	Contents  string `json:"contents"`
	Size      int    `json:"size"`
	Encoding  string `json:"encoding"`
	Truncated bool   `json:"truncated,omitempty"`
}

// CallbackUploadFileInput defines input for uploading a file to a callback host.
type CallbackUploadFileInput struct {
	CallbackID      string `json:"callback_id"`
	RemotePath      string `json:"remote_path,omitempty"`
	FilePath        string `json:"file_path,omitempty"`
	FileName        string `json:"file_name,omitempty"`
	Contents        string `json:"contents,omitempty"`
	Command         string `json:"command,omitempty"`
	Parameters      string `json:"parameters,omitempty"`
	PayloadType     string `json:"payload_type,omitempty"`
	ParameterGroup  string `json:"parameter_group,omitempty"`
	TaskingLocation string `json:"tasking_location,omitempty"`
	WaitTimeout     int    `json:"wait_timeout,omitempty"`
	Async           bool   `json:"async,omitempty"`
}

// CallbackUploadFileOutput defines output for uploading a file to a callback host.
type CallbackUploadFileOutput struct {
	TaskID      string `json:"task_id"`
	Status      string `json:"status"`
	Completed   bool   `json:"completed"`
	Summary     string `json:"summary,omitempty"`
	FileID      string `json:"file_id"`
	FileName    string `json:"file_name,omitempty"`
	RemotePath  string `json:"remote_path,omitempty"`
	StagedState string `json:"staged_state,omitempty"`
	Background  bool   `json:"background,omitempty"`
}

// CallbackDownloadFileInput defines input for downloading a file from a callback.
type CallbackDownloadFileInput struct {
	CallbackID      string `json:"callback_id"`
	RemotePath      string `json:"remote_path,omitempty"`
	Command         string `json:"command,omitempty"`
	Parameters      string `json:"parameters,omitempty"`
	PayloadType     string `json:"payload_type,omitempty"`
	ParameterGroup  string `json:"parameter_group,omitempty"`
	TaskingLocation string `json:"tasking_location,omitempty"`
	WaitTimeout     int    `json:"wait_timeout,omitempty"`
	Async           bool   `json:"async,omitempty"`
}

// CallbackDownloadFileOutput defines output for downloading a file from a callback.
type CallbackDownloadFileOutput struct {
	TaskID     string `json:"task_id"`
	Status     string `json:"status"`
	Completed  bool   `json:"completed"`
	Summary    string `json:"summary,omitempty"`
	FileID     string `json:"file_id"`
	FileName   string `json:"file_name,omitempty"`
	RemotePath string `json:"remote_path,omitempty"`
	Host       string `json:"host,omitempty"`
	Contents   string `json:"contents"`
	Size       int    `json:"size"`
	Encoding   string `json:"encoding"`
	Truncated  bool   `json:"truncated,omitempty"`
	Background bool   `json:"background,omitempty"`
}
