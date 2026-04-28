package intent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	homeclawcfg "github.com/home-ai-union/homeocto/pkg/homeocto/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// intentClassifyPrompt is the system prompt used for intent classification.
// The intent types here correspond 1-to-1 with the IntentType constants in intent.go.
const intentClassifyPrompt = `����һ�����ܼҾ����ֵ���ͼʶ������������û����룬�����������ѡ����ƥ�����ͼ��

## �豸������
- device.control.single:  ���Ƶ��������豸�����ơ��ؿյ������¶ȵ�26�ȣ�
- device.control.scene:   �����������龰ģʽ��˯��ģʽ�����š��ؼҡ���Ӱģʽ��
- device.control.global:  �����л�ĳ���豸ִ��ͬһ�������ص����еơ�ȫ�ݿյ����ͣ�
- device.control.correct: �����ղŵĲ��������ԣ�����̨�ơ�����յ�ƹص��

## �豸������
- device.add:          ������豸��ϵͳ
- device.scan:         ɨ��/���־������豸
- device.remove:       ɾ��/�Ƴ��豸
- device.rename:       �������豸
- device.move:         ���豸�ƶ�����������
- device.query.status: ��ѯ�豸��ǰ״̬�����Ƿ��š��յ��¶��Ƕ��٣�

## �ռ������
- space.define:  ����/�����ҵĿռ�ṹ�����п��������ҡ��鷿��
- space.rename:  �����������ռ�
- space.query:   ��ѯ�ռ�ṹ��ĳ������������Щ�豸

## �û�������
- user.add:             ��Ӽ�ͥ��Ա
- user.remove:          ɾ����ͥ��Ա
- user.query:           ��ѯ��Ա�б��ĳ��Ա��Ϣ

## ϵͳ������
- config.skill.enable:  ����ĳ�� Skill ���
- config.skill.disable: ����ĳ�� Skill ���

## �Ի���
- chat.greeting: �ʺ򡢴��к�
- chat.help:     ѯ������ʲô��ʹ�ð���
- chat.confirm:  ȷ�ϲ������õġ�ȷ�ϡ��ǵġ�û���⣩
- chat.cancel:   ȡ�����������ˡ����ˡ�ȡ����

## �������
- ֻ��� JSON����Ҫ���κζ������ݡ�
- confidence ��ʾ��Ը���ͼ�жϵ����Ŷȣ���Χ 0.0�C1.0��
- entities ��ȡ����ͼ��صĹؼ�ʵ�壬�����ֶΣ�
  - device_name: �豸���ƣ���"̨��"��"�����յ�"��
  - action: ��������"on"��"off"��"set"��
  - value: Ŀ��ֵ�����¶�"26"������"50%"��
  - space_name: �ռ�/��������
  - member_name: ��ͥ��Ա����
  - workflow_name: ����������
  - skill_name: Skill ����
- �������ʵ���� entities Ϊ�ն��� {}��
- ���޷��ж���ͼ�����Ŷȼ��ͣ�intent �� "unknown"��

�����ʽ��
{
    "intent": "<intent_type>",
    "confidence": <0.0-1.0>,
    "entities": {}
}`

// llmClassifier implements IntentClassifier by calling a small language model.
type llmClassifier struct {
	provider  providers.LLMProvider
	cfg       *homeclawcfg.HomeclawConfig
	modelName string // resolved model identifier sent to the provider
}

// NewLLMClassifier creates an IntentClassifier that uses the given LLMProvider
// (expected to be a small / lightweight model) for intent recognition.
// modelName is the model identifier passed to provider.Chat().
func NewLLMClassifier(
	provider providers.LLMProvider,
	cfg *homeclawcfg.HomeclawConfig,
	modelName string,
) IntentClassifier {
	return &llmClassifier{
		provider:  provider,
		cfg:       cfg,
		modelName: modelName,
	}
}

// Classify sends the userInput to the small model and parses the JSON response.
// On error or low confidence it returns IntentUnknown so the agent loop can
// fall through to the large-model handler.
func (c *llmClassifier) Classify(ctx context.Context, userInput string) (IntentResult, error) {
	unknown := IntentResult{Type: IntentUnknown, Confidence: 0}

	if c.provider == nil {
		return unknown, nil
	}

	userMsg := fmt.Sprintf("�û�����: %s", userInput)

	messages := []providers.Message{
		{Role: "system", Content: intentClassifyPrompt},
		{Role: "user", Content: userMsg},
	}

	resp, err := c.provider.Chat(ctx, messages, nil, c.modelName, map[string]any{
		"max_tokens":  256,
		"temperature": 0.0,
	})
	if err != nil {
		// Degrade gracefully: classification failure falls through to large model.
		return unknown, fmt.Errorf("intent classifier: %w", err)
	}

	if resp == nil || len(resp.Content) == 0 {
		return unknown, nil
	}

	raw := extractJSON(resp.Content)
	if raw == "" {
		return unknown, fmt.Errorf("intent classifier: no JSON in response: %q", resp.Content)
	}

	var result IntentResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return unknown, fmt.Errorf("intent classifier: parse response: %w", err)
	}

	// Apply confidence threshold (hardcoded default).
	const threshold = 0.7
	if result.Confidence < threshold {
		return unknown, nil
	}

	if result.Type == "" {
		return unknown, nil
	}

	return result, nil
}

// extractJSON attempts to extract a JSON object from a larger string.
// The model sometimes wraps the JSON in markdown code fences.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Strip markdown code fences if present.
	if idx := strings.Index(s, "```"); idx >= 0 {
		s = s[idx+3:]
		if strings.HasPrefix(s, "json") {
			s = s[4:]
		}
		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
	}

	// Find the first '{' ... '}' block.
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < start {
		return ""
	}
	return strings.TrimSpace(s[start : end+1])
}
