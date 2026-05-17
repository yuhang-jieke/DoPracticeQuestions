package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type AIConfig struct {
	APIKey  string
	APIURL  string
	Model   string
	UseMock bool
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	MaxTokens      int             `json:"max_tokens"`
	Temperature    float64         `json:"temperature"`
	Stream         bool            `json:"stream"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type AIEvaluationResult struct {
	Score        float64 `json:"score"`
	Analysis     string  `json:"analysis"`
	Strengths    string  `json:"strengths"`
	Weaknesses   string  `json:"weaknesses"`
	Reference    string  `json:"reference_answer"`
	Improvements string  `json:"improvements"`
}

type AIRequest struct {
	QuestionTitle   string   `json:"question_title"`
	QuestionContent string   `json:"question_content"`
	UserAnswer      string   `json:"user_answer"`
	QuestionType    string   `json:"question_type"`
	PreviousScore   *float64 `json:"previous_score,omitempty"`
	PreviousAnswer  string   `json:"previous_answer,omitempty"`
	IsEdit          bool     `json:"is_edit"`
}

type ClassifyResult struct {
	Category  string `json:"category"`
	Status    string `json:"status"`
	Rewritten string `json:"rewritten"`
	Reason    string `json:"reason"`
}

func NewAIConfig(apiKey, apiURL, model string, useMock bool) *AIConfig {
	return &AIConfig{
		APIKey:  apiKey,
		APIURL:  apiURL,
		Model:   model,
		UseMock: useMock,
	}
}

func (c *AIConfig) TestConnection() error {
	chatReq := ChatRequest{
		Model:       c.Model,
		Messages:    []Message{{Role: "user", Content: "hi"}},
		MaxTokens:   50,
		Temperature: 0,
		Stream:      false,
	}
	body, _ := json.Marshal(chatReq)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.APIURL, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("无法连接到 API 地址: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	ct := resp.Header.Get("Content-Type")

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("API Key 无效或无权访问 (状态码 %d)", resp.StatusCode)
	}

	if strings.Contains(ct, "text/html") {
		return fmt.Errorf("API 地址返回了网页而非 API 数据 (Content-Type: %s)，请确认填写的 URL 是完整的 API 端点", ct)
	}

	if resp.StatusCode != http.StatusOK {
		bodyStr := string(respBody)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		return fmt.Errorf("API 返回错误 (状态码 %d): %s", resp.StatusCode, bodyStr)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		log.Printf("TestConnection JSON解析失败 | Content-Type: %s | Body: %s", ct, string(respBody))
		return fmt.Errorf("API 返回了非 JSON 数据，请确认填写的 URL 是正确的 chat completions 端点")
	}

	if len(chatResp.Choices) == 0 {
		return fmt.Errorf("API 返回了空的响应，请检查模型名称 '%s' 是否在该服务商中可用", c.Model)
	}

	return nil
}

func (c *AIConfig) EvaluateAnswer(req *AIRequest) (*AIEvaluationResult, error) {
	if c.UseMock {
		return c.mockEvaluation(req)
	}
	var result AIEvaluationResult
	err := c.EvaluateAnswerStream(req, func(ev StreamEvent) error {
		switch ev.Type {
		case "score":
			if m, ok := ev.Data.(map[string]interface{}); ok {
				if s, ok := m["score"].(float64); ok {
					result.Score = s
				}
			}
		case "chunk":
			if m, ok := ev.Data.(map[string]interface{}); ok {
				f, _ := m["field"].(string)
				t, _ := m["text"].(string)
				switch f {
				case "analysis":
					result.Analysis += t
				case "strengths":
					result.Strengths += t
				case "weaknesses":
					result.Weaknesses += t
				case "improvements":
					result.Improvements += t
				case "reference":
					result.Reference += t
				}
			}
		}
		return nil
	})
	return &result, err
}

func (c *AIConfig) EvaluateRaw(prompt string) (string, error) {
	chatReq := ChatRequest{
		Model:       c.Model,
		Messages:    []Message{{Role: "system", Content: "你是一个专业的教学分析助手。"}, {Role: "user", Content: prompt}},
		MaxTokens:   2048,
		Temperature: 0.7,
		Stream:      false,
	}
	body, _ := json.Marshal(chatReq)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.APIURL, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("AI请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	ct := resp.Header.Get("Content-Type")

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI接口返回错误 (状态码 %d): %s", resp.StatusCode, string(respBody))
	}

	if strings.Contains(ct, "text/html") {
		return "", fmt.Errorf("AI接口返回了网页而非API数据，请检查API地址配置")
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		log.Printf("解析AI响应失败 | URL: %s | Content-Type: %s | Body前200字符: %.200s", c.APIURL, ct, string(respBody))
		return "", fmt.Errorf("AI接口返回了非预期的数据格式")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (c *AIConfig) ClassifyQuestion(title, tags string, categories []string) (*ClassifyResult, error) {
	prompt := "你是题库审核助手。以下是已有分类列表：\n" + strings.Join(categories, "、") + "\n\n"
	prompt += `请对以下内容做三件事：分类、判断是否为面试/技术题目、必要时改写。

## 判断标准：
- valid：是正经面试/技术题，表述清晰完整
- invalid：跟面试/技术完全无关（闲聊、灌水、无意义内容），在reason字段说明原因
- rewritten：有面试/技术意图但表述太简陋，请帮忙扩写成一条规范完整的面试题

## 改写要求：
- 保留原意图和核心问题
- 补充必要的上下文和技术深度
- 控制在一段话以内（不超过200字）

请严格以JSON格式返回：
{
  "category": "已有分类之一，匹配不到则填'综合技术'",
  "status": "valid 或 invalid 或 rewritten",
  "rewritten": "改写后内容，status为valid或invalid时留空",
  "reason": "invalid时填写原因，其他状态留空"
}

题目内容：` + title + "\n"
	if tags != "" {
		prompt += "技术点：" + tags + "\n"
	}
	raw, err := c.EvaluateRaw(prompt)
	if err != nil {
		return nil, err
	}
	var result ClassifyResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		cleaned := raw
		if start := strings.Index(raw, "{"); start >= 0 {
			if end := strings.LastIndex(raw, "}"); end > start {
				cleaned = raw[start : end+1]
			}
		}
		if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
			return &ClassifyResult{Category: "综合技术", Status: "valid"}, nil
		}
	}
	return &result, nil
}

func (c *AIConfig) mockEvaluation(req *AIRequest) (*AIEvaluationResult, error) {
	score := 7.5
	if req.QuestionType == "non-tech" {
		score = 6.5
	}

	var analysis string
	if req.IsEdit {
		analysis = fmt.Sprintf("这是您编辑后的版本。与之前版本（%.1f分）相比，本次回答在结构上有所改进，但在关键点的阐述上还可以更加深入。", *req.PreviousScore)
	} else {
		analysis = "您的回答整体思路清晰，覆盖了主要知识点。建议在深度和具体实例方面进一步丰富。"
	}

	return &AIEvaluationResult{
		Score:        score,
		Analysis:     analysis,
		Strengths:    "回答结构清晰，重点突出，表达流畅。",
		Weaknesses:   "可以在深度上进一步加强，建议结合实际案例来说明。",
		Reference:    "（mock模式）这是AI综合多位优秀回答生成的参考答案，建议从原理、应用场景、优缺点三个维度展开回答。",
		Improvements: "建议补充具体的代码示例/案例分析，并注意回答的逻辑递进关系。",
	}, nil
}

type StreamEvent struct {
	Type string
	Data interface{}
}

type StreamCallback func(event StreamEvent) error

func (c *AIConfig) EvaluateAnswerStream(req *AIRequest, cb StreamCallback) error {
	if c.UseMock {
		return c.mockStreamEvaluation(req, cb)
	}
	return c.realStreamEvaluation(req, cb)
}

func (c *AIConfig) realStreamEvaluation(req *AIRequest, cb StreamCallback) error {
	prompt := buildStreamPrompt(req)
	chatReq := ChatRequest{
		Model:          c.Model,
		Messages:       []Message{{Role: "user", Content: prompt}},
		MaxTokens:      4096,
		Temperature:    0.3,
		Stream:         true,
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}
	body, _ := json.Marshal(chatReq)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.APIURL, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("AI请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("AI接口返回错误 (状态码 %d): %s", resp.StatusCode, string(respBody))
	}

	var fullText strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4096), 256*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		dataStr := strings.TrimPrefix(line, "data: ")
		if dataStr == "[DONE]" {
			break
		}
		var streamResp struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(dataStr), &streamResp); err != nil {
			continue
		}
		if len(streamResp.Choices) == 0 {
			continue
		}
		chunk := streamResp.Choices[0].Delta.Content
		if chunk == "" {
			continue
		}
		fullText.WriteString(chunk)
	}

	raw := fullText.String()
	result, err := parseEvaluationJSON(raw)
	if err != nil {
		log.Printf("AI评估JSON解析失败: %v | 原始输出: %.300s", err, raw)
		return fmt.Errorf("AI返回了非预期的数据格式，请重试")
	}

	result.Analysis = stripTitlePrefix(result.Analysis)
	result.Strengths = reorderItems(stripTitlePrefix(result.Strengths))
	result.Weaknesses = reorderItems(stripTitlePrefix(result.Weaknesses))
	result.Improvements = reorderItems(stripTitlePrefix(result.Improvements))
	result.Reference = stripTitlePrefix(result.Reference)

	return emitJSONResult(result, cb)
}

func (c *AIConfig) mockStreamEvaluation(req *AIRequest, cb StreamCallback) error {
	score := 7.5
	if req.QuestionType == "non-tech" {
		score = 6.5
	}

	var analysis string
	if req.IsEdit {
		analysis = fmt.Sprintf("这是您编辑后的版本。与之前版本（%.1f分）相比，本次回答在结构上有所改进，但在关键点的阐述上还可以更加深入。", *req.PreviousScore)
	} else {
		analysis = "您的回答整体思路清晰，覆盖了主要知识点。建议在深度和具体实例方面进一步丰富。"
	}

	result := &AIEvaluationResult{
		Score:        score,
		Analysis:     analysis,
		Strengths:    "1. 准确性高：概念定义准确\n2. 深度突出：延伸底层原理\n3. 广度优秀：覆盖多场景\n4. 结构化良好：逻辑递进清晰",
		Weaknesses:   "1. 部分描述偏理论，缺少举例\n2. 未提及性能隐患与并发风险\n3. 高阶场景内容较概括",
		Improvements: "1. 补充贴合业务的实操案例\n2. 补充机制存在的性能问题与异常风险\n3. 完善高阶场景的具体实现细节",
		Reference:    "这是AI综合多位优秀回答生成的参考答案，建议从原理、应用场景、优缺点三个维度展开回答。",
	}

	return emitJSONResult(result, cb)
}

// ====================== 修复格式问题的核心：提示词 ======================
func buildStreamPrompt(req *AIRequest) string {
	techType := "技术"
	if req.QuestionType == "non-tech" {
		techType = "非技术"
	}

	var sb strings.Builder
	sb.WriteString("你是资深专业面试官，客观公正评估用户面试回答，**严格只输出纯标准JSON格式**，严禁输出JSON之外任何多余内容、注释、解释、markdown代码块、表情符号、自定义标题。\n\n")

	// 定义固定JSON结构，不约束条目数量
	sb.WriteString(`输出JSON结构严格如下：
{
  "score": 分数(1-10区间，保留一位小数),
  "analysis": "一句话综合评价，概括回答整体水平、准确性与完整度",
  "strengths": "根据回答实际亮点自由撰写，有几条写几条，无需凑数，每条单独换行，以数字1.开头独立编号",
  "weaknesses": "根据回答真实存在问题如实撰写，问题多则多写，少则少写，不强行凑条数，每条单独换行，以数字1.开头独立编号",
  "improvements": "严格对应不足逐条给出改进方案，有几条不足就写几条建议，逐条匹配，每条单独换行，以数字1.开头独立编号",
  "reference_answer": "贴合题目考点撰写标准面试回答，篇幅自由，简单题精简、复杂题详实，适合口头作答"
}
`)

	// 核心强制规则
	sb.WriteString("\n执行硬性规则：\n")
	sb.WriteString("1. strengths、weaknesses、improvements 三者互相独立，各自编号均从1重新开始，禁止跨列表延续序号\n")
	sb.WriteString("2. 所有字段内容内，**绝对禁止出现：综合分析、优点、不足、改进建议、参考答案**这类标题文字\n")
	sb.WriteString("3. 不固定任何条目数量，完全依据用户回答质量自适应生成内容条数\n")
	sb.WriteString("4. 内容禁止使用JSON非法特殊字符，标点统一使用中文标点\n")
	sb.WriteString("5. 仅返回纯净JSON字符串，无任何前缀后缀多余文字\n\n")

	// 传入答题上下文
	sb.WriteString(fmt.Sprintf("面试题目：%s\n题目详情：%s\n题目类型：%s\n考生回答内容：%s\n\n",
		req.QuestionTitle, req.QuestionContent, techType, req.UserAnswer))

	// 编辑对比内容
	if req.IsEdit && req.PreviousAnswer != "" {
		sb.WriteString(fmt.Sprintf("考生上一轮作答内容：%s\n\n", req.PreviousAnswer))
		if req.PreviousScore != nil {
			sb.WriteString(fmt.Sprintf("上一轮评估得分：%.1f/10，请对比两次作答差异优化评估\n\n", *req.PreviousScore))
		}
	}

	return sb.String()
}

// ======================================================================

func parseEvaluationJSON(raw string) (*AIEvaluationResult, error) {
	var result AIEvaluationResult

	raw = strings.TrimSpace(raw)
	if start := strings.Index(raw, "{"); start != -1 {
		if end := strings.LastIndex(raw, "}"); end != -1 && end > start {
			raw = raw[start : end+1]
		}
	}

	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func reorderItems(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	var items []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		content := trimmed
		for i, r := range trimmed {
			if r >= '0' && r <= '9' || r == ' ' || r == '.' || r == '、' || r == ')' || r == '-' {
				continue
			}
			content = trimmed[i:]
			break
		}
		items = append(items, strings.TrimSpace(content))
	}

	var sb strings.Builder
	for i, item := range items {
		if item == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%d. %s", i+1, item))
		if i < len(items)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

var titlePrefixes = []string{
	"综合分析", "✅ 优点", "✅优点", "📌 不足", "📌不足",
	"💡 改进建议", "💡改进建议", "💡 改进", "💡改进",
	"📝 参考答案", "📝参考答案", "得分", "合格",
	"：", ":",
}

func stripTitlePrefix(text string) string {
	trimmed := strings.TrimSpace(text)
	for _, prefix := range titlePrefixes {
		trimmed = strings.TrimPrefix(trimmed, prefix)
	}
	return strings.TrimSpace(trimmed)
}

func emitJSONResult(result *AIEvaluationResult, cb StreamCallback) error {
	isQualified := result.Score >= 7

	if err := cb(StreamEvent{
		Type: "score",
		Data: map[string]interface{}{"score": result.Score, "is_qualified": isQualified},
	}); err != nil {
		return err
	}

	sections := []struct {
		field string
		text  string
	}{
		{"analysis", result.Analysis + "\n"},
		{"strengths", result.Strengths + "\n"},
		{"weaknesses", result.Weaknesses + "\n"},
		{"improvements", result.Improvements + "\n"},
		{"reference", result.Reference + "\n"},
	}

	for _, sec := range sections {
		runes := []rune(sec.text)
		for i := 0; i < len(runes); i += 3 {
			end := i + 3
			if end > len(runes) {
				end = len(runes)
			}
			ev := StreamEvent{
				Type: "chunk",
				Data: map[string]interface{}{"field": sec.field, "text": string(runes[i:end])},
			}
			if err := cb(ev); err != nil {
				return err
			}
		}
	}

	return cb(StreamEvent{Type: "done", Data: nil})
}
