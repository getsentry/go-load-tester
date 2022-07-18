package utils

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"math/rand"
	"os"
)

// ProjectProvider can be used to get project Ids and keys for testing
type ProjectProvider interface {
	GetNumberOfProjects() int
	GetProjectId(maxProjects int) string
	GetProjectKey(projectId string) string
}

// ProjectApiKeyProvider is a provider of API keys for projects.
// API keys can be used to change ProjectConfigs via the ProjectDetails endpoint.
type ProjectApiKeyProvider interface {
	GetApiKey(projectId int) string
}

type ProjectProviderWithApiKey interface {
	ProjectProvider
	ProjectApiKeyProvider
}

type RandomProjectProvider struct{}

func (provider RandomProjectProvider) GetNumberOfProjects() int {
	// any number of projects
	return -1
}

func (provider RandomProjectProvider) GetProjectId(maxProjects int) string {
	return fmt.Sprintf("%d", rand.Intn(maxProjects)+1)
}

func (provider RandomProjectProvider) GetProjectKey(projectId int) string {
	tmp := fmt.Sprintf("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa%d", projectId)
	return tmp[len(tmp)-32:]
}

type projectInfo struct {
	ProjectId     string `json:"project_id,omitempty"`
	ProjectKey    string `json:"project_key"`
	ProjectApiKey string `json:"access_token,omitempty"`
}

type FileProjectProvider struct {
	projectInfo map[string]projectInfo
	projectIds  []string
}

func (provider FileProjectProvider) GetNumberOfProjects() int {
	return len(provider.projectIds)
}

func (provider FileProjectProvider) GetProjectId(maxProjects int) string {
	idx := rand.Intn(Min(maxProjects, len(provider.projectIds)))
	return provider.projectIds[idx]
}

func (provider FileProjectProvider) GetProjectKey(projectId string) string {
	return provider.projectInfo[projectId].ProjectKey
}

func (provider FileProjectProvider) GetApiKey(projectId string) string {
	return provider.projectInfo[projectId].ProjectApiKey
}

// LoadFileProjectProvider loads projects from a json file containing a list of projectId to projectKey mappings.
func LoadFileProjectProvider(filePath string) (*FileProjectProvider, error) {
	jsonFile, err := os.Open(filePath)
	if err != nil {
		log.Err(err).Msg("Failed to open project file")
		return nil, err
	}
	defer func() { _ = jsonFile.Close() }()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var projectInfos map[string]projectInfo
	err = json.Unmarshal([]byte(byteValue), &projectInfos)
	if err != nil {
		log.Err(err).Msg("Failed to parse project file")
		return nil, err
	}

	var projectIds = make([]string, 0, len(projectInfos))

	//consolidate project info ( the projectInfos were not deserialize)
	for projId, projInfo := range projectInfos {
		projInfo.ProjectId = projId
		projectIds = append(projectIds, projId)
	}

	return &FileProjectProvider{projectInfo: projectInfos, projectIds: projectIds}, nil
}
