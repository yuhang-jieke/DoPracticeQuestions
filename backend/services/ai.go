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
	"regexp"
	"strconv"
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

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	Stream      bool      `json:"stream"`
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
	return c.realEvaluation(req)
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
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("AI无响应")
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

func buildPrompt(req *AIRequest) string {
	techType := "技术"
	if req.QuestionType == "non-tech" {
		techType = "非技术"
	}

	prompt := fmt.Sprintf(`你是一个专业的面试官，请评估以下面试回答。

## 题目信息
- 题目：%s
- 详细内容：%s
- 题目类型：%s

## 用户的回答：
%s
`,
		req.QuestionTitle, req.QuestionContent, techType, req.UserAnswer)

	if req.IsEdit && req.PreviousAnswer != "" {
		prompt += fmt.Sprintf(`
## 用户之前的回答（编辑前的版本）：
%s
`, req.PreviousAnswer)
		if req.PreviousScore != nil {
			prompt += fmt.Sprintf("## 用户之前的评分：%.1f/10\n", *req.PreviousScore)
		}
		prompt += `
请注意比较两次回答的差异，如果这次回答不如之前，要明确指出哪些方面退步了。
`
	}

	if req.QuestionType == "non-tech" {
		prompt += `
请按照企业级面试标准进行评估，侧重以下维度：
1. 完整性：回答是否全面覆盖了问题要点
2. 逻辑性：表达是否清晰有条理
3. 说服力：回答是否有说服力
4. 专业性：是否使用了恰当的职场表达
5. STAR法则运用：情境、任务、行动、结果是否清晰

请给出具体的面试回答建议和范例话术。
`
	} else {
		prompt += `
请按照技术面试标准进行评估，侧重以下维度：
1. 准确性：技术点是否正确
2. 深度：理解是否深入
3. 广度：是否考虑了相关知识点
4. 结构化：表达是否清晰有条理

综合分析多位优秀回答后，给出一个高质量的参考答案。
`
	}

	prompt += `
请以JSON格式返回评估结果，格式严格如下：
{
  "score": <1-10的整数或小数>,
  "analysis": "综合分析评价",
  "strengths": "回答的优点",
  "weaknesses": "回答的不足和改进建议",
  "reference_answer": "综合优秀回答给出的参考答案",
  "improvements": "具体的改进建议"
}

评分标准参考：
- 9-10分：回答极其出色，深入浅出，结构完美，有独到见解
- 7-8分：回答优秀，内容准确全面，条理清晰
- 5-6分：回答基本正确，覆盖了核心知识点，但深度或广度不足
- 3-4分：回答有部分正确内容，但存在明显不足或缺失
- 1-2分：回答偏离题目或内容非常有限

注意：7分代表"良好合格的回答"，符合面试基本要求即可给予7分以上。请严格输出JSON，不要带markdown代码块标记。
`
	return prompt
}

