package cmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetDocAnnotation(t *testing.T) {

	testCases := []struct {
		docString      string
		expectedResult map[string]string
	}{
		{
			docString:      `//abc @doc(hello world)`,
			expectedResult: map[string]string{"data": "hello world"},
		},
		{
			docString:      `@doc("hello world")`,
			expectedResult: map[string]string{"data": "hello world"},
		},
		{
			docString:      `@doc({"scope": "hello", "x":"world"})`,
			expectedResult: map[string]string{"scope": "hello", "x": "world"},
		},
	}

	for _, testCase := range testCases {
		result := getDocAnnotation(testCase.docString)
		if diff := cmp.Diff(testCase.expectedResult, result); diff != "" {
			t.Errorf("Failed to get doc annotation (-expect +actual)\n %s", diff)
		}
	}
}

func TestRemoveDocAnnotation(t *testing.T) {

	testCases := []struct {
		docString      string
		expectedResult string
	}{
		{
			docString:      "//abc @doc(hello world)",
			expectedResult: "//abc ",
		},
		{
			docString:      "@doc(hello world)",
			expectedResult: "",
		},
		{
			docString:      "@doc(hello world) def",
			expectedResult: " def",
		},
		{
			docString:      "abc-@doc(hello world)-def",
			expectedResult: "abc--def",
		},
		{
			docString:      "abc-@doc(hello world)-def-@doc(hello world)-ghi",
			expectedResult: "abc--def--ghi",
		},
		{
			docString:      "abc",
			expectedResult: "abc",
		},
	}

	for _, testCase := range testCases {
		result := removeDocAnnotation(testCase.docString)
		if result != testCase.expectedResult {
			t.Errorf("failed to remove annotation exptected=%s, got=%s", result, testCase.expectedResult)
		}
	}
}
