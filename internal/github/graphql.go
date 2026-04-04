package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// graphql executes a GraphQL query against the GitHub API.
func (c *Client) graphql(ctx context.Context, query string, variables map[string]any, result any) error {
	body, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/graphql", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create graphql request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("graphql request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read graphql response: %w", err)
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return fmt.Errorf("unmarshal graphql response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}

	return json.Unmarshal(gqlResp.Data, result)
}

// labelNode is the shape of a GraphQL label node.
type labelNode struct {
	Name string `json:"name"`
}

// extractLabels extracts label names from GraphQL label nodes.
func extractLabels(nodes []labelNode) []string {
	var labels []string
	for _, l := range nodes {
		labels = append(labels, l.Name)
	}
	return labels
}

// actor is a reusable type for GitHub actors (users, bots, etc.).
type actor struct {
	Login string `json:"login"`
}

func (a *actor) GetLogin() string {
	if a == nil {
		return ""
	}
	return a.Login
}
