/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package netlify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sprintframework/sprint"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
)

const defaultBaseURL = "https://api.netlify.com/api/v1"
const defaultPublicIDURL = "https://api.ipify.org"

// Client Netlify API client.
type Client struct {
	HTTPClient *http.Client
	BaseURL    string

	token string
}

// NewClient creates a new Client.
func NewClient(token string) *Client {
	return &Client{
		HTTPClient: http.DefaultClient,
		BaseURL:    defaultBaseURL,
		token:      token,
	}
}

// GetRecords gets a DNS records.
func (c *Client) GetRecords(zoneID string) ([]*sprint.DNSRecord, error) {
	zoneID = strings.ReplaceAll(zoneID, ".", "_")
	endpoint, err := c.createEndpoint("dns_zones", zoneID, "dns_records")
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code: %s: %s", resp.Status, string(body))
	}

	var records []*sprint.DNSRecord
	err = json.Unmarshal(body, &records)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response body: %w", err)
	}

	return records, nil
}

// CreateRecord creates a DNS records.
func (c *Client) CreateRecord(zoneID string, record *sprint.DNSRecord) (*sprint.DNSRecord, error) {
	zoneID = strings.ReplaceAll(zoneID, ".", "_")
	endpoint, err := c.createEndpoint("dns_zones", zoneID, "dns_records")
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}

	marshaledRecord, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(marshaledRecord))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("invalid status code: %s: %s", resp.Status, string(body))
	}

	var recordResp sprint.DNSRecord
	err = json.Unmarshal(body, &recordResp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response body: %w", err)
	}

	return &recordResp, nil
}

// RemoveRecord removes a DNS records.
func (c *Client) RemoveRecord(zoneID, recordID string) error {
	zoneID = strings.ReplaceAll(zoneID, ".", "_")
	endpoint, err := c.createEndpoint("dns_zones", zoneID, "dns_records", recordID)
	if err != nil {
		return fmt.Errorf("failed to parse endpoint: %w", err)
	}

	req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("API call failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("invalid status code: %s: %s", resp.Status, string(body))
	}

	return nil
}

func (c *Client) GetPublicIP() (addr string, err error) {
	res, err := http.Get("https://api.ipify.org")
	if err != nil {
		return  "", err
	}
	ip, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(ip), nil
}

func (c *Client) createEndpoint(parts ...string) (string, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	endpoint, err := base.Parse(path.Join(base.Path, path.Join(parts...)))
	if err != nil {
		return "", fmt.Errorf("failed to parse endpoint path: %w", err)
	}

	return endpoint.String(), nil
}

