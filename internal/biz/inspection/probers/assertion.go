package probers

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// Assertion defines a single assertion rule.
type Assertion struct {
	Name      string `json:"name"`
	Source    string `json:"source"`    // body / header
	Path      string `json:"path"`      // body: GJSON path, header: header name
	Condition string `json:"condition"` // == > >= < <= contains notcontains regexp notregexp
	Value     string `json:"value"`
}

// AssertionResult holds the outcome of a single assertion evaluation.
type AssertionResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Actual  string `json:"actual"`
	Error   string `json:"error"`
}

// EvaluateAssertions evaluates all assertions against the response body and headers.
func EvaluateAssertions(assertions []Assertion, body string, headers http.Header) []AssertionResult {
	results := make([]AssertionResult, 0, len(assertions))
	for _, a := range assertions {
		r := AssertionResult{Name: a.Name}
		actual, err := extractValue(a.Source, a.Path, body, headers)
		if err != nil {
			r.Error = err.Error()
			results = append(results, r)
			continue
		}
		r.Actual = actual
		r.Success = evaluateCondition(actual, a.Condition, a.Value)
		if !r.Success && r.Error == "" {
			r.Error = fmt.Sprintf("expected %s %s, got %s", a.Condition, a.Value, actual)
		}
		results = append(results, r)
	}
	return results
}

func extractValue(source, path, body string, headers http.Header) (string, error) {
	switch source {
	case "body":
		// Convert $.x.y style to gjson path x.y
		gjsonPath := path
		if strings.HasPrefix(gjsonPath, "$.") {
			gjsonPath = gjsonPath[2:]
		} else if strings.HasPrefix(gjsonPath, "$") {
			gjsonPath = gjsonPath[1:]
		}
		result := gjson.Get(body, gjsonPath)
		if !result.Exists() {
			return "", fmt.Errorf("path %s not found in body", path)
		}
		return result.String(), nil
	case "header":
		val := headers.Get(path)
		if val == "" {
			return "", fmt.Errorf("header %s not found", path)
		}
		return val, nil
	default:
		return "", fmt.Errorf("unknown assertion source: %s", source)
	}
}

func evaluateCondition(actual, condition, expected string) bool {
	switch condition {
	case "==":
		return actual == expected
	case "contains":
		return strings.Contains(actual, expected)
	case "notcontains":
		return !strings.Contains(actual, expected)
	case "regexp":
		re, err := regexp.Compile(expected)
		if err != nil {
			return false
		}
		return re.MatchString(actual)
	case "notregexp":
		re, err := regexp.Compile(expected)
		if err != nil {
			return false
		}
		return !re.MatchString(actual)
	case ">", ">=", "<", "<=":
		av, errA := strconv.ParseFloat(actual, 64)
		ev, errE := strconv.ParseFloat(expected, 64)
		if errA != nil || errE != nil {
			return false
		}
		switch condition {
		case ">":
			return av > ev
		case ">=":
			return av >= ev
		case "<":
			return av < ev
		case "<=":
			return av <= ev
		}
	}
	return false
}
