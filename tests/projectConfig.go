package tests

import (
	"container/list"
	"math/rand"
	"sync"
	"time"
)

// ProjectConfigJob is how a projectConfigJob is parametrize
type ProjectConfigJob struct {
	NumRelays int `json:"numRelays" yaml:"numRelays"`
	//NumProjects the number of projects used in the attack
	NumProjects int `json:"numProjects" yaml:"numProjects"`
	//MinBatchSize the minimum number of project in a project config request
	MinBatchSize int `json:"minBatchSize" yaml:"minBatchSize"`
	//MaxBatchSize the maximum number of projects in a project config request
	MaxBatchSize int `json:"maxBatchSize" yaml:"maxBatchSize"`
}

// projectConfigRun defines the state data during a ProjectConfiguration load test
type projectConfigRun struct {
	// the configuration of the attack
	config ProjectConfigJob
	// virtual relays used in the attack
	relays []virtualRelay
	// index of the next virtual Relay to be used in the attack
	nextRelayIdx int
	// lock to be used when manipulating projectConfigRun (specifically nextRelayIdx)
	lock sync.Mutex
}

// represents a project id together with the last update date
type projectDate struct {
	id         int
	lastUpdate time.Time
}

// represents a virtualRelay (with the cache of pending projects and projects already in the cache)
type virtualRelay struct {
	// projects that are in the pending state (requested but not yet available, will be re-requested at first
	// opportunity)
	pendingProjects map[int]bool
	// projects that have been cached (together with the last cached date)
	cachedProjects map[int]time.Time
	// a list with the project that have been cached in the order that they have been cached
	// it is used as an easy way to find expired projects (without walking through the whole cacheProjects dict)
	cachedProjectDates list.List
	// lock for mutual exclusion during virtual Relay operations
	lock sync.Mutex
}

func newProjectConfigRun(config ProjectConfigJob) *projectConfigRun {
	var retVal = &projectConfigRun{
		config: config,
		relays: make([]virtualRelay, config.NumRelays),
	}

	for idx := 0; idx < len(retVal.relays); idx++ {
		retVal.relays[idx].InitVirtualRelay()
	}

	return retVal
}

func (run *projectConfigRun) GetNextRelay() (*virtualRelay, error) {
	if run == nil {
		panic("null projectConfig Run")
	}
	if run.relays == nil {
		panic("invalid projectConfigRun, relays slice is nil")
	}

	run.lock.Lock()
	defer run.lock.Unlock()

	retVal := &run.relays[run.nextRelayIdx]

	run.nextRelayIdx = (run.nextRelayIdx + 1) % len(run.relays)

	return retVal, nil
}

func (vr *virtualRelay) InitVirtualRelay() {
	vr.pendingProjects = make(map[int]bool)
	vr.cachedProjects = make(map[int]time.Time)
}

func NewVirtualRelay() *virtualRelay {
	retVal := new(virtualRelay)
	retVal.InitVirtualRelay()
	return retVal
}

func (vr *virtualRelay) GetProjectsForRequest(numProjects int, expiryTime time.Duration, maxProjId int) []int {
	return getProjectsForRequest(vr, numProjects, expiryTime, maxProjId, time.Now(), rand.Intn(maxProjId))
}

// getProjectsForRequest internal version of GetProjectsForRequest for testing (no time.Now or random stuff)
func getProjectsForRequest(vr *virtualRelay, numProjects int, expiryTime time.Duration, maxProjId int,
	now time.Time, randomBaseProjectId int) []int {
	if vr == nil {
		panic("nil virtual Relay")
	}

	vr.lock.Lock()
	defer vr.lock.Unlock()

	//cleanup expired projects (they can be queried again)
	vr.cleanExpiredProjects(expiryTime, now)

	// expected number of projects
	retVal := make([]int, 0, numProjects)

	// first add to the request the pending projects (maybe they have been resolved)
	for k := range vr.pendingProjects {
		retVal = append(retVal, k)
		if len(retVal) == numProjects {
			//enough projects for our request
			return retVal
		}
	}

	for idx := 0; idx < maxProjId; idx++ {
		projectId := (idx+randomBaseProjectId)%maxProjId + 1
		if _, ok := vr.pendingProjects[projectId]; !ok {
			if _, ok := vr.cachedProjects[projectId]; !ok {
				// project id not pending and not in cache we can use it
				retVal = append(retVal, projectId)
				//we have enough projects for our request
				if len(retVal) == numProjects {
					return retVal
				}
			}
		}
	}
	//return what we have (probably not enough project ids)
	return retVal
}

// UpdateProjectStates updates the project states with the result from a getProjects response
func (vr *virtualRelay) UpdateProjectStates(pendingProjects []int, resolvedProjects []int) {
	updateProjectStates(vr, pendingProjects, resolvedProjects, time.Now())
}

// updateProjectStates updates the list of cached projects setting their refresh date to now
// if the projects were already cached the old values are removed and the new values are inserted
func updateProjectStates(vr *virtualRelay, pendingProjects []int, resolvedProjects []int, now time.Time) {
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
