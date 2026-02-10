package logger

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

type aiNetPayload struct {
	Model    string      `json:"model"`
	Messages []aiContent `json:"messages"`
}

type aiContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiNetResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (l *CustomLogger) ConsultSecurityAI(trafficData string) string {
	apiKey := os.Getenv("API_KEY")
	apiUrl := ""

	prompt := "Analyze the following network metadata for potential DDoS patterns. Respond with ONLY 'CLEAN' or 'ATTACK'. Metadata: " + trafficData

	body := aiNetPayload{
		Model: "gpt-3.5-turbo",
		Messages: []aiContent{
			{
				Role:    "system",
				Content: "You are an expert Cybersecurity AI focused on Real-time Traffic Analysis and DDoS mitigation.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}
	jsonData, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		l.Log(ERROR, "AI Sentinel: Failed to connect to API")
		return "SERVICE_ERROR"
	}
	defer resp.Body.Close()

	var result aiNetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		l.Log(ERROR, "AI Sentinel: Failed to parse API response")
		return "PARSE_ERROR"
	}

	if len(result.Choices) > 0 {
		answer := strings.ToUpper(strings.TrimSpace(result.Choices[0].Message.Content))

		// Ensure the system reacts correctly even if the AI is wordy
		if strings.Contains(answer, "ATTACK") {
			return "ATTACK"
		}
	}

	return "CLEAN"

}
