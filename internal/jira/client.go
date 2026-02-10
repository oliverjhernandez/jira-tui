// Package jira provides a client for interacting with the Jira API.
package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
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
	ID                string
	Key               string
	Summary           string
	Status            string
	Type              string
	Assignee          string
	Priority          Priority
	Description       string
	Reporter          string
	Comments          []Comment
	Parent            *Parent
	IsLinkedToChange  bool
	ChangeIssueLinkID string
	OriginalEstimate  string
	Created           string
	Updated           string
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

type IssueLink struct {
	ID           string       `json:"id"`
	InwardIssue  *LinkedIssue `json:"inwardIssue,omitempty"`
	OutwardIssue *LinkedIssue `json:"outwardIssue,omitempty"`
	Type         Link         `json:"type"`
}

const MonthlyChangeIssue = "IN-912"

type LinkedIssue struct {
	ID     string         `json:"id"`
	Key    string         `json:"key"`
	Self   string         `json:"self"`
	Fields map[string]any `json:"fields"`
}

type Link struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
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
	IssueLinks       []IssueLink     `json:"issueLinks"`
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

func (c *Client) doJiraRequest(ctx context.Context, method, endpoint string, queryParams url.Values, body any, result any, expectedStatus ...int) error {
	apiURL := fmt.Sprintf("%s%s", c.jiraURL, endpoint)
	if len(queryParams) > 0 {
		apiURL = fmt.Sprintf("%s?%s", apiURL, queryParams.Encode())
	}

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.jiraEmail, c.jiraToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if len(expectedStatus) == 0 {
		expectedStatus = []int{http.StatusOK, http.StatusCreated}
	}

	statusOK := slices.Contains(expectedStatus, resp.StatusCode)

	if !statusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) doTempoRequest(ctx context.Context, method, endpoint string, queryParams url.Values, body any, result any, expectedStatus ...int) error {
	apiURL := fmt.Sprintf("%s%s", c.tempoURL, endpoint)

	if len(queryParams) > 0 {
		apiURL = fmt.Sprintf("%s?%s", apiURL, queryParams.Encode())
	}

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Tempo uses Bearer token, not BasicAuth
	req.Header.Set("Authorization", "Bearer "+c.tempoToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if len(expectedStatus) == 0 {
		expectedStatus = []int{http.StatusOK, http.StatusCreated}
	}

	statusOK := slices.Contains(expectedStatus, resp.StatusCode)

	if !statusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) GetMySelf(ctx context.Context) (*User, error) {
	var result User
	err := c.doJiraRequest(
		ctx,
		"GET",
		"/rest/api/3/myself",
		nil,
		nil,
		&result,
	)

	return &result, err
}

func (c *Client) SearchIssuesJql(ctx context.Context, jql string) ([]Issue, error) {
	params := url.Values{}
	params.Add("jql", jql)
	params.Add("maxResults", "100")
	params.Add("fields", "id,summary,status,issuetype,assignee,priority")

	var searchResp searchResponse

	err := c.doJiraRequest(
		ctx,
		"GET",
		"/rest/api/3/search/jql",
		params,
		nil,
		&searchResp,
	)

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

	return result, err
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
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s", issueKey)
	params := url.Values{}
	params.Add("fields", "id,summary,description,status,issuetype,assignee,reporter,comment,priority,parent,issuelinks,timeoriginalestimate,created,updated")

	var issue jiraIssue
	err := c.doJiraRequest(ctx, "GET", apiURL, params, nil, &issue)

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

	for _, link := range issue.Fields.IssueLinks {
		var linkedKey string

		if link.InwardIssue != nil {
			linkedKey = link.InwardIssue.Key
		} else if link.OutwardIssue != nil {
			linkedKey = link.OutwardIssue.Key
		}

		if linkedKey == MonthlyChangeIssue {
			detail.IsLinkedToChange = true
			detail.ChangeIssueLinkID = link.ID
			break
		}
	}

	if issue.Fields.OriginalEstimate != nil {
		detail.OriginalEstimate = formatSecondsToTime(*issue.Fields.OriginalEstimate)
	}

	detail.Created = issue.Fields.Created
	detail.Updated = issue.Fields.Updated

	return detail, err
}

func (c *Client) GetTransitions(ctx context.Context, issueKey string) ([]Transition, error) {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey)

	var result struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				Name string `json:"name"`
			} `json:"to"`
		} `json:"transitions"`
	}

	err := c.doJiraRequest(ctx, "GET", apiURL, nil, nil, &result)

	transitions := make([]Transition, 0, len(result.Transitions))
	for _, t := range result.Transitions {
		transitions = append(transitions, Transition{
			ID:   t.ID,
			Name: t.To.Name,
		})
	}

	return transitions, err
}

func (c *Client) GetUsers(ctx context.Context, issueKey string) ([]User, error) {
	apiURL := fmt.Sprintf("/rest/api/3/user/assignable/search?issueKey=%s", issueKey)

	var result []User

	err := c.doJiraRequest(
		ctx,
		"GET",
		apiURL,
		nil,
		nil,
		&result,
	)

	users := make([]User, 0, len(result))
	for _, t := range result {
		users = append(users, User{
			ID:   t.ID,
			Name: t.Name,
		})
	}

	return users, err
}

