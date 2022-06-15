package tests

import (
	"github.com/getsentry/go-load-tester/utils"
	"github.com/google/go-cmp/cmp"
	"testing"
	"time"
)

func TestGetNextRelay(t *testing.T) {
	numRelays := 7
	numProjects := 100
	run := newProjectConfigRun(ProjectConfigJob{NumRelays: numRelays, NumProjects: numProjects})
	for idx := 0; idx < 500; idx++ {
		//test that we iterate round-robin through the available relays
		relay, err := run.GetNextRelay()
		if err != nil {
			t.Errorf("failed to get next relay on loop %d", idx)
		}
		if relay != &run.relays[idx%numRelays] {
			t.Errorf("unexpected relay returned and loop %d", idx)
		}
	}
}

func TestGetProjectsForRequestEmptyRelay(t *testing.T) {
	vr := NewVirtualRelay()
	numProjects := 5
	maxProjectId := 100
	expiryTime := time.Minute * 5
	now := time.Date(2010, 1, 12, 10, 0, 0, 0, time.UTC)

	type testExpectations struct {
		base     int
		expected []int
	}

	testCases := []testExpectations{
		{0, []int{1, 2, 3, 4, 5}},
		{1, []int{2, 3, 4, 5, 6}},
		{1001, []int{2, 3, 4, 5, 6}},
		{50, []int{51, 52, 53, 54, 55}},
		{97, []int{98, 99, 100, 1, 2}},
	}

	for _, testCase := range testCases {
		response := getProjectsForRequest(vr, numProjects, expiryTime, maxProjectId, now, testCase.base)

		if diff := cmp.Diff(response, testCase.expected); diff != "" {
			t.Errorf("Unpexpected projects returned (-expect +actual)\n %s", diff)
		}
	}
}

func TestGetProjectsForRequestPendingConfigs(t *testing.T) {
	numProjects := 5
	maxProjectId := 100
	expiryTime := time.Minute * 5
	now := time.Date(2010, 1, 12, 10, 0, 0, 0, time.UTC)

	type testExpectations struct {
		base            int
		pendingRequests []int
		newExpected     []int
	}

	testCases := []testExpectations{
		{0, []int{1, 3, 5}, []int{2, 4}},
		// when more pending than requested are pending just return some pending projects
		{17, []int{2, 3, 4, 70, 71, 72}, []int{}},
		// test it wraps right
		{1002, []int{4, 5, 6}, []int{3, 7}},
		// rest respects the base
		{17, []int{70, 71, 72}, []int{18, 19}},
	}

	for _, testCase := range testCases {
		vr := NewVirtualRelay()
		for _, projectId := range testCase.pendingRequests {
			vr.pendingProjects[projectId] = true
		}
		response := getProjectsForRequest(vr, numProjects, expiryTime, maxProjectId, now, testCase.base)

		//check that the first entries are taken from the pending
		pending := utils.Min(len(testCase.pendingRequests), numProjects)

		for idx := 0; idx < pending; idx++ {
			if _, ok := vr.pendingProjects[response[idx]]; !ok {
				t.Errorf("expected a pending request but found %d ", response[idx])
			}
		}

		//check that if there are any entries left they are taken in order from base upward
		rest := response[pending:]

		if diff := cmp.Diff(rest, testCase.newExpected); diff != "" {
			t.Errorf("Unpexpected projects returned (-expect +actual)\n %s", diff)
		}
	}
}

