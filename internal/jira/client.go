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
	"slices"
	"strconv"
)

type Client struct {
	Client     *http.Client
	jiraURL    string
	jiraToken  string
	tempoURL   string
	tempoToken string
	jiraEmail  string
}

// TODO: map to correct types
type Issue struct {
	ID               string
	Key              string
	Summary          string
	Status           string
	Type             string
	Assignee         string
	Reporter         Reporter
	Priority         Priority
	Parent           *Parent
	Project          Project
	Description      *ContentDoc
	OriginalEstimate string
}

type IssueDetail struct {
	ID               string
	Key              string
	Project          Project
	Summary          string
	Status           string
	Type             string
	Assignee         string
	Priority         Priority
	Description      *ContentDoc
	Reporter         string
	Comments         []Comment
	Parent           *Parent
	IssueLinks       []IssueLink
	OriginalEstimate string
	Created          string
	Updated          string
	SubTasks         []Issue
	Worklogs         []Worklog
}

type IssueType struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Scope       *IssueTypeScope `json:"scope,omitempty"`
}

type IssueTypeScope struct {
	Project struct {
		ID string `json:"id"`
	} `json:"project"`
	Type string `json:"type"`
}

type Reporter struct {
	ID string `json:"id"`
}

// NOTE: improve
type Project struct {
	ID   string
	Name string
	Key  string
}

type Comment struct {
	ID           string
	Author       string
	EmailAddress string
	Body         *ContentDoc
	Created      string
	Updated      string
}

type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID    string `json:"accountId"`
	Name  string `json:"displayName"`
	Email string `json:"emailAddress"`
}

type Priority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
type issuesSearchResponse struct {
	Issues []jiraIssue `json:"issues"`
}

type projectsSearchResponse struct {
	Projects []jiraProject `json:"values"`
}

type worklogsResponse struct {
	Results []Worklog `json:"results"`
}

type jiraIssue struct {
	Key    string      `json:"key"`
	ID     string      `json:"id"`
	Fields issueFields `json:"fields"`
}

type jiraProject struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type issueFields struct {
	Summary          string         `json:"summary"`
	Project          Project        `json:"project"`
	Description      *ContentDoc    `json:"description"`
	Status           statusField    `json:"status"`
	Type             typeField      `json:"issuetype"`
	Assignee         *UserField     `json:"assignee"`
	Reporter         *UserField     `json:"reporter"`
	Comment          *commentList   `json:"comment"`
	Priority         *priorityField `json:"priority"`
	Parent           *parentField   `json:"parent"`
	IssueLinks       []IssueLink    `json:"issueLinks"`
	OriginalEstimate *int           `json:"timeoriginalestimate"`
	Created          string         `json:"created"`
	Updated          string         `json:"updated"`
}

type IssueLink struct {
	ID           string       `json:"id"`
	InwardIssue  *LinkedIssue `json:"inwardIssue,omitempty"`
	OutwardIssue *LinkedIssue `json:"outwardIssue,omitempty"`
	Type         Link         `json:"type"`
}

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

type LinkType int

const (
	Relates LinkType = iota
	Blocks
	Duplicates
)

func (l LinkType) String() string {
	switch l {
	case Relates:
		return "Relates"
	case Blocks:
		return "Blocks"
	case Duplicates:
		return "Duplicates"
	default:
		return ""
	}
}

type ContentDoc struct {
	Type    string        `json:"type"`
	Version int           `json:"version"`
	Content []ContentNode `json:"content"`
}

type ContentNode struct {
	Type    string        `json:"type"`
	Text    string        `json:"text,omitempty"`
	Content []ContentNode `json:"content,omitempty"`
	Attrs   *contentAttrs `json:"attrs,omitempty"`
	Marks   []mark        `json:"marks,omitempty"`
}

type mark struct {
	Type string `json:"type"`
}

type contentAttrs struct {
	Text string `json:"text,omitempty"`
	ID   string `json:"id,omitempty"`
	Alt  string `json:"alt"`
}

type statusField struct {
	Name string `json:"name"`
}

type typeField struct {
	Name string `json:"name"`
}

type priorityField struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type parentField struct {
	ID         string `json:"id"`
	ParentType string `json:"parent_type"`
	Key        string `json:"key"`
}

type UserField struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type commentList struct {
	Comments []jiraComment `json:"comments"`
}

type jiraComment struct {
	ID      string      `json:"id"`
	Author  UserField   `json:"author"`
	Body    *ContentDoc `json:"body"`
	Created string      `json:"created"`
	Updated string      `json:"updated"`
}

