package main

import (
	"net/http"
	"testing"
)

func TestNormalizeGitUrl(t *testing.T) {
	cases := []struct {
		gitUrl, normalizedUrl string
	}{
		{"http://example.com/jsmith/example", "example.com/jsmith/example"},
		{"https://example.com/jsmith/example", "example.com/jsmith/example"},
		{"git@example.com:jsmith/example.git", "example.com/jsmith/example"},
		{"", ""},
	}
	for _, c := range cases {
		got := NormalizeGitUrl(c.gitUrl)
		if got != c.normalizedUrl {
			t.Errorf("NormalizeGitUrl(%q) == %q, expected %q", c.gitUrl, got, c.normalizedUrl)
		}
	}
}

func TestValidateGitLabHeader(t *testing.T) {

	cases := []struct {
		//headerKey, headerValue string
		headers       map[string]string
		errorExpected bool
	}{
		{map[string]string{
			"X-Gitlab-Event": "System Hook",
			"X-Gitlab-Token": "valid",
		}, false},
		{map[string]string{
			"X-Gitlab-Event": "System Hook",
			"X-Gitlab-Token": "invalid",
		}, true},
		{map[string]string{
			"X-Gitlab-Event": "Something",
			"X-Gitlab-Token": "valid",
		}, true},
		{map[string]string{}, true},
	}
	for _, c := range cases {
		gitLabSettings := GitLabSettings{
			Token: "valid",
		}
		request, err := http.NewRequest("GET", "blah", nil)
		if err != nil {
			t.Error(err)
		}
		for key, value := range c.headers {
			request.Header.Add(key, value)
		}

		var headerError = ValidateGitLabHeader(request, gitLabSettings)

		if c.errorExpected != (headerError != nil) {
			t.Errorf("ValidateGitLabHeader. Expected headerError '%v', but got %q", c.errorExpected, headerError)
		}
	}
}
