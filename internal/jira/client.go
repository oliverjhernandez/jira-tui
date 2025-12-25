// Package jira provides a client for interacting with the Jira API.
package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	baseURL string
	client  *http.Client
	email   string
	token   string
}

type Issue struct {
	Key     string
	Summary string
	Status  string
	Type    string
}

type IssueDetail struct {
	Key         string
	Summary     string
	Status      string
	Type        string
	Description string
	Assignee    string
	Reporter    string
	Comments    []Comment
}

type Comment struct {
	Author  string
	Body    string
	Created string
}

type Transition struct {
	ID   string
	Name string
}

func NewClient(baseURL, email, token string) (*Client, error) {
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{},
		email:   email,
		token:   token,
	}, nil
}

// Response structs for the v3 API
type searchResponse struct {
	Issues []jiraIssue `json:"issues"`
}

type jiraIssue struct {
	Key    string      `json:"key"`
	Fields issueFields `json:"fields"`
}

type issueFields struct {
	Summary     string          `json:"summary"`
	Description *descriptionDoc `json:"description"`
	Status      statusField     `json:"status"`
	Type        typeField       `json:"issuetype"`
	Assignee    *userField      `json:"assignee"`
	Reporter    *userField      `json:"reporter"`
	Comment     *commentList    `json:"comment"`
}

type descriptionDoc struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type    string        `json:"type"`
	Content []contentNode `json:"content,omitempty"`
	Text    string        `json:"text,omitempty"`
}

type contentNode struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type statusField struct {
	Name string `json:"name"`
}

type typeField struct {
	Name string `json:"name"`
}

type userField struct {
	DisplayName string `json:"displayName"`
}

type commentList struct {
	Comments []jiraComment `json:"comments"`
}

type jiraComment struct {
	Author  userField       `json:"author"`
	Body    *descriptionDoc `json:"body"`
	Created string          `json:"created"`
}

func (c *Client) GetMyIssues(ctx context.Context) ([]Issue, error) {
	jql := "assignee = currentUser() AND resolution = Unresolved ORDER BY created DESC"

	apiURL := fmt.Sprintf("%s/rest/api/3/search/jql", c.baseURL)
	params := url.Values{}
	params.Add("jql", jql)
	params.Add("maxResults", "50")
	params.Add("fields", "summary,status,issuetype")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var searchResp searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make([]Issue, 0, len(searchResp.Issues))
	for _, issue := range searchResp.Issues {
		result = append(result, Issue{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Status:  issue.Fields.Status.Name,
			Type:    issue.Fields.Type.Name,
		})
	}

	return result, nil
}

func (c *Client) GetIssueDetail(ctx context.Context, issueKey string) (*IssueDetail, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s", c.baseURL, issueKey)
	params := url.Values{}
	params.Add("fields", "summary,description,status,issuetype,assignee,reporter,comment")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var issue jiraIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	detail := &IssueDetail{
		Key:         issue.Key,
		Summary:     issue.Fields.Summary,
		Status:      issue.Fields.Status.Name,
		Type:        issue.Fields.Type.Name,
		Description: extractText(issue.Fields.Description),
	}

	if issue.Fields.Assignee != nil {
		detail.Assignee = issue.Fields.Assignee.DisplayName
	} else {
		detail.Assignee = "Unassigned"
	}

	if issue.Fields.Reporter != nil {
		detail.Reporter = issue.Fields.Reporter.DisplayName
	}

	if issue.Fields.Comment != nil {
		for _, comment := range issue.Fields.Comment.Comments {
			detail.Comments = append(detail.Comments, Comment{
				Author:  comment.Author.DisplayName,
				Body:    extractText(comment.Body),
				Created: comment.Created,
			})
		}
	}

	return detail, nil
}

func (c *Client) GetTransitions(ctx context.Context, issueKey string) ([]Transition, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", c.baseURL, issueKey)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				Name string `json:"name"`
			} `json:"to"`
		} `json:"transitions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	transitions := make([]Transition, 0, len(result.Transitions))
	for _, t := range result.Transitions {
		transitions = append(transitions, Transition{
			ID:   t.ID,
			Name: t.To.Name,
		})
	}

	return transitions, nil
}

func (c *Client) DoTransition(ctx context.Context, issueKey, transitionID string) error {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", c.baseURL, issueKey)

	body := map[string]any{
		"transition": map[string]string{
			"id": transitionID,
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) PostDescription(ctx context.Context, issueKey string, description string) error {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s", c.baseURL, issueKey)

	body := map[string]any{
		"fields": map[string]any{
			"description": map[string]any{
				"type":    "doc",
				"version": 1,
				"content": []map[string]any{
					{
						"type": "paragraph",
						"content": []map[string]any{
							{
								"type": "text",
								"text": description,
							},
						},
					},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func extractText(doc *descriptionDoc) string {
	if doc == nil {
		return ""
	}

	var text string
	for _, block := range doc.Content {
		text += extractBlockText(block) + "\n"
	}
	return text
}

func extractBlockText(block contentBlock) string {
	if block.Text != "" {
		return block.Text
	}

	var text string
	for _, node := range block.Content {
		if node.Text != "" {
			text += node.Text
		}
	}
	return text
}
