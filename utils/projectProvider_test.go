package utils

import (
	"math/rand"
	"os"
	"testing"
)

func createTestProjectInfoFile() (string, error) {
	data := `
{
	"11": {"project_key": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1","access_token":"abc1"},
	"12": {"project_key": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2","access_token":"abc2"},
	"13": {"project_key": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa3","access_token":"abc3"},
	"14": {"project_key": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa4","access_token":"abc4"}
}
`
	f, err := os.CreateTemp("", "projectInfo-*.json")

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
		key := provider.GetProjectKey(projectId)
		accessToken := provider.GetApiKey(projectId)
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

	projectIds := map[string]bool{}
	currentProjectId := ""
	// iterate through more than the length
	for i := 0; i < 6; i++ {
		projectId := provider.GetNextProjectId(7, currentProjectId)
		if projectId == "" {
			t.Fatal("Empty projectId returned")
		}
		projectIds[projectId] = true
		currentProjectId = projectId
	}
	// check we iterated through all projects and cycled back to the first one
	if len(projectIds) != 4 {
		t.Fatal("Not all projects were returned expected 4 got", len(projectIds))
	}
}
