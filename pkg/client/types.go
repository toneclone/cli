package client

import "time"

// API Key Scope types
type APIKeyScope string

const (
	// Read permissions
	ScopePersonasRead  APIKeyScope = "personas:read"
	ScopeKnowledgeRead APIKeyScope = "knowledge:read"
	ScopeTrainingRead  APIKeyScope = "training:read"
	ScopeFilesRead     APIKeyScope = "files:read"
	ScopeWritingRead   APIKeyScope = "writing:read"
	ScopeUserRead      APIKeyScope = "user:read"

	// Write permissions
	ScopePersonasWrite  APIKeyScope = "personas:write"
	ScopeKnowledgeWrite APIKeyScope = "knowledge:write"
	ScopeTrainingWrite  APIKeyScope = "training:write"
	ScopeFilesWrite     APIKeyScope = "files:write"
	ScopeWritingWrite   APIKeyScope = "writing:write"
	ScopeUserWrite      APIKeyScope = "user:write"

	// Text generation
	ScopeTextGenerate APIKeyScope = "text:generate"

	// Admin permissions
	ScopeAdmin APIKeyScope = "admin:all"

	// Wildcard permission
	ScopeAll APIKeyScope = "*"
)

// Persona represents a writing persona
type Persona struct {
	PersonaID         string    `json:"personaId"`
	Name              string    `json:"name"`
	LastUsedAt        time.Time `json:"lastUsedAt"`
	LastModifiedAt    time.Time `json:"lastModifiedAt"`
	Status            string    `json:"status"`
	TrainingStatus    string    `json:"trainingStatus"`
	PersonaType       string    `json:"personaType"`
	VoiceEvolution    bool      `json:"voiceEvolution"`
	PromptDescription string    `json:"personaPromptDescription,omitempty"`
	IsBuiltIn         bool      `json:"isBuiltIn,omitempty"`
}

// PersonaListResponse represents the response from listing personas
type PersonaListResponse struct {
	Personas []Persona `json:"personas"`
}

// KnowledgeCard represents a writing knowledge card
type KnowledgeCard struct {
	PK              string    `json:"PK"`
	SK              string    `json:"SK"`
	KnowledgeCardID string    `json:"knowledgeCardId"`
	UserID          string    `json:"userId"`
	Name            string    `json:"name"`
	Instructions    string    `json:"instructions"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// KnowledgeCardListResponse represents the response from listing knowledge cards
type KnowledgeCardListResponse struct {
	KnowledgeCards []KnowledgeCard `json:"knowledgeCards"`
}

// TrainingFile represents a training file
type TrainingFile struct {
	PK              string    `json:"PK"`
	SK              string    `json:"SK"`
	UserID          string    `json:"userId"`
	FileID          string    `json:"fileId"`
	FileName        string    `json:"filename"`
	FileType        string    `json:"fileType"`
	FileSize        int64     `json:"size"`
	CreatedAt       time.Time `json:"createdAt"`
	ModifiedAt      time.Time `json:"modifiedAt"`
	S3Key           string    `json:"s3Key"`
	ContentType     string    `json:"contentType"`
	Source          string    `json:"source"`
	UsedForTraining bool      `json:"usedForTraining"`
	PersonaID       string    `json:"personaId,omitempty"`
}

// TrainingJob represents a training job
type TrainingJob struct {
	JobID           string    `json:"jobId"`
	PersonaID       string    `json:"personaId"`
	FileIDs         []string  `json:"fileIds"`
	TotalFiles      int       `json:"totalFiles"`
	FilesProcessed  int       `json:"filesProcessed"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	OpenAIJobID     string    `json:"openAIJobID,omitempty"`
	OpenAIJobStatus string    `json:"openAIJobStatus,omitempty"`
	BaseModel       string    `json:"baseModel,omitempty"`
}

// UploadTextRequest represents a text upload request
type UploadTextRequest struct {
	Content  string `json:"content"`
	Filename string `json:"filename"`
	Source   string `json:"source"`
}

// TrainingFileListResponse represents the response from listing training files
type TrainingFileListResponse struct {
	Files []TrainingFile `json:"files"`
}

// GenerateTextRequest represents a text generation request
type GenerateTextRequest struct {
	Prompt           string   `json:"prompt"`
	PersonaID        string   `json:"personaId"`
	KnowledgeCardID  string   `json:"knowledgeCardId,omitempty"`
	KnowledgeCardIDs []string `json:"knowledgeCardIds,omitempty"`
	Context          string   `json:"context,omitempty"`
	SessionID        string   `json:"sessionId,omitempty"`
	Document         string   `json:"document,omitempty"`
	Selection        string   `json:"selection,omitempty"`
	Formality        int      `json:"formality,omitempty"`
	ReadingLevel     int      `json:"readingLevel,omitempty"`
	Length           int      `json:"length,omitempty"`
	Model            string   `json:"model,omitempty"`
	Streaming        *bool    `json:"streaming,omitempty"`
}

// GenerateTextResponse represents a text generation response
type GenerateTextResponse struct {
	Text            string `json:"text"`
	PersonaID       string `json:"personaId,omitempty"`
	KnowledgeCardID string `json:"knowledgeCardId,omitempty"`
	Model           string `json:"model,omitempty"`
	Tokens          int    `json:"tokens,omitempty"`
}

// WritingSession represents a writing session
type WritingSession struct {
	SessionID       string    `json:"sessionId"`
	Title           string    `json:"title,omitempty"`
	Content         string    `json:"content,omitempty"`
	PersonaID       string    `json:"personaId,omitempty"`
	KnowledgeCardID string    `json:"knowledgeCardId,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	LastModifiedAt  time.Time `json:"lastModifiedAt"`
	Status          string    `json:"status"`
}

// WritingSessionListResponse represents the response from listing writing sessions
type WritingSessionListResponse struct {
	Sessions []WritingSession `json:"sessions"`
}

// User represents user information
type User struct {
	UserID    string    `json:"userId"`
	Email     string    `json:"email"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	Plan      string    `json:"plan,omitempty"`
}

// APIKey represents an API key (for management)
type APIKey struct {
	KeyID      string        `json:"keyId"`
	Name       string        `json:"name"`
	Prefix     string        `json:"prefix"`
	Scopes     []APIKeyScope `json:"scopes"`
	Status     string        `json:"status"`
	CreatedAt  time.Time     `json:"createdAt"`
	LastUsedAt *time.Time    `json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time    `json:"expiresAt,omitempty"`
	UsageCount int64         `json:"usageCount"`
}

// APIKeyListResponse represents the response from listing API keys
type APIKeyListResponse struct {
	Keys []APIKey `json:"keys"`
}

// Common request/response types

// ListOptions represents common listing options
type ListOptions struct {
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Sort   string `json:"sort,omitempty"`
	Filter string `json:"filter,omitempty"`
}

// PaginatedResponse represents a paginated response
type PaginatedResponse[T any] struct {
	Items      []T  `json:"items"`
	Total      int  `json:"total"`
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
	HasMore    bool `json:"hasMore"`
	NextOffset int  `json:"nextOffset,omitempty"`
}
