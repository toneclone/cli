package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/internal/config"
	"github.com/toneclone/cli/pkg/client"
)

var (
	// Training command flags
	trainingFormat    string
	trainingPersona   string
	trainingFile      string
	trainingText      string
	trainingFilename  string
	trainingDirectory string
	trainingRecursive bool
	trainingConfirm   bool
	trainingVerbose   bool
	trainingFileID    string
	trainingBatchSize int
)

// trainingCmd represents the training command
var trainingCmd = &cobra.Command{
	Use:   "training",
	Short: "Manage training data for personas",
	Long: `Manage training data for personas - upload files and associate with personas.

Training data is used to customize personas with your own writing style and content.
Files can be uploaded and associated with personas to improve their writing quality.

Examples:
  toneclone training list
  toneclone training add --file=document.txt --persona=professional
  toneclone training add --text="Sample content" --persona=casual
  toneclone training associate --file-id=file-123 --persona=writer`,
}

// listTrainingCmd represents the list subcommand
var listTrainingCmd = &cobra.Command{
	Use:   "list",
	Short: "List training files",
	Long: `List all training files for the authenticated user.

Files can be filtered by persona association and sorted by various criteria.

Examples:
  toneclone training list
  toneclone training list --persona=professional
  toneclone training list --format=json`,
	RunE: runListTraining,
}

// addTrainingCmd represents the add subcommand
var addTrainingCmd = &cobra.Command{
	Use:   "add",
	Short: "Add training data",
	Long: `Add training data by uploading files or text content.

Files can be uploaded from local filesystem or text can be provided directly.
Files are automatically associated with the specified persona.

Examples:
  toneclone training add --file=document.txt --persona=professional
  toneclone training add --text="Sample content" --persona=casual --filename=sample.txt
  toneclone training add --directory=./docs --persona=writer --recursive`,
	RunE: runAddTraining,
}

// removeTrainingCmd represents the remove subcommand
var removeTrainingCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove training data",
	Long: `Remove training data by deleting files or disassociating from personas.

Files can be completely deleted or just disassociated from specific personas.

Examples:
  toneclone training remove --file-id=file-123 --persona=professional
  toneclone training remove --file-id=file-123 --confirm`,
	RunE: runRemoveTraining,
}

// associateTrainingCmd represents the associate subcommand
var associateTrainingCmd = &cobra.Command{
	Use:   "associate",
	Short: "Associate files with personas",
	Long: `Associate existing files with personas for training.

Files must be uploaded before they can be associated with personas.

Examples:
  toneclone training associate --file-id=file-123 --persona=professional
  toneclone training associate --file-id=file-123,file-456 --persona=writer`,
	RunE: runAssociateTraining,
}

// disassociateTrainingCmd represents the disassociate subcommand
var disassociateTrainingCmd = &cobra.Command{
	Use:   "disassociate",
	Short: "Disassociate files from personas",
	Long: `Disassociate files from personas without deleting the files.

Files remain available for association with other personas.

Examples:
  toneclone training disassociate --file-id=file-123 --persona=professional
  toneclone training disassociate --file-id=file-123,file-456 --persona=writer`,
	RunE: runDisassociateTraining,
}