func TestGetProjectsForRequestWithCache(t *testing.T) {
	numProjects := 5
	maxProjectId := 100
	expiryTime := time.Minute * 5
	now := time.Date(2010, 1, 12, 10, 0, 0, 0, time.UTC)

	type testExpectations struct {
		base           int
		cachedRequests []int
		expected       []int
	}

	testCases := []testExpectations{
		{0, []int{1, 3, 5}, []int{2, 4, 6, 7, 8}},
		{21, []int{2, 3, 4, 21, 22, 23, 26, 31, 71}, []int{24, 25, 27, 28, 29}},
	}

	for _, testCase := range testCases {
		vr := NewVirtualRelay()
		updateProjectStates(vr, []int{}, testCase.cachedRequests, now.Add(-time.Second))
		response := getProjectsForRequest(vr, numProjects, expiryTime, maxProjectId, now, testCase.base)

		if diff := cmp.Diff(response, testCase.expected); diff != "" {
			t.Errorf("Unpexpected projects returned (-expect +actual)\n %s", diff)
		}
	}

}
func TestGetProjectsForRequestWithCacheAndPending(t *testing.T) {
	numProjects := 5
	maxProjectId := 100
	expiryTime := time.Minute * 5
	now := time.Date(2010, 1, 12, 10, 0, 0, 0, time.UTC)

	type testExpectations struct {
		base                 int
		cachedRequests       []int
		expiredCacheRequests []int
		pendingRequests      []int
		expected             []int
	}

	testCases := []testExpectations{
		{
			base:                 0,
			cachedRequests:       []int{1, 3, 5},
			expiredCacheRequests: []int{2, 4, 6},
			pendingRequests:      []int{80, 81, 7},
			expected:             []int{80, 81, 7, 2, 4},
		},

		{
			base:                 7,
			cachedRequests:       []int{9, 10, 11},
			expiredCacheRequests: []int{7, 8, 9, 10, 12},
			pendingRequests:      []int{1, 80},
			expected:             []int{1, 80, 8, 12, 13},
		},
	}

	for _, testCase := range testCases {
		vr := NewVirtualRelay()
		updateProjectStates(vr, []int{}, testCase.expiredCacheRequests, now.Add(-time.Hour))
		updateProjectStates(vr, testCase.pendingRequests, testCase.cachedRequests, now.Add(-time.Second))
		response := getProjectsForRequest(vr, numProjects, expiryTime, maxProjectId, now, testCase.base)

		expectedMap := make(map[int]bool)
		for _, projId := range testCase.expected {
			expectedMap[projId] = true
		}
		responseMap := make(map[int]bool)
		for _, projId := range response {
			responseMap[projId] = true
		}

		if diff := cmp.Diff(responseMap, expectedMap); diff != "" {
			t.Errorf("Unpexpected projects returned (-expect +actual)\n %s", diff)
		}
	}
}

func TestCleanExpiredProjects(t *testing.T) {
	expiryTime := time.Minute * 5
	now := time.Date(2010, 1, 12, 10, 0, 0, 0, time.UTC)

	veryExpired := now.Add(-expiryTime * 10)
	expired := now.Add(-expiryTime - time.Second)
	recent := now.Add(-time.Minute)
	veryRecent := now.Add(-time.Second)

	type testExpectations struct {
		// veryOld expired cached projects
		veryOld []int
		// old expired cached projects
		old []int
		// new still valid cached projects
		new []int
		// verNew still valid cached projects
		veryNew []int
		// expected contents of the linked list as seen when using PopFront
		expectedListFront []int
		// total number of cached projects (after clean)
		expectedCached int
	}

	testCases := []testExpectations{
		{
			veryOld:           []int{1, 2, 3, 4},
			old:               []int{3, 4, 5, 6},
			new:               []int{5, 6, 7, 8},
			veryNew:           []int{8, 9, 10},
			expectedCached:    6,
			expectedListFront: []int{10, 9, 8, 8, 7, 6, 5},
		},
		{
			veryOld:           []int{1, 2, 3, 4},
			old:               []int{3, 4, 5, 6},
			new:               []int{},
			veryNew:           []int{},
			expectedListFront: []int{},
			expectedCached:    0,
		},
		{
			veryOld:           []int{1},
			old:               []int{4},
			new:               []int{5, 6, 7, 8},
			veryNew:           []int{10, 11, 12},
			expectedListFront: []int{12, 11, 10, 8, 7, 6, 5},
			expectedCached:    7,
		},
	}

	for _, testCase := range testCases {
		vr := NewVirtualRelay()

		updateProjectStates(vr, []int{}, testCase.veryOld, veryExpired)
		updateProjectStates(vr, []int{}, testCase.old, expired)
		updateProjectStates(vr, []int{}, testCase.new, recent)
		updateProjectStates(vr, []int{}, testCase.veryNew, veryRecent)

		vr.cleanExpiredProjects(expiryTime, now)

		for _, ids := range [][]int{testCase.new, testCase.veryNew} {
			for _, id := range ids {
				if _, ok := vr.cachedProjects[id]; !ok {
					t.Errorf("Could not find project %d in cached projects", id)
				}
			}
			if len(vr.cachedProjects) != testCase.expectedCached {
				t.Errorf("Expected %d in cache found %d", testCase.expectedCached, len(vr.cachedProjects))
			}
		}
		for _, expectedProjId := range testCase.expectedListFront {
			projDate, err := vr.cachedProjectDates.PopFront()
			if err != nil {
				t.Errorf("Expecting %d in cachedProjectDates got error:\n%v", expectedProjId, err)
			}
			if projDate.id != expectedProjId {
				t.Errorf("Expecting project: %d got: %d", expectedProjId, projDate.id)
			}
		}
	}
}
