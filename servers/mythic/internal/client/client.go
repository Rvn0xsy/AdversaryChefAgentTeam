// Package client provides the Mythic C2 HTTP/GraphQL/WebSocket client.
/*
 * Mythic C2 RPC Client
 * Handles connection and authentication to Mythic server
 */

package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	// external config dependency removed
	"log"
	"github.com/gorilla/websocket"
)

const maxDownloadedFileBytes = 256 * 1024

const mythicDebugLogFileName = "mythic-debug.log"

// Client represents Mythic C2 server connection
type Client struct {
	server     string
	apiKey     string
	wsConn     *websocket.Conn
	httpClient *http.Client
	mu         sync.Mutex
	Callbacks  map[string]*Callback
}

// Callback represents an agent callback
type Callback struct {
	ID           string `json:"id"`
	InternalID   int    `json:"internal_id,omitempty"`
	DisplayID    int    `json:"display_id,omitempty"`
	AgentID      string `json:"agent_id"`
	Host         string `json:"host"`
	User         string `json:"user"`
	PID          int    `json:"pid"`
	IP           string `json:"ip"`
	ExternalIP   string `json:"external_ip,omitempty"`
	ProcessName  string `json:"process_name,omitempty"`
	Description  string `json:"description,omitempty"`
	OS           string `json:"os,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Domain       string `json:"domain,omitempty"`
	ExtraInfo    string `json:"extra_info,omitempty"`
	SleepInfo    string `json:"sleep_info,omitempty"`
	PayloadType  string `json:"payload_type,omitempty"`
	PayloadUUID  string `json:"payload_uuid,omitempty"`
	Operator     string `json:"operator,omitempty"`
	Active       bool   `json:"active,omitempty"`
	LastCheck    string `json:"last_check"`
	InitCallback string `json:"init_callback,omitempty"`
	Status       string `json:"status"`
}

// PayloadConfig holds configuration for payload creation
type PayloadConfig struct {
	PayloadType string            `json:"payload_type"` // apollo, hermes, etc.
	Format      string            `json:"format"`       // exe, dll, shellcode, etc.
	Parameters  map[string]string `json:"parameters"`
	Tag         string            `json:"tag,omitempty"`
}

// TaskConfig holds configuration for tasking an agent
type TaskConfig struct {
	CallbackID string `json:"callback_id"`
	Command    string `json:"command"`
	Parameters string `json:"parameters,omitempty"`
}

// NewClient creates a new Mythic C2 client. Prefer NewClientWithConfig for explicit parameters.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		Callbacks:  make(map[string]*Callback),
	}
}

// NewClientWithConfig creates a new Mythic C2 client with explicit config
func NewClientWithConfig(server, apiKey string) *Client {
	return &Client{
		server:     server,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		Callbacks:  make(map[string]*Callback),
	}
}

// Connect establishes connection to Mythic C2 server
func (c *Client) Connect(ctx context.Context) error {
	if err := c.ensureConfigured(); err != nil {
		return err
	}

	wsURL, err := c.websocketURL()
	if err != nil {
		return err
	}

	c.wsConn, _, err = websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Mythic WebSocket: %w", err)
	}

	return nil
}

// Disconnect closes the connection
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.wsConn != nil {
		return c.wsConn.Close()
	}
	return nil
}

// IsConnected returns true if connected
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.wsConn != nil
}

// GetServer returns the Mythic server address
func (c *Client) GetServer() string {
	return c.server
}

// ValidateConfig checks whether Mythic server and API key are available.
func (c *Client) ValidateConfig() error {
	return c.ensureConfigured()
}

func mythicDebugLog(ctx context.Context, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	log.Printf("[mythic] %s", message)
	_ = ctx
}

func (c *Client) ensureConfigured() error {
	if strings.TrimSpace(c.server) == "" || strings.TrimSpace(c.apiKey) == "" {
		return fmt.Errorf("Mythic not configured: server=%s, apiKey set=%v", c.server, strings.TrimSpace(c.apiKey) != "")
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 120 * time.Second}
	}
	return nil
}

func (c *Client) baseURL() (string, error) {
	if err := c.ensureConfigured(); err != nil {
		return "", err
	}

	raw := strings.TrimSpace(c.server)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid Mythic server URL %q: %w", c.server, err)
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func (c *Client) websocketURL() (string, error) {
	base, err := c.baseURL()
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	default:
		parsed.Scheme = "wss"
	}
	parsed.Path = "/ws"
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func (c *Client) graphqlURL() (string, error) {
	base, err := c.baseURL()
	if err != nil {
		return "", err
	}
	return base + "/graphql/", nil
}

func (c *Client) apiURL(path string) (string, error) {
	base, err := c.baseURL()
	if err != nil {
		return "", err
	}
	return base + "/" + strings.TrimLeft(path, "/"), nil
}

func (c *Client) authHeaders() http.Header {
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("apitoken", c.apiKey)
	return headers
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func (c *Client) executeGraphQL(ctx context.Context, query string, variables map[string]any, target any) error {
	if err := c.ensureConfigured(); err != nil {
		return err
	}

	endpoint, err := c.graphqlURL()
	if err != nil {
		return err
	}

	body, err := json.Marshal(&graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create GraphQL request: %w", err)
	}
	req.Header = c.authHeaders()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GraphQL request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read GraphQL response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("GraphQL request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var envelope graphqlResponse
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("failed to decode GraphQL response: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", envelope.Errors[0].Message)
	}
	if target == nil {
		return nil
	}
	if len(envelope.Data) == 0 {
		return fmt.Errorf("GraphQL response did not include data")
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		return fmt.Errorf("failed to decode GraphQL data: %w", err)
	}
	return nil
}

func (c *Client) executeRESTWebhook(ctx context.Context, path string, payload any, target any) error {
	if err := c.ensureConfigured(); err != nil {
		return err
	}

	endpoint, err := c.apiURL(path)
	if err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal REST payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create REST request: %w", err)
	}
	req.Header = c.authHeaders()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("REST request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read REST response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("REST request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if target == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, target); err != nil {
		return fmt.Errorf("failed to decode REST response: %w", err)
	}
	return nil
}

func (c *Client) CreatePayload(ctx context.Context, input *CreatePayloadInput) (*CreatePayloadOutput, error) {
	if input == nil || strings.TrimSpace(input.PayloadType) == "" {
		return nil, fmt.Errorf("payload_type is required")
	}

	payloadDefinition := map[string]any{
		"payload_type": input.PayloadType,
		"description":  firstNonEmpty(input.Description, input.Tag, fmt.Sprintf("Payload created via AdversaryChef: %s", input.PayloadType)),
		"filename":     derivePayloadFilename(input),
	}
	if len(input.Commands) > 0 {
		payloadDefinition["commands"] = append([]string(nil), input.Commands...)
	}
	if c2Profiles := buildC2Profiles(input.C2Profiles); len(c2Profiles) > 0 {
		payloadDefinition["c2_profiles"] = c2Profiles
	}
	if buildParams := buildPayloadBuildParameters(input); len(buildParams) > 0 {
		payloadDefinition["build_parameters"] = buildParams
	}

	encodedDefinition, err := json.Marshal(payloadDefinition)
	if err != nil {
		return nil, fmt.Errorf("failed to encode payload definition: %w", err)
	}

	var mutationResponse struct {
		CreatePayload struct {
			Status string `json:"status"`
			Error  string `json:"error"`
			UUID   string `json:"uuid"`
		} `json:"createPayload"`
	}
	err = c.executeGraphQL(ctx, `mutation CreatePayload($payload_definition: String!) {
  createPayload(payloadDefinition: $payload_definition) {
    status
    error
    uuid
  }
}`,
		map[string]any{"payload_definition": string(encodedDefinition)},
		&mutationResponse,
	)
	if err != nil {
		return nil, err
	}
	if mutationResponse.CreatePayload.Status != "success" {
		return nil, fmt.Errorf("failed to create payload: %s", strings.TrimSpace(mutationResponse.CreatePayload.Error))
	}

	payload, err := c.GetPayloadByUUID(ctx, mutationResponse.CreatePayload.UUID)
	if err != nil {
		return nil, err
	}
	return &CreatePayloadOutput{
		PayloadUUID: payload.UUID,
		Status:      firstNonEmpty(payload.BuildPhase, mutationResponse.CreatePayload.Status),
	}, nil
}

type payloadRecord struct {
	UUID        string
	BuildPhase  string
	BuildMsg    string
	Filename    string
	OS          string
	PayloadType string
	Tag         string
}

func (c *Client) GetPayloadByUUID(ctx context.Context, uuid string) (*payloadRecord, error) {
	if strings.TrimSpace(uuid) == "" {
		return nil, fmt.Errorf("payload uuid is required")
	}
	var queryResponse struct {
		Payload []struct {
			UUID         string `json:"uuid"`
			Description  string `json:"description"`
			OS           string `json:"os"`
			BuildPhase   string `json:"build_phase"`
			BuildMessage string `json:"build_message"`
			PayloadType  struct {
				Name string `json:"name"`
			} `json:"payloadtype"`
		} `json:"payload"`
	}
	err := c.executeGraphQL(ctx, `query GetPayloadByUUID($uuid: String!) {
  payload(where: {uuid: {_eq: $uuid}}) {
    uuid
    description
    os
    build_phase
    build_message
    payloadtype {
      name
    }
  }
}`,
		map[string]any{"uuid": uuid},
		&queryResponse,
	)
	if err != nil {
		return nil, err
	}
	if len(queryResponse.Payload) == 0 {
		return nil, fmt.Errorf("payload %s not found", uuid)
	}
	payload := queryResponse.Payload[0]
	return &payloadRecord{
		UUID:        payload.UUID,
		BuildPhase:  payload.BuildPhase,
		BuildMsg:    payload.BuildMessage,
		OS:          payload.OS,
		PayloadType: payload.PayloadType.Name,
	}, nil
}

func (c *Client) IssueTask(ctx context.Context, input *TaskAgentInput) (*TaskAgentOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("task input is required")
	}
	callbackID, err := strconv.Atoi(strings.TrimSpace(input.CallbackID))
	if err != nil || callbackID <= 0 {
		return nil, fmt.Errorf("callback_id must be a positive integer display ID")
	}

	// Auto-resolve parameter_group_name when the caller hasn't specified one.
	// Apollo's HTTP C2 profile defaults to "Default", but many commands (e.g. sleep,
	// load) use a different group.  Without this, Mythic rejects the task with
	// "args don't belong to the Default parameter group", wasting iterations.
	parameterGroup := input.ParameterGroup
	if parameterGroup == "" && input.Command != "" {
		if resolved, err := c.resolveParameterGroup(ctx, input.CallbackID, input.Command); err == nil && resolved != "" {
			parameterGroup = resolved
			mythicDebugLog(ctx, "IssueTask auto-resolved parameter_group callback_id=%s command=%s group=%s", input.CallbackID, input.Command, parameterGroup)
		}
	}

	payload := map[string]any{
		"input": map[string]any{
			"callback_id": callbackID,
			"command":     input.Command,
			"params":      input.Parameters,
		},
	}
	if input.TaskingLocation != "" {
		payload["input"].(map[string]any)["tasking_location"] = input.TaskingLocation
	}
	if parameterGroup != "" {
		payload["input"].(map[string]any)["parameter_group_name"] = parameterGroup
	}
	if input.PayloadType != "" {
		payload["input"].(map[string]any)["payload_type"] = input.PayloadType
	}
	if len(input.FileIDs) > 0 {
		payload["input"].(map[string]any)["files"] = append([]string(nil), input.FileIDs...)
	}
	mythicDebugLog(ctx, "IssueTask request callback_id=%s command=%s parameter_group=%s payload_type=%s tasking_location=%s file_ids=%v params=%s",
		input.CallbackID,
		input.Command,
		input.ParameterGroup,
		input.PayloadType,
		input.TaskingLocation,
		input.FileIDs,
		truncateForDebug(input.Parameters, 512),
	)

	var webhookResponse struct {
		Status    string `json:"status"`
		Error     string `json:"error"`
		ID        int    `json:"id"`
		DisplayID int    `json:"display_id"`
	}
	err = c.executeRESTWebhook(ctx, "api/v1.4/create_task_webhook", payload, &webhookResponse)
	if err != nil {
		mythicDebugLog(ctx, "IssueTask webhook error callback_id=%s command=%s err=%v", input.CallbackID, input.Command, err)
		return nil, err
	}
	if webhookResponse.Status != "success" {
		mythicDebugLog(ctx, "IssueTask webhook non-success callback_id=%s command=%s status=%s error=%s", input.CallbackID, input.Command, webhookResponse.Status, webhookResponse.Error)
		errText := strings.TrimSpace(webhookResponse.Error)
		errMsg := fmt.Sprintf("failed to issue task: %s", errText)
		// When Mythic rejects parameters due to group mismatch, add a diagnostic hint
		// so the agent knows to inspect the command's actual parameter groups instead of
		// blindly retrying with guesswork.
		if strings.Contains(errText, "don't belong to") || strings.Contains(errText, "parameter group") {
			errMsg += "\n\nHINT: Call mythic_get_callback_commands for this callback to find the correct parameter_group for this command. Do NOT retry with guessed variations — inspect first, then use the exact group name from the command metadata."
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	var taskQuery struct {
		Task []struct {
			DisplayID   int    `json:"display_id"`
			CommandName string `json:"command_name"`
			Status      string `json:"status"`
			Stdout      string `json:"stdout"`
			Stderr      string `json:"stderr"`
		} `json:"task"`
	}
	err = c.executeGraphQL(ctx, `query GetTask($display_id: Int!) {
  task(where: {display_id: {_eq: $display_id}}, limit: 1) {
    display_id
    command_name
    status
    stdout
    stderr
  }
}`,
		map[string]any{"display_id": webhookResponse.DisplayID},
		&taskQuery,
	)
	if err != nil {
		mythicDebugLog(ctx, "IssueTask follow-up query error task_display_id=%d err=%v", webhookResponse.DisplayID, err)
		return nil, err
	}
	result := &TaskAgentOutput{TaskID: strconv.Itoa(webhookResponse.DisplayID), Status: webhookResponse.Status}
	if len(taskQuery.Task) > 0 {
		result.Status = firstNonEmpty(taskQuery.Task[0].Status, result.Status)
		raw := strings.TrimSpace(strings.TrimSpace(taskQuery.Task[0].Stdout) + "\n" + strings.TrimSpace(taskQuery.Task[0].Stderr))
		result.Response = stripArgsWarning(raw)
	}
	mythicDebugLog(ctx, "IssueTask result task_id=%s status=%s response=%s", result.TaskID, result.Status, truncateForDebug(result.Response, 512))
	return result, nil
}

func (c *Client) RunTask(ctx context.Context, input *RunTaskInput) (*RunTaskOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("run task input is required")
	}
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode == "" {
		mode = "auto"
	}
	if mode != "auto" && mode != "foreground" && mode != "background" {
		return nil, fmt.Errorf("mode must be one of foreground, background, or auto")
	}
	if mode == "auto" {
		if isForegroundTask(input.Command, input.Parameters) {
			mode = "foreground"
		} else {
			mode = "background"
		}
	}

	task, err := c.IssueTask(ctx, &TaskAgentInput{
		CallbackID:      input.CallbackID,
		Command:         input.Command,
		Parameters:      input.Parameters,
		PayloadType:     input.PayloadType,
		ParameterGroup:  input.ParameterGroup,
		TaskingLocation: input.TaskingLocation,
		FileIDs:         input.FileIDs,
	})
	if err != nil {
		return nil, err
	}
	result := &RunTaskOutput{
		TaskID: task.TaskID,
		Status: task.Status,
		Mode:   mode,
	}
	if task.Response != "" {
		result.Summary = task.Response
	}
	if mode == "background" {
		result.Background = true
		status, statusErr := c.GetTaskStatus(ctx, &GetTaskStatusInput{TaskID: task.TaskID})
		if statusErr != nil {
			result.Error = statusErr.Error()
			return result, nil
		}
		result.Status = status.Status
		result.Completed = status.Completed
		result.Command = status.Command
		result.DisplayParams = status.DisplayParams
		if status.Completed {
			result.Background = false
		}
		return result, nil
	}

	timeout := input.WaitTimeout
	if timeout <= 0 {
		timeout = 60
	}
	waited, waitErr := c.WaitForTask(ctx, &WaitForTaskInput{TaskID: task.TaskID, Timeout: timeout})
	if waitErr != nil {
		status, statusErr := c.GetTaskStatus(ctx, &GetTaskStatusInput{TaskID: task.TaskID})
		if statusErr == nil {
			result.Status = status.Status
			result.Completed = status.Completed
			result.Command = status.Command
			result.DisplayParams = status.DisplayParams
			result.Background = !status.Completed
		}
		result.Error = waitErr.Error()
		return result, nil
	}
	result.Status = waited.Status
	result.Completed = waited.Completed
	result.Command = waited.Command
	result.Summary = waited.Summary
	if waited.Completed {
		output, outputErr := c.GetTaskOutput(ctx, &GetTaskOutputInput{TaskID: task.TaskID})
		if outputErr == nil {
			result.Outputs = output.Outputs
			result.Summary = firstNonEmpty(output.Combined, result.Summary)
		}
	}
	return result, nil
}

func isForegroundTask(command, parameters string) bool {
	command = strings.ToLower(strings.TrimSpace(command))
	parameters = strings.ToLower(strings.TrimSpace(parameters))
	if command == "" {
		return false
	}
	backgroundCommands := map[string]struct{}{
		"nmap": {}, "naabu": {}, "masscan": {}, "bloodhound": {}, "sharphound": {},
		"portscan": {}, "netstat": {},
	}
	if _, ok := backgroundCommands[command]; ok {
		return false
	}
	if command == "shell" || command == "run" {
		for _, marker := range []string{"nmap", "naabu", "masscan", "bloodhound", "sharphound", "sleep ", "timeout ", "ping -t"} {
			if strings.Contains(parameters, marker) {
				return false
			}
		}
	}
	foregroundCommands := map[string]struct{}{
		"whoami": {}, "hostname": {}, "pwd": {}, "ls": {}, "dir": {}, "ps": {},
		"ipconfig": {}, "ifconfig": {}, "cat": {}, "type": {}, "cp": {}, "copy": {},
		"mv": {}, "move": {}, "rm": {}, "del": {}, "mkdir": {}, "rmdir": {},
		"getuid": {}, "getpid": {}, "getprivs": {}, "shell": {}, "run": {},
	}
	_, ok := foregroundCommands[command]
	return ok
}

type taskRecord struct {
	ID             int // internal Mythic task ID (used to query responses)
	DisplayID      int
	Command        string
	Status         string
	DisplayParams  string
	OriginalParams string
	ResponseCount  int
	Stdout         string
	Stderr         string
	Completed      bool
	Timestamp      string
}

// ListTasksByCallback returns all tasks for a callback (non-blocking).
func (c *Client) ListTasksByCallback(ctx context.Context, callbackID string, limit int) ([]taskRecord, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	cbDisplayID, err := strconv.Atoi(strings.TrimSpace(callbackID))
	if err != nil || cbDisplayID <= 0 {
		return nil, fmt.Errorf("callback_id must be a positive integer display ID")
	}
	// Resolve display ID to internal ID for the task query
	internalID, _, err := c.resolveCallbackInternalID(ctx, callbackID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve callback %s: %w", callbackID, err)
	}
	if limit <= 0 {
		limit = 50
	}
	var taskQuery struct {
		Task []struct {
			ID             int    `json:"id"`
			DisplayID      int    `json:"display_id"`
			CommandName    string `json:"command_name"`
			Status         string `json:"status"`
			DisplayParams  string `json:"display_params"`
			OriginalParams string `json:"original_params"`
			ResponseCount  int    `json:"response_count"`
			Stdout         string `json:"stdout"`
			Stderr         string `json:"stderr"`
			Completed      bool   `json:"completed"`
			Timestamp      string `json:"timestamp"`
		} `json:"task"`
	}
	err = c.executeGraphQL(ctx, `query ListTasksByCallback($callback_id: Int!, $limit: Int!) {
  task(where: {callback_id: {_eq: $callback_id}}, limit: $limit, order_by: {display_id: desc}) {
    id
    display_id
    command_name
    status
    display_params
    original_params
    response_count
    stdout
    stderr
    completed
    timestamp
  }
}`,
		map[string]any{"callback_id": internalID, "limit": limit},
		&taskQuery,
	)
	if err != nil {
		return nil, err
	}
	tasks := make([]taskRecord, 0, len(taskQuery.Task))
	for _, t := range taskQuery.Task {
		tasks = append(tasks, taskRecord{
			ID:             t.ID,
			DisplayID:      t.DisplayID,
			Command:        t.CommandName,
			Status:         t.Status,
			DisplayParams:  t.DisplayParams,
			OriginalParams: t.OriginalParams,
			ResponseCount:  t.ResponseCount,
			Stdout:         t.Stdout,
			Stderr:         t.Stderr,
			Completed:      t.Completed,
			Timestamp:      t.Timestamp,
		})
	}
	return tasks, nil
}

type taskResponseRecord struct {
	Response string
}

// queryTaskResponses fetches task response_text entries by internal task ID.
func (c *Client) queryTaskResponses(ctx context.Context, internalTaskID int) ([]taskResponseRecord, error) {
	var queryResponse struct {
		Response []struct {
			ResponseText string `json:"response_text"`
		} `json:"response"`
	}
	err := c.executeGraphQL(ctx, `query GetTaskResponses($task_id: Int!) {
  response(where: {task_id: {_eq: $task_id}}, order_by: {id: asc}) {
    response_text
  }
}`,
		map[string]any{"task_id": internalTaskID},
		&queryResponse,
	)
	if err != nil {
		mythicDebugLog(ctx, "queryTaskResponses error internal_id=%d err=%v", internalTaskID, err)
		return nil, err
	}
	responses := make([]taskResponseRecord, 0, len(queryResponse.Response))
	for _, response := range queryResponse.Response {
		responses = append(responses, taskResponseRecord{Response: response.ResponseText})
	}
	mythicDebugLog(ctx, "queryTaskResponses internal_id=%d count=%d", internalTaskID, len(responses))
	return responses, nil
}

func combineTaskOutputs(task *taskRecord, responses []taskResponseRecord) []string {
	outputs := make([]string, 0, 2+len(responses))
	if task != nil {
		if stdout := strings.TrimSpace(task.Stdout); stdout != "" {
			if cleaned := stripArgsWarning(decodeMythicText(stdout)); cleaned != "" {
				outputs = append(outputs, cleaned)
			}
		}
		if stderr := strings.TrimSpace(task.Stderr); stderr != "" {
			if cleaned := stripArgsWarning(decodeMythicText(stderr)); cleaned != "" {
				outputs = append(outputs, cleaned)
			}
		}
	}
	for _, response := range responses {
		if content := strings.TrimSpace(response.Response); content != "" {
			if cleaned := stripArgsWarning(decodeMythicText(content)); cleaned != "" {
				outputs = append(outputs, cleaned)
			}
		}
	}
	return outputs
}

func (c *Client) queryTaskByDisplayID(ctx context.Context, displayID int) (*taskRecord, error) {
	var taskQuery struct {
		Task []struct {
			ID             int    `json:"id"`
			DisplayID      int    `json:"display_id"`
			CommandName    string `json:"command_name"`
			Status         string `json:"status"`
			DisplayParams  string `json:"display_params"`
			OriginalParams string `json:"original_params"`
			ResponseCount  int    `json:"response_count"`
			Stdout         string `json:"stdout"`
			Stderr         string `json:"stderr"`
			Completed      bool   `json:"completed"`
		} `json:"task"`
	}
	err := c.executeGraphQL(ctx, `query GetTask($display_id: Int!) {
  task(where: {display_id: {_eq: $display_id}}, limit: 1) {
    id
    display_id
    command_name
    status
		display_params
		original_params
		response_count
    stdout
    stderr
    completed
  }
}`,
		map[string]any{"display_id": displayID},
		&taskQuery,
	)
	if err != nil {
		mythicDebugLog(ctx, "queryTaskByDisplayID error task_id=%d err=%v", displayID, err)
		return nil, err
	}
	if len(taskQuery.Task) == 0 {
		mythicDebugLog(ctx, "queryTaskByDisplayID task_id=%d not found", displayID)
		return nil, fmt.Errorf("task %d not found", displayID)
	}
	task := taskQuery.Task[0]
	mythicDebugLog(ctx, "queryTaskByDisplayID task_id=%d internal_id=%d status=%s completed=%v response_count=%d display_params=%s original_params=%s stdout=%s stderr=%s", task.DisplayID, task.ID, task.Status, task.Completed, task.ResponseCount, truncateForDebug(task.DisplayParams, 256), truncateForDebug(task.OriginalParams, 256), truncateForDebug(task.Stdout, 256), truncateForDebug(task.Stderr, 256))
	return &taskRecord{
		ID:             task.ID,
		DisplayID:      task.DisplayID,
		Command:        task.CommandName,
		Status:         task.Status,
		DisplayParams:  task.DisplayParams,
		OriginalParams: task.OriginalParams,
		ResponseCount:  task.ResponseCount,
		Stdout:         task.Stdout,
		Stderr:         task.Stderr,
		Completed:      task.Completed,
	}, nil
}

func (c *Client) GetTaskStatus(ctx context.Context, input *GetTaskStatusInput) (*GetTaskStatusOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	if input == nil || strings.TrimSpace(input.TaskID) == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	taskID, err := strconv.Atoi(strings.TrimSpace(input.TaskID))
	if err != nil || taskID <= 0 {
		return nil, fmt.Errorf("task_id must be a positive integer display ID")
	}
	task, err := c.queryTaskByDisplayID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return &GetTaskStatusOutput{
		TaskID:        strconv.Itoa(task.DisplayID),
		Status:        task.Status,
		Completed:     task.Completed,
		Command:       task.Command,
		DisplayParams: task.DisplayParams,
		Stderr:        task.Stderr,
	}, nil
}

func (c *Client) WaitForTask(ctx context.Context, input *WaitForTaskInput) (*WaitForTaskOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("wait input is required")
	}
	taskID, err := strconv.Atoi(strings.TrimSpace(input.TaskID))
	if err != nil || taskID <= 0 {
		return nil, fmt.Errorf("task_id must be a positive integer display ID")
	}
	timeoutSeconds := input.Timeout
	if timeoutSeconds <= 0 {
		timeoutSeconds = 60
	}
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	mythicDebugLog(ctx, "WaitForTask start task_id=%s timeout=%ds", input.TaskID, timeoutSeconds)
	for {
		task, err := c.queryTaskByDisplayID(ctx, taskID)
		if err != nil {
			mythicDebugLog(ctx, "WaitForTask query error task_id=%s err=%v", input.TaskID, err)
			return nil, err
		}
		status := strings.ToLower(strings.TrimSpace(task.Status))
		if task.Completed || status == "success" || status == "error" || status == "completed" || status == "processed" {
			responses, responseErr := c.queryTaskResponses(ctx, task.ID)
			if responseErr != nil {
				mythicDebugLog(ctx, "WaitForTask response query error task_id=%s err=%v", input.TaskID, responseErr)
			}
			outputs := combineTaskOutputs(task, responses)
			summary := stripArgsWarning(strings.Join(outputs, "\n"))
			if summary == "" {
				summary = "task completed (no output)"
			}
			mythicDebugLog(ctx, "WaitForTask complete task_id=%s status=%s completed=%v response_count=%d outputs=%d summary=%s", input.TaskID, task.Status, task.Completed || status == "success" || status == "completed" || status == "processed", task.ResponseCount, len(outputs), truncateForDebug(summary, 512))
			return &WaitForTaskOutput{
				TaskID:    strconv.Itoa(task.DisplayID),
				Status:    task.Status,
				Completed: task.Completed || status == "success" || status == "completed" || status == "processed",
				Command:   task.Command,
				Summary:   summary,
			}, nil
		}
		if time.Now().After(deadline) {
			mythicDebugLog(ctx, "WaitForTask timeout task_id=%s", input.TaskID)
			return nil, fmt.Errorf("timeout after %d seconds", timeoutSeconds)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

func (c *Client) GetTaskOutput(ctx context.Context, input *GetTaskOutputInput) (*GetTaskOutputOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("task output input is required")
	}
	taskID, err := strconv.Atoi(strings.TrimSpace(input.TaskID))
	if err != nil || taskID <= 0 {
		return nil, fmt.Errorf("task_id must be a positive integer display ID")
	}
	task, err := c.queryTaskByDisplayID(ctx, taskID)
	if err != nil {
		mythicDebugLog(ctx, "GetTaskOutput query error task_id=%s err=%v", input.TaskID, err)
		return nil, err
	}
	responses, responseErr := c.queryTaskResponses(ctx, task.ID)
	if responseErr != nil {
		mythicDebugLog(ctx, "GetTaskOutput response query error task_id=%s err=%v", input.TaskID, responseErr)
	}
	outputs := combineTaskOutputs(task, responses)
	combined := stripArgsWarning(strings.Join(outputs, "\n"))
	if combined == "" {
		combined = "task completed (no output)"
	}
	result := &GetTaskOutputOutput{
		TaskID:    strconv.Itoa(task.DisplayID),
		Status:    task.Status,
		Outputs:   outputs,
		Combined:  combined,
		Completed: task.Completed,
	}
	mythicDebugLog(ctx, "GetTaskOutput result task_id=%s status=%s response_count=%d outputs=%d combined=%s", result.TaskID, result.Status, task.ResponseCount, len(result.Outputs), truncateForDebug(result.Combined, 512))
	return result, nil
}

func (c *Client) GetCallbacks(ctx context.Context, filter *GetCallbacksInput) (*GetCallbacksOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	mythicDebugLog(ctx, "GetCallbacks start agent_filter=%s", strings.TrimSpace(func() string {
		if filter == nil {
			return ""
		}
		return filter.AgentID
	}()))
	var queryResponse struct {
		Callback []struct {
			ID              int    `json:"id"`
			DisplayID       int    `json:"display_id"`
			AgentCallbackID string `json:"agent_callback_id"`
			InitCallback    string `json:"init_callback"`
			LastCheckin     string `json:"last_checkin"`
			User            string `json:"user"`
			Host            string `json:"host"`
			PID             int    `json:"pid"`
			IP              string `json:"ip"`
			ExternalIP      string `json:"external_ip"`
			ProcessName     string `json:"process_name"`
			Description     string `json:"description"`
			Active          bool   `json:"active"`
			Locked          bool   `json:"locked"`
			OS              string `json:"os"`
			Architecture    string `json:"architecture"`
			Domain          string `json:"domain"`
			ExtraInfo       string `json:"extra_info"`
			SleepInfo       string `json:"sleep_info"`
			Payload         struct {
				UUID        string `json:"uuid"`
				PayloadType struct {
					Name string `json:"name"`
				} `json:"payloadtype"`
			} `json:"payload"`
			Operator struct {
				Username string `json:"username"`
			} `json:"operator"`
		} `json:"callback"`
	}
	err := c.executeGraphQL(ctx, `query GetAllCallbacks {
  callback(order_by: {id: desc}) {
    id
    display_id
    agent_callback_id
    init_callback
    last_checkin
    user
    host
    pid
    ip
    external_ip
    process_name
    description
    active
    locked
    os
    architecture
    domain
    extra_info
    sleep_info
    payload {
      uuid
      payloadtype {
        name
      }
    }
    operator {
      username
    }
  }
}`,
		nil,
		&queryResponse,
	)
	if err != nil {
		mythicDebugLog(ctx, "GetCallbacks query error err=%v", err)
		return nil, err
	}

	result := &GetCallbacksOutput{Callbacks: make([]*Callback, 0, len(queryResponse.Callback))}
	wantedAgentID := ""
	if filter != nil {
		wantedAgentID = strings.TrimSpace(filter.AgentID)
	}
	for _, item := range queryResponse.Callback {
		callback := &Callback{
			ID:           strconv.Itoa(item.DisplayID),
			InternalID:   item.ID,
			DisplayID:    item.DisplayID,
			AgentID:      item.AgentCallbackID,
			Host:         item.Host,
			User:         item.User,
			PID:          item.PID,
			IP:           strings.Join(parseIPString(item.IP), ","),
			ExternalIP:   item.ExternalIP,
			ProcessName:  item.ProcessName,
			Description:  item.Description,
			OS:           item.OS,
			Architecture: item.Architecture,
			Domain:       item.Domain,
			ExtraInfo:    item.ExtraInfo,
			SleepInfo:    item.SleepInfo,
			PayloadType:  item.Payload.PayloadType.Name,
			PayloadUUID:  item.Payload.UUID,
			Operator:     item.Operator.Username,
			Active:       item.Active,
			InitCallback: item.InitCallback,
			LastCheck:    item.LastCheckin,
			Status:       callbackStatus(item.Active),
		}
		if wantedAgentID != "" && callback.AgentID != wantedAgentID {
			continue
		}
		result.Callbacks = append(result.Callbacks, callback)
	}
	result.Count = len(result.Callbacks)
	mythicDebugLog(ctx, "GetCallbacks result count=%d", result.Count)
	return result, nil
}

func (c *Client) resolveCallbackInternalID(ctx context.Context, callbackID string) (int, string, error) {
	trimmed := strings.TrimSpace(callbackID)
	if trimmed == "" {
		return 0, "", fmt.Errorf("callback_id is required")
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return 0, "", fmt.Errorf("callback_id must be a positive integer")
	}

	callbacks, err := c.GetCallbacks(ctx, &GetCallbacksInput{})
	if err != nil {
		return 0, "", err
	}
	for _, callback := range callbacks.Callbacks {
		if callback.InternalID == parsed || callback.DisplayID == parsed {
			mythicDebugLog(ctx, "resolveCallbackInternalID callback_id=%s ids internal_id=%d display_id=%s via callback cache", trimmed, callback.InternalID, callback.ID)
			return callback.InternalID, callback.ID, nil
		}
	}

	var queryResponse struct {
		Callback []struct {
			ID        int `json:"id"`
			DisplayID int `json:"display_id"`
		} `json:"callback"`
	}
	err = c.executeGraphQL(ctx, `query ResolveCallback($display_id: Int!, $id: Int!) {
  callback(where: {_or: [{display_id: {_eq: $display_id}}, {id: {_eq: $id}}]}, limit: 1) {
    id
    display_id
  }
}`,
		map[string]any{"display_id": parsed, "id": parsed},
		&queryResponse,
	)
	if err != nil {
		mythicDebugLog(ctx, "resolveCallbackInternalID callback_id=%s graphql error=%v", trimmed, err)
		return 0, "", err
	}
	if len(queryResponse.Callback) == 0 {
		mythicDebugLog(ctx, "resolveCallbackInternalID callback_id=%s not found", trimmed)
		return 0, "", fmt.Errorf("callback %s not found", trimmed)
	}
	mythicDebugLog(ctx, "resolveCallbackInternalID callback_id=%s ids internal_id=%d display_id=%d via graphql", trimmed, queryResponse.Callback[0].ID, queryResponse.Callback[0].DisplayID)
	return queryResponse.Callback[0].ID, strconv.Itoa(queryResponse.Callback[0].DisplayID), nil
}

func (c *Client) GetCallbackCommands(ctx context.Context, input *GetCallbackCommandsInput) (*GetCallbackCommandsOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("callback commands input is required")
	}
	internalID, displayID, err := c.resolveCallbackInternalID(ctx, input.CallbackID)
	if err != nil {
		mythicDebugLog(ctx, "GetCallbackCommands resolve error callback_id=%s err=%v", input.CallbackID, err)
		return nil, err
	}
	mythicDebugLog(ctx, "GetCallbackCommands start callback_id=%s internal_id=%d", displayID, internalID)

	var queryResponse struct {
		LoadedCommands []struct {
			Command struct {
				Cmd                 string   `json:"cmd"`
				SupportedUIFeatures []string `json:"supported_ui_features"`
				PayloadType         struct {
					Name string `json:"name"`
				} `json:"payloadtype"`
				CommandParameters []struct {
					Name               string `json:"name"`
					CLIName            string `json:"cli_name"`
					DisplayName        string `json:"display_name"`
					Type               string `json:"type"`
					ParameterGroupName string `json:"parameter_group_name"`
					Required           bool   `json:"required"`
					DefaultValue       string `json:"default_value"`
					Description        string `json:"description"`
				} `json:"commandparameters"`
			} `json:"command"`
		} `json:"loadedcommands"`
	}
	err = c.executeGraphQL(ctx, `query GetLoadedCommands($callback_id: Int!) {
  loadedcommands(where: {callback_id: {_eq: $callback_id}}) {
    command {
      cmd
      supported_ui_features
      payloadtype {
        name
      }
      commandparameters(order_by: {ui_position: asc}) {
        name
        cli_name
        display_name
        type
        parameter_group_name
        required
        default_value
        description
      }
    }
  }
}`,
		map[string]any{"callback_id": internalID},
		&queryResponse,
	)
	if err != nil {
		mythicDebugLog(ctx, "GetCallbackCommands query error callback_id=%s err=%v", displayID, err)
		return nil, err
	}

	commands := make([]CallbackCommandInfo, 0, len(queryResponse.LoadedCommands))
	for _, loadedCommand := range queryResponse.LoadedCommands {
		parameters := make([]CallbackCommandParameter, 0, len(loadedCommand.Command.CommandParameters))
		groupsSeen := map[string]struct{}{}
		groups := make([]string, 0, len(loadedCommand.Command.CommandParameters))
		for _, parameter := range loadedCommand.Command.CommandParameters {
			parameters = append(parameters, CallbackCommandParameter{
				Name:               parameter.Name,
				CLIName:            parameter.CLIName,
				DisplayName:        parameter.DisplayName,
				Type:               parameter.Type,
				ParameterGroupName: parameter.ParameterGroupName,
				Required:           parameter.Required,
				DefaultValue:       parameter.DefaultValue,
				Description:        parameter.Description,
			})
			group := strings.TrimSpace(parameter.ParameterGroupName)
			if group == "" {
				continue
			}
			if _, exists := groupsSeen[group]; exists {
				continue
			}
			groupsSeen[group] = struct{}{}
			groups = append(groups, group)
		}
		commands = append(commands, CallbackCommandInfo{
			CommandName:         loadedCommand.Command.Cmd,
			PayloadType:         loadedCommand.Command.PayloadType.Name,
			SupportedUIFeatures: append([]string(nil), loadedCommand.Command.SupportedUIFeatures...),
			ParameterGroups:     groups,
			Parameters:          parameters,
		})
	}

	result := &GetCallbackCommandsOutput{
		CallbackID: displayID,
		Commands:   commands,
		Count:      len(commands),
	}
	mythicDebugLog(ctx, "GetCallbackCommands result callback_id=%s count=%d commands=%s", result.CallbackID, result.Count, summarizeCommandNames(result.Commands))
	return result, nil
}

// resolveParameterGroup looks up the correct parameter_group for a command on a
// callback.  Returns the first group name found, or "" when the command has no
// groups / isn't loaded / can't be queried.
func (c *Client) resolveParameterGroup(ctx context.Context, callbackID, command string) (string, error) {
	cmds, err := c.GetCallbackCommands(ctx, &GetCallbackCommandsInput{CallbackID: callbackID})
	if err != nil {
		return "", err
	}
	for _, cmd := range cmds.Commands {
		if strings.EqualFold(cmd.CommandName, command) && len(cmd.ParameterGroups) > 0 {
			return cmd.ParameterGroups[0], nil
		}
	}
	return "", fmt.Errorf("command %q not found or has no parameter groups on callback %s", command, callbackID)
}

// stripArgsWarning removes Apollo's informational "args aren't being used"
// warnings from task output.  These are emitted when optional parameters belong
// to a different parameter group (e.g. pe_file in "New PE" while the task uses
// "Default").  The task still executes normally — the warning is noise that
// confuses the agent into thinking it made a mistake.
func stripArgsWarning(raw string) string {
	// Match lines like:
	//   The following args aren't being used because they don't belong to the Default parameter group:
	//   {"pe_file": null}
	const prefix = "The following args aren't being used because they don't belong to the"
	idx := strings.Index(raw, prefix)
	if idx == -1 {
		return raw
	}
	// Find the end of this block — the JSON block after the prefix line.
	end := strings.Index(raw[idx:], "\n\n")
	if end == -1 {
		// Single line or trailing; just cut from the prefix.
		before := strings.TrimSpace(raw[:idx])
		return before
	}
	before := strings.TrimSpace(raw[:idx])
	after := strings.TrimSpace(raw[idx+end:])
	if before == "" {
		return after
	}
	return before + "\n" + after
}

// GetFiles lists files registered in Mythic (the file store, NOT callback-host files).
func (c *Client) GetFiles(ctx context.Context, input *GetFilesInput) (*GetFilesOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	limit := 100
	if input != nil && input.Limit > 0 {
		limit = input.Limit
	}
	var queryResponse struct {
		FileMeta []struct {
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
			Filename            string `json:"filename_text"`
			MD5                 string `json:"md5"`
			SHA1                string `json:"sha1"`
			Size                int    `json:"size"`
			Comment             string `json:"comment"`
			OperatorID          int    `json:"operator_id"`
			Timestamp           string `json:"timestamp"`
			Deleted             bool   `json:"deleted"`
			TaskDisplayID       *int   `json:"task_id"`
		} `json:"filemeta"`
	}
	query := `query GetFiles($limit: Int!) {
  filemeta(order_by: {id: desc}, limit: $limit, where: {deleted: {_eq: false}}) {
    id
    agent_file_id
    total_chunks
    chunks_received
    complete
    path
    full_remote_path
    host
    is_payload
    is_screenshot
    is_download_from_agent
    filename_text
    md5
    sha1
    size
    comment
    operator_id
    timestamp
    deleted
    task_id
  }
}`
	if err := c.executeGraphQL(ctx, query, map[string]any{"limit": limit}, &queryResponse); err != nil {
		return nil, fmt.Errorf("list files failed: %w", err)
	}

	result := &GetFilesOutput{Files: make([]*FileMeta, 0, len(queryResponse.FileMeta))}
	for _, f := range queryResponse.FileMeta {
		taskDisplayID := ""
		if f.TaskDisplayID != nil {
			taskDisplayID = strconv.Itoa(*f.TaskDisplayID)
		}
		result.Files = append(result.Files, &FileMeta{
			ID:                  f.ID,
			AgentFileID:         f.AgentFileID,
			TotalChunks:         f.TotalChunks,
			ChunksReceived:      f.ChunksReceived,
			Complete:            f.Complete,
			Path:                f.Path,
			FullRemotePath:      f.FullRemotePath,
			Host:                f.Host,
			IsPayload:           f.IsPayload,
			IsScreenshot:        f.IsScreenshot,
			IsDownloadFromAgent: f.IsDownloadFromAgent,
			Filename:            decodeMythicText(f.Filename),
			MD5:                 f.MD5,
			SHA1:                f.SHA1,
			Size:                int64(f.Size),
			Comment:             f.Comment,
			OperatorID:          f.OperatorID,
			Timestamp:           f.Timestamp,
			Deleted:             f.Deleted,
			TaskDisplayID:       taskDisplayID,
		})
	}
	result.Count = len(result.Files)
	return result, nil
}

func (c *Client) UploadFile(ctx context.Context, input *UploadFileInput) (*UploadFileOutput, error) {
	if strings.TrimSpace(input.FileName) == "" {
		return nil, fmt.Errorf("file_name is required")
	}
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}

	var fileBytes []byte
	if fp := strings.TrimSpace(input.FilePath); fp != "" {
		var err error
		fileBytes, err = os.ReadFile(fp)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %q: %w", fp, err)
		}
	} else {
		contents := strings.TrimSpace(input.Contents)
		if contents == "" {
			return nil, fmt.Errorf("file_path or contents is required")
		}
		var err error
		fileBytes, err = base64.StdEncoding.DecodeString(contents)
		if err != nil {
			return nil, fmt.Errorf("contents must be base64 encoded: %w", err)
		}
	}

	endpoint, err := c.apiURL("api/v1.4/task_upload_file_webhook")
	if err != nil {
		return nil, err
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", input.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload form: %w", err)
	}
	if _, err := part.Write(fileBytes); err != nil {
		return nil, fmt.Errorf("failed to write upload form: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize upload form: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("apitoken", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var uploadResp struct {
		AgentFileID string `json:"agent_file_id"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		return nil, fmt.Errorf("failed to decode upload response: %w", err)
	}
	if uploadResp.AgentFileID == "" {
		return nil, fmt.Errorf("upload response did not include agent_file_id")
	}
	return &UploadFileOutput{FileID: uploadResp.AgentFileID, Status: firstNonEmpty(uploadResp.Status, "uploaded")}, nil
}

