package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type CrucibleRepositoryList struct {
	Start    uint32
	Size     uint32
	LastPage bool `json:"lastPage"`
	Values   []CrucibleRepository
}

type CrucibleRepository struct {
	Name string
	Git  CrucibleRepositoryGitInformation
}

type CrucibleRepositoryGitInformation struct {
	Location           string
	NormalizedLocation string
}

type CrucibleSettings struct {
	ApiBaseUrl             string
	ApiKey                 string
	Username               string
	Password               string
	ProjectRefreshInterval int
	ProjectLimit           int
}

type CrucibleRepositoriesCache struct {
	mutex        sync.RWMutex
	repositories map[string]string
}

func (cache *CrucibleRepositoriesCache) getRepositoriesCount() int {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	return len(cache.repositories)
}

func (cache *CrucibleRepositoriesCache) isEmpty() bool {
	return cache.getRepositoriesCount() == 0
}

func (cache *CrucibleRepositoriesCache) updateFactory(settings CrucibleSettings, normalizeGitUrl func(string) string) func() {
	return func() {
		var repositoriesMap = make(map[string]string)
		var start uint32
		var lastPage = false

		for !lastPage {
			repositories := settings.getCrucibleRepositories(start)

			lastPage = repositories.LastPage
			start = repositories.Start + repositories.Size

			for _, repo := range repositories.Values {
				normalizedUrl := normalizeGitUrl(repo.Git.Location)
				repositoriesMap[normalizedUrl] = repo.Name
			}
		}
		log.Printf("found %d repositories in Crucible\n", len(repositoriesMap))

		cache.mutex.Lock()
		defer cache.mutex.Unlock()
		cache.repositories = repositoriesMap
	}
}

func (cache *CrucibleRepositoriesCache) getRepositoryName(url string) string {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	return cache.repositories[url]
}

func (settings *CrucibleSettings) getCrucibleRepositories(start uint32) CrucibleRepositoryList {
	repositoriesUrl := fmt.Sprintf("%s/admin/repositories/?start=%d&limit=%d", settings.ApiBaseUrl, start, settings.ProjectLimit)

	request, err := http.NewRequest("GET", repositoriesUrl, nil)
	if err != nil {
		panic(err)
	}
	request.SetBasicAuth(settings.Username, settings.Password)
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
		fmt.Println(string(b))
		panic(err)
	}
	return repositories
}

func TriggerCrucibleSync(projectId string, client http.Client, crucible *CrucibleSettings) error {

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
