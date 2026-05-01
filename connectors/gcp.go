package connectors

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
)

// GCPConfig holds connection configuration for GCP.
// Credentials are resolved via Application Default Credentials (ADC):
//   - GOOGLE_APPLICATION_CREDENTIALS env var pointing to a service-account JSON key, OR
//   - gcloud auth application-default login
type GCPConfig struct {
	// ProjectID is the GCP project to audit.  Falls back to GCP_PROJECT_ID env var when empty.
	ProjectID string `json:"project_id"`

	// CredentialsFile is an optional explicit path to a service-account JSON key.
	// When empty, ADC resolution is used.
	CredentialsFile string `json:"credentials_file"`
}

// GCPClient is the authenticated surface passed to GCP scanners.
type GCPClient struct {
	ProjectID string

	// ClientOptions holds the resolved google API options so each scanner
	// can construct its own service client without re-authenticating.
	ClientOptions []option.ClientOption
}

// ConnectGCP authenticates to GCP and returns a GCPClient ready for scanning.
func ConnectGCP(ctx context.Context, gcpCfg GCPConfig) (*GCPClient, error) {
	// Resolve project ID
	projectID := gcpCfg.ProjectID
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT_ID")
	}
	if projectID == "" {
		return nil, fmt.Errorf("GCP project ID is required: set GCPConfig.ProjectID or GCP_PROJECT_ID env var")
	}

	var opts []option.ClientOption

	if gcpCfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(gcpCfg.CredentialsFile))
	} else {
		// Use ADC — respects GOOGLE_APPLICATION_CREDENTIALS automatically
		creds, err := google.FindDefaultCredentials(ctx,
			cloudresourcemanager.CloudPlatformScope,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find GCP default credentials: %w", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}

	return &GCPClient{
		ProjectID:     projectID,
		ClientOptions: opts,
	}, nil
}