func (c *Client) DownloadFile(ctx context.Context, input *DownloadFileInput) (*DownloadFileOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("download file input is required")
	}
	fileID := strings.TrimSpace(input.FileID)
	if fileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}

	body, err := c.downloadFileBody(ctx, fileID, true)
	if err != nil {
		return nil, err
	}

	truncated := len(body) > maxDownloadedFileBytes
	if truncated {
		body = body[:maxDownloadedFileBytes]
	}

	contents := body
	if len(body) > 0 && body[0] == '{' {
		var wrapped struct {
			File string `json:"file"`
		}
		if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.File != "" {
			decoded, decodeErr := base64.StdEncoding.DecodeString(wrapped.File)
			if decodeErr == nil {
				contents = decoded
			}
		}
	}

	encoded := base64.StdEncoding.EncodeToString(contents)
	return &DownloadFileOutput{
		FileID:    fileID,
		Contents:  encoded,
		Size:      len(contents),
		Encoding:  "base64",
		Truncated: truncated,
	}, nil
}

// DeleteFile marks a file as deleted in Mythic's file store (soft delete).
func (c *Client) DeleteFile(ctx context.Context, input *DeleteFileInput) (*DeleteFileOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	fileID := strings.TrimSpace(input.FileID)
	if fileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}

	var mutationResponse struct {
		UpdateFileMeta struct {
			AffectedRows int `json:"affected_rows"`
		} `json:"update_filemeta"`
	}
	err := c.executeGraphQL(ctx, `mutation DeleteFile($file_id: String!) {
  update_filemeta(where: {agent_file_id: {_eq: $file_id}}, _set: {deleted: true}) {
    affected_rows
  }
}`, map[string]any{"file_id": fileID}, &mutationResponse)
	if err != nil {
		return nil, fmt.Errorf("delete file failed: %w", err)
	}
	if mutationResponse.UpdateFileMeta.AffectedRows == 0 {
		return nil, fmt.Errorf("file not found: %s", fileID)
	}
	return &DeleteFileOutput{FileID: fileID, Status: "deleted"}, nil
}

