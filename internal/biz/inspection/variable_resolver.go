package inspection

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var variablePattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

// VariableResolver resolves {{name}} references in probe config fields.
type VariableResolver struct {
	variableRepo ProbeVariableRepo
}

func NewVariableResolver(repo ProbeVariableRepo) *VariableResolver {
	return &VariableResolver{variableRepo: repo}
}

// ExtractVariableNames extracts all {{name}} references from the given texts.
func ExtractVariableNames(texts ...string) []string {
	seen := make(map[string]bool)
	var names []string
	for _, text := range texts {
		matches := variablePattern.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			name := m[1]
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}

// Resolve replaces all {{name}} references in text with variable values.
func (r *VariableResolver) Resolve(ctx context.Context, text string, allowedGroupIDs []uint) (string, error) {
	names := ExtractVariableNames(text)
	if len(names) == 0 {
		return text, nil
	}
	vars, err := r.variableRepo.GetByNames(ctx, names, allowedGroupIDs)
	if err != nil {
		return "", fmt.Errorf("query variables: %w", err)
	}
	varMap := make(map[string]string, len(vars))
	for _, v := range vars {
		varMap[v.Name] = v.Value
	}
	result := variablePattern.ReplaceAllStringFunc(text, func(match string) string {
		name := match[2 : len(match)-2]
		if val, ok := varMap[name]; ok {
			return val
		}
		return match // keep unresolved
	})
	return result, nil
}

// ResolveConfig resolves variable references in a ProbeConfig, returning a shallow copy.
func (r *VariableResolver) ResolveConfig(ctx context.Context, cfg *ProbeConfig) (*ProbeConfig, error) {
	texts := []string{cfg.Target, cfg.URL, cfg.Headers, cfg.Params, cfg.Body, cfg.ProxyURL}
	names := ExtractVariableNames(texts...)
	if len(names) == 0 {
		return cfg, nil
	}

	// Parse config's GroupIDs as allowed scope
	allowedGroupIDs := parseGroupIDs(cfg.GroupIDs)

	vars, err := r.variableRepo.GetByNames(ctx, names, allowedGroupIDs)
	if err != nil {
		return nil, fmt.Errorf("query variables: %w", err)
	}
	varMap := make(map[string]string, len(vars))
	for _, v := range vars {
		varMap[v.Name] = v.Value
	}

	replacer := func(text string) string {
		return variablePattern.ReplaceAllStringFunc(text, func(match string) string {
			name := match[2 : len(match)-2]
			if val, ok := varMap[name]; ok {
				return val
			}
			return match
		})
	}

	// Shallow copy
	resolved := *cfg
	resolved.Target = replacer(cfg.Target)
	resolved.URL = replacer(cfg.URL)
	resolved.Headers = replacer(cfg.Headers)
	resolved.Params = replacer(cfg.Params)
	resolved.Body = replacer(cfg.Body)
	resolved.ProxyURL = replacer(cfg.ProxyURL)
	return &resolved, nil
}

// ResolveText replaces all {{name}} references in text. extraVars take priority over system variables.
func (r *VariableResolver) ResolveText(ctx context.Context, text string, extraVars map[string]string, allowedGroupIDs []uint) (string, error) {
	names := ExtractVariableNames(text)
	if len(names) == 0 {
		return text, nil
	}
	// Only query system variables for names not in extraVars
	var sysNames []string
	for _, n := range names {
		if _, ok := extraVars[n]; !ok {
			sysNames = append(sysNames, n)
		}
	}
	varMap := make(map[string]string, len(names))
	if len(sysNames) > 0 && r.variableRepo != nil {
		vars, err := r.variableRepo.GetByNames(ctx, sysNames, allowedGroupIDs)
		if err != nil {
			return "", fmt.Errorf("query variables: %w", err)
		}
		for _, v := range vars {
			varMap[v.Name] = v.Value
		}
	}
	// extraVars override system variables
	for k, v := range extraVars {
		varMap[k] = v
	}
	result := variablePattern.ReplaceAllStringFunc(text, func(match string) string {
		name := match[2 : len(match)-2]
		if val, ok := varMap[name]; ok {
			return val
		}
		return match
	})
	return result, nil
}

// ResolveMap resolves all {{name}} references in map values.
func (r *VariableResolver) ResolveMap(ctx context.Context, m map[string]string, extraVars map[string]string, allowedGroupIDs []uint) (map[string]string, error) {
	if len(m) == 0 {
		return m, nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		resolved, err := r.ResolveText(ctx, v, extraVars, allowedGroupIDs)
		if err != nil {
			return nil, err
		}
		result[k] = resolved
	}
	return result, nil
}

func parseGroupIDs(s string) []uint {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	ids := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if id, err := strconv.ParseUint(p, 10, 64); err == nil && id > 0 {
			ids = append(ids, uint(id))
		}
	}
	return ids
}