type Worklog struct {
	ID          int    `json:"tempoWorklogId"`
	Time        int    `json:"timeSpentSeconds"`
	StartDate   string `json:"startDate"`
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

var (
	PriorityOrder = map[string]int{
		"Highest": 0,
		"High":    1,
		"Medium":  2,
		"Low":     3,
		"Lowest":  4,
	}
)

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
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("failed to close response body: %v", closeErr)
		}
	}()

	if len(expectedStatus) == 0 {
		expectedStatus = []int{http.StatusOK, http.StatusCreated, http.StatusNoContent}
	}

	statusOK := slices.Contains(expectedStatus, resp.StatusCode)

	if !statusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)

		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

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
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("failed to close response body: %v", closeErr)
		}
	}()

	if len(expectedStatus) == 0 {
		expectedStatus = []int{http.StatusOK, http.StatusCreated, http.StatusNoContent}
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
	params.Add("fields", "id,summary,description,status,issuetype,assignee,parent,priority,project,timeoriginalestimate")

	var searchResp issuesSearchResponse

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

		i := Issue{
			ID:       issue.ID,
			Key:      issue.Key,
			Summary:  issue.Fields.Summary,
			Status:   issue.Fields.Status.Name,
			Type:     issue.Fields.Type.Name,
			Assignee: assignee,
			Project:  issue.Fields.Project,
		}

		if issue.Fields.Priority != nil {
			i.Priority = Priority{
				ID:   issue.Fields.Priority.ID,
				Name: issue.Fields.Priority.Name,
			}
		}

		if issue.Fields.Parent != nil {
			i.Parent = &Parent{
				issue.Fields.Parent.ID,
				issue.Fields.Parent.Key,
				issue.Fields.Parent.ParentType,
			}
		}

		if issue.Fields.Description != nil {
			i.Description = issue.Fields.Description
		}

		if issue.Fields.OriginalEstimate != nil {
			i.OriginalEstimate = strconv.Itoa(*issue.Fields.OriginalEstimate)
		}

		result = append(result, i)
	}

	return result, err
}

func (c *Client) GetMyIssues(ctx context.Context) ([]Issue, error) {
	jql := "assignee = currentUser() AND resolution = Unresolved ORDER BY status DESC"
	return c.SearchIssuesJql(ctx, jql)
}

func (c *Client) GetSubTasks(ctx context.Context, parentKey string) ([]Issue, error) {
	jql := fmt.Sprintf("parent = %s ORDER BY status DESC", parentKey)
	return c.SearchIssuesJql(ctx, jql)
}

func (c *Client) GetProjects(ctx context.Context) ([]jiraProject, error) {
	apiURL := "/rest/api/3/project/search"

	var projects projectsSearchResponse
	err := c.doJiraRequest(
		ctx,
		"GET",
		apiURL,
		nil,
		nil,
		&projects,
		http.StatusOK,
	)

	return projects.Projects, err
}

func (c *Client) GetIssueTypes(ctx context.Context) ([]IssueType, error) {
	apiURL := "/rest/api/3/issuetype"

	var issueTypes []IssueType
	err := c.doJiraRequest(
		ctx,
		"GET",
		apiURL,
		nil,
		nil,
		&issueTypes,
		http.StatusOK,
	)

	return issueTypes, err
}

func (c *Client) GetIssueDetail(ctx context.Context, issueKey string) (*IssueDetail, error) {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s", issueKey)
	params := url.Values{}
	params.Add("fields", "id,summary,description,project,status,issuetype,assignee,reporter,comment,priority,parent,issuelinks,timeoriginalestimate,created,updated")

	var issue jiraIssue
	err := c.doJiraRequest(
		ctx,
		"GET",
		apiURL,
		params,
		nil,
		&issue,
		http.StatusOK,
	)

	detail := &IssueDetail{
		ID:          issue.ID,
		Key:         issue.Key,
		Project:     issue.Fields.Project,
		Type:        issue.Fields.Type.Name,
		Summary:     issue.Fields.Summary,
		Status:      issue.Fields.Status.Name,
		Description: issue.Fields.Description,
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
				ID:      comment.ID,
				Author:  comment.Author.DisplayName,
				Body:    comment.Body,
				Created: comment.Created,
				Updated: comment.Updated,
			})
		}
	}

	if issue.Fields.Priority != nil {
		detail.Priority = Priority{
			Name: issue.Fields.Priority.Name,
		}
	}

	if issue.Fields.IssueLinks != nil {
		detail.IssueLinks = issue.Fields.IssueLinks
	}

	if issue.Fields.OriginalEstimate != nil {
		detail.OriginalEstimate = strconv.Itoa(*issue.Fields.OriginalEstimate)
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

	err := c.doJiraRequest(
		ctx,
		"GET",
		apiURL,
		nil,
		nil,
		&result,
		http.StatusOK,
	)

	transitions := make([]Transition, 0, len(result.Transitions))
	for _, t := range result.Transitions {
		transitions = append(transitions, Transition{
			ID:   t.ID,
			Name: t.To.Name,
		})
	}

	return transitions, err
}