// UpdateFileComment sets or clears the comment on a file in Mythic's file store.
func (c *Client) UpdateFileComment(ctx context.Context, input *UpdateFileCommentInput) (*UpdateFileCommentOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	fileID := strings.TrimSpace(input.FileID)
	if fileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}

	var mutationResponse struct {
		UpdateFileMeta struct {
			AffectedRows int `json:"affected_rows"`
		} `json:"update_filemeta"`
	}
	err := c.executeGraphQL(ctx, `mutation UpdateFileComment($file_id: String!, $comment: String!) {
  update_filemeta(where: {agent_file_id: {_eq: $file_id}}, _set: {comment: $comment}) {
    affected_rows
  }
}`, map[string]any{"file_id": fileID, "comment": input.Comment}, &mutationResponse)
	if err != nil {
		return nil, fmt.Errorf("update file comment failed: %w", err)
	}
	if mutationResponse.UpdateFileMeta.AffectedRows == 0 {
		return nil, fmt.Errorf("file not found: %s", fileID)
	}
	return &UpdateFileCommentOutput{FileID: fileID, Comment: input.Comment, Status: "updated"}, nil
}

func (c *Client) downloadFileBody(ctx context.Context, fileID string, allowDirectFallback bool) ([]byte, error) {
	endpoint, err := c.apiURL("api/v1.4/files/download/" + fileID)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("apitoken", c.apiKey)
	mythicDebugLog(ctx, "downloadFileBody start file_id=%s endpoint=%s", fileID, endpoint)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		mythicDebugLog(ctx, "downloadFileBody request error file_id=%s err=%v", fileID, err)
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadedFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read download response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		if allowDirectFallback && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) {
			mythicDebugLog(ctx, "downloadFileBody auth failure file_id=%s status=%d, falling back to direct download", fileID, resp.StatusCode)
			return c.downloadFileBodyDirect(ctx, fileID)
		}
		mythicDebugLog(ctx, "downloadFileBody failure file_id=%s status=%d body=%s", fileID, resp.StatusCode, truncateForDebug(string(body), 256))
		return nil, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	mythicDebugLog(ctx, "downloadFileBody success file_id=%s bytes=%d", fileID, len(body))
	return body, nil
}

