package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"strconv"
	"sync"

	"github.com/rs/zerolog/log"
)

// ProjectFreqProfile defines the frequency relative to other profiles
// in which a project is picked.
// Example: [ {numProjects:5, relativeFreq:1}, {numProjects:3, relativeFreq:2}]
// With the example above the project provider will return 8 projects (5+3) and
// each of the 5 projects in the first group would be return 3 times less often
// than each of the 3 project in the second group.
type ProjectFreqProfile interface {
	GetNumProjects() int
	GetRelativeFreqWeight() float64
}

// projectChoiceRatio is a utility type that makes it easier to generate project
// choices, it contains the same information as ProjectFreqProfile but in a more
// convenient format (convenient for generation as opposed to specification).
type projectChoiceRatio struct {
	lastProjectIndex int
	aggregatedRatio  float64
}

// ProjectProvider can be used to get project Ids and keys for testing
type ProjectProvider interface {
	// GetNumberOfProjects returns the number of projects that can be used
	GetNumberOfProjects() int
	// GetProjectId returns a random project id
	GetProjectId(maxProjects int) string
	// GetProjectIdV2 returns a random project id weighted by the specified project
	// profiles. With this function you are able to specify that some projects may
	// be called with a greater frequency than other projects. Profiles is a list
	// of elements containing the number of projects and their relative frequency ratio
	GetProjectIdV2(profiles []ProjectFreqProfile) (string, int, error)
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

func (provider RandomProjectProvider) GetProjectIdV2(profiles []ProjectFreqProfile) (string, int, error) {
	idx, profileIdx, err := indexFromProfiles(profiles)
	if err != nil {
		return "", 0, err
	}
	return fmt.Sprintf("%d", idx), profileIdx, nil
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

func (provider FileProjectProvider) GetProjectIdV2(profiles []ProjectFreqProfile) (string, int, error) {
	numProjectsRequired := projectsRequired(profiles)
	if len(provider.projectIds) < numProjectsRequired {
		return "", 0, fmt.Errorf("not enough projects available for the requested profile. requested: %d,  available %d",
			numProjectsRequired, len(provider.projectIds))
	}
	idx, profileIdx, err := indexFromProfiles(profiles)
	if err != nil {
		return "", 0, err
	}
	// should never happen (first check should have caused this)
	if idx >= len(provider.projectIds) {
		return "", 0, errors.New("internal error, failed to calculate project index")
	}
	return provider.projectIds[idx], profileIdx, nil
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

// projectsRequired returns the number of projects required for the specified profile
func projectsRequired(profiles []ProjectFreqProfile) int {
	if profiles == nil || len(profiles) == 0 {
		return 0
	}
	retVal := 0
	for _, profile := range profiles {
		retVal += profile.GetNumProjects()
	}

	return retVal
}

// FreqProfilesToProjectChoiceWeights changes from []ProjectFreqProfile which is easier to specify
// to []ProjectChoiceRatio which is easier to work with when generating random project ids
func freqProfilesToProjectChoiceWeights(profiles []ProjectFreqProfile) []projectChoiceRatio {
	if profiles == nil || len(profiles) == 0 {
		return make([]projectChoiceRatio, 0)
	}
	retVal := make([]projectChoiceRatio, 0, len(profiles))

	lastProjectIdx := 0
	for idx := 0; idx < len(profiles); idx++ {
		numProjects := profiles[idx].GetNumProjects()
		lastProjectIdx = lastProjectIdx + numProjects
		aggregatedRatio := profiles[idx].GetRelativeFreqWeight() * float64(numProjects)
		if idx > 0 {
			aggregatedRatio += retVal[idx-1].aggregatedRatio
		}
		retVal = append(retVal, projectChoiceRatio{
			lastProjectIndex: lastProjectIdx,
			aggregatedRatio:  aggregatedRatio,
		})
	}
	return retVal
}

func indexFromProfiles(profiles []ProjectFreqProfile) (int, int, error) {
	freqProfiles := freqProfilesToProjectChoiceWeights(profiles)
	numProfiles := len(freqProfiles)

	if numProfiles == 0 {
		return 0, 0, errors.New("no profiles passed to GetProjectIdV2")
	}

	maxVal := freqProfiles[numProfiles-1].aggregatedRatio

	val := rand.Float64() * maxVal

	// find the profile that will be returned
	for idx, profile := range freqProfiles {
		if val <= profile.aggregatedRatio {
			// found the profile now return a project index in this profile
			lastProjIdx := profile.lastProjectIndex
			firstProjIdx := 0
			if idx > 0 {
				firstProjIdx = freqProfiles[idx-1].lastProjectIndex
			}
			projIdx := firstProjIdx + 1 + rand.Intn(lastProjIdx-firstProjIdx)
			return projIdx, idx, nil
		}
	}
	// should not happen (unless there's a bug)
	return 0, 0, errors.New("internal error failed to generate project index")
}
