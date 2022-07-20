package tests

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getsentry/go-load-tester/utils"
	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// ProjectConfigJob is how a projectConfigJob is parametrize
type ProjectConfigJob struct {
	// NumRelays is the number of relays to use
	NumRelays int
	// NumProjects to use in the requests
	NumProjects int
	// MinBatchSize the minimum number of project in a project config request
	MinBatchSize int
	// MaxBatchSize the maximum number of projects in a project config request
	MaxBatchSize int
	// BatchInterval is the duration of validity of a project config
	BatchInterval time.Duration
	// ProjectsInvalidated externally invalidated per second (calls to project details to invalidate the project config)
	ProjectsInvalidated int
	// InvalidatedPer is the unit of duration in which to invalidate ProjectsInvalidated
	InvalidatedPer time.Duration
}

// projectConfigLoadTester defines the state data during a ProjectConfiguration load test
type projectConfigLoadTester struct {
	// the base url for the target
	url string
	// the configuration of the attack
	config ProjectConfigJob
	// virtual relays used in the attack
	relays []virtualRelay
	// index of the next virtual Relay to be used in the attack
	nextRelayIdx int
	// lock to be used when manipulating projectConfigLoadTester (specifically nextRelayIdx)
	lock sync.Mutex
}

// represents a project id together with the last update date
type projectDate struct {
	id         string
	lastUpdate time.Time
}

// represents a virtualRelay (with the cache of pending projects and projects already in the cache)
type virtualRelay struct {
	// projects that are in the pending state (requested but not yet available, will be re-requested at first
	// opportunity)
	pendingProjects map[string]bool
	// projects that have been cached (together with the last cached date)
	cachedProjects map[string]time.Time
	// a list with the project that have been cached in the order that they have been cached
	// it is used as an easy way to find expired projects (without walking through the whole cacheProjects dict)
	cachedProjectDates list.List
	// lock for mutual exclusion during virtual Relay operations
	lock sync.Mutex
}

func newProjectConfigLoadTester(url string, rawProjectConfigParams json.RawMessage) *projectConfigLoadTester {
	var projectConfigParams ProjectConfigJob
	err := json.Unmarshal(rawProjectConfigParams, &projectConfigParams)
	if err != nil {
		log.Error().Err(err).Msgf("error unmarshalling projectConfigJob \nraw data\n%s", rawProjectConfigParams)
	}
	return projectConfigLoadTesterFromJob(projectConfigParams)
}

func projectConfigLoadTesterFromJob(job ProjectConfigJob) *projectConfigLoadTester {
	var retVal = &projectConfigLoadTester{
		config: job,
		relays: make([]virtualRelay, job.NumRelays),
	}

	for idx := 0; idx < len(retVal.relays); idx++ {
		retVal.relays[idx].InitVirtualRelay()
	}

	return retVal
}

// TODO
func (lt *projectConfigLoadTester) GetTargeter() vegeta.Targeter {
	return func(target *vegeta.Target) error {
		if target == nil {
			return vegeta.ErrNilTarget
		}
		/*
			config := lt.config
			target.Method = "POST"

			relay, err := lt.GetNextRelay()

			if err != nil {
				log.Error().Err(err).Msg("Could not get virtual relay")
				return err
			}

			//TODO redo virtual relay to work with the ProjectProvider
			relay.GetProjectsForRequest(config.NumProjects, config.BatchInterval, 111)
		*/
		//TODO finish here
		return nil
	}
}

// TODO
func (lt *projectConfigLoadTester) ProcessResult(result *vegeta.Result) {
	return
}

// projectConfigLoadSplitter divides the load for each worker by:
// 	* dividing the number of total calls per worker
// 	* dividing the number of relays per worker
// 	* dividing the number of invalidation calls per worker
func projectConfigLoadSplitter(masterParams TestParams, numWorkers int) ([]TestParams, error) {
	if numWorkers <= 0 {
		return nil, fmt.Errorf("invalid number of workers %d need at least 1", numWorkers)
	}
	// divide attack intensity among workers
	newParams := masterParams
	newParams.Per = time.Duration(numWorkers) * masterParams.Per
	var projConfigJob ProjectConfigJob
	err := json.Unmarshal(masterParams.Params, &projConfigJob)
	if err != nil {
		log.Error().Err(err).Msg("error unmarshalling projectConfigJob")
		return nil, err
	}
	splitRelays, err := utils.Divide(projConfigJob.NumRelays, numWorkers)
	if err != nil {
		log.Error().Err(err).Msg("error splitting the number of relays among workers")
		return nil, err
	}
	projConfigJob.InvalidatedPer = projConfigJob.InvalidatedPer * time.Duration(numWorkers)
	retVal := make([]TestParams, 0, numWorkers)
	for idx := 0; idx < numWorkers; idx++ {
		// distribute the relays among the workers
		projConfigJob.NumRelays = splitRelays[idx]
		newParams.Params, err = json.Marshal(projConfigJob)
		retVal = append(retVal, newParams)
	}
	return retVal, nil
}