func (c *Client) PostAssignee(ctx context.Context, issueKey, assigneeID string) error {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s/assignee", issueKey)

	body := map[string]any{
		"accountId": assigneeID,
	}

	err := c.doJiraRequest(ctx, "PUT", apiURL, nil, body, nil)

	return err
}

func (c *Client) PostTransition(ctx context.Context, issueKey, transitionID string) error {
	return c.PostTransitionWithFields(ctx, issueKey, transitionID, nil)
}

func (c *Client) PostTransitionWithFields(ctx context.Context, issueKey, transitionID string, fields map[string]any) error {
	return c.PostTransitionWithComment(ctx, issueKey, transitionID, fields, "")
}

func (c *Client) PostTransitionWithComment(ctx context.Context, issueKey, transitionID string, fields map[string]any, comment string) error {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey)

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

	err := c.doJiraRequest(ctx, "POST", apiURL, nil, body, nil)

	return err
}

func (c *Client) UpdateDescription(ctx context.Context, issueKey string, description string) error {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s", issueKey)

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

	err := c.doJiraRequest(
		ctx,
		"PUT",
		apiURL,
		nil,
		body,
		nil,
	)

	return err
}

func (c *Client) UpdatePriority(ctx context.Context, issueKey string, priority string) error {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s", issueKey)

	body := map[string]any{
		"fields": map[string]any{
			"priority": map[string]any{
				"name": priority,
			},
		},
	}

	err := c.doJiraRequest(
		ctx,
		"PUT",
		apiURL,
		nil,
		body,
		nil,
	)

	return err
}

func (c *Client) UpdateOriginalEstimate(ctx context.Context, issueKey string, estimate string) error {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s", issueKey)

	body := map[string]any{
		"fields": map[string]any{
			"timetracking": map[string]any{
				"originalEstimate": estimate,
			},
		},
	}

	err := c.doJiraRequest(
		ctx,
		"PUT",
		apiURL,
		nil,
		body,
		nil,
	)

	return err
}

func (c *Client) GetPriorities(ctx context.Context) ([]Priority, error) {
	var result []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	err := c.doJiraRequest(
		ctx,
		"GET",
		"/rest/api/3/priority",
		nil,
		nil,
		&result,
	)

	priorities := make([]Priority, 0, len(result))
	for _, t := range result {
		priorities = append(priorities, Priority{
			ID:   t.ID,
			Name: t.Name,
		})
	}

	return priorities, err
}

func (c *Client) GetStatuses(ctx context.Context, projects []string) ([]Status, error) {
	statuses := []Status{}
	seen := make(map[string]bool)

	for _, p := range projects {
		apiURL := fmt.Sprintf("/rest/api/3/project/%s/statuses", p)

		var result []struct {
			Statuses []Status `json:"statuses"`
		}

		err := c.doJiraRequest(
			ctx,
			"GET",
			apiURL,
			nil,
			nil,
			&result,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get statuses for project %s: %w", p, err)
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
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey)

	content, err := parseCommentContent(comment, usersCache)
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

	err = c.doJiraRequest(
		ctx,
		"POST",
		apiURL,
		nil,
		body,
		nil,
	)

	return err
}

func (c *Client) GetWorkLogs(ctx context.Context, issueID string) ([]WorkLog, error) {
	apiURL := fmt.Sprintf("/4/worklogs/issue/%s", issueID)
	var result worklogsResponse
	err := c.doTempoRequest(
		ctx,
		"GET",
		apiURL,
		nil,
		nil,
		&result,
	)

	return result.Results, err
}

func (c *Client) PostWorkLog(ctx context.Context, issueID, date, accountID string, time int) error {
	body := map[string]any{
		"issueId":          issueID,
		"timeSpentSeconds": time,
		"startDate":        date,
		"authorAccountId":  accountID,
	}

	err := c.doTempoRequest(
		ctx,
		"POST",
		"/4/worklogs",
		nil,
		body,
		nil,
	)

	return err
}

func (c *Client) PostIssueLink(ctx context.Context, fromKey, toKey string) error {
	body := map[string]any{
		"type": map[string]string{
			"name": "Relates",
		},
		"inwardIssue": map[string]string{
			"key": fromKey,
		},
		"outwardIssue": map[string]string{
			"key": toKey,
		},
	}

	err := c.doJiraRequest(
		ctx,
		"POST",
		"/rest/api/3/issueLink",
		nil,
		body,
		nil,
	)

	return err
}

func (c *Client) DeleteIssueLink(ctx context.Context, linkID string) error {
	apiURL := fmt.Sprintf("/rest/api/3/issueLink/%s", linkID)

	err := c.doJiraRequest(
		ctx,
		"DELETE",
		apiURL,
		nil,
		nil,
		nil,
		http.StatusNoContent,
	)

	return err
}
