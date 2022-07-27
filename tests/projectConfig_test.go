package tests

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/getsentry/go-load-tester/utils"
)

// getNow returns a time that is always the same
func getNow() time.Time {
	return time.Date(2010, 1, 12, 10, 0, 0, 0, time.UTC)
}

func TestGetNextRelay(t *testing.T) {
	numRelays := 7
	numProjects := 100
	run := projectConfigLoadTesterFromJob(ProjectConfigJob{NumRelays: numRelays, NumProjects: numProjects, ProjectInvalidationRatio: 0.0001}, "the-url")
	var lastSeq uint64 = 0
	for idx := 0; idx < 500; idx++ {
		// test that we iterate round-robin through the available relays
		seq, _ := run.GetRequestSequence()
		if lastSeq != 0 {
			if seq != lastSeq+1 {
				t.Errorf("expected sequence %d but found %d", lastSeq+1, seq)
			}
		}
		lastSeq = seq
		relay, err := run.RelayFromSequence(seq)
		if err != nil {
			t.Errorf("failed to get next relay on loop %d", idx)
		}
		intSeq := int(seq)
		if relay != &run.relays[intSeq%numRelays] {
			t.Errorf("unexpected relay returned and loop %d", idx)
		}
	}
}

func TestGetProjectsForRequestEmptyRelay(t *testing.T) {
	vr := NewVirtualRelay()
	projectProvider := utils.RandomProjectProvider{}
	numProjects := 5
	maxProjectId := 100
	expiryTime := time.Minute * 5
	now := getNow()

	type testExpectations struct {
		base     string
		expected []string
	}

	testCases := []testExpectations{
		{"", []string{"1", "2", "3", "4", "5"}},
		{"1", []string{"2", "3", "4", "5", "6"}},
		{"1001", []string{"2", "3", "4", "5", "6"}},
		{"50", []string{"51", "52", "53", "54", "55"}},
		{"97", []string{"98", "99", "100", "1", "2"}},
	}

	for _, testCase := range testCases {
		response := getProjectsForRequest(vr, numProjects, expiryTime, maxProjectId, now, testCase.base, projectProvider)

		if diff := cmp.Diff(response, testCase.expected); diff != "" {
			t.Errorf("Unpexpected projects returned (-expect +actual)\n %s", diff)
		}
	}
}

