package xsolla

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type HTTPClient struct {
	baseURL    string
	projectID  string
	merchantID string
	apiKey     string
	client     *http.Client
}

func NewHTTPClient(baseURL, projectID, merchantID, apiKey string) *HTTPClient {
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"), projectID: projectID,
		merchantID: merchantID, apiKey: apiKey, client: &http.Client{},
	}
}

func (c *HTTPClient) CreatePaymentToken(ctx context.Context, request PaymentTokenRequest) (string, error) {
	body, err := json.Marshal(request)

	if err != nil {
		return "", fmt.Errorf("marshal payment token request: %w", err)
	}
	url := fmt.Sprintf("%s/v3/project/%s/admin/payment/token", c.baseURL, c.projectID)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))

	if err != nil {
		return "", fmt.Errorf("create payment token request: %w", err)
	}
	httpRequest.SetBasicAuth(c.merchantID, c.apiKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(httpRequest)

	if err != nil {
		return "", fmt.Errorf("call xsolla token API: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)

	if err != nil {
		return "", fmt.Errorf("read xsolla token response: %w", err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("xsolla token API returned %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("decode xsolla token response: %w", err)
	}

	if result.Token == "" {
		return "", fmt.Errorf("xsolla token response did not contain token")
	}
	return result.Token, nil
}