func (c *Client) downloadFileBodyDirect(ctx context.Context, fileID string) ([]byte, error) {
	endpoint, err := c.apiURL("direct/download/" + fileID)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create direct download request: %w", err)
	}
	req.Header.Set("apitoken", c.apiKey)
	mythicDebugLog(ctx, "downloadFileBodyDirect start file_id=%s endpoint=%s", fileID, endpoint)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		mythicDebugLog(ctx, "downloadFileBodyDirect request error file_id=%s err=%v", fileID, err)
		return nil, fmt.Errorf("direct download request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadedFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read direct download response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		mythicDebugLog(ctx, "downloadFileBodyDirect failure file_id=%s status=%d body=%s", fileID, resp.StatusCode, truncateForDebug(string(body), 256))
		return nil, fmt.Errorf("direct download failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	mythicDebugLog(ctx, "downloadFileBodyDirect success file_id=%s bytes=%d", fileID, len(body))
	return body, nil
}

type taskFileRecord struct {
	AgentFileID         string
	FilenameText        string
	FullRemotePath      string
	Host                string
	IsDownloadFromAgent bool
	Deleted             bool
	Timestamp           time.Time
}

func (c *Client) queryLatestTaskFile(ctx context.Context, taskDisplayID int, requireDownload bool) (*taskFileRecord, error) {
	var queryResponse struct {
		Task []struct {
			Filemeta []struct {
				AgentFileID         string `json:"agent_file_id"`
				FilenameText        string `json:"filename_text"`
				FullRemotePathText  string `json:"full_remote_path_text"`
				Host                string `json:"host"`
				IsDownloadFromAgent bool   `json:"is_download_from_agent"`
				Deleted             bool   `json:"deleted"`
				Timestamp           string `json:"timestamp"`
			} `json:"filemeta"`
		} `json:"task"`
	}
	err := c.executeGraphQL(ctx, `query GetTaskFiles($display_id: Int!) {
  task(where: {display_id: {_eq: $display_id}}, limit: 1) {
    filemeta(order_by: {id: desc}) {
      agent_file_id
      filename_text
      full_remote_path_text
      host
      is_download_from_agent
      deleted
      timestamp
    }
  }
}`,
		map[string]any{"display_id": taskDisplayID},
		&queryResponse,
	)
	if err != nil {
		return nil, err
	}
	if len(queryResponse.Task) == 0 {
		return nil, fmt.Errorf("task %d not found", taskDisplayID)
	}
	for _, item := range queryResponse.Task[0].Filemeta {
		if strings.TrimSpace(item.AgentFileID) == "" || item.Deleted {
			continue
		}
		if requireDownload && !item.IsDownloadFromAgent {
			continue
		}
		record := &taskFileRecord{
			AgentFileID:         item.AgentFileID,
			FilenameText:        decodeMythicText(item.FilenameText),
			FullRemotePath:      decodeMythicText(item.FullRemotePathText),
			Host:                item.Host,
			IsDownloadFromAgent: item.IsDownloadFromAgent,
			Deleted:             item.Deleted,
		}
		if item.Timestamp != "" {
			if parsed, parseErr := time.Parse(time.RFC3339Nano, item.Timestamp); parseErr == nil {
				record.Timestamp = parsed
			} else if parsed, parseErr := time.Parse(time.RFC3339, item.Timestamp); parseErr == nil {
				record.Timestamp = parsed
			}
		}
		return record, nil
	}
	if requireDownload {
		return nil, fmt.Errorf("task %d did not produce a downloadable file", taskDisplayID)
	}
	return nil, fmt.Errorf("task %d did not produce a file", taskDisplayID)
}

func (c *Client) waitForTaskFile(ctx context.Context, taskDisplayID int, timeoutSeconds int, requireDownload bool) (*taskFileRecord, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 60
	}
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	for {
		record, err := c.queryLatestTaskFile(ctx, taskDisplayID, requireDownload)
		if err == nil {
			return record, nil
		}
		if !strings.Contains(err.Error(), "did not produce") {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout after %d seconds waiting for task %d file", timeoutSeconds, taskDisplayID)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

func (c *Client) CallbackUploadFile(ctx context.Context, input *CallbackUploadFileInput) (*CallbackUploadFileOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("callback upload input is required")
	}
	callbackID := strings.TrimSpace(input.CallbackID)
	if callbackID == "" {
		return nil, fmt.Errorf("callback_id is required")
	}
	filePath := strings.TrimSpace(input.FilePath)
	contents := strings.TrimSpace(input.Contents)
	if filePath == "" && contents == "" {
		return nil, fmt.Errorf("file_path or contents is required")
	}
	fileName := strings.TrimSpace(input.FileName)
	if fileName == "" {
		fileName = deriveRemoteFilename(input.RemotePath)
	}
	if fileName == "" {
		return nil, fmt.Errorf("file_name or remote_path filename is required")
	}

	staged, err := c.UploadFile(ctx, &UploadFileInput{
		CallbackID: callbackID,
		FileName:   fileName,
		FilePath:   filePath,
		Contents:   contents,
	})
	if err != nil {
		mythicDebugLog(ctx, "CallbackUploadFile stage error callback_id=%s file_name=%s file_path=%s err=%v", callbackID, fileName, filePath, err)
		return nil, err
	}
	mythicDebugLog(ctx, "CallbackUploadFile staged callback_id=%s file_name=%s staged_file_id=%s", callbackID, fileName, staged.FileID)

	params := strings.TrimSpace(input.Parameters)
	if params == "" {
		payload, marshalErr := json.Marshal(map[string]string{
			"filename":    fileName,
			"remote_path": strings.TrimSpace(input.RemotePath),
		})
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to encode callback upload parameters: %w", marshalErr)
		}
		params = string(payload)
	}

	task, err := c.IssueTask(ctx, &TaskAgentInput{
		CallbackID:      callbackID,
		Command:         firstNonEmpty(input.Command, "upload"),
		Parameters:      params,
		PayloadType:     strings.TrimSpace(input.PayloadType),
		ParameterGroup:  firstNonEmpty(strings.TrimSpace(input.ParameterGroup), "ExistingFile"),
		TaskingLocation: firstNonEmpty(strings.TrimSpace(input.TaskingLocation), "file_browser"),
		FileIDs:         []string{staged.FileID},
	})
	if err != nil {
		mythicDebugLog(ctx, "CallbackUploadFile task error callback_id=%s remote_path=%s err=%v", callbackID, strings.TrimSpace(input.RemotePath), err)
		return nil, err
	}

	// Async mode: return immediately with task_id, don't block on wait.
	if input.Async {
		mythicDebugLog(ctx, "CallbackUploadFile async dispatch task_id=%s", task.TaskID)
		return &CallbackUploadFileOutput{
			TaskID:      task.TaskID,
			Status:      task.Status,
			Completed:   false,
			Summary:     fmt.Sprintf("upload task %s dispatched in background (staged file: %s). Use mythic_get_task_status to track completion.", task.TaskID, staged.FileID),
			FileID:      staged.FileID,
			FileName:    fileName,
			RemotePath:  strings.TrimSpace(input.RemotePath),
			StagedState: staged.Status,
			Background:  true,
		}, nil
	}

	waitResult, err := c.WaitForTask(ctx, &WaitForTaskInput{TaskID: task.TaskID, Timeout: input.WaitTimeout})
	if err != nil {
		mythicDebugLog(ctx, "CallbackUploadFile wait error task_id=%s err=%v", task.TaskID, err)
		return nil, err
	}
	mythicDebugLog(ctx, "CallbackUploadFile result task_id=%s status=%s summary=%s", task.TaskID, waitResult.Status, truncateForDebug(waitResult.Summary, 512))

	return &CallbackUploadFileOutput{
		TaskID:      task.TaskID,
		Status:      waitResult.Status,
		Completed:   waitResult.Completed,
		Summary:     firstNonEmpty(waitResult.Summary, task.Response),
		FileID:      staged.FileID,
		FileName:    fileName,
		RemotePath:  strings.TrimSpace(input.RemotePath),
		StagedState: staged.Status,
	}, nil
}

func (c *Client) CallbackDownloadFile(ctx context.Context, input *CallbackDownloadFileInput) (*CallbackDownloadFileOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("callback download input is required")
	}
	callbackID := strings.TrimSpace(input.CallbackID)
	if callbackID == "" {
		return nil, fmt.Errorf("callback_id is required")
	}
	params := strings.TrimSpace(input.Parameters)
	if params == "" {
		params = strings.TrimSpace(input.RemotePath)
	}
	if params == "" {
		return nil, fmt.Errorf("remote_path or parameters is required")
	}

	task, err := c.IssueTask(ctx, &TaskAgentInput{
		CallbackID:      callbackID,
		Command:         firstNonEmpty(input.Command, "download"),
		Parameters:      params,
		PayloadType:     strings.TrimSpace(input.PayloadType),
		ParameterGroup:  strings.TrimSpace(input.ParameterGroup),
		TaskingLocation: firstNonEmpty(strings.TrimSpace(input.TaskingLocation), "file_browser"),
	})
	if err != nil {
		mythicDebugLog(ctx, "CallbackDownloadFile task error callback_id=%s remote_path=%s err=%v", callbackID, strings.TrimSpace(input.RemotePath), err)
		return nil, err
	}

	// Async mode: return immediately with task_id, don't block on wait.
	if input.Async {
		mythicDebugLog(ctx, "CallbackDownloadFile async dispatch task_id=%s", task.TaskID)
		return &CallbackDownloadFileOutput{
			TaskID:     task.TaskID,
			Status:     task.Status,
			Completed:  false,
			Summary:    fmt.Sprintf("download task %s dispatched in background. Use mythic_get_task_status to track, then mythic_get_files + mythic_download_file to retrieve the file when complete.", task.TaskID),
			RemotePath: strings.TrimSpace(input.RemotePath),
			Background: true,
		}, nil
	}

	waitResult, err := c.WaitForTask(ctx, &WaitForTaskInput{TaskID: task.TaskID, Timeout: input.WaitTimeout})
	if err != nil {
		mythicDebugLog(ctx, "CallbackDownloadFile wait error task_id=%s err=%v", task.TaskID, err)
		return nil, err
	}

	taskDisplayID, err := strconv.Atoi(strings.TrimSpace(task.TaskID))
	if err != nil || taskDisplayID <= 0 {
		return nil, fmt.Errorf("task_id must be a positive integer display ID")
	}
	fileRecord, err := c.waitForTaskFile(ctx, taskDisplayID, input.WaitTimeout, true)
	if err != nil {
		mythicDebugLog(ctx, "CallbackDownloadFile waitForTaskFile error task_id=%s err=%v", task.TaskID, err)
		return nil, err
	}
	mythicDebugLog(ctx, "CallbackDownloadFile file record task_id=%s file_id=%s remote_path=%s host=%s", task.TaskID, fileRecord.AgentFileID, fileRecord.FullRemotePath, fileRecord.Host)
	file, err := c.DownloadFile(ctx, &DownloadFileInput{FileID: fileRecord.AgentFileID})
	if err != nil {
		mythicDebugLog(ctx, "CallbackDownloadFile DownloadFile error task_id=%s file_id=%s err=%v", task.TaskID, fileRecord.AgentFileID, err)
		return nil, err
	}
	mythicDebugLog(ctx, "CallbackDownloadFile result task_id=%s status=%s outputs_size=%d", task.TaskID, waitResult.Status, file.Size)

	return &CallbackDownloadFileOutput{
		TaskID:     task.TaskID,
		Status:     waitResult.Status,
		Completed:  waitResult.Completed,
		Summary:    firstNonEmpty(waitResult.Summary, task.Response),
		FileID:     file.FileID,
		FileName:   firstNonEmpty(fileRecord.FilenameText, deriveRemoteFilename(input.RemotePath)),
		RemotePath: firstNonEmpty(fileRecord.FullRemotePath, strings.TrimSpace(input.RemotePath)),
		Host:       fileRecord.Host,
		Contents:   file.Contents,
		Size:       file.Size,
		Encoding:   file.Encoding,
		Truncated:  file.Truncated,
	}, nil
}

func callbackStatus(active bool) string {
	if active {
		return "active"
	}
	return "inactive"
}

func parseIPString(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func truncateForDebug(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit <= 0 || utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit]) + "..."
}

func summarizeCommandNames(commands []CallbackCommandInfo) string {
	if len(commands) == 0 {
		return ""
	}
	names := make([]string, 0, len(commands))
	for _, command := range commands {
		names = append(names, command.CommandName)
	}
	return strings.Join(names, ",")
}

func decodeMythicText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return value
	}
	if !utf8.Valid(decoded) {
		return value
	}
	decodedText := strings.TrimSpace(string(decoded))
	if decodedText == "" {
		return value
	}
	return decodedText
}

func deriveRemoteFilename(remotePath string) string {
	trimmed := strings.TrimSpace(remotePath)
	trimmed = strings.TrimRight(trimmed, `/\\`)
	if trimmed == "" {
		return ""
	}
	lastSlash := strings.LastIndexAny(trimmed, `/\\`)
	if lastSlash < 0 {
		return trimmed
	}
	return strings.TrimSpace(trimmed[lastSlash+1:])
}

func normalizePayloadOS(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "linux":
		return "Linux"
	case "macos", "darwin":
		return "macOS"
	case "windows", "win":
		return "Windows"
	default:
		return raw
	}
}

func derivePayloadFilename(input *CreatePayloadInput) string {
	if input == nil {
		return "payload"
	}
	if filename := strings.TrimSpace(input.Filename); filename != "" {
		return filename
	}
	name := strings.TrimSpace(input.PayloadType)
	if name == "" {
		name = "payload"
	}
	format := strings.TrimSpace(input.Format)
	if format == "" {
		return name
	}
	if strings.HasPrefix(format, ".") {
		return name + format
	}
	return name + "." + format
}

func buildPayloadBuildParameters(input *CreatePayloadInput) []map[string]any {
	params := make([]map[string]any, 0)
	reserved := map[string]struct{}{
		"selected_os": {},
	}
	for key, value := range input.Parameters {
		if _, skip := reserved[key]; skip {
			continue
		}
		params = append(params, map[string]any{"name": key, "value": value})
	}
	for key, value := range input.BuildParameters {
		params = append(params, map[string]any{"name": key, "value": value})
	}
	return params
}

func buildC2Profiles(profiles []C2ProfileInput) []map[string]any {
	result := make([]map[string]any, 0, len(profiles))
	for _, profile := range profiles {
		name := strings.TrimSpace(profile.Name)
		if name == "" {
			continue
		}
		params := make(map[string]any, len(profile.Parameters))
		for key, value := range profile.Parameters {
			params[key] = value
		}
		result = append(result, map[string]any{
			"c2_profile":            name,
			"c2_profile_is_p2p":     false,
			"c2_profile_parameters": params,
		})
	}
	return result
}