func init() {
	rootCmd.AddCommand(trainingCmd)

	// Add subcommands
	trainingCmd.AddCommand(listTrainingCmd)
	trainingCmd.AddCommand(addTrainingCmd)
	trainingCmd.AddCommand(removeTrainingCmd)
	trainingCmd.AddCommand(associateTrainingCmd)
	trainingCmd.AddCommand(disassociateTrainingCmd)

	// List command flags
	listTrainingCmd.Flags().StringVar(&trainingFormat, "format", "table", "output format: table, json")
	listTrainingCmd.Flags().StringVar(&trainingPersona, "persona", "", "filter by persona name or ID")

	// Add command flags
	addTrainingCmd.Flags().StringVar(&trainingFile, "file", "", "file to upload")
	addTrainingCmd.Flags().StringVar(&trainingText, "text", "", "text content to upload")
	addTrainingCmd.Flags().StringVar(&trainingPersona, "persona", "", "persona to associate with")
	addTrainingCmd.Flags().StringVar(&trainingFilename, "filename", "", "filename for text content")
	addTrainingCmd.Flags().StringVar(&trainingDirectory, "directory", "", "directory to upload files from")
	addTrainingCmd.Flags().BoolVar(&trainingRecursive, "recursive", false, "recursively upload files from directory")
	addTrainingCmd.Flags().BoolVar(&trainingVerbose, "verbose", false, "verbose output")

	// Remove command flags
	removeTrainingCmd.Flags().StringVar(&trainingFileID, "file-id", "", "file ID to remove")
	removeTrainingCmd.Flags().StringVar(&trainingPersona, "persona", "", "persona to disassociate from")
	removeTrainingCmd.Flags().BoolVar(&trainingConfirm, "confirm", false, "skip confirmation prompt")

	// Associate command flags
	associateTrainingCmd.Flags().StringVar(&trainingFileID, "file-id", "", "file ID(s) to associate (comma-separated)")
	associateTrainingCmd.Flags().StringVar(&trainingPersona, "persona", "", "persona to associate with")
	associateTrainingCmd.MarkFlagRequired("file-id")
	associateTrainingCmd.MarkFlagRequired("persona")

	// Disassociate command flags
	disassociateTrainingCmd.Flags().StringVar(&trainingFileID, "file-id", "", "file ID(s) to disassociate (comma-separated)")
	disassociateTrainingCmd.Flags().StringVar(&trainingPersona, "persona", "", "persona to disassociate from")
	disassociateTrainingCmd.MarkFlagRequired("file-id")
	disassociateTrainingCmd.MarkFlagRequired("persona")
}

func runListTraining(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current API key
	keyConfig, err := cfg.GetCurrentKey()
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	// Create API client
	apiClient := client.NewToneCloneClientFromConfig(
		keyConfig.BaseURL,
		keyConfig.Key,
		30*time.Second,
	)

	ctx := context.Background()

	// Get training files
	files, err := apiClient.Training.ListFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to list training files: %w", err)
	}

	// Filter by persona if specified
	if trainingPersona != "" {
		persona, err := validatePersona(ctx, apiClient, trainingPersona)
		if err != nil {
			return fmt.Errorf("persona validation failed: %w", err)
		}

		// Get files associated with this persona
		personaFiles, err := apiClient.Personas.ListFiles(ctx, persona.PersonaID)
		if err != nil {
			return fmt.Errorf("failed to list persona files: %w", err)
		}

		files = filterFilesByPersona(files, personaFiles)
	}

	// Output files
	if trainingFormat == "json" {
		return outputTrainingFilesJSON(files)
	}

	return outputTrainingFilesTable(files)
}

