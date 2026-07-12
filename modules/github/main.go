package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"server-go/common"

	"golang.org/x/oauth2"
)

const defaultAPIEndpoint = "https://api.github.com"

var oauthEndpoint = oauth2.Endpoint{
	AuthURL:   "https://github.com/login/oauth/authorize",
	TokenURL:  "https://github.com/login/oauth/access_token",
	AuthStyle: oauth2.AuthStyleInParams,
}

type apiError struct {
	Message string `json:"message"`
}

type GithubUser struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	NodeID    string `json:"node_id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

type RepositoryOwner struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	NodeID    string `json:"node_id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
}

type Repository struct {
	ID              int64            `json:"id"`
	NodeID          string           `json:"node_id"`
	Name            string           `json:"name"`
	FullName        string           `json:"full_name"`
	Description     *string          `json:"description"`
	Private         bool             `json:"private"`
	Fork            bool             `json:"fork"`
	Archived        bool             `json:"archived"`
	Disabled        bool             `json:"disabled"`
	HTMLURL         string           `json:"html_url"`
	Homepage        *string          `json:"homepage"`
	Language        *string          `json:"language"`
	StargazersCount int              `json:"stargazers_count"`
	ForksCount      int              `json:"forks_count"`
	OpenIssuesCount int              `json:"open_issues_count"`
	Owner           *RepositoryOwner `json:"owner"`
}

func ExchangeCode(code string) (*oauth2.Token, error) {
	if common.Config.Github == nil {
		return nil, errors.New("github config is missing")
	}

	conf := &oauth2.Config{
		Endpoint:     oauthEndpoint,
		RedirectURL:  common.Config.Origin + "/api/reviewdb/oauth/github",
		ClientID:     common.Config.Github.ClientID,
		ClientSecret: common.Config.Github.ClientSecret,
	}

	return conf.Exchange(context.Background(), code)
}

func GetUserInfo(accessToken string) (*GithubUser, error) {
	var user GithubUser
	if err := getJSON("/user", accessToken, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func GetRepository(owner string, repo string) (*Repository, error) {
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" {
		return nil, errors.New("invalid github repository")
	}

	var repository Repository
	if err := getJSON("/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(repo), "", &repository); err != nil {
		return nil, err
	}
	return &repository, nil
}

func GetRepositoryByID(repositoryID int64) (*Repository, error) {
	if repositoryID <= 0 {
		return nil, errors.New("invalid github repository ID")
	}

	var repository Repository
	if err := getJSON(fmt.Sprintf("/repositories/%d", repositoryID), "", &repository); err != nil {
		return nil, err
	}
	return &repository, nil
}

func getJSON(path string, accessToken string, out any) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, apiEndpoint()+path, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ReviewDB")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr apiError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Message != "" {
			return fmt.Errorf("github api error: %s", apiErr.Message)
		}
		return fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func apiEndpoint() string {
	if common.Config.Github != nil && common.Config.Github.ApiEndpoint != "" {
		return strings.TrimRight(common.Config.Github.ApiEndpoint, "/")
	}
	return defaultAPIEndpoint
}
