[![Actions Status](https://github.com/AOEpeople/gitlab-crucible-bridge/workflows/Test%20and%20build%20Docker%20image/badge.svg)](https://github.com/AOEpeople/gitlab-crucible-bridge/actions)
| [Docker Image](https://hub.docker.com/r/aoepeople/gitlab-crucible-bridge/)
| [Binaries](https://github.com/AOEpeople/gitlab-crucible-bridge/releases)
# GitLab Crucible Bridge
This project provided some glue code to trigger a SCM refresh in Crucible via GitLab webhooks.

## Why?
To trigger an SCM refresh in Crucible you need the Crucible project ID.
GitLab sends the git url within its webhooks.
To be able to leverage GitLab webhooks as a trigger for refreshing Crucible we need to convert between git urls and Crucible project IDs.
This tool periodically (with configurable time periods) downloads a project list from Crucible, normalizes the git url (to be able to handle http(s) and ssh urls) and uses that list whenever a GitLab webhook comes in to trigger a refresh. 

## Why do I need an API key and credentials?
The app uses the `incremental-index` endpoint in Crucible which is the only endpoint which can be accessed via API key for now. For downloading the projects list we need to use real credentials.

# Compatibility
Tested with:
* GitLab 10.2.x
* Crucible 4.2.0

# Configuration
## GitLab itself
You can use GitLab's "System Hooks". There is no special configuration in each GitLab repository needed.
All you have to do is to configure a "System Hook" for **Push events** or **Tag push events** and provide a **Secret Token**.

## This application

**All of the following settings are required!**

| Environment Variable | Description | Example |
| -------------------- | ----------- | ------- |
|`CRUCIBLE_API_BASE_URL`|The base url to the REST service endpoint|`https://crucible.example.com/cru/rest-service-fecru`|
|`CRUCIBLE_API_KEY`|The API key which is used for triggering a refresh in Crucible. Look at the [Crucible documentation](https://confluence.atlassian.com/fisheye/setting-the-rest-api-token-317197023.html) on how to generate that.|XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX|
|`CRUCIBLE_USERNAME`|Username of a Crucible user which can access the project list|username|
|`CRUCIBLE_PASSWORD`|Password of a Crucible user which can access the project list|password|
|`CRUCIBLE_PROJECT_REFRESH_INTERVAL`|How often the repository list should be refreshed. (In minutes)|60|
|`CRUCIBLE_PROJECT_LIMIT`|Limit of how many projects should be fetched from Crucible in one request.|100|
|`GITLAB_TOKEN`|A token in the GitLab webhook which will be used for validation in the app.|some_token|
