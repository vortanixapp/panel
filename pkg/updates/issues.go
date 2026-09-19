package updates

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Issue struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	State       string `json:"state"`
	StateReason string `json:"state_reason"`
	URL         string `json:"url"`
	Comments    int    `json:"comments"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type githubIssue struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	State       string    `json:"state"`
	StateReason *string   `json:"state_reason"`
	HTMLURL     string    `json:"html_url"`
	Comments    int       `json:"comments"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
	PullRequest *struct{} `json:"pull_request"`
}

func SearchIssues(ctx context.Context, repo string, words []string, limit int) ([]Issue, error) {
	query := "repo:" + repo + " is:issue"
	params := url.Values{}
	if len(words) == 0 {
		query += " is:open"
		params.Set("sort", "updated")
		params.Set("order", "desc")
	} else {
		query += " in:title,body " + strings.Join(words, " OR ")
	}
	params.Set("q", query)
	params.Set("per_page", strconv.Itoa(limit))

	var payload struct {
		Items []githubIssue `json:"items"`
	}
	if err := githubAPI(ctx, "/search/issues?"+params.Encode(), &payload); err != nil {
		if errors.Is(err, errGitHubNotFound) {
			return nil, fmt.Errorf("репозиторий %s не найден на GitHub", repo)
		}
		return nil, err
	}

	list := make([]Issue, 0, len(payload.Items))
	for _, item := range payload.Items {
		if item.PullRequest != nil {
			continue
		}
		reason := ""
		if item.StateReason != nil {
			reason = *item.StateReason
		}
		list = append(list, Issue{
			Number:      item.Number,
			Title:       item.Title,
			State:       item.State,
			StateReason: reason,
			URL:         item.HTMLURL,
			Comments:    item.Comments,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
		if len(list) >= limit {
			break
		}
	}
	return list, nil
}
