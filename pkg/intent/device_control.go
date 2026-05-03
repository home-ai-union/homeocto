package intent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/home-ai-union/homeocto/pkg/data"
	"github.com/home-ai-union/homeocto/pkg/workflow"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// DeviceControlIntent handles all device.control.* intents.
//
// Processing logic (mirrors ��ͼʶ�����.md ��4.1):
//  1. Query the WorkflowStore for a workflow whose triggers match the user input.
//  2. If a match is found, execute it via workflow.Engine and return the result.
//  3. If no match is found, return Handled=false so the agent loop falls through
//     to the large model, which will generate a new workflow.
type DeviceControlIntent struct {
	workflowStore data.WorkflowStore
	engine        workflow.Engine
	// provider is the small LLM used for workflow matching.
	provider providers.LLMProvider
	// modelName is the model identifier passed to provider when making calls.
	modelName string
}

// NewDeviceControlIntent creates a DeviceControlIntent.
// workflowStore and engine are required; provider/modelName are used for
// LLM-based workflow matching and may be nil/empty (falls back gracefully).
func NewDeviceControlIntent(store data.WorkflowStore, eng workflow.Engine, provider providers.LLMProvider, modelName string) *DeviceControlIntent {
	return &DeviceControlIntent{
		workflowStore: store,
		engine:        eng,
		provider:      provider,
		modelName:     modelName,
	}
}

// Types implements Intent.
// DeviceControlIntent handles all device.control.* subtypes.
func (d *DeviceControlIntent) Types() []IntentType {
	return []IntentType{
		IntentDeviceControlSingle,
		IntentDeviceControlScene,
		IntentDeviceControlGlobal,
		IntentDeviceControlCorrect,
	}
}

// Run executes the device control intent.
func (d *DeviceControlIntent) Run(ctx context.Context, ictx IntentContext) IntentResponse {
	if d.workflowStore == nil || d.engine == nil {
		// Infrastructure not available �C fall through to large model.
		return IntentResponse{Handled: false}
	}

	// Try to find a matching workflow via LLM-based matching.
	wf, err := d.matchWorkflow(ctx, ictx.UserInput)
	if err != nil {
		logger.ErrorCF("intent", "device control workflow match error",
			map[string]any{"error": err.Error(), "input": ictx.UserInput})
		return IntentResponse{Handled: false}
	}

	if wf == nil {
		// No match found �C ask the large model to generate a workflow.
		logger.InfoCF("intent", "no matching workflow, falling through to large model",
			map[string]any{"input": ictx.UserInput})
		return IntentResponse{Handled: false}
	}

	// Execute the matched workflow.
	execCtx := data.ExecutionContext{
		MemberName: ictx.SenderID,
		TriggerBy:  "intent",
		Input: map[string]interface{}{
			"user_input": ictx.UserInput,
			"entities":   ictx.Result.Entities,
		},
	}

	record, err := d.engine.Execute(ctx, wf, execCtx)
	if err != nil {
		logger.ErrorCF("intent", "workflow execution failed",
			map[string]any{"workflow_id": wf.ID, "error": err.Error()})
		return IntentResponse{
			Handled:  true,
			Response: fmt.Sprintf("ִ�й�������%s��ʱ���ִ������Ժ����ԡ�", wf.Name),
			Error:    err,
		}
	}

	msg := buildExecutionSummary(wf.Name, record)
	return IntentResponse{Handled: true, Response: msg}
}

// matchWorkflow finds the best-matching workflow for the given user input
// using the small LLM provider.  It builds a concise catalogue of enabled
// workflows (id / name / description) and asks the model to return the best
// matching ID as JSON {"id":"<workflow-id>"}.  If no workflow fits, the model
// should return {"id":""} and the method returns nil.
// Falls back to nil (no match) on any error so the caller can fall through
// safely to the large model.
func (d *DeviceControlIntent) matchWorkflow(ctx context.Context, userInput string) (*data.WorkflowDef, error) {
	metas, err := d.workflowStore.GetAllMeta()
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}

	// Build enabled-only catalogue.
	var catalogue strings.Builder
	for _, m := range metas {
		if !m.Enabled {
			continue
		}
		catalogue.WriteString(fmt.Sprintf("- id=%q  name=%q  description=%q\n", m.ID, m.Name, m.Description))
	}
	if catalogue.Len() == 0 {
		return nil, nil
	}

	systemPrompt := `����һ��������ƥ�����֡������û�ָ��Ӻ�ѡ�������б���ѡ����ƥ���һ����
ֻ��� JSON����ʽΪ {"id":"<workflow-id>"}��
���û�к��ʵĹ���������� {"id":""}��
��Ҫ����κ��������ݡ�`

	userMsg := fmt.Sprintf("��ѡ��������\n%s\n�û�ָ�%s", catalogue.String(), userInput)

	messages := []providers.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}

	mn := d.modelName
	if mn == "" {
		mn = d.provider.GetDefaultModel()
	}

	resp, err := d.provider.Chat(ctx, messages, nil, mn, map[string]any{
		"max_tokens":  64,
		"temperature": 0.0,
	})
	if err != nil {
		logger.WarnCF("intent", "workflow match LLM error, falling through",
			map[string]any{"error": err.Error()})
		return nil, nil
	}
	if resp == nil || resp.Content == "" {
		return nil, nil
	}

	// Extract JSON from response.
	raw := extractJSON(resp.Content)
	if raw == "" {
		return nil, nil
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil || result.ID == "" {
		return nil, nil
	}

	wf, err := d.workflowStore.GetByID(result.ID)
	if err != nil {
		return nil, fmt.Errorf("load workflow %s: %w", result.ID, err)
	}
	return wf, nil
}

// buildExecutionSummary creates a brief human-readable summary of a workflow run.
func buildExecutionSummary(workflowName string, record *data.ExecutionRecord) string {
	if record == nil {
		return fmt.Sprintf("��ִ�й�������%s����", workflowName)
	}
	if !record.Success {
		return fmt.Sprintf("��������%s��ִ��ʧ�ܣ�%s", workflowName, record.Error)
	}
	return fmt.Sprintf("����ɣ�%s����ִ�� %d ������", workflowName, len(record.StepExecutions))
}
