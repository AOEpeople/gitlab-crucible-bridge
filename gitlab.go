package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type GitLabHook struct {
	EventName string            `json:"event_name"`
	Project   GitLabHookProject `json:"project"`
}

type GitLabHookProject struct {
	WebUrl string `json:"web_url"`
}

type GitLabSettings struct {
	Token     string
	HostNames []string
}

func NormalizeGitUrl(url string, gitLabHostNames []string) string {
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "ssh://")
	url = strings.TrimPrefix(url, "git@")
	if len(gitLabHostNames) > 1 {
		canonicalHostName := gitLabHostNames[0]
		for _, altHostName := range gitLabHostNames[1:] {
			url = strings.Replace(url, altHostName, canonicalHostName, -1)
		}
	}
	url = strings.TrimSuffix(url, ".git")
	url = strings.Replace(url, ":", "/", -1)
	return url
}

func ValidateGitLabHeader(request *http.Request, gitLabSettings GitLabSettings) error {
	eventHeader := request.Header.Get("X-Gitlab-Event")

	if eventHeader != "System Hook" {
		return errors.New("no valid GitLab Hook Header found")
	}
	tokenHeader := request.Header.Get("X-Gitlab-Token")

	if tokenHeader != gitLabSettings.Token {
		return fmt.Errorf("invalid GitLab token: %v", tokenHeader)
	}

	return nil
}

func GetNormalizedGitUrlFromRequest(request *http.Request, gitLabSettings GitLabSettings) (string, error) {
	headerError := ValidateGitLabHeader(request, gitLabSettings)
	if headerError != nil {
		return "", headerError
	}

	var gitLabHook GitLabHook
	decoderError := json.NewDecoder(request.Body).Decode(&gitLabHook)
	if decoderError != nil {
		return "", fmt.Errorf("could not read body: %v", decoderError)
	}

	return NormalizeGitUrl(gitLabHook.Project.WebUrl, gitLabSettings.HostNames), nil
}
