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

	if err := writer.WriteField("source", "cli"); err != nil {
		return nil, fmt.Errorf("failed to add source field: %w", err)
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	var uploadedFile TrainingFile
	if err := t.client.PostMultipart(ctx, "/files", writer.FormDataContentType(), body.Bytes(), &uploadedFile); err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}
	return &uploadedFile, nil
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
