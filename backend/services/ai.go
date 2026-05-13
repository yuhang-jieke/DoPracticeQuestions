package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
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
	QuestionTitle    string   `json:"question_title"`
	QuestionContent  string   `json:"question_content"`
	UserAnswer       string   `json:"user_answer"`
	QuestionType     string   `json:"question_type"`
	ReferenceAnswers []string `json:"reference_answers,omitempty"`
	PreviousScore    *float64 `json:"previous_score,omitempty"`
	PreviousAnswer   string   `json:"previous_answer,omitempty"`
	IsEdit           bool     `json:"is_edit"`
}

func NewAIConfig(apiKey, apiURL, model string, useMock bool) *AIConfig {
	return &AIConfig{
		APIKey:  apiKey,
		APIURL:  apiURL,
		Model:   model,
		UseMock: useMock,
	}
}

func (c *AIConfig) EvaluateAnswer(req *AIRequest) (*AIEvaluationResult, error) {
	if c.UseMock {
		return c.mockEvaluation(req)
	}
	return c.realEvaluation(req)
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

	if len(req.ReferenceAnswers) > 0 {
		prompt += "\n## 本题其他优秀用户的回答（供参考）：\n"
		for i, ans := range req.ReferenceAnswers {
			prompt += fmt.Sprintf("\n优秀回答 %d:\n%s\n", i+1, ans)
		}
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
	}

	body, _ := json.Marshal(chatReq)
	httpReq, _ := http.NewRequest("POST", c.APIURL, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("AI请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI接口返回错误 (状态码 %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("解析AI响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("AI无响应")
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

	if len(req.ReferenceAnswers) > 0 {
		analysis += " 参考了其他优秀回答者的思路，综合来看您在某些细节上还有提升空间。"
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
