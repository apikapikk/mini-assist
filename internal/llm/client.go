package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// loadDotEnv memuat file .env ke environment variables jika ada
func loadDotEnv(paths ...string) {
	if len(paths) == 0 {
		paths = []string{".env", "../.env", "../../.env"}
	}
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				val = strings.Trim(val, `"'`)
				if os.Getenv(key) == "" {
					os.Setenv(key, val)
				}
			}
		}
		return
	}
}

// Config konfigurasi client LLM
type Config struct {
	APIKey string
	Model  string // default: gemini-2.5-flash atau gemini-1.5-flash
}

// Client pembungkus panggilan API LLM
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient membuat instance LLM Client baru
func NewClient(cfg Config) *Client {
	loadDotEnv()

	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("LLM_API_KEY")
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("GEMINI_API_KEY")
		}
	}
	if cfg.Model == "" {
		cfg.Model = os.Getenv("LLM_MODEL")
		if cfg.Model == "" {
			cfg.Model = "gemini-2.5-flash"
		}
	}

	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ChatMessage merepresentasikan satu pesan dalam riwayat obrolan
type ChatMessage struct {
	Role string `json:"role"` // "user" atau "model"
	Text string `json:"text"`
}

// ChatParams parameter untuk obrolan multi-turn
type ChatParams struct {
	SystemPrompt string
	Messages     []ChatMessage
	Temperature  float64
}

// PromptParams parameter untuk generate teks satu kali (single-turn)
type PromptParams struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	SystemInstruction *geminiContent         `json:"system_instruction,omitempty"`
	Contents          []geminiContent        `json:"contents"`
	GenerationConfig  map[string]interface{} `json:"generationConfig,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

type geminiRateLimitDetail struct {
	Type       string `json:"@type"`
	RetryDelay string `json:"retryDelay"`
	Violations []struct {
		QuotaMetric string `json:"quotaMetric"`
		QuotaID     string `json:"quotaId"`
	} `json:"violations"`
}

type geminiErrorBody struct {
	Error struct {
		Code    int                     `json:"code"`
		Message string                  `json:"message"`
		Status  string                  `json:"status"`
		Details []geminiRateLimitDetail `json:"details"`
	} `json:"error"`
}

func parseRateLimitInfo(bodyBytes []byte) (isDaily bool, delay time.Duration, infoMsg string) {
	var eb geminiErrorBody
	_ = json.Unmarshal(bodyBytes, &eb)

	msgLower := strings.ToLower(eb.Error.Message)
	isDaily = strings.Contains(msgLower, "perday") || strings.Contains(msgLower, "per day") || strings.Contains(msgLower, "daily")

	delay = 15 * time.Second
	for _, d := range eb.Error.Details {
		for _, v := range d.Violations {
			vLower := strings.ToLower(v.QuotaID + " " + v.QuotaMetric)
			if strings.Contains(vLower, "perday") || strings.Contains(vLower, "day") {
				isDaily = true
			}
		}
		if d.RetryDelay != "" {
			if parsed, err := time.ParseDuration(d.RetryDelay); err == nil && parsed > 0 {
				delay = parsed
			}
		}
	}

	if isDaily {
		infoMsg = "Batas kuota harian (RPD) untuk model ini telah habis. Kuota free tier di-reset pukul 07:00 WIB besok, atau kamu bisa mengganti model ke 'gemini-1.5-flash' di .env."
	} else {
		infoMsg = fmt.Sprintf("Batas laju per menit (RPM) tercapai. Menunggu %.0f detik...", delay.Seconds())
	}
	return isDaily, delay, infoMsg
}

// GenerateText memanggil Gemini API untuk menghasilkan teks satu kali
func (c *Client) GenerateText(ctx context.Context, params PromptParams) (string, error) {
	return c.Chat(ctx, ChatParams{
		SystemPrompt: params.SystemPrompt,
		Messages: []ChatMessage{
			{Role: "user", Text: params.UserPrompt},
		},
		Temperature: params.Temperature,
	})
}

// Chat memanggil Gemini API dengan riwayat percakapan multi-turn
func (c *Client) Chat(ctx context.Context, params ChatParams) (string, error) {
	if c.cfg.APIKey == "" {
		return "", fmt.Errorf("API key Gemini belum dikonfigurasi (set LLM_API_KEY atau GEMINI_API_KEY di .env)")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		c.cfg.Model, c.cfg.APIKey)

	var contents []geminiContent
	for _, m := range params.Messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Text}},
		})
	}

	reqBody := geminiRequest{
		Contents: contents,
		GenerationConfig: map[string]interface{}{
			"temperature":     params.Temperature,
			"maxOutputTokens": 300,
		},
	}

	isGemma := strings.Contains(strings.ToLower(c.cfg.Model), "gemma")
	if params.SystemPrompt != "" {
		if isGemma {
			if len(contents) > 0 {
				contents[0].Parts[0].Text = params.SystemPrompt + "\n\n" + contents[0].Parts[0].Text
			} else {
				contents = append(contents, geminiContent{
					Role:  "user",
					Parts: []geminiPart{{Text: params.SystemPrompt}},
				})
			}
		} else {
			reqBody.SystemInstruction = &geminiContent{
				Parts: []geminiPart{{Text: params.SystemPrompt}},
			}
		}
	}

	reqBody.Contents = contents

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("gagal marshal request: %w", err)
	}

	maxRetries := 2
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
		if err != nil {
			return "", fmt.Errorf("gagal membuat http request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("gagal memanggil Gemini API: %w", err)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("gagal membaca respon API: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			isDaily, delay, infoMsg := parseRateLimitInfo(bodyBytes)
			if isDaily {
				return "", fmt.Errorf("%s", infoMsg)
			}

			// Informasikan ke terminal agar pengguna tidak bingung menunggu
			fmt.Fprintf(os.Stderr, "\n[Info Rate-Limit Gemini]: %s\n", infoMsg)

			sleepDuration := delay + 1*time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(sleepDuration):
				fmt.Fprintln(os.Stderr, "[Mencoba mengirim kembali...]")
				continue
			}
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("API Gemini mengembalikan status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var gResp geminiResponse
		if err := json.Unmarshal(bodyBytes, &gResp); err != nil {
			return "", fmt.Errorf("gagal unmarshal respon Gemini: %w", err)
		}

		if gResp.Error != nil {
			return "", fmt.Errorf("error dari Gemini: %s", gResp.Error.Message)
		}

		if len(gResp.Candidates) == 0 || len(gResp.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("tidak ada kandidat balasan dari Gemini")
		}

		return strings.TrimSpace(gResp.Candidates[0].Content.Parts[0].Text), nil
	}

	return "", fmt.Errorf("batas laju (RPM) Gemini masih belum pulih setelah retry. Coba kirim pesan kembali dalam beberapa saat.")
}
