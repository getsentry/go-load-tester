package utils

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
)

func createTestProjectInfoFile() (string, error) {
	data := `
[
	{"project_id":"11", "project_key": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1","access_token":"abc1", "project_slug": "proj-11", "organization_slug": "org-11"},
	{"project_id":"12", "project_key": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2","access_token":"abc2", "project_slug": "proj-12", "organization_slug": "org-12"},
	{"project_id":"13", "project_key": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa3","access_token":"abc3", "project_slug": "proj-13", "organization_slug": "org-13"},
	{"project_id":"14", "project_key": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa4","access_token":"abc4", "project_slug": "proj-14", "organization_slug": "org-14"}
]
`
	f, err := os.CreateTemp("", "ProjectInfo-*.json")

	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	_, err = f.WriteString(data)
	if err != nil {
		return "", err
	}
	return f.Name(), nil
}

func deleteTestProjectInfoFile(fileName string) {
	_ = os.Remove(fileName)
}

func getProjectProviderUtil(t *testing.T) *FileProjectProvider {
	fileName, err := createTestProjectInfoFile()
	if err != nil {
		t.Fatal(err)
		return nil
	}
	defer deleteTestProjectInfoFile(fileName)

	provider, err := LoadFileProjectProvider(fileName)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	return provider
}

func TestFileProjectProviderRandomAccess(t *testing.T) {
	provider := getProjectProviderUtil(t)
	if provider == nil {
		return // error already logged
	}

	// make the test reproducible
	rand.Seed(1)

	for i := 0; i < 10; i++ {
		projectId := provider.GetProjectId(7)
		projInfo := provider.GetProjectInfo(projectId)
		key := projInfo.ProjectKey
		accessToken := projInfo.ProjectApiKey
		switch projectId {
		case "11":
			if key != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1" || accessToken != "abc1" {
				t.Fatal("ProjectId 11 is not correct")
			}
		case "12":
			if key != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2" || accessToken != "abc2" {
				t.Fatal("ProjectId 12 is not correct")
			}
		case "13":
			if key != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa3" || accessToken != "abc3" {
				t.Fatal("ProjectId 13 is not correct")
			}
		case "14":
			if key != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa4" || accessToken != "abc4" {
				t.Fatal("ProjectId 14 is not correct")
			}
		default:
			t.Fatal("ProjectId is not in the expected range")
		}
	}
}

func TestFileProjectProviderSequentialAccess(t *testing.T) {

	provider := getProjectProviderUtil(t)
	if provider == nil {
		return // error already logged
	}

	numProjects := provider.GetNumberOfProjects()

	for _, maxProjects := range []int{1, 3, 4, 5, 7} {
		currentProjectId := "11"
		wrapAround := Min(numProjects, maxProjects)
		for i := 0; i < 5; i++ {
			projectId := provider.GetNextProjectId(maxProjects, currentProjectId)
			if projectId == "" {
				t.Fatal("Empty projectId returned")
			}
			projIdNum, err := strconv.Atoi(projectId)
			if err != nil {
				t.Fatal("Invalid project id ", err)
			}
			// we start at 11 (for i == 0 ) and the next projectId would be 12, we only have 4 projects
			// we can calculate the expected next project id from this
			expectedNextProjectId := 11 + (i+1)%wrapAround
			if expectedNextProjectId != projIdNum {
				t.Fatal("Unexpected next projectId expected, got", expectedNextProjectId, projIdNum)
			}
			currentProjectId = projectId
		}
	}
}