func (lt *projectConfigLoadTester) GetNextRelay() (*virtualRelay, error) {
	if lt == nil {
		panic("null projectConfig Run")
	}
	if lt.relays == nil {
		panic("invalid projectConfigLoadTester, relays slice is nil")
	}

	lt.lock.Lock()
	defer lt.lock.Unlock()

	retVal := &lt.relays[lt.nextRelayIdx]

	lt.nextRelayIdx = (lt.nextRelayIdx + 1) % len(lt.relays)

	return retVal, nil
}

func (vr *virtualRelay) InitVirtualRelay() {
	vr.pendingProjects = make(map[string]bool)
	vr.cachedProjects = make(map[string]time.Time)
}

func NewVirtualRelay() *virtualRelay {
	retVal := new(virtualRelay)
	retVal.InitVirtualRelay()
	return retVal
}

// GetProjectsForRequest returns a list of projectIDs that should be requested next, this takes in account
// the pending projects and the cached projects.
func (vr *virtualRelay) GetProjectsForRequest(numProjects int, expiryTime time.Duration, maxNumProjects int,
	projectProvider utils.ProjectProvider) []string {

	baseProjectId := projectProvider.GetProjectId(maxNumProjects)
	return getProjectsForRequest(vr, numProjects, expiryTime, maxNumProjects, time.Now(), baseProjectId, projectProvider)
}

// getProjectsForRequest internal version of GetProjectsForRequest for testing (no time.Now or random stuff)
// function only used for testing (with injected now), normal usage should go through the struct member function
func getProjectsForRequest(vr *virtualRelay, numProjects int, expiryTime time.Duration, maxNumProjects int,
	now time.Time, baseProjectId string, provider utils.ProjectProvider) []string {

	if vr == nil {
		panic("nil virtual Relay")
	}

	vr.lock.Lock()
	defer vr.lock.Unlock()

	//cleanup expired projects (they can be queried again)
	vr.cleanExpiredProjects(expiryTime, now)

	// expected number of projects
	retVal := make([]string, 0, numProjects)

	// first add to the request the pending projects (maybe they have been resolved)
	for k := range vr.pendingProjects {
		retVal = append(retVal, k)
		if len(retVal) == numProjects {
			//enough projects for our request
			return retVal
		}
	}

	firstSuggestion := provider.GetNextProjectId(maxNumProjects, baseProjectId)
	projectId := firstSuggestion
	for len(retVal) < numProjects {
		//check the suggestion is not already in the list or in the cached projects
		if _, ok := vr.pendingProjects[projectId]; !ok {
			if _, ok := vr.cachedProjects[projectId]; !ok {
				// project id not pending and not in cache we can use it
				retVal = append(retVal, projectId)
			}
		}
		projectId = provider.GetNextProjectId(maxNumProjects, projectId)
		if projectId == firstSuggestion {
			//we have looped around the list, we can't find enough projects, return what we have
			return retVal
		}
	}
	//we have enough projects for our request
	return retVal
}

// UpdateProjectStates updates the project states with the result from a getProjects response
func (vr *virtualRelay) UpdateProjectStates(pendingProjects []string, resolvedProjects []string) {
	updateProjectStates(vr, pendingProjects, resolvedProjects, time.Now())
}

// updateProjectStates updates the list of cached projects setting their refresh date to now
// if the projects were already cached the old values are removed and the new values are inserted
// function only used for testing (with injected now), normal usage should go through the struct member function
func updateProjectStates(vr *virtualRelay, pendingProjects []string, resolvedProjects []string, now time.Time) {
	if vr == nil {
		panic("nil virtual Relay")
	}

	vr.lock.Lock()
	defer vr.lock.Unlock()

	// update the list of pending project ids
	for _, pendingProjectId := range pendingProjects {
		vr.pendingProjects[pendingProjectId] = true
	}

	for _, projectId := range resolvedProjects {
		vr.cachedProjects[projectId] = now
		vr.cachedProjectDates.PushFront(projectDate{id: projectId, lastUpdate: now})
		// remove resolved projects from the list of pending projects (since they are not pending anymore)
		delete(vr.pendingProjects, projectId)
	}
}

