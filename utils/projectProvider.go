package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"strconv"
	"sync"

	"github.com/rs/zerolog/log"
)

// ProjectProvider can be used to get project Ids and keys for testing
type ProjectProvider interface {
	// GetNumberOfProjects returns the number of projects that can be used
	GetNumberOfProjects() int
	// GetProjectId returns a random project id
	GetProjectId(maxProjects int) string
	// GetNextProjectId returns the next project id given the last used project id
	GetNextProjectId(maxProjects int, currentProjectId string) string
	GetProjectInfo(projectId string) ProjectInfo
}

var setDefaultProvider sync.Once
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
	return fmt.Sprintf("%d", currentProjectIdInt%maxProjects+1)
}

func (provider RandomProjectProvider) GetProjectInfo(projectId string) ProjectInfo {
	tmp := fmt.Sprintf("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa%s", projectId)
	tmp = tmp[len(tmp)-32:]
	return ProjectInfo{
		ProjectId:        projectId,
		ProjectKey:       tmp,
		ProjectSlug:      fmt.Sprintf("project-%s", projectId),
		OrganizationSlug: fmt.Sprintf("organization-%s", projectId),
	}
}

type ProjectInfo struct {
	ProjectId        string `json:"project_id" yaml:"project_id"`
	ProjectKey       string `json:"project_key" yaml:"project_key"`
	ProjectApiKey    string `json:"access_token,omitempty" yaml:"access_token,omitempty"`
	ProjectSlug      string `json:"project_slug,omitempty" yaml:"project_slug,omitempty"`
	OrganizationSlug string `json:"organization_slug,omitempty" yaml:"organization_slug,omitempty"`
}

type FileProjectProvider struct {
	projectInfo    map[string]ProjectInfo
	projectIdToIdx map[string]int
	projectIds     []string
}

func (provider FileProjectProvider) GetNumberOfProjects() int {
	return len(provider.projectIds)
}

func (provider FileProjectProvider) GetProjectId(maxProjects int) string {
	idx := rand.Intn(Min(maxProjects, len(provider.projectIds)))
	return provider.projectIds[idx]
}

func (provider FileProjectProvider) GetProjectInfo(projectId string) ProjectInfo {
	return provider.projectInfo[projectId]
}

func (provider FileProjectProvider) GetNextProjectId(maxProjects int, currentProjectId string) string {
	if len(provider.projectIds) == 0 {
		return ""
	}
	if currentProjectId == "" {
		return provider.projectIds[0]
	}
	currentProjectIdx, ok := provider.projectIdToIdx[currentProjectId]
	if !ok {
		log.Error().Msgf("Unknown project id '%s' returning the first project id", currentProjectId)
		return provider.projectIds[0]
	}

	// go to the next project id, wrap around at maxProjects or the actual number of projects
	nextProjectIdx := (currentProjectIdx + 1) % Min(maxProjects, len(provider.projectIds))

	return provider.projectIds[nextProjectIdx]
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

	var projects []ProjectInfo
	err = json.Unmarshal([]byte(byteValue), &projects)
	if err != nil {
		log.Err(err).Msg("Failed to parse project file")
		return nil, err
	}

	var projectIds = make([]string, 0, len(projects))
	var projectInfos = make(map[string]ProjectInfo, len(projects))
	var projectIdToIdx = make(map[string]int, len(projects))

	// consolidate project info ( the projectInfos were not deserialize)
	for idx, projInfo := range projects {
		projId := projInfo.ProjectId
		projectInfos[projId] = projInfo
		projectIds = append(projectIds, projId)
		projectIdToIdx[projId] = idx
	}

	return &FileProjectProvider{projectInfo: projectInfos, projectIds: projectIds, projectIdToIdx: projectIdToIdx}, nil
}

func RegisterProjectProvider(projectsFileName string) error {
	if len(projectsFileName) > 0 {
		var err error
		log.Info().Msgf("Loading projects from file: %s", projectsFileName)
		projectProvider, err = LoadFileProjectProvider(projectsFileName)
		if err != nil {
			log.Error().Err(err).Msgf("Could not load projects from file: %s", projectsFileName)
			return err
		}
		log.Info().Msgf("Loaded %d projects from project file.", projectProvider.GetNumberOfProjects())
	}
	return nil
}

// GetProjectProvider returns the current project provider.
// Note: this function is not thread safe but since this is only initialised
// once at the start of the program, it should be fine.
func GetProjectProvider() ProjectProvider {
	setDefaultProvider.Do(func() {
		if projectProvider == nil {
			projectProvider = RandomProjectProvider{}
		}
	})
	return projectProvider
}
