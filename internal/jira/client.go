// Package jira provides a client for interacting with the Jira API.
package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Client struct {
	Client     *http.Client
	jiraURL    string
	jiraToken  string
	tempoURL   string
	tempoToken string
	jiraEmail  string
}

type Issue struct {
	ID       string
	Key      string
	Summary  string
	Status   string
	Type     string
	Assignee string
	Priority string
}

type IssueDetail struct {
	ID               string
	Key              string
	Summary          string
	Status           string
	Type             string
	Assignee         string
	Priority         Priority
	Description      string
	Reporter         string
	Comments         []Comment
	Parent           *Parent
	OriginalEstimate string
	Created          string
	Updated          string
}

type Comment struct {
	Author       string
	EmailAddress string
	Body         string
	Created      string
}

type Transition struct {
	ID   string
	Name string
}

type User struct {
	ID    string `json:"accountId"`
	Name  string `json:"displayName"`
	Email string `json:"emailAddress"`
}

type Priority struct {
	ID   string
	Name string
}

type Parent struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Type string `json:"type"`
}

func NewClient(jiraBaseURL, email, jiraToken, tempoBaseURL, tempoToken string) (*Client, error) {
	return &Client{
		Client:     &http.Client{},
		jiraURL:    jiraBaseURL,
		jiraToken:  jiraToken,
		tempoURL:   tempoBaseURL,
		tempoToken: tempoToken,
		jiraEmail:  email,
	}, nil
}

// Response structs for the v3 API
type searchResponse struct {
	Issues []jiraIssue `json:"issues"`
}

type worklogsResponse struct {
	Results []WorkLog `json:"results"`
}

type jiraIssue struct {
	Key    string      `json:"key"`
	ID     string      `json:"id"`
	Fields issueFields `json:"fields"`
}

type issueFields struct {
	Summary          string          `json:"summary"`
	Description      *descriptionDoc `json:"description"`
	Status           statusField     `json:"status"`
	Type             typeField       `json:"issuetype"`
	Assignee         *userField      `json:"assignee"`
	Reporter         *userField      `json:"reporter"`
	Comment          *commentList    `json:"comment"`
	Priority         *priorityField  `json:"priority"`
	Parent           *parentField    `json:"parent"`
	OriginalEstimate *int            `json:"timeoriginalestimate"`
	Created          string          `json:"created"`
	Updated          string          `json:"updated"`
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
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Attrs attrs  `json:"attrs"`
}

type attrs struct {
	Text string `json:"text,omitempty"`
	ID   string `json:"id,omitempty"`
}

type statusField struct {
	Name string `json:"name"`
}

type typeField struct {
	Name string `json:"name"`
}

type priorityField struct {
	Name string `json:"name"`
}

type parentField struct {
	ID         string `json:"id"`
	ParentType string `json:"parent_type"`
	Key        string `json:"key"`
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

type WorkLog struct {
	ID          int    `json:"tempoWorklogId"`
	Time        int    `json:"timeSpentSeconds"`
	Author      Author `json:"author"`
	Description string `json:"description"`
	UpdatedAt   string `json:"updatedAt"`
}

type Author struct {
	AccountID string `json:"accountId"`
}

type Status struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	StatusCategory StatusCategory `json:"statusCategory"`
}

type StatusCategory struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

func (c *Client) GetMySelf() (*User, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/myself", c.jiraURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.jiraEmail, c.jiraToken)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	var result User
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) SearchIssuesJql(ctx context.Context, jql string) ([]Issue, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/search/jql", c.jiraURL)
	params := url.Values{}
	params.Add("jql", jql)
	params.Add("maxResults", "100")
	params.Add("fields", "id,summary,status,issuetype,assignee,priority")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("JSON: %v", string(bodyBytes))
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var searchResp searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make([]Issue, 0, len(searchResp.Issues))

	for _, issue := range searchResp.Issues {
		var assignee string

		if issue.Fields.Assignee == nil {
			assignee = "Unassigned"
		} else {
			assignee = issue.Fields.Assignee.DisplayName
		}

		result = append(result, Issue{
			ID:       issue.ID,
			Key:      issue.Key,
			Summary:  issue.Fields.Summary,
			Status:   issue.Fields.Status.Name,
			Type:     issue.Fields.Type.Name,
			Assignee: assignee,
			Priority: issue.Fields.Priority.Name,
		})
	}

	return result, nil
}

