package tests

import (
	"container/list"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"
	"gopkg.in/yaml.v2"

	"github.com/getsentry/go-load-tester/utils"
)

// ProjectConfigJob is how a projectConfigJob is parametrized
//
// Here's an example of project config parameters:
// ```json
// {
//   "numRelays": 50,
//   "numProjects": 10000,
//   "minBatchSize": 10,
//   "maxBatchSize": 100,
//   "BatchInterval": "5s",
//   "projectInvalidationRatio": 0.001,
//   "RelayPublicKey": "ftFuDNBFm8-kPpuCuaWMio_mJAW2txCFCsaLMHn2vv0",
//   "RelayPrivateKey": "uZUtRaayN8uuuTTOjbs5EDfqWNwyDfFro6TERx6Wfhs",
//   "RelayId": "aaa12340-a123-123b-4567-0afe1f27e066",
// }
// ```
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
	// The ratio from the number of requests that are invalidation requests (should be between 0 and 1).
	ProjectInvalidationRatio float64
	// RelayPublicKey public key for Relay authentication
	RelayPublicKey string
	// RelayPrivateKey private key for Relay authentication
	RelayPrivateKey string
	// RelayId is the id of the Relay used for authentication
	RelayId string
}

type requestType int

const (
	ProjectConfigRequest     requestType = 0
	InvalidateProjectRequest requestType = 1
)