func runAddTraining(cmd *cobra.Command, args []string) error {
	// Validate input
	if trainingFile == "" && trainingText == "" && trainingDirectory == "" {
		return fmt.Errorf("one of --file, --text, or --directory must be specified")
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current API key
	keyConfig, err := cfg.GetCurrentKey()
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	// Create API client
	apiClient := client.NewToneCloneClientFromConfig(
		keyConfig.BaseURL,
		keyConfig.Key,
		30*time.Second,
	)

	ctx := context.Background()

	// Validate persona if specified
	var persona *client.Persona
	if trainingPersona != "" {
		persona, err = validatePersona(ctx, apiClient, trainingPersona)
		if err != nil {
			return fmt.Errorf("persona validation failed: %w", err)
		}
	}

	// Handle different input types
	if trainingText != "" {
		return addTextTraining(ctx, apiClient, persona)
	}

	if trainingFile != "" {
		return addFileTraining(ctx, apiClient, persona)
	}

	if trainingDirectory != "" {
		return addDirectoryTraining(ctx, apiClient, persona)
	}

	return nil
}

func runRemoveTraining(cmd *cobra.Command, args []string) error {
	if trainingFileID == "" {
		return fmt.Errorf("--file-id is required")
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current API key
	keyConfig, err := cfg.GetCurrentKey()
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	// Create API client
	apiClient := client.NewToneCloneClientFromConfig(
		keyConfig.BaseURL,
		keyConfig.Key,
		30*time.Second,
	)

	ctx := context.Background()

	// If persona is specified, disassociate instead of delete
	if trainingPersona != "" {
		persona, err := validatePersona(ctx, apiClient, trainingPersona)
		if err != nil {
			return fmt.Errorf("persona validation failed: %w", err)
		}

		fileIDs := strings.Split(trainingFileID, ",")
		for i, id := range fileIDs {
			fileIDs[i] = strings.TrimSpace(id)
		}

		err = apiClient.Personas.DisassociateFiles(ctx, persona.PersonaID, fileIDs)
		if err != nil {
			return fmt.Errorf("failed to disassociate files: %w", err)
		}

		fmt.Printf("✓ Files disassociated from persona '%s'\n", persona.Name)
		return nil
	}

	// Get file info for confirmation
	file, err := apiClient.Training.GetFile(ctx, trainingFileID)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Confirm deletion
	if !trainingConfirm {
		fmt.Printf("Are you sure you want to delete file '%s' (%s)? [y/N]: ", file.FileName, file.FileID)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete file
	err = apiClient.Training.DeleteFile(ctx, trainingFileID)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	fmt.Printf("✓ File '%s' deleted successfully\n", file.FileName)
	return nil
}

func runAssociateTraining(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current API key
	keyConfig, err := cfg.GetCurrentKey()
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	// Create API client
	apiClient := client.NewToneCloneClientFromConfig(
		keyConfig.BaseURL,
		keyConfig.Key,
		30*time.Second,
	)

	ctx := context.Background()

	// Validate persona
	persona, err := validatePersona(ctx, apiClient, trainingPersona)
	if err != nil {
		return fmt.Errorf("persona validation failed: %w", err)
	}

	// Parse file IDs
	fileIDs := strings.Split(trainingFileID, ",")
	for i, id := range fileIDs {
		fileIDs[i] = strings.TrimSpace(id)
	}

	// Associate files
	err = apiClient.Personas.AssociateFiles(ctx, persona.PersonaID, fileIDs)
	if err != nil {
		return fmt.Errorf("failed to associate files: %w", err)
	}

	fmt.Printf("✓ %d file(s) associated with persona '%s'\n", len(fileIDs), persona.Name)
	return nil
}

func runDisassociateTraining(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current API key
	keyConfig, err := cfg.GetCurrentKey()
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	// Create API client
	apiClient := client.NewToneCloneClientFromConfig(
		keyConfig.BaseURL,
		keyConfig.Key,
		30*time.Second,
	)

	ctx := context.Background()

	// Validate persona
	persona, err := validatePersona(ctx, apiClient, trainingPersona)
	if err != nil {
		return fmt.Errorf("persona validation failed: %w", err)
	}

	// Parse file IDs
	fileIDs := strings.Split(trainingFileID, ",")
	for i, id := range fileIDs {
		fileIDs[i] = strings.TrimSpace(id)
	}

	// Disassociate files
	err = apiClient.Personas.DisassociateFiles(ctx, persona.PersonaID, fileIDs)
	if err != nil {
		return fmt.Errorf("failed to disassociate files: %w", err)
	}

	fmt.Printf("✓ %d file(s) disassociated from persona '%s'\n", len(fileIDs), persona.Name)
	return nil
}


// Helper functions

func addTextTraining(ctx context.Context, apiClient *client.ToneCloneClient, persona *client.Persona) error {
	filename := trainingFilename
	if filename == "" {
		filename = "text_content.txt"
	}

	request := &client.UploadTextRequest{
		Content:  trainingText,
		Filename: filename,
		Source:   "cli",
	}

	file, err := apiClient.Training.UploadText(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to upload text: %w", err)
	}

	fmt.Printf("✓ Text uploaded successfully\n")
	fmt.Printf("  File ID: %s\n", file.FileID)
	fmt.Printf("  Filename: %s\n", file.FileName)
	fmt.Printf("  Size: %d bytes\n", file.FileSize)

	// Associate with persona if specified
	if persona != nil {
		err = apiClient.Personas.AssociateFiles(ctx, persona.PersonaID, []string{file.FileID})
		if err != nil {
			return fmt.Errorf("failed to associate with persona: %w", err)
		}
		fmt.Printf("  Associated with persona: %s\n", persona.Name)
	}

	return nil
}

func addFileTraining(ctx context.Context, apiClient *client.ToneCloneClient, persona *client.Persona) error {
	// Check if file exists
	if _, err := os.Stat(trainingFile); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", trainingFile)
	}

	// Open file
	file, err := os.Open(trainingFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	filename := filepath.Base(trainingFile)

	if trainingVerbose {
		fmt.Printf("Uploading file: %s (%d bytes)\n", filename, fileInfo.Size())
	}

	// Upload file
	uploadedFile, err := apiClient.Training.UploadFile(ctx, file, filename)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	fmt.Printf("✓ File uploaded successfully\n")
	fmt.Printf("  File ID: %s\n", uploadedFile.FileID)
	fmt.Printf("  Filename: %s\n", uploadedFile.FileName)
	fmt.Printf("  Size: %d bytes\n", uploadedFile.FileSize)

	// Associate with persona if specified
	if persona != nil {
		err = apiClient.Personas.AssociateFiles(ctx, persona.PersonaID, []string{uploadedFile.FileID})
		if err != nil {
			return fmt.Errorf("failed to associate with persona: %w", err)
		}
		fmt.Printf("  Associated with persona: %s\n", persona.Name)
	}

	return nil
}

func addDirectoryTraining(ctx context.Context, apiClient *client.ToneCloneClient, persona *client.Persona) error {
	// Check if directory exists
	if _, err := os.Stat(trainingDirectory); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", trainingDirectory)
	}

	var files []string

	// Walk directory
	err := filepath.Walk(trainingDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip subdirectories if not recursive
		if !trainingRecursive {
			dir := filepath.Dir(path)
			if dir != trainingDirectory {
				return nil
			}
		}

		// Filter by file extension (optional)
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".txt" || ext == ".md" || ext == ".doc" || ext == ".docx" || ext == ".pdf" {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no supported files found in directory")
	}

	fmt.Printf("Found %d files to upload\n", len(files))

	// Process files in batches for better efficiency
	const batchSize = 10
	var totalUploaded int
	var totalAssociated int

	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]
		batchNum := (i / batchSize) + 1
		totalBatches := (len(files) + batchSize - 1) / batchSize

		if trainingVerbose {
			fmt.Printf("Processing batch %d/%d (%d files)\n", batchNum, totalBatches, len(batch))
		}

		// Prepare file uploads for this batch
		var fileUploads []client.FileUpload

		for _, filePath := range batch {
			filename := filepath.Base(filePath)
			
			// Open file
			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("  ✗ Failed to open %s: %v\n", filename, err)
				continue
			}

			fileUploads = append(fileUploads, client.FileUpload{
				Filename: filename,
				Reader:   file,
			})
		}

		// Skip if no valid files in this batch
		if len(fileUploads) == 0 {
			continue
		}

		// Get persona ID for batch upload
		var personaID string
		if persona != nil {
			personaID = persona.PersonaID
		}

		// Upload batch with integrated persona association
		response, err := apiClient.Training.UploadFileBatch(ctx, fileUploads, personaID, "cli")
		
		// Close all files
		for _, upload := range fileUploads {
			if closer, ok := upload.Reader.(io.Closer); ok {
				closer.Close()
			}
		}

		if err != nil {
			fmt.Printf("  ✗ Batch upload failed: %v\n", err)
			continue
		}

		// Report results for this batch
		for _, result := range response.Files {
			if result.Status == "success" {
				fmt.Printf("  ✓ %s uploaded", result.Filename)
				if result.FileID != "" {
					fmt.Printf(" (ID: %s)", result.FileID)
				}
				if result.Associated {
					fmt.Printf(" and associated with persona")
				}
				fmt.Printf("\n")
			} else {
				fmt.Printf("  ✗ %s failed: %s\n", result.Filename, result.Error)
			}
		}

		totalUploaded += response.Summary.Uploaded
		totalAssociated += response.Summary.Associated
	}

	fmt.Printf("✓ %d files uploaded successfully", totalUploaded)
	if persona != nil && totalAssociated > 0 {
		fmt.Printf(", %d associated with persona '%s'", totalAssociated, persona.Name)
	}
	fmt.Printf("\n")

	return nil
}

