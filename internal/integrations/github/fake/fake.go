package fake

import (
	"context"

	"email-subscription-service/internal/domain"
)

type Client struct {
	Exists bool
	Err    error
	LatestTag string
	HasLatest bool
}

func (c Client) RepoExists(ctx context.Context, repo domain.Repo) (bool, error) {
	if c.Err != nil {
		return false, c.Err
	}
	return c.Exists, nil
}

func (c Client) LatestReleaseTag(ctx context.Context, repo domain.Repo) (tag string, ok bool, err error) {
	if c.Err != nil {
		return "", false, c.Err
	}
	if !c.HasLatest {
		return "", false, nil
	}
	return c.LatestTag, true, nil
}