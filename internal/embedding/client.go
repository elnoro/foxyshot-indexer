package embedding

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
)

var ErrInvalidRequest = errors.New("invalid response code from embeddings api")

type Client struct {
	baseURL string

	log *slog.Logger
}

func NewClient(baseURL string, log *slog.Logger) *Client {
	return &Client{baseURL: baseURL, log: log}
}

type embReq struct {
	File string `json:"file"`
}

type embResp struct {
	Embedding []float32 `json:"embedding"`
}

func (c *Client) CreateEmbeddingForFile(filePath string) (domain.Embedding, error) {
	c.log.Info("creating embedding for file", slog.String("file", filePath))
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file locally, %w", err)
	}

	base64EncodedData := base64.StdEncoding.EncodeToString(fileData)

	return c.CreateEmbeddingFromBase64(base64EncodedData)
}

func (c *Client) CreateEmbeddingFromBase64(data string) (domain.Embedding, error) {
	req := embReq{File: data}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request, %w", err)
	}

	resp, err := http.DefaultClient.Post(c.baseURL+"/encode-base64", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("request failed, %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logResp, _ := httputil.DumpResponse(resp, true)

		c.log.Error("embeddings api response",
			slog.String("resp", string(logResp)),
		)

		return nil, ErrInvalidRequest
	}

	var jsonResp embResp

	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		return nil, fmt.Errorf("cannote decode response, %w", err)
	}

	return jsonResp.Embedding, nil
}
