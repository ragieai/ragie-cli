package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const BaseURL = "https://api.ragie.ai"

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type Document struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

type ListOptions struct {
	Filter    map[string]interface{} `json:"filter,omitempty"`
	PageSize  int                    `json:"page_size,omitempty"`
	Cursor    string                 `json:"cursor,omitempty"`
	Partition string                 `json:"partition,omitempty"`
}

type ListResponse struct {
	Documents  []Document `json:"documents"`
	Pagination struct {
		NextCursor string `json:"next_cursor"`
	} `json:"pagination"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (c *Client) CreateDocumentRaw(partition string, name string, data string, metadata map[string]interface{}) (*Document, error) {
	payload := map[string]interface{}{
		"name":     name,
		"data":     data,
		"metadata": metadata,
	}

	if partition != "" {
		payload["partition"] = partition
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/documents/raw", BaseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var doc Document
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

func (c *Client) ListDocuments(opts ListOptions) (*ListResponse, error) {
	query := url.Values{}
	if opts.Filter != nil {
		filterJSON, err := json.Marshal(opts.Filter)
		if err != nil {
			return nil, err
		}
		query.Set("filter", string(filterJSON))
	}
	if opts.PageSize > 0 {
		query.Set("page_size", fmt.Sprintf("%d", opts.PageSize))
	}
	if opts.Cursor != "" {
		query.Set("cursor", opts.Cursor)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/documents?%s", BaseURL, query.Encode()), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	if opts.Partition != "" {
		req.Header.Set("Partition", opts.Partition)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}

	return &listResp, nil
}

func (c *Client) DeleteDocument(id string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/documents/%s", BaseURL, id), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	return nil
}
