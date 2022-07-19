package utils

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"strconv"
)

// ProjectProvider can be used to get project Ids and keys for testing
type ProjectProvider interface {
	// GetNumberOfProjects returns the number of projects that can be used
	GetNumberOfProjects() int
	// GetProjectId returns a random project id
	GetProjectId(maxProjects int) string
	// GetNextProjectId returns the next project id given the last used project id
	GetNextProjectId(maxProjects int, currentProjectId string) string
	GetProjectKey(projectId string) string
	GetApiKey(projectId string) string
}

var projectProvider ProjectProvider

type RandomProjectProvider struct{}

func (provider RandomProjectProvider) GetNumberOfProjects() int {
	// any number of projects
	return math.MaxInt - 1000 // give it a bit of space
}

func (provider RandomProjectProvider) GetProjectId(maxProjects int) string {
	return fmt.Sprintf("%d", rand.Intn(maxProjects)+1)
}

func (provider RandomProjectProvider) GetNextProjectId(maxProjects int, currentProjectId string) string {
	currentProjectIdInt, err := strconv.Atoi(currentProjectId)
	if err != nil {
		log.Error().Err(err).Msg("error parsing project id")
		return "1"
	}
	return fmt.Sprintf("%d", (currentProjectIdInt+1)%maxProjects+1)
}

func (provider RandomProjectProvider) GetProjectKey(projectId string) string {
	tmp := fmt.Sprintf("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa%d", projectId)
	return tmp[len(tmp)-32:]
}

func (provider RandomProjectProvider) GetApiKey(projectId string) string {
	return ""
}

type projectInfo struct {
	ProjectId     string `json:"project_id,omitempty"`
	ProjectKey    string `json:"project_key"`
	ProjectApiKey string `json:"access_token,omitempty"`
}

type FileProjectProvider struct {
	projectInfo   map[string]projectInfo
	nextProjectId map[string]string
	projectIds    []string
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

func (provider FileProjectProvider) GetNextProjectId(maxProjects int, currentProjectId string) string {
	return provider.nextProjectId[currentProjectId]
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
	var nextProjectIdMap = make(map[string]string)

	//consolidate project info ( the projectInfos were not deserialize)
	previousProjectId := ""
	for projId, projInfo := range projectInfos {
		projInfo.ProjectId = projId
		projectIds = append(projectIds, projId)
		if previousProjectId != "" {
			nextProjectIdMap[previousProjectId] = projId
		}
		previousProjectId = projId
	}
	if len(projectIds) > 0 {
		// wrap around
		nextProjectIdMap[previousProjectId] = projectIds[0]
	}

	return &FileProjectProvider{projectInfo: projectInfos, projectIds: projectIds, nextProjectId: nextProjectIdMap}, nil
}

func RegisterProjectProvider(projectsFileName string) {
	if len(projectsFileName) > 0 {
		var err error
		log.Info().Msgf("Loading projects from file: %s", projectsFileName)
		projectProvider, err = LoadFileProjectProvider(projectsFileName)
		if err != nil {
			log.Error().Err(err).Msgf("Could not load projects from file: %s", projectsFileName)
		}
	} else {
		projectProvider = RandomProjectProvider{}
	}
}

// GetProjectProvider returns the current project provider.
// Note: this function is not thread safe but since this is only initialised
// once at the start of the program, it should be fine.
func GetProjectProvider() ProjectProvider {
	return projectProvider
}
