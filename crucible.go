package main

import (
	"net/http"
	"net/url"
	"encoding/json"
	"fmt"
	"log"
	"errors"
	"sync"
)

type CrucibleProject struct {
	ID       string
	Location string
}

type CrucibleRepositoryList struct {
	Values []CrucibleRepository
}

type CrucibleRepository struct {
	Name string
	Git  CrucibleRepositoryGitInformation
}

type CrucibleRepositoryGitInformation struct {
	Location           string
	NormalizedLocation string
}

type CrucibleLoginResponse struct {
	Token string
}

type CrucibleSettings struct {
	ApiBaseUrl             string
	ApiKey                 string
	Username               string
	Password               string
	ProjectRefreshInterval int
	ProjectLimit           int
	muProjects sync.RWMutex
	cachedCrucibleProjects []CrucibleProject
}

func (settings *CrucibleSettings) getProjects() []CrucibleProject {
	settings.muProjects.RLock()
	defer settings.muProjects.RUnlock()
	return settings.cachedCrucibleProjects
}

func (settings *CrucibleSettings) updateCachedProjects() {
	authToken, err := login(settings.ApiBaseUrl, settings.Username, settings.Password)
	if err != nil {
		panic(err)
	}

	repositoriesUrl := fmt.Sprintf("%s/admin/repositories/?FEAUTH=%s&limit=%d", settings.ApiBaseUrl, authToken, settings.ProjectLimit)

	request, err := http.NewRequest("GET", repositoriesUrl, nil)
	if err != nil {
		panic(err)
	}
	response, err := client.Do(request)
	if err != nil {
		panic(fmt.Sprintf("downloading projects from Crucible failed: %v", err))
	}
	defer response.Body.Close()
	var b []byte
	if _, err := response.Body.Read(b); err != nil {
		panic(err)
	}

	var repositories CrucibleRepositoryList
	if err := json.NewDecoder(response.Body).Decode(&repositories); err != nil {
		panic(err)
	}

	var projects []CrucibleProject
	for _, repo := range repositories.Values {
		project := CrucibleProject{
			ID:       repo.Name,
			Location: NormalizeGitUrl(repo.Git.Location),
		}
		projects = append(projects, project)
	}
	log.Println(fmt.Sprintf("found %d projects in Crucible", len(projects)))

	settings.muProjects.Lock()
	defer settings.muProjects.Unlock()
	settings.cachedCrucibleProjects = projects
}

func login(crucibleBaseUrl string, username string, password string) (string, error) {
	authUrl := crucibleBaseUrl + "/auth/login"

	response, err := http.PostForm(authUrl, url.Values{"userName": {username}, "password": {password}})
	if err != nil {
		return "", err
	}
	if response.StatusCode == http.StatusUnauthorized {
		return "", errors.New("login failed")
	}
	var loginResponse CrucibleLoginResponse
	if err := json.NewDecoder(response.Body).Decode(&loginResponse); err != nil {
		return "", err
	}
	return loginResponse.Token, nil
}

func TriggerCrucibleSync(projectId string, client http.Client, crucible CrucibleSettings) error {

	triggerUrl := fmt.Sprintf("%s/admin/repositories/%s/incremental-index", crucible.ApiBaseUrl, projectId)

	request, err := http.NewRequest("PUT", triggerUrl, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("X-Api-Key", crucible.ApiKey)
	response, err := client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode > 299 {
		return fmt.Errorf("triggering Crucible failed: %s", response.Status)
	}
	return nil
}
