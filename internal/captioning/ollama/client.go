package ollama

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

const (
	model  = "llava"
	prompt = "Describe this screenshot, please"
)

var ErrBadResponse = errors.New("ollama returned non-200 code")

type Client struct {
	url string
	log *slog.Logger
}

func NewClient(url string, log *slog.Logger) (*Client, error) {
	cl := &Client{url: url, log: log}
	err := cl.test()
	if err != nil {
		return nil, err
	}

	return cl, nil
}

func (o *Client) test() error {
	reqBody := fmt.Sprintf(`{ "name": "%s" }'`, model)
	buffer := bytes.NewBufferString(reqBody)
	resp, err := http.Post(o.url+"/api/show", "application/json", buffer)
	if err != nil {
		return fmt.Errorf("ollama unavailable, %s %w", o.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("model unavailable, %w", ErrBadResponse)
	}

	return nil
}

func (o *Client) Caption(filename string) (string, error) {
	o.log.Info("captioning file", slog.String("file", filename))
	imageBytes, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("cannot read file %s, %w", filename, err)
	}

	encodedImage := base64.StdEncoding.EncodeToString(imageBytes)

	req := captionReq{
		Model:  model,
		Prompt: prompt,
		Images: []string{encodedImage},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request to ollama, %w", err)
	}

	resp, err := http.Post(o.url+"/api/generate", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("cannot get response from ollama, %s, %w", o.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("calling api, %s, code %d, %w", o.url, resp.StatusCode, ErrBadResponse)
	}

	var caption captionResp
	err = json.NewDecoder(resp.Body).Decode(&caption)
	if err != nil {
		return "", fmt.Errorf("decoding response, %w", err)
	}

	return caption.Response, nil
}

type captionReq struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images"`
	Stream bool     `json:"stream"`
}

type captionResp struct {
	Response string `json:"response"`
}