func (c *Client) GetMyIssues(ctx context.Context) ([]Issue, error) {
	jql := "assignee = currentUser() AND resolution = Unresolved ORDER BY status DESC"
	return c.SearchIssuesJql(ctx, jql)
}

func (c *Client) GetEpicChildren(ctx context.Context, epicKey string) ([]Issue, error) {
	jql := fmt.Sprintf("parent = %s ORDER BY status DESC", epicKey)
	return c.SearchIssuesJql(ctx, jql)
}

func (c *Client) GetIssueDetail(ctx context.Context, issueKey string) (*IssueDetail, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s", c.jiraURL, issueKey)
	params := url.Values{}
	params.Add("fields", "id,summary,description,status,issuetype,assignee,reporter,comment,priority,parent,timeoriginalestimate,created,updated")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %s", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var issue jiraIssue
	// body2 := resp.Body
	// bodyBytes, _ := io.ReadAll(body2)
	// prettyJSON, _ := json.MarshalIndent(string(bodyBytes), "", " ")
	// log.Printf("JSON: %s", string(prettyJSON))

	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	detail := &IssueDetail{
		ID:          issue.ID,
		Key:         issue.Key,
		Type:        issue.Fields.Type.Name,
		Summary:     issue.Fields.Summary,
		Status:      issue.Fields.Status.Name,
		Description: extractText(issue.Fields.Description),
	}

	if issue.Fields.Parent != nil {
		detail.Parent = &Parent{
			issue.Fields.Parent.ID,
			issue.Fields.Parent.Key,
			issue.Fields.Parent.ParentType,
		}
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

	if issue.Fields.Priority != nil {
		detail.Priority = Priority{
			Name: issue.Fields.Priority.Name,
		}
	}

	if issue.Fields.OriginalEstimate != nil {
		detail.OriginalEstimate = formatSecondsToTime(*issue.Fields.OriginalEstimate)
	}

	detail.Created = issue.Fields.Created
	detail.Updated = issue.Fields.Updated

	return detail, nil
}

func formatSecondsToTime(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return ""
}

func (c *Client) GetTransitions(ctx context.Context, issueKey string) ([]Transition, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", c.jiraURL, issueKey)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

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

func (c *Client) GetUsers(ctx context.Context, issueKey string) ([]User, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/user/assignable/search?issueKey=%s", c.jiraURL, issueKey)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result []User

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	users := make([]User, 0, len(result))
	for _, t := range result {
		users = append(users, User{
			ID:   t.ID,
			Name: t.Name,
		})
	}

	return users, nil
}

func (c *Client) PostAssignee(ctx context.Context, issueKey, assigneeID string) error {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s/assignee", c.jiraURL, issueKey)

	body := map[string]any{
		"accountId": assigneeID,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		log.Printf("failed to execute request: %s", err.Error())
		return nil // TODO: manage error upwards
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return nil
	}

	return nil
}

func (c *Client) PostTransition(ctx context.Context, issueKey, transitionID string) error {
	return c.PostTransitionWithFields(ctx, issueKey, transitionID, nil)
}

func (c *Client) PostTransitionWithFields(ctx context.Context, issueKey, transitionID string, fields map[string]any) error {
	return c.PostTransitionWithComment(ctx, issueKey, transitionID, fields, "")
}

func (c *Client) PostTransitionWithComment(ctx context.Context, issueKey, transitionID string, fields map[string]any, comment string) error {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", c.jiraURL, issueKey)

	body := map[string]any{
		"transition": map[string]string{
			"id": transitionID,
		},
	}

	if fields != nil {
		body["fields"] = fields
	}

	if comment != "" {
		body["update"] = map[string]any{
			"comment": []map[string]any{
				{
					"add": map[string]any{
						"body": map[string]any{
							"type":    "doc",
							"version": 1,
							"content": []map[string]any{
								{
									"type": "paragraph",
									"content": []map[string]any{
										{
											"type": "text",
											"text": comment,
										},
									},
								},
							},
						},
					},
				},
			},
		}
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) UpdateDescription(ctx context.Context, issueKey string, description string) error {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s", c.jiraURL, issueKey)

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

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) UpdatePriority(ctx context.Context, issueKey string, priority string) error {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s", c.jiraURL, issueKey)

	body := map[string]any{
		"fields": map[string]any{
			"priority": map[string]any{
				"name": priority,
			},
		},
	}

	// NOTE: sending a request probably should be a function on its own
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) UpdateOriginalEstimate(ctx context.Context, issueKey string, estimate string) error {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s", c.jiraURL, issueKey)

	body := map[string]any{
		"fields": map[string]any{
			"timetracking": map[string]any{
				"originalEstimate": estimate,
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

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) GetPriorities(ctx context.Context) ([]Priority, error) {
	apiURL := fmt.Sprintf("%s/rest/api/3/priority", c.jiraURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Body: %+v", string(bodyBytes))
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	priorities := make([]Priority, 0, len(result))
	for _, t := range result {
		priorities = append(priorities, Priority{
			ID:   t.ID,
			Name: t.Name,
		})
	}

	return priorities, nil
}

func (c *Client) GetStatuses(ctx context.Context, projects []string) ([]Status, error) {
	statuses := []Status{}
	seen := make(map[string]bool)

	for _, p := range projects {
		apiURL := fmt.Sprintf("%s/rest/api/3/project/%s/statuses", c.jiraURL, p)

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.SetBasicAuth(c.jiraEmail, c.jiraToken)
		req.Header.Set("Accept", "application/json")

		resp, err := c.Client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("error: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			log.Printf("Body: %+v", string(bodyBytes))
			return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var result []struct {
			Statuses []Status `json:"statuses"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		for _, r := range result {
			for _, s := range r.Statuses {
				if !seen[s.ID] {
					seen[s.ID] = true
					statuses = append(statuses, s)
				}
			}
		}
	}

	return statuses, nil
}

func (c *Client) PostComment(ctx context.Context, issueKey string, comment string, usersCache []User) error {
	apiURL := fmt.Sprintf("%s/rest/api/3/issue/%s/comment", c.jiraURL, issueKey)

	content, err := c.parseCommentContent(comment, usersCache)
	if err != nil {
		return fmt.Errorf("failed to parse comment.: %w", err)
	}

	body := map[string]any{
		"body": map[string]any{
			"type":    "doc",
			"version": 1,
			"content": []map[string]any{
				{
					"content": content,
					"type":    "paragraph",
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, bodyBytes, "", "  ")
	log.Printf("Full request body:\n%s", prettyJSON.String())

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) GetWorkLogs(ctx context.Context, issueID string) ([]WorkLog, error) {
	apiURL := fmt.Sprintf("%s/4/worklogs/issue/%s", c.tempoURL, issueID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.tempoToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Body: %+v", string(bodyBytes))
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var wl worklogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&wl); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return wl.Results, nil
}

func (c *Client) PostWorkLog(ctx context.Context, issueID, date, accountID string, time int) error {
	apiURL := fmt.Sprintf("%s/4/worklogs", c.tempoURL)

	body := map[string]any{
		"issueId":          issueID,
		"timeSpentSeconds": time,
		"startDate":        date,
		"authorAccountId":  accountID,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.tempoToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) parseCommentContent(comment string, users []User) ([]map[string]any, error) {
	mentionRegex := regexp.MustCompile(`@\[([^\]]+)\]`)

	matches := mentionRegex.FindAllStringSubmatchIndex(comment, -1)

	if len(matches) == 0 {
		return []map[string]any{
			{
				"type": "text",
				"text": comment,
			},
		}, nil
	}

	var content []map[string]any
	lastEnd := 0

	for _, match := range matches {
		matchStart := match[0]
		matchEnd := match[1]
		nameStart := match[2]
		nameEnd := match[3]

		if matchStart > lastEnd {
			content = append(content, map[string]any{
				"type": "text",
				"text": comment[lastEnd:matchStart],
			})
		}

		displayName := comment[nameStart:nameEnd]

		var accountID string
		for _, user := range users {
			if user.Name == displayName {
				accountID = user.ID
				break
			}
		}

		if accountID == "" {
			content = append(content, map[string]any{
				"type": "text",
				"text": comment[matchStart:matchEnd],
			})
		} else {
			content = append(content, map[string]any{
				"type": "mention",
				"attrs": map[string]string{
					"id":   accountID,
					"text": "@" + displayName,
				},
			})
		}

		lastEnd = matchEnd
	}

	if lastEnd < len(comment) {
		content = append(content, map[string]any{
			"type": "text",
			"text": comment[lastEnd:],
		})
	}

	return content, nil
}
