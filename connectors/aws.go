package connectors

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	stscreds "github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type AWSConfig struct {
	AccessKey  string `json:"access_key"`
	SecretKey  string `json:"secret_key"`
	Region     string `json:"region"`
	RoleARN    string `json:"role_arn"`
	ExternalID string `json:"external_id"`
}

func ConnectAws(ctx context.Context, awsCfg AWSConfig) (aws.Config, error) {
	
	// Default configuration loaders
	optFns := []func(*config.LoadOptions) error{}

	if awsCfg.Region != "" {
		optFns = append(optFns, config.WithRegion(awsCfg.Region))
	}

	if awsCfg.AccessKey != "" && awsCfg.SecretKey != "" {
		provider := credentials.NewStaticCredentialsProvider(awsCfg.AccessKey, awsCfg.SecretKey, "")
		optFns = append(optFns, config.WithCredentialsProvider(provider))
	}

	cfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return aws.Config{}, err
	}

	if awsCfg.RoleARN == "" {
		return cfg, nil
	}

	stsClient := sts.NewFromConfig(cfg)

	var assumeProvider *stscreds.AssumeRoleProvider
	if awsCfg.ExternalID != "" {
		assumeProvider = stscreds.NewAssumeRoleProvider(stsClient, awsCfg.RoleARN, func(o *stscreds.AssumeRoleOptions) {
			o.ExternalID = aws.String(awsCfg.ExternalID)
		})
	} else {
		assumeProvider = stscreds.NewAssumeRoleProvider(stsClient, awsCfg.RoleARN)
	}

	cfg.Credentials = aws.NewCredentialsCache(assumeProvider)

	return cfg, nil
}