// projectConfigLoadTester defines the state data during a ProjectConfiguration load test
type projectConfigLoadTester struct {
	// the base url for the target
	url string
	// the configuration of the attack
	config ProjectConfigJob
	// virtual relays used in the attack
	relays []virtualRelay
	// keeps a request sequence (to figure out what relay was used)
	reqSequence uint64
	// keeps a count of how many invalidation requests were sent
	invalidationRequestsSent uint64
	// lock to be used when manipulating projectConfigLoadTester (specifically nextRelayIdx)
	lock sync.Mutex
	// relayPrivateKey is the private key used to sign the request
	relayPrivateKey ed25519.PrivateKey
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

// projectConfigResponse represents the response from a getProjects request
type projectConfigResponse struct {
	Pending []string                   `json:"pending"`
	Configs map[string]json.RawMessage `json:"configs"`
}

func newProjectConfigLoadTester(url string, rawProjectConfigParams json.RawMessage) *projectConfigLoadTester {
	var projectConfigParams ProjectConfigJob
	err := json.Unmarshal(rawProjectConfigParams, &projectConfigParams)
	if err != nil {
		log.Error().Err(err).Msgf("error unmarshalling projectConfigJob \nraw data\n%s", rawProjectConfigParams)
	}
	return projectConfigLoadTesterFromJob(projectConfigParams, url)
}

func projectConfigLoadTesterFromJob(job ProjectConfigJob, url string) *projectConfigLoadTester {
	var retVal = &projectConfigLoadTester{
		url:    url,
		config: job,
		relays: make([]virtualRelay, job.NumRelays),
	}

	for idx := 0; idx < len(retVal.relays); idx++ {
		retVal.relays[idx].InitVirtualRelay()
	}

	return retVal
}

type projectConfigRequest struct {
	PublicKeys []string `json:"publicKeys"`
	FullConfig bool     `json:"fullConfig"`
	NoCache    *bool    `json:"noCache,omitempty"` // not used yet
}

func (lt *projectConfigLoadTester) GetRelayPrivateKey() (ed25519.PrivateKey, error) {
	lt.lock.Lock()
	defer lt.lock.Unlock()
	if lt.relayPrivateKey != nil {
		return lt.relayPrivateKey, nil
	} else {
		var privateKey, err = utils.PrivateKeyFromString(lt.config.RelayPublicKey, lt.config.RelayPrivateKey)
		lt.relayPrivateKey = privateKey
		return privateKey, err
	}
}

func (lt *projectConfigLoadTester) GetTargeter() (vegeta.Targeter, uint64) {

	var privateKey, pkError = lt.GetRelayPrivateKey()
	var reqSequence, reqType = lt.GetRequestSequence()

	getInvalidationRequest := func(target *vegeta.Target) error {

		projectProvider := utils.GetProjectProvider()
		numProjects := projectProvider.GetNumberOfProjects()
		projectId := projectProvider.GetProjectId(numProjects)
		projectInfo := projectProvider.GetProjectInfo(projectId)
		apiKey := projectInfo.ProjectApiKey
		orgSlug := projectInfo.OrganizationSlug
		projSlug := projectInfo.ProjectSlug

		authHeader := fmt.Sprintf(" Bearer %s", apiKey)
		target.Header = make(http.Header)
		target.Header.Set("Content-Type", "application/json")
		target.Header.Set("Authorization", authHeader)

		target.Method = "POST"
		target.URL = fmt.Sprintf("%s/api/0/projects/%s/%s/", lt.url, orgSlug, projSlug)

		// generate a unique change in the project config in order to invalidate it
		body := fmt.Sprintf(`{"safeFields": ["x-%d"]}`, reqSequence)
		target.Body = []byte(body)
		return nil
	}

	getProjectRequest := func(target *vegeta.Target) error {
		if target == nil {
			return vegeta.ErrNilTarget
		}

		if pkError != nil {
			return pkError
		}

		var relay, err = lt.RelayFromSequence(reqSequence)

		if err != nil {
			log.Error().Err(err).Msg("error getting relay")
			return err
		}

		config := lt.config
		target.Method = "POST"

		url := lt.url
		if !strings.HasSuffix(url, "/") {
			url += "/"
		}
		url += "api/0/relays/projectconfigs/?version=3"

		if err != nil {
			log.Error().Err(err).Msg("Could not get virtual relay")
			return err
		}

		batchSize := config.MinBatchSize + rand.Intn(config.MaxBatchSize-config.MinBatchSize)

		projectProvider := utils.GetProjectProvider()
		projectIds := relay.GetProjectsForRequest(batchSize, config.BatchInterval, config.NumProjects, projectProvider)

		if len(projectIds) == 0 {
			return errors.New("no projects available for virtual relay")
		}

		projectKeys := make([]string, len(projectIds))
		for _, projectId := range projectIds {
			projectInfo := projectProvider.GetProjectInfo(projectId)
			projectKey := projectInfo.ProjectKey
			projectKeys = append(projectKeys, projectKey)
		}

		req := projectConfigRequest{
			PublicKeys: projectKeys,
			FullConfig: true,
		}
		body, err := json.Marshal(req)
		if err != nil {
			log.Error().Err(err).Msg("Could not marshal project config request")
			return err
		}

		now := time.Now().UTC()
		signature, err := utils.RelayAuthSign(privateKey, body, now)
		if err != nil {
			log.Error().Err(err).Msg("Could not sign request")
			return err
		}

		target.Header = make(http.Header)
		target.Header.Set("Content-Type", "application/json")
		target.Header.Set("X-Sentry-Relay-Signature", signature)
		target.Header.Set("X-Sentry-Relay-Id", config.RelayId)
		target.Body = body
		return nil
	}

	switch reqType {
	case InvalidateProjectRequest:
		return getInvalidationRequest, reqSequence
	default:
		return getProjectRequest, reqSequence
	}
}

func (lt *projectConfigLoadTester) ProcessResult(result *vegeta.Result, seq uint64) {
	var relay, err = lt.RelayFromSequence(seq)
	if err != nil {
		log.Error().Err(err).Msg("error getting relay")
		return
	}

	var configResponse projectConfigResponse
	err = json.Unmarshal(result.Body, &configResponse)
	if err != nil {
		// it's probably a project invalidation response (don't bother with it)
		return
	}

	// get all resolvedProjects from configResponse.Configs
	var resolvedProjects = make([]string, 0, len(configResponse.Configs))
	for k := range configResponse.Configs {
		resolvedProjects = append(resolvedProjects, k)
	}
	relay.UpdateProjectStates(configResponse.Pending, resolvedProjects)

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
	retVal := make([]TestParams, 0, numWorkers)
	for idx := 0; idx < numWorkers; idx++ {
		// distribute the relays among the workers
		projConfigJob.NumRelays = splitRelays[idx]
		newParams.Params, err = json.Marshal(projConfigJob)
		retVal = append(retVal, newParams)
	}
	return retVal, nil
}

func (lt *projectConfigLoadTester) GetRequestSequence() (uint64, requestType) {

	lt.lock.Lock()
	defer lt.lock.Unlock()
	lt.reqSequence++
	if float64(lt.reqSequence)*lt.config.ProjectInvalidationRatio > float64(lt.invalidationRequestsSent) {
		// we are falling behind with invalidation requests send one now.
		lt.invalidationRequestsSent++
		return lt.reqSequence, InvalidateProjectRequest
	}
	return lt.reqSequence, ProjectConfigRequest
}

func (lt *projectConfigLoadTester) RelayFromSequence(sequence uint64) (*virtualRelay, error) {
	if lt == nil {
		panic("null projectConfig Run")
	}
	if lt.relays == nil {
		panic("invalid projectConfigLoadTester, relays slice is nil")
	}

	lt.lock.Lock()
	defer lt.lock.Unlock()

	return &lt.relays[sequence%uint64(len(lt.relays))], nil
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

	// cleanup expired projects (they can be queried again)
	vr.cleanExpiredProjects(expiryTime, now)

	// expected number of projects
	retVal := make([]string, 0, numProjects)

	// first add to the request the pending projects (maybe they have been resolved)
	for k := range vr.pendingProjects {
		retVal = append(retVal, k)
		if len(retVal) == numProjects {
			// enough projects for our request
			return retVal
		}
	}

	firstSuggestion := provider.GetNextProjectId(maxNumProjects, baseProjectId)
	projectId := firstSuggestion
	for len(retVal) < numProjects {
		// check the suggestion is not already in the list or in the cached projects
		if _, ok := vr.pendingProjects[projectId]; !ok {
			if _, ok := vr.cachedProjects[projectId]; !ok {
				// project id not pending and not in cache we can use it
				retVal = append(retVal, projectId)
			}
		}
		projectId = provider.GetNextProjectId(maxNumProjects, projectId)
		if projectId == firstSuggestion {
			// we have looped around the list, we can't find enough projects, return what we have
			return retVal
		}
	}
	// we have enough projects for our request
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
	NumRelays                int     `json:"numRelays" yaml:"numRelays"`
	NumProjects              int     `json:"numProjects" yaml:"numProjects"`
	MinBatchSize             int     `json:"minBatchSize" yaml:"minBatchSize"`
	MaxBatchSize             int     `json:"maxBatchSize" yaml:"maxBatchSize"`
	BatchInterval            string  `json:"batchInterval" yaml:"batchInterval"`
	ProjectInvalidationRatio float64 `json:"projectInvalidationRatio" yaml:"projectInvalidationRatio"`
	RelayPublicKey           string  `json:"relayPublicKey" yaml:"relayPublicKey"`
	RelayPrivateKey          string  `json:"relayPrivateKey" yaml:"relayPrivateKey"`
	RelayId                  string  `json:"relayId" yaml:"relayId"`
}

func (pc projectConfigJobRaw) into(result *ProjectConfigJob) error {
	result.NumRelays = pc.NumRelays
	result.NumProjects = pc.NumProjects
	result.MinBatchSize = pc.MinBatchSize
	result.MaxBatchSize = pc.MaxBatchSize
	result.ProjectInvalidationRatio = pc.ProjectInvalidationRatio
	result.RelayPublicKey = pc.RelayPublicKey
	result.RelayPrivateKey = pc.RelayPrivateKey
	result.RelayId = pc.RelayId

	if len(pc.BatchInterval) >= 0 {
		batchInterval, err := time.ParseDuration(pc.BatchInterval)
		if err != nil {
			return err
		}
		result.BatchInterval = batchInterval
	}
	return nil
}

func (pcj ProjectConfigJob) intoRaw() projectConfigJobRaw {
	return projectConfigJobRaw{
		NumRelays:                pcj.NumRelays,
		NumProjects:              pcj.NumProjects,
		MinBatchSize:             pcj.MinBatchSize,
		MaxBatchSize:             pcj.MaxBatchSize,
		BatchInterval:            pcj.BatchInterval.String(),
		ProjectInvalidationRatio: pcj.ProjectInvalidationRatio,
		RelayPublicKey:           pcj.RelayPublicKey,
		RelayPrivateKey:          pcj.RelayPrivateKey,
		RelayId:                  pcj.RelayId,
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
