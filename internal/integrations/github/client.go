package github

import (
	"context"

	"email-subscription-service/internal/domain"
)

type Client interface {
	RepoExists(ctx context.Context, repo domain.Repo) (bool, error)
	LatestReleaseTag(ctx context.Context, repo domain.Repo) (tag string, ok bool, err error)
}