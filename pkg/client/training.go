package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// TrainingClient handles training-related API operations
type TrainingClient struct {
	client *Client
}

// NewTrainingClient creates a new training client
func NewTrainingClient(client *Client) *TrainingClient {
	return &TrainingClient{client: client}
}

// ListFiles retrieves all training files for the user
func (t *TrainingClient) ListFiles(ctx context.Context) ([]TrainingFile, error) {
	resp, err := t.client.makeRequest(ctx, "GET", "/files", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	// Handle empty response (no files)
	if resp.ContentLength == 0 {
		return []TrainingFile{}, nil
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle empty body content
	if len(respBody) == 0 {
		return []TrainingFile{}, nil
	}

	var files []TrainingFile
	if err := json.Unmarshal(respBody, &files); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return files, nil
}

// GetFile retrieves a specific training file by ID
func (t *TrainingClient) GetFile(ctx context.Context, fileID string) (*TrainingFile, error) {
	var file TrainingFile
	err := t.client.Get(ctx, fmt.Sprintf("/files/%s", fileID), &file)
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s: %w", fileID, err)
	}
	return &file, nil
}

// UploadFile uploads a binary file
func (t *TrainingClient) UploadFile(ctx context.Context, file io.Reader, filename string) (*TrainingFile, error) {
	// Create a buffer to hold the multipart form data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Create form file field
	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy file content to form
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", t.client.baseURL+"/files", &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+t.client.apiKey)
	req.Header.Set("User-Agent", "ToneClone-CLI/1.0")

	// Make request
	resp, err := t.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	// Parse response
	var uploadedFile TrainingFile
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	err = json.Unmarshal(respBody, &uploadedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &uploadedFile, nil
}

// FileUpload represents a file to be uploaded
type FileUpload struct {
	Filename string
	Reader   io.Reader
}

// BatchFileResult represents the result of uploading a single file in a batch
type BatchFileResult struct {
	FileID     string `json:"file_id,omitempty"`
	Filename   string `json:"filename"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	Size       int64  `json:"size,omitempty"`
	Associated bool   `json:"associated"`
}

// BatchUploadResponse represents the response from batch file upload
type BatchUploadResponse struct {
	Files     []BatchFileResult `json:"files"`
	PersonaID string           `json:"persona_id,omitempty"`
	Summary   struct {
		Total      int `json:"total"`
		Uploaded   int `json:"uploaded"`
		Associated int `json:"associated"`
		Failed     int `json:"failed"`
	} `json:"summary"`
}

// UploadFileBatch uploads multiple files in a single request
func (t *TrainingClient) UploadFileBatch(ctx context.Context, files []FileUpload, personaID, source string) (*BatchUploadResponse, error) {
	// Create a buffer to hold the multipart form data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add files to form
	for _, file := range files {
		fileWriter, err := writer.CreateFormFile("files", file.Filename)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file for %s: %w", file.Filename, err)
		}

		// Reset reader if it's seekable
		if seeker, ok := file.Reader.(io.Seeker); ok {
			seeker.Seek(0, io.SeekStart)
		}

		_, err = io.Copy(fileWriter, file.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file content for %s: %w", file.Filename, err)
		}
	}

	// Add persona_id if provided
	if personaID != "" {
		err := writer.WriteField("persona_id", personaID)
		if err != nil {
			return nil, fmt.Errorf("failed to add persona_id field: %w", err)
		}
	}

	// Add source
	if source != "" {
		err := writer.WriteField("source", source)
		if err != nil {
			return nil, fmt.Errorf("failed to add source field: %w", err)
		}
	}

	// Close the multipart writer
	err := writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", t.client.baseURL+"/files/batch", &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+t.client.apiKey)
	req.Header.Set("User-Agent", "ToneClone-CLI/1.0")

	// Make request
	resp, err := t.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload files: %w", err)
	}
	defer resp.Body.Close()

	// Read response body first
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusPartialContent {
		// Try to parse error response for more details
		var errorResp map[string]interface{}
		if json.Unmarshal(respBody, &errorResp) == nil {
			if msg, ok := errorResp["error"].(string); ok {
				return nil, fmt.Errorf("batch upload failed with status %d: %s", resp.StatusCode, msg)
			}
			if msg, ok := errorResp["message"].(string); ok {
				return nil, fmt.Errorf("batch upload failed with status %d: %s", resp.StatusCode, msg)
			}
		}
		return nil, fmt.Errorf("batch upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var batchResponse BatchUploadResponse
	err = json.Unmarshal(respBody, &batchResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &batchResponse, nil
}

// UploadText uploads text content
func (t *TrainingClient) UploadText(ctx context.Context, request *UploadTextRequest) (*TrainingFile, error) {
	resp, err := t.client.makeRequest(ctx, "POST", "/files/text", request)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	// Handle empty response (backend issue)
	if resp.ContentLength == 0 {
		// Return a placeholder file object since the upload succeeded but no response
		return &TrainingFile{
			FileID:      "unknown",
			FileName:    request.Filename,
			FileSize:    int64(len(request.Content)),
			ContentType: "text",
			CreatedAt:   time.Now(),
		}, nil
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle empty body content
	if len(respBody) == 0 {
		return &TrainingFile{
			FileID:      "unknown",
			FileName:    request.Filename,
			FileSize:    int64(len(request.Content)),
			ContentType: "text",
			CreatedAt:   time.Now(),
		}, nil
	}

	var file TrainingFile
	if err := json.Unmarshal(respBody, &file); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &file, nil
}

// DeleteFile deletes a training file
func (t *TrainingClient) DeleteFile(ctx context.Context, fileID string) error {
	err := t.client.Delete(ctx, fmt.Sprintf("/files/%s", fileID))
	if err != nil {
		return fmt.Errorf("failed to delete file %s: %w", fileID, err)
	}
	return nil
}

// ListJobs retrieves all training jobs for the user
func (t *TrainingClient) ListJobs(ctx context.Context) ([]TrainingJob, error) {
	var jobs []TrainingJob
	err := t.client.Get(ctx, "/training/jobs", &jobs)
	if err != nil {
		return nil, fmt.Errorf("failed to list training jobs: %w", err)
	}
	return jobs, nil
}

// GetJob retrieves a specific training job by ID
func (t *TrainingClient) GetJob(ctx context.Context, jobID string) (*TrainingJob, error) {
	var job TrainingJob
	err := t.client.Get(ctx, fmt.Sprintf("/training/jobs/%s", jobID), &job)
	if err != nil {
		return nil, fmt.Errorf("failed to get training job %s: %w", jobID, err)
	}
	return &job, nil
}

// CreateJob creates a new training job
func (t *TrainingClient) CreateJob(ctx context.Context, personaID string, fileIDs []string) (*TrainingJob, error) {
	request := map[string]interface{}{
		"persona_id": personaID,
	}

	if len(fileIDs) > 0 {
		request["file_ids"] = fileIDs
	}

	var job TrainingJob
	err := t.client.Post(ctx, "/training/jobs", request, &job)
	if err != nil {
		return nil, fmt.Errorf("failed to create training job: %w", err)
	}
	return &job, nil
}

// CreatePersonaTrainingJob creates a training job for a persona using all associated files
func (t *TrainingClient) CreatePersonaTrainingJob(ctx context.Context, personaID string) (*TrainingJob, error) {
	var job TrainingJob
	err := t.client.Post(ctx, fmt.Sprintf("/training/personas/%s", personaID), nil, &job)
	if err != nil {
		return nil, fmt.Errorf("failed to create persona training job: %w", err)
	}
	return &job, nil
}