func (c *AIConfig) realEvaluation(req *AIRequest) (*AIEvaluationResult, error) {
	prompt := buildPrompt(req)

	chatReq := ChatRequest{
		Model: c.Model,
		Messages: []Message{
			{Role: "system", Content: "你是一个专业的面试评估助手，擅长评估面试回答质量并给出建设性建议。"},
			{Role: "user", Content: prompt},
		},
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
		return nil, fmt.Errorf("AI请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	ct := resp.Header.Get("Content-Type")

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI接口返回错误 (状态码 %d): %s", resp.StatusCode, string(respBody))
	}

	if strings.Contains(ct, "text/html") {
		return nil, fmt.Errorf("AI接口返回了网页而非API数据，请检查API地址配置")
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		log.Printf("解析AI响应失败 | URL: %s | Content-Type: %s | Body前200字符: %.200s", c.APIURL, ct, string(respBody))
		return nil, fmt.Errorf("AI接口返回了非预期的数据格式")
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("AI无响应，请检查模型名称 '%s' 是否在该服务商中可用", c.Model)
	}

	return parseEvaluation(chatResp.Choices[0].Message.Content)
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

func parseEvaluation(content string) (*AIEvaluationResult, error) {
	var result AIEvaluationResult
	if err := json.Unmarshal([]byte(content), &result); err == nil {
		return &result, nil
	}

	cleaned := content
	if len(content) > 7 {
		start := -1
		end := -1
		for i := 0; i < len(content)-3; i++ {
			if content[i] == '{' {
				start = i
				break
			}
		}
		for i := len(content) - 1; i >= 0; i-- {
			if content[i] == '}' {
				end = i + 1
				break
			}
		}
		if start >= 0 && end > start {
			cleaned = content[start:end]
		}
	}

	if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
		return &result, nil
	}

	return &AIEvaluationResult{
		Score:      5.0,
		Analysis:   "AI评估完成",
		Strengths:  "请参考详细分析",
		Weaknesses: "请参考详细分析",
		Reference:  "参考答案生成中...",
	}, nil
}

// ──────────── streaming evaluation ────────────

type StreamEvent struct {
	Type string
	Data interface{}
}

type StreamCallback func(event StreamEvent) error

type streamParser struct {
	fullText    strings.Builder
	sentLen     int
	currSection string
	scoreSent   bool
	score       float64
	isQualified bool
}

var (
	scoreLineRe = regexp.MustCompile(`得分[：:][^\n]*|合格[：:][^\n]*`)
	sectionRe   = regexp.MustCompile(`✅\s*优点[：:]?\s*|📌\s*不足[：:]?\s*|💡\s*改进建议[：:]?\s*|💡\s*改进[：:]?\s*|综合分析[：:]?\s*|参考答案[：:]?\s*|##(?:✅|📌|💡)\S*|\s*##\s*|#{1,3}\s*(?:✅|📌|💡)?\s*(?:综合分析|优点|不足|改进建议|参考答案)[：:]?\s*|[一二三四五]、\s*(?:✅|📌|💡)?\s*(?:综合分析|优点|不足|改进建议|参考答案)[：:]?\s*|✅|📌|💡|优点\d+\.?\s*|不足\d+\.?\s*|建议\d+\.?\s*`)
	scoreRe     = regexp.MustCompile(`得分[：:]\s*(\d+(?:\.\d+)?)`)
	cleanHashRe = regexp.MustCompile(`(^|\n)\s*##\s*`)
)

func filterMarkers(text string) string {
	text = scoreLineRe.ReplaceAllString(text, "")
	text = sectionRe.ReplaceAllString(text, "")
	text = cleanHashRe.ReplaceAllString(text, "$1")
	return strings.TrimSpace(text)
}

func extractScore(text string) (float64, bool) {
	m := scoreRe.FindStringSubmatch(text)
	if m == nil {
		return 0, false
	}
	s, err := strconv.ParseFloat(m[1], 64)
	return s, err == nil
}

func lastSectionHead(text string) string {
	markers := []struct {
		key   string
		field string
	}{
		{"综合分析", "analysis"},
		{"✅ 优点", "strengths"}, {"优点", "strengths"},
		{"📌 不足", "weaknesses"}, {"不足", "weaknesses"},
		{"💡 改进建议", "improvements"}, {"改进建议", "improvements"},
		{"参考答案", "reference"},
	}
	last := ""
	lastIdx := -1
	for _, m := range markers {
		if idx := strings.LastIndex(text, m.key); idx > lastIdx {
			lastIdx = idx
			last = m.field
		}
	}
	return last
}

func (p *streamParser) feed(chunk string) []StreamEvent {
	p.fullText.WriteString(chunk)
	text := p.fullText.String()
	var events []StreamEvent

	if !p.scoreSent {
		if s, ok := extractScore(text); ok {
			p.score = s
			p.isQualified = s >= 7
			p.scoreSent = true
			events = append(events, StreamEvent{
				Type: "score",
				Data: map[string]interface{}{"score": s, "is_qualified": p.isQualified},
			})
		}
	}

	lastHead := lastSectionHead(text)
	if lastHead != "" {
		p.currSection = lastHead
	}

	lastNL := strings.LastIndex(text, "\n")
	if lastNL < 0 || lastNL < p.sentLen {
		return events
	}
	end := lastNL + 1
	raw := text[p.sentLen:end]
	p.sentLen = end

	filtered := filterMarkers(raw)
	if filtered != "" {
		events = append(events, StreamEvent{
			Type: "chunk",
			Data: map[string]interface{}{"field": p.currSection, "text": filtered},
		})
	}

	return events
}

func (p *streamParser) done() StreamEvent {
	return StreamEvent{
		Type: "done",
		Data: map[string]interface{}{
			"score":        p.score,
			"is_qualified": p.isQualified,
		},
	}
}

func (c *AIConfig) EvaluateAnswerStream(req *AIRequest, cb StreamCallback) error {
	if c.UseMock {
		return c.mockStreamEvaluation(req, cb)
	}
	return c.realStreamEvaluation(req, cb)
}

func (c *AIConfig) realStreamEvaluation(req *AIRequest, cb StreamCallback) error {
	prompt := buildStreamPrompt(req)
	chatReq := ChatRequest{
		Model:       c.Model,
		Messages:    []Message{{Role: "system", Content: "你是一个专业的面试评估助手，擅长评估面试回答质量并给出建设性建议。"}, {Role: "user", Content: prompt}},
		MaxTokens:   4096,
		Temperature: 0.7,
		Stream:      true,
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

	parser := &streamParser{currSection: "analysis"}
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
		for _, ev := range parser.feed(chunk) {
			if err := cb(ev); err != nil {
				return err
			}
		}
	}

	// Flush buffered text then signal done
	for _, ev := range parser.feed("\n") {
		if err := cb(ev); err != nil {
			return err
		}
	}
	if err := cb(parser.done()); err != nil {
		return err
	}
	return nil
}

func (c *AIConfig) mockStreamEvaluation(req *AIRequest, cb StreamCallback) error {
	score := 7.5
	if req.QuestionType == "non-tech" {
		score = 6.5
	}

	sections := []struct {
		field string
		text  string
	}{
		{"score", fmt.Sprintf("得分：%.1f\n合格：%s\n", score, map[bool]string{true: "是", false: "否"}[score >= 7])},
		{"analysis", "综合分析\n您的回答整体思路清晰，覆盖了主要知识点，对比往期回答表现稳定。\n\n"},
		{"strengths", "✅ 优点\n1. 准确性高：概念定义准确，核心特性无错误\n2. 深度突出：延伸底层原理与运行机制\n3. 广度优秀：覆盖多场景与业务选型\n4. 结构化良好：逻辑递进清晰\n\n"},
		{"weaknesses", "📌 不足\n1. 部分描述偏理论，缺少举例\n2. 未提及性能隐患与并发风险\n3. 高阶场景内容较概括\n\n"},
		{"improvements", "💡 改进建议\n1. 补充1-2个贴合业务的实操案例\n2. 补充机制存在的性能问题与异常风险\n3. 完善高阶场景的具体实现细节\n\n"},
		{"reference", "参考答案\n这是AI综合多位优秀回答生成的参考答案，建议从原理、应用场景、优缺点三个维度展开。\n"},
	}

	for _, sec := range sections {
		if sec.field == "score" {
			ev := StreamEvent{Type: "score", Data: map[string]interface{}{"score": score, "is_qualified": score >= 7}}
			if err := cb(ev); err != nil {
				return err
			}
			continue
		}
		runes := []rune(sec.text)
		for i := 0; i < len(runes); i += 3 {
			end := i + 3
			if end > len(runes) {
				end = len(runes)
			}
			chunk := string(runes[i:end])
			ev := StreamEvent{Type: "chunk", Data: map[string]interface{}{"field": sec.field, "text": chunk}}
			if err := cb(ev); err != nil {
				return err
			}
			time.Sleep(50 * time.Millisecond)
		}
	}

	return cb(StreamEvent{Type: "done", Data: map[string]interface{}{"score": score, "is_qualified": score >= 7}})
}

func buildStreamPrompt(req *AIRequest) string {
	techType := "技术"
	if req.QuestionType == "non-tech" {
		techType = "非技术"
	}

	var sb strings.Builder
	sb.WriteString("你是专业的面试评估助手。请评估以下面试回答。\n\n")
	sb.WriteString(fmt.Sprintf("## 题目\n%s\n\n", req.QuestionTitle))
	sb.WriteString(fmt.Sprintf("## 题目详情\n%s\n\n", req.QuestionContent))
	sb.WriteString(fmt.Sprintf("## 题目类型\n%s\n\n", techType))
	sb.WriteString(fmt.Sprintf("## 用户回答\n%s\n\n", req.UserAnswer))

	if req.IsEdit && req.PreviousAnswer != "" {
		sb.WriteString(fmt.Sprintf("## 用户之前的回答\n%s\n\n", req.PreviousAnswer))
		if req.PreviousScore != nil {
			sb.WriteString(fmt.Sprintf("之前评分：%.1f/10。请注意对比两次差异。\n\n", *req.PreviousScore))
		}
	}

	evalDims := "准确性、深度、广度、结构化"
	if req.QuestionType == "non-tech" {
		evalDims = "完整性、逻辑性、说服力、专业性、STAR法则运用"
	}

	sb.WriteString(fmt.Sprintf("评估维度：%s。\n\n", evalDims))
	sb.WriteString("请严格按以下格式输出（不要输出JSON，禁止使用Markdown标记如##）：\n\n")
	sb.WriteString("得分：<只写一个1-10的数字，不要写/10>\n")
	sb.WriteString("合格：<只写是或否>\n\n")
	sb.WriteString("综合分析\n<整体评价，直接写正文>\n\n")
	sb.WriteString("✅ 优点\n1. xxx（每条换行，数字前不写任何文字）\n2. xxx\n3. xxx\n4. xxx\n\n")
	sb.WriteString("📌 不足\n1. xxx\n2. xxx\n3. xxx\n\n")
	sb.WriteString("💡 改进建议\n1. xxx\n2. xxx\n3. xxx\n\n")
	sb.WriteString("参考答案\n<直接写内容>\n\n")
	sb.WriteString("严格禁止：\n- 段落间或末尾不要加单独的✅📌💡\n- 编号仅写\"1.\"\"2.\"格式，前面不许加任何文字（禁止写\"优点1.\"\"不足2.\"\"建议3.\"）\n- 每个编号项占独立一行\n- 任何位置不写##\n- 标题不加冒号\n- 得分只写数字禁止写\"9.0/10\"")

	return sb.String()
}