func filterFilesByPersona(files []client.TrainingFile, personaFiles []client.TrainingFile) []client.TrainingFile {
	personaFileMap := make(map[string]bool)
	for _, pf := range personaFiles {
		personaFileMap[pf.FileID] = true
	}

	var filtered []client.TrainingFile
	for _, f := range files {
		if personaFileMap[f.FileID] {
			filtered = append(filtered, f)
		}
	}

	return filtered
}

func filterJobsByStatus(jobs []client.TrainingJob, status string) []client.TrainingJob {
	var filtered []client.TrainingJob
	for _, job := range jobs {
		if strings.EqualFold(job.Status, status) {
			filtered = append(filtered, job)
		}
	}
	return filtered
}

func filterJobsByPersona(jobs []client.TrainingJob, personaID string) []client.TrainingJob {
	var filtered []client.TrainingJob
	for _, job := range jobs {
		if job.PersonaID == personaID {
			filtered = append(filtered, job)
		}
	}
	return filtered
}

func watchJobStatus(ctx context.Context, apiClient *client.ToneCloneClient, jobID string) error {
	fmt.Printf("Watching job %s (press Ctrl+C to stop)\n\n", jobID)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastStatus string

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			job, err := apiClient.Training.GetJob(ctx, jobID)
			if err != nil {
				fmt.Printf("Error getting job status: %v\n", err)
				continue
			}

			if job.Status != lastStatus {
				fmt.Printf("[%s] Status: %s", time.Now().Format("15:04:05"), job.Status)
				if job.FilesProcessed > 0 {
					fmt.Printf(" (%d/%d files processed)", job.FilesProcessed, job.TotalFiles)
				}
				fmt.Println()
				lastStatus = job.Status
			}

			// Stop watching if job is complete
			if job.Status == "Ready" || job.Status == "Error" {
				fmt.Printf("\nJob completed with status: %s\n", job.Status)
				return nil
			}
		}
	}
}

