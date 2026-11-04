package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/longkey1/gojira/internal/jsonutil"
	"github.com/longkey1/gojira/internal/models"
)

type sprintPage struct {
	MaxResults int             `json:"maxResults"`
	StartAt    int             `json:"startAt"`
	IsLast     bool            `json:"isLast"`
	Values     []models.Sprint `json:"values"`
}

func (c *Client) listSprintsPage(ctx context.Context, boardID int, states []string, startAt int) (*sprintPage, error) {
	params := url.Values{}
	params.Set("startAt", strconv.Itoa(startAt))
	params.Set("maxResults", strconv.Itoa(defaultMaxResults))
	if len(states) > 0 {
		params.Set("state", strings.Join(states, ","))
	}
	endpoint := fmt.Sprintf("/rest/agile/1.0/board/%d/sprint?%s", boardID, params.Encode())

	req, err := c.newRequest("GET", endpoint)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	body = jsonutil.SanitizeJSON(body)

	var page sprintPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &page, nil
}

// ListSprints returns all sprints of a board, following startAt/isLast pagination.
// An empty states slice returns sprints in every state.
func (c *Client) ListSprints(ctx context.Context, boardID int, states []string) ([]models.Sprint, error) {
	sprints := []models.Sprint{}
	startAt := 0

	for {
		page, err := c.listSprintsPage(ctx, boardID, states, startAt)
		if err != nil {
			return nil, err
		}

		sprints = append(sprints, page.Values...)

		// An empty page guards against looping forever if the server never reports isLast.
		if page.IsLast || len(page.Values) == 0 {
			break
		}

		startAt += len(page.Values)
	}

	return sprints, nil
}