// RemoveCallback removes callbacks by display ID via hard delete.
func (c *Client) RemoveCallback(ctx context.Context, input *RemoveCallbackInput) (*RemoveCallbackOutput, error) {
	if err := c.ensureConfigured(); err != nil {
		return nil, err
	}
	if input == nil || len(input.CallbackIDs) == 0 {
		return nil, fmt.Errorf("callback_ids is required")
	}

	mythicDebugLog(ctx, "RemoveCallback start callback_ids=%v", input.CallbackIDs)
	ids := make([]int, 0, len(input.CallbackIDs))
	for _, raw := range input.CallbackIDs {
		displayID, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || displayID <= 0 {
			return nil, fmt.Errorf("callback_id must be a positive integer display ID, got %q", raw)
		}
		ids = append(ids, displayID)
	}

	if err := c.deleteCallbacksHard(ctx, ids); err != nil {
		mythicDebugLog(ctx, "RemoveCallback hard delete error ids=%v err=%v", ids, err)
		return nil, fmt.Errorf("failed to delete callbacks: %w", err)
	}
	mythicDebugLog(ctx, "RemoveCallback hard delete succeeded ids=%v", ids)
	return &RemoveCallbackOutput{
		Status:  "deleted",
		Removed: len(ids),
	}, nil
}