// cleanExpiredProjects removes all projects from the front of the queue that have been added before the
// maximum allowed time (i.e. now-expiryTime)
func (vr *virtualRelay) cleanExpiredProjects(expiryTime time.Duration, now time.Time) {
	cutoff := now.Add(-expiryTime)
	for elm := vr.cachedProjectDates.Back(); elm != nil; elm = vr.cachedProjectDates.Back() {
		val := elm.Value.(projectDate)
		if val.lastUpdate.Before(cutoff) {
			// value is too old, pop it
			// this is a candidate for delete (if there is a more recent update it will
			// have overridden the date in the map )
			lastUpdate, ok := vr.cachedProjects[val.id]
			if ok && lastUpdate.Before(cutoff) {
				delete(vr.cachedProjects, val.id)
			}
			vr.cachedProjectDates.Remove(elm)
		} else {
			return
		}
	}
}

type projectConfigJobRaw struct {
	NumRelays           int    `json:"numRelays" yaml:"numRelays"`
	MinBatchSize        int    `json:"minBatchSize" yaml:"minBatchSize"`
	MaxBatchSize        int    `json:"maxBatchSize" yaml:"maxBatchSize"`
	BatchInterval       string `json:"batchInterval" yaml:"batchInterval"`
	ProjectsInvalidated int    `json:"projectsInvalidated" yaml:"projectsInvalidated"`
	InvalidatedPer      string `json:"invalidatedPer" yaml:"invalidatedPer"`
}

func (pc projectConfigJobRaw) into(result *ProjectConfigJob) error {
	result.NumRelays = pc.NumRelays
	result.MinBatchSize = pc.MinBatchSize
	result.MaxBatchSize = pc.MaxBatchSize
	result.ProjectsInvalidated = pc.ProjectsInvalidated

	if len(pc.BatchInterval) >= 0 {
		batchInterval, err := time.ParseDuration(pc.BatchInterval)
		if err != nil {
			return err
		}
		result.BatchInterval = batchInterval
	}
	if len(pc.InvalidatedPer) >= 0 {
		per, err := time.ParseDuration(pc.InvalidatedPer)
		if err != nil {
			return err
		}
		result.InvalidatedPer = per
	}
	return nil
}

func (pcj ProjectConfigJob) intoRaw() projectConfigJobRaw {
	return projectConfigJobRaw{
		NumRelays:           pcj.NumRelays,
		MinBatchSize:        pcj.MinBatchSize,
		MaxBatchSize:        pcj.MaxBatchSize,
		BatchInterval:       pcj.BatchInterval.String(),
		ProjectsInvalidated: pcj.ProjectsInvalidated,
		InvalidatedPer:      pcj.InvalidatedPer.String(),
	}
}

func (pcj *ProjectConfigJob) UnmarshalJSON(b []byte) error {
	if pcj == nil {
		return errors.New("nil value passed as deserialization target")
	}
	var raw projectConfigJobRaw
	var err error
	if err = json.Unmarshal(b, &raw); err != nil {
		return err
	}
	return raw.into(pcj)
}

func (pcj ProjectConfigJob) MarshalJSON() ([]byte, error) {
	return json.Marshal(pcj.intoRaw())
}

func (pcj *ProjectConfigJob) UnmarshalYaml(b []byte) error {
	if pcj == nil {
		return errors.New("nil value passed as deserialization target")
	}
	var raw projectConfigJobRaw

	var err error
	if err = yaml.Unmarshal(b, &raw); err != nil {
		return err
	}
	return raw.into(pcj)
}

func (pcj ProjectConfigJob) MarshalYaml() ([]byte, error) {
	return yaml.Marshal(pcj.intoRaw())
}

func init() {
	// can we do it less ugly here?
	var loadTestBuilder LoadTesterBuilder = func(targetUrl string, params json.RawMessage) LoadTester {
		return newProjectConfigLoadTester(targetUrl, params)
	}
	RegisterTestType("projectConfig", loadTestBuilder, projectConfigLoadSplitter)
}