func outputTrainingFilesTable(files []client.TrainingFile) error {
	if len(files) == 0 {
		fmt.Println("No training files found.")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "FILENAME\tSIZE\tCONTENT TYPE\tUPLOADED\tUSED FOR TRAINING\tID")
	fmt.Fprintln(w, "--------\t----\t------------\t--------\t-----------------\t--")

	// Rows
	for _, file := range files {
		uploaded := formatTime(file.CreatedAt)
		used := "No"
		if file.UsedForTraining {
			used = "Yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			file.FileName,
			formatFileSize(file.FileSize),
			file.ContentType,
			uploaded,
			used,
			file.FileID,
		)
	}

	return nil
}

func outputTrainingFilesJSON(files []client.TrainingFile) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"files": files,
		"count": len(files),
	})
}

func outputTrainingJobsTable(jobs []client.TrainingJob) error {
	if len(jobs) == 0 {
		fmt.Println("No training jobs found.")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "JOB ID\tPERSONA ID\tSTATUS\tPROGRESS\tCREATED\tUPDATED")
	fmt.Fprintln(w, "------\t----------\t------\t--------\t-------\t-------")

	// Rows
	for _, job := range jobs {
		created := formatTime(job.CreatedAt)
		updated := formatTime(job.UpdatedAt)
		progress := fmt.Sprintf("%d/%d", job.FilesProcessed, job.TotalFiles)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			job.JobID,
			job.PersonaID,
			job.Status,
			progress,
			created,
			updated,
		)
	}

	return nil
}

func outputTrainingJobsJSON(jobs []client.TrainingJob) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

func outputJobStatusDetails(job *client.TrainingJob) error {
	fmt.Printf("Training Job Details\n")
	fmt.Printf("====================\n")
	fmt.Printf("Job ID:           %s\n", job.JobID)
	fmt.Printf("Persona ID:       %s\n", job.PersonaID)
	fmt.Printf("Status:           %s\n", job.Status)
	fmt.Printf("Progress:         %d/%d files processed\n", job.FilesProcessed, job.TotalFiles)
	fmt.Printf("Created:          %s\n", formatTime(job.CreatedAt))
	fmt.Printf("Updated:          %s\n", formatTime(job.UpdatedAt))
	if job.OpenAIJobID != "" {
		fmt.Printf("OpenAI Job ID:    %s\n", job.OpenAIJobID)
		fmt.Printf("OpenAI Status:    %s\n", job.OpenAIJobStatus)
	}
	if job.BaseModel != "" {
		fmt.Printf("Base Model:       %s\n", job.BaseModel)
	}

	return nil
}

func outputJobStatusJSON(job *client.TrainingJob) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(job)
}

func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}