func TestGetProjectsForRequestPendingConfigs(t *testing.T) {
	numProjects := 5
	maxProjectId := 100
	expiryTime := time.Minute * 5
	now := getNow()
	projectProvider := utils.RandomProjectProvider{}

	type testExpectations struct {
		base            string
		pendingRequests []string
		newExpected     []string
	}

	testCases := []testExpectations{
		{"", []string{"1", "3", "5"}, []string{"2", "4"}},
		// when more pending than requested are pending just return some pending projects
		{"17", []string{"2", "3", "4", "70", "71", "72"}, []string{}},
		// test it wraps right
		{"1002", []string{"4", "5", "6"}, []string{"3", "7"}},
		// rest respects the base
		{"17", []string{"70", "71", "72"}, []string{"18", "19"}},
	}

	for _, testCase := range testCases {
		vr := NewVirtualRelay()
		for _, projectId := range testCase.pendingRequests {
			vr.pendingProjects[projectId] = true
		}
		response := getProjectsForRequest(vr, numProjects, expiryTime, maxProjectId, now, testCase.base, projectProvider)

		// check that the first entries are taken from the pending
		pending := utils.Min(len(testCase.pendingRequests), numProjects)

		for idx := 0; idx < pending; idx++ {
			if _, ok := vr.pendingProjects[response[idx]]; !ok {
				t.Errorf("expected a pending request but found %s ", response[idx])
			}
		}

		// check that if there are any entries left they are taken in order from base upward
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
	now := getNow()
	projectProvider := utils.RandomProjectProvider{}

	type testExpectations struct {
		base           string
		cachedRequests []string
		expected       []string
	}

	testCases := []testExpectations{
		{"0", []string{"1", "3", "5"}, []string{"2", "4", "6", "7", "8"}},
		{"21", []string{"2", "3", "4", "21", "22", "23", "26", "31", "71"},
			[]string{"24", "25", "27", "28", "29"}},
	}

	for _, testCase := range testCases {
		vr := NewVirtualRelay()
		updateProjectStates(vr, []string{}, testCase.cachedRequests, now.Add(-time.Second))
		response := getProjectsForRequest(vr, numProjects, expiryTime, maxProjectId, now, testCase.base, projectProvider)

		if diff := cmp.Diff(response, testCase.expected); diff != "" {
			t.Errorf("Unpexpected projects returned (-expect +actual)\n %s", diff)
		}
	}

}
func TestGetProjectsForRequestWithCacheAndPending(t *testing.T) {
	numProjects := 5
	maxProjectId := 100
	expiryTime := time.Minute * 5
	now := getNow()
	projectProvider := utils.RandomProjectProvider{}

	type testExpectations struct {
		base                 string
		cachedRequests       []string
		expiredCacheRequests []string
		pendingRequests      []string
		expected             []string
	}

	testCases := []testExpectations{
		{
			base:                 "",
			cachedRequests:       []string{"1", "3", "5"},
			expiredCacheRequests: []string{"2", "4", "6"},
			pendingRequests:      []string{"80", "81", "7"},
			expected:             []string{"80", "81", "7", "2", "4"},
		},

		{
			base:                 "7",
			cachedRequests:       []string{"9", "10", "11"},
			expiredCacheRequests: []string{"7", "8", "9", "10", "12"},
			pendingRequests:      []string{"1", "80"},
			expected:             []string{"1", "80", "8", "12", "13"},
		},
	}

	for _, testCase := range testCases {
		vr := NewVirtualRelay()
		updateProjectStates(vr, []string{}, testCase.expiredCacheRequests, now.Add(-time.Hour))
		updateProjectStates(vr, testCase.pendingRequests, testCase.cachedRequests, now.Add(-time.Second))
		response := getProjectsForRequest(vr, numProjects, expiryTime, maxProjectId, now, testCase.base, projectProvider)

		expectedMap := make(map[string]bool)
		for _, projId := range testCase.expected {
			expectedMap[projId] = true
		}
		responseMap := make(map[string]bool)
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
	now := getNow()

	veryExpired := now.Add(-expiryTime * 10)
	expired := now.Add(-expiryTime - time.Second)
	recent := now.Add(-time.Minute)
	veryRecent := now.Add(-time.Second)

	type testExpectations struct {
		// veryOld expired cached projects
		veryOld []string
		// old expired cached projects
		old []string
		// new still valid cached projects
		new []string
		// verNew still valid cached projects
		veryNew []string
		// expected contents of the linked list as seen when using PopFront
		expectedListFront []string
	}

	testCases := []testExpectations{
		{
			veryOld:           []string{"1", "2", "3", "4"},
			old:               []string{"3", "4", "5", "6"},
			new:               []string{"5", "6", "7", "8"},
			veryNew:           []string{"8", "9", "10"},
			expectedListFront: []string{"10", "9", "8", "8", "7", "6", "5"},
		},
		{
			veryOld:           []string{"1", "2", "3", "4"},
			old:               []string{"3", "4", "5", "6"},
			new:               []string{},
			veryNew:           []string{},
			expectedListFront: []string{},
		},
		{
			veryOld:           []string{"1"},
			old:               []string{"4"},
			new:               []string{"5", "6", "7", "8"},
			veryNew:           []string{"10", "11", "12"},
			expectedListFront: []string{"12", "11", "10", "8", "7", "6", "5"},
		},
	}

	for _, testCase := range testCases {
		vr := NewVirtualRelay()

		updateProjectStates(vr, []string{}, testCase.veryOld, veryExpired)
		updateProjectStates(vr, []string{}, testCase.old, expired)
		updateProjectStates(vr, []string{}, testCase.new, recent)
		updateProjectStates(vr, []string{}, testCase.veryNew, veryRecent)

		vr.cleanExpiredProjects(expiryTime, now)

		expectedCacheProjects := make(map[string]bool)
		// check that we have the expected projects cached
		for _, ids := range [][]string{testCase.new, testCase.veryNew} {
			for _, id := range ids {
				expectedCacheProjects[id] = true
				if _, ok := vr.cachedProjects[id]; !ok {
					t.Errorf("Could not find project %s in cached projects", id)
				}
			}
		}
		if len(vr.cachedProjects) != len(expectedCacheProjects) {
			t.Errorf("Expected %d in cache found %d", len(expectedCacheProjects), len(vr.cachedProjects))
		}
		elm := vr.cachedProjectDates.Front()
		for _, expectedProjId := range testCase.expectedListFront {
			if elm == nil {
				t.Errorf("Expected %s in cachedProjectDates got nothing", expectedProjId)
				break
			}
			projDate := elm.Value.(projectDate)
			if projDate.id != expectedProjId {
				t.Errorf("Expecting project: %s got: %s", expectedProjId, projDate.id)
			}
			elm = elm.Next()
		}
		if elm != nil {
			t.Errorf("chached project date queue contains more elements than expected")
		}
	}
}