func (c *Client) GetAssignableUsers(ctx context.Context, issueKey string) ([]User, error) {
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

func (c Client) GetAllUsers(ctx context.Context) ([]User, error) {
	apiURL := "/rest/api/3/users/search"
	params := url.Values{}
	params.Add("maxResults", "200")

	var result []User

	err := c.doJiraRequest(ctx,
		"GET",
		apiURL,
		params,
		nil,
		&result,
	)

	users := make([]User, len(result))
	for _, u := range result {
		users = append(users, User{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		})
	}

	return users, err
}

func (c *Client) PostAssignee(ctx context.Context, issueKey, assigneeID string) error {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s/assignee", issueKey)

	body := map[string]any{
		"accountId": assigneeID,
	}

	err := c.doJiraRequest(
		ctx,
		"PUT",
		apiURL,
		nil,
		body,
		nil,
		http.StatusNoContent,
	)

	return err
}

func (c *Client) PostTransition(ctx context.Context, issueKey, transitionID string, fields map[string]any, comment, worklogTime string) error {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey)

	body := map[string]any{
		"transition": map[string]string{
			"id": transitionID,
		},
	}

	if fields != nil {
		body["fields"] = fields
	}

	update := make(map[string]any)

	if comment != "" {
		update["comment"] = []map[string]any{
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
		}
	}

	if worklogTime != "" {
		update["worklog"] = []map[string]any{
			{
				"add": map[string]any{
					"timeSpent": worklogTime,
				},
			},
		}
	}

	if len(update) > 0 {
		body["update"] = update
	}

	err := c.doJiraRequest(
		ctx,
		"POST",
		apiURL,
		nil,
		body,
		nil,
		http.StatusNoContent,
	)

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
		http.StatusNoContent,
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
		http.StatusNoContent,
	)

	return err
}

func (c *Client) UpdateOriginalEstimate(ctx context.Context, issueKey string, estimate string) error {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s", issueKey)

	// TODO: validate estimate
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
		http.StatusNoContent,
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

func (c *Client) GetStatuses(ctx context.Context, projects []Project) ([]Status, error) {
	statuses := []Status{}
	seen := make(map[string]bool)

	for _, p := range projects {
		apiURL := fmt.Sprintf("/rest/api/3/project/%s/statuses", p.ID)

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
			return nil, fmt.Errorf("failed to get statuses for project %s: %w", p.Name, err)
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

func (c *Client) PutComment(ctx context.Context, issueKey, commentID, comment string, usersCache []User) error {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s/comment/%s", issueKey, commentID)

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
		"PUT",
		apiURL,
		nil,
		body,
		nil,
	)

	return err
}

func (c *Client) DeleteComment(ctx context.Context, issueKey, commentID string) error {
	apiURL := fmt.Sprintf("/rest/api/3/issue/%s/comment/%s", issueKey, commentID)

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

func (c *Client) GetWorkLogs(ctx context.Context, issueID string) ([]Worklog, error) {
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

func (c *Client) PostWorkLog(ctx context.Context, issueID, startDate, accountID, description string, time int) error {
	body := map[string]any{
		"authorAccountId":  accountID,
		"description":      description,
		"issueId":          issueID,
		"startDate":        startDate,
		"timeSpentSeconds": time,
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

func (c *Client) PutWorkLog(ctx context.Context, worklogID, issueID, startDate, accountID, description string, time int) error {
	apiURL := fmt.Sprintf("/4/worklogs/%s", worklogID)
	body := map[string]any{
		"authorAccountId":  accountID,
		"description":      description,
		"issueId":          issueID,
		"startDate":        startDate,
		"timeSpentSeconds": time,
	}

	err := c.doTempoRequest(
		ctx,
		"PUT",
		apiURL,
		nil,
		body,
		nil,
	)

	return err
}

func (c *Client) DeleteWorkLog(ctx context.Context, worklogID string) error {
	apiURL := fmt.Sprintf("/4/worklogs/%s", worklogID)

	err := c.doTempoRequest(
		ctx,
		"DELETE",
		apiURL,
		nil,
		nil,
		nil,
	)

	return err
}

func (c *Client) PostIssueLink(ctx context.Context, fromKey, toKey string, linkType LinkType) error {
	body := map[string]any{
		"type": map[string]string{
			"name": linkType.String(),
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

func (c *Client) PostNewIssue(
	ctx context.Context,
	projectID,
	issueTypeID,
	originalEstimate,
	summary,
	parentKey,
	assigneeID,
	priorityID string,
	description *ContentDoc,
	dueDate string,

) error {
	apiURL := "/rest/api/3/issue"

	fields := map[string]any{
		"assignee": map[string]any{
			"id": assigneeID,
		},
		"description": description,
		"duedate":     dueDate,
		"issuetype": map[string]any{
			"id": issueTypeID,
		},
		"priority": map[string]any{
			"id": priorityID,
		},
		"project": map[string]any{
			"id": projectID,
		},
		"summary": summary,
	}

	if parentKey != "" {
		fields["parent"] = map[string]any{
			"key": parentKey,
		}
	}

	body := map[string]any{
		"fields": fields,
	}

	if originalEstimate != "" {
		body["timetracking"] = map[string]any{
			"originalEstimate": originalEstimate,
		}
	}

	err := c.doJiraRequest(
		ctx,
		"POST",
		apiURL,
		nil,
		body,
		nil,
		http.StatusCreated,
	)

	return err
}
