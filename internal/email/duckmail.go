package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"reg_go/internal/storage"
)

// DuckMailConfig DuckMail 配置
type DuckMailConfig struct {
	Name   string `json:"name"`   // 配置名称（用户自定义）
	APIURL string `json:"apiUrl"` // API 基础 URL，例如 https://api.duckmail.sbs
	APIKey string `json:"apiKey"` // Bearer API Key（创建账号时使用）
	Domain string `json:"domain"` // 固定域名，例如 nsync1.133558.xyz
}

// DuckMailClient DuckMail REST 客户端（使用标准 net/http + 代理）
type DuckMailClient struct {
	config  DuckMailConfig
	client  *http.Client
}

// duckMailAccountResp 创建账号响应（mail.tm 格式）
type duckMailAccountResp struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// duckMailTokenResp 获取 token 响应
type duckMailTokenResp struct {
	Token string `json:"token"`
}

// duckMailMessagesResp 消息列表响应（hydra 格式）
type duckMailMessagesResp struct {
	Members []struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	} `json:"hydra:member"`
}

// duckMailMessageDetail 消息详情
type duckMailMessageDetail struct {
	ID   string      `json:"id"`
	Text string      `json:"text"`
	HTML interface{} `json:"html"`
}

// newDuckMailHTTPClient 创建携带代理配置的 HTTP 客户端
func newDuckMailHTTPClient() *http.Client {
	transport := &http.Transport{
		ResponseHeaderTimeout: 20 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
	}

	// 读取全局代理配置
	proxyURL := storage.GetProxy()
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   25 * time.Second,
	}
}

// NewDuckMailClient 创建 DuckMail 客户端
func NewDuckMailClient(config DuckMailConfig) *DuckMailClient {
	return &DuckMailClient{
		config: config,
		client: newDuckMailHTTPClient(),
	}
}

// generateDuckMailUsername 生成 12 位随机用户名
func generateDuckMailUsername() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// generateDuckMailPassword 生成 16 位随机密码
func generateDuckMailPassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// doRequest 发送 HTTP 请求（context-aware）
func (c *DuckMailClient) doRequest(ctx context.Context, method, path string, body interface{}, bearerToken string) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reqBody = bytes.NewReader(b)
	}

	endpoint := strings.TrimRight(c.config.APIURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	} else if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	return data, resp.StatusCode, err
}

// CreateAccount 在 DuckMail 创建临时邮箱，返回 (emailAddress, bearerToken, error)
func (c *DuckMailClient) CreateAccount(ctx context.Context) (string, string, error) {
	username := generateDuckMailUsername()
	address := fmt.Sprintf("%s@%s", username, c.config.Domain)
	password := generateDuckMailPassword()

	// Step 1: 注册账号
	payload1 := map[string]interface{}{
		"address":   address,
		"password":  password,
		"expiresIn": 3600,
	}
	data1, status1, err := c.doRequest(ctx, "POST", "/accounts", payload1, "")
	if err != nil {
		return "", "", fmt.Errorf("创建账号请求失败: %w", err)
	}
	if status1 != 200 && status1 != 201 {
		return "", "", fmt.Errorf("创建账号失败 (HTTP %d): %s", status1, string(data1))
	}

	// Step 2: 获取 token
	payload2 := map[string]interface{}{
		"address":  address,
		"password": password,
	}
	data2, status2, err := c.doRequest(ctx, "POST", "/token", payload2, "")
	if err != nil {
		return "", "", fmt.Errorf("获取 token 失败: %w", err)
	}
	if status2 != 200 {
		return "", "", fmt.Errorf("获取 token 失败 (HTTP %d): %s", status2, string(data2))
	}

	var tokenResp duckMailTokenResp
	if err := json.Unmarshal(data2, &tokenResp); err != nil {
		return "", "", fmt.Errorf("解析 token 响应失败: %w", err)
	}
	if tokenResp.Token == "" {
		return "", "", fmt.Errorf("token 为空，响应: %s", string(data2))
	}

	return address, tokenResp.Token, nil
}

// fetchMessages 获取收件箱消息列表
func (c *DuckMailClient) fetchMessages(ctx context.Context, bearerToken string) ([]struct {
	ID      string
	Subject string
}, error) {
	data, status, err := c.doRequest(ctx, "GET", "/messages", nil, bearerToken)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("获取消息列表失败 (HTTP %d)", status)
	}

	var result duckMailMessagesResp
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	out := make([]struct {
		ID      string
		Subject string
	}, len(result.Members))
	for i, m := range result.Members {
		out[i].ID = m.ID
		out[i].Subject = m.Subject
	}
	return out, nil
}