func (c *Client) deleteCallbacksHard(ctx context.Context, ids []int) error {
	mythicDebugLog(ctx, "deleteCallbacksHard request ids=%v", ids)
	var mutationResponse struct {
		DeleteTasksAndCallbacks struct {
			Status          string `json:"status"`
			Error           string `json:"error"`
			FailedCallbacks []int  `json:"failed_callbacks"`
			FailedTasks     []int  `json:"failed_tasks"`
		} `json:"deleteTasksAndCallbacks"`
	}
	err := c.executeGraphQL(ctx, `mutation DeleteComponents($callbacks: [Int], $tasks: [Int]) {
  deleteTasksAndCallbacks(callbacks: $callbacks, tasks: $tasks) {
    status
    error
    failed_callbacks
    failed_tasks
  }
}`,
		map[string]any{"callbacks": ids, "tasks": []int{}},
		&mutationResponse,
	)
	if err != nil {
		mythicDebugLog(ctx, "deleteCallbacksHard graphql error ids=%v err=%v", ids, err)
		return err
	}
	resp := mutationResponse.DeleteTasksAndCallbacks
	mythicDebugLog(ctx, "deleteCallbacksHard response ids=%v status=%s error=%s failed_callbacks=%v", ids, resp.Status, resp.Error, resp.FailedCallbacks)
	if resp.Status != "success" {
		return fmt.Errorf("deleteTasksAndCallbacks failed: %s", strings.TrimSpace(resp.Error))
	}
	if len(resp.FailedCallbacks) > 0 {
		return fmt.Errorf("failed to delete callbacks: %v", resp.FailedCallbacks)
	}
	return nil
}

// SendMessage sends a message to Mythic
func (c *Client) SendMessage(ctx context.Context, msg interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.wsConn == nil {
		return fmt.Errorf("not connected to Mythic")
	}

	return c.wsConn.WriteJSON(msg)
}

// ReceiveMessage receives a message from Mythic
func (c *Client) ReceiveMessage(ctx context.Context, msg interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.wsConn == nil {
		return fmt.Errorf("not connected to Mythic")
	}

	return c.wsConn.ReadJSON(msg)
}
