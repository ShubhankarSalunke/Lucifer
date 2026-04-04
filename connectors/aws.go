package connectors

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	stscreds "github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func ConnectAws(ctx context.Context, roleArn string, externalId string) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return aws.Config{}, err
	}

	if roleArn == "" {
		return cfg, nil
	}

	stsClient := sts.NewFromConfig(cfg)

	var provider *stscreds.AssumeRoleProvider
	if externalId != "" {
		provider = stscreds.NewAssumeRoleProvider(stsClient, roleArn, func(o *stscreds.AssumeRoleOptions) {
			o.ExternalID = aws.String(externalId)
		})
	} else {
		provider = stscreds.NewAssumeRoleProvider(stsClient, roleArn)
	}

	cfg.Credentials = aws.NewCredentialsCache(provider)

	return cfg, nil
}