// fetchMessageDetail 获取单封邮件详情
func (c *DuckMailClient) fetchMessageDetail(ctx context.Context, msgID, bearerToken string) (*duckMailMessageDetail, error) {
	data, status, err := c.doRequest(ctx, "GET", "/messages/"+msgID, nil, bearerToken)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("获取消息详情失败 (HTTP %d)", status)
	}

	var detail duckMailMessageDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

// TestConnection 测试连接（创建一个临时账号）
func (c *DuckMailClient) TestConnection() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	addr, token, err := c.CreateAccount(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("[DuckMail] 连接测试成功，邮箱: %s (token 长度: %d)", addr, len(token))
	return []string{c.config.Domain}, nil
}

// DuckMailProvider 实现 TempEmailService 接口
type DuckMailProvider struct {
	client      *DuckMailClient
	address     string
	bearerToken string
	checkedIDs  map[string]bool
}

// NewDuckMailProvider 创建 DuckMail Provider（在 context 下创建账号）
func NewDuckMailProvider(config DuckMailConfig) (*DuckMailProvider, error) {
	client := NewDuckMailClient(config)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	addr, token, err := client.CreateAccount(ctx)
	if err != nil {
		return nil, fmt.Errorf("DuckMail 创建邮箱失败: %w", err)
	}
	log.Printf("[DuckMail] 邮箱创建完成: %s", addr)

	return &DuckMailProvider{
		client:      client,
		address:     addr,
		bearerToken: token,
		checkedIDs:  make(map[string]bool),
	}, nil
}

// GetAddress 返回邮箱地址
func (p *DuckMailProvider) GetAddress() string {
	return p.address
}

// WaitForCode 轮询等待 6 位数字验证码（context-aware，支持 Stop 取消）
func (p *DuckMailProvider) WaitForCode(timeoutSec, intervalSec int) (string, error) {
	if intervalSec <= 0 {
		intervalSec = 3
	}
	if timeoutSec <= 0 {
		timeoutSec = 120
	}

	codeRegex := regexp.MustCompile(`\b(\d{6})\b`)
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)

	attempt := 0
	for time.Now().Before(deadline) {
		attempt++

		// 用短 context 发起请求，这样取消信号能快速生效
		reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		messages, err := p.client.fetchMessages(reqCtx, p.bearerToken)
		cancel()

		if err != nil {
			if attempt%5 == 0 {
				log.Printf("[DuckMail] 获取消息失败: %v", err)
			}
		} else {
			for _, item := range messages {
				if p.checkedIDs[item.ID] {
					continue
				}
				p.checkedIDs[item.ID] = true

				// 从 subject 提取
				if code := extractCode6(item.Subject, codeRegex); code != "" {
					log.Printf("[DuckMail] 从主题提取到验证码: %s", code)
					return code, nil
				}

				// 获取详情
				detCtx, detCancel := context.WithTimeout(context.Background(), 10*time.Second)
				detail, detErr := p.client.fetchMessageDetail(detCtx, item.ID, p.bearerToken)
				detCancel()

				if detErr != nil {
					log.Printf("[DuckMail] 获取消息详情失败: %v", detErr)
					continue
				}

				// 从纯文本提取
				if code := extractCode6(detail.Text, codeRegex); code != "" {
					log.Printf("[DuckMail] 从正文提取到验证码: %s", code)
					return code, nil
				}

				// 从 HTML 提取
				htmlText := extractHTMLText(detail.HTML)
				if htmlText != "" {
					stripped := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(htmlText, " ")
					if code := extractCode6(stripped, codeRegex); code != "" {
						log.Printf("[DuckMail] 从 HTML 提取到验证码: %s", code)
						return code, nil
					}
				}
			}
		}

		if attempt%5 == 0 {
			remaining := int(time.Until(deadline).Seconds())
			log.Printf("[DuckMail] 等待验证码... (剩余 %ds)", remaining)
		}

		// 可中断的 sleep
		select {
		case <-time.After(time.Duration(intervalSec) * time.Second):
		}
	}

	return "", fmt.Errorf("等待 DuckMail 验证码超时 (%ds)", timeoutSec)
}

// extractCode6 从文本中提取 6 位数字
func extractCode6(text string, re *regexp.Regexp) string {
	m := re.FindStringSubmatch(text)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// extractHTMLText 从 HTML 字段（string 或 []interface{}）提取文字
func extractHTMLText(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []interface{}:
		parts := make([]string, 0, len(val))
		for _, p := range val {
			if s, ok := p.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}
