package intermediate

import (
	"fmt"
	"sort"
	"strings"
)

type PhysicalNameSet struct {
	names map[string]string
}

func NewPhysicalNameSet(candidates []string) *PhysicalNameSet {
	if len(candidates) == 0 {
		return nil
	}

	set := &PhysicalNameSet{names: make(map[string]string)}
	for _, candidate := range candidates {
		addPhysicalCandidate(set.names, candidate)
	}

	if len(set.names) == 0 {
		return nil
	}

	return set
}

func addPhysicalCandidate(dest map[string]string, name string) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return
	}

	canonical := CanonicalIdentifier(trimmed)
	if canonical == "" {
		return
	}

	if _, exists := dest[canonical]; !exists {
		dest[canonical] = trimmed
	}

	if idx := strings.Index(canonical, "."); idx > 0 && idx < len(canonical)-1 {
		short := canonical[idx+1:]
		if _, exists := dest[short]; !exists {
			dest[short] = trimmed
		}
	}
}

func (s *PhysicalNameSet) Resolve(name string) (string, bool) {
	if s == nil {
		return "", false
	}

	canonical := CanonicalIdentifier(name)
	if canonical == "" {
		return "", false
	}

	resolved, ok := s.names[canonical]
	if !ok {
		return "", false
	}

	return strings.TrimSpace(resolved), true
}

type planTableDescriber struct {
	mapping  map[string]TableReferenceInfo
	byQuery  map[string][]TableReferenceInfo
	physical *PhysicalNameSet
}

func newPlanTableDescriber(mapping map[string]TableReferenceInfo, physicalNames []string) *planTableDescriber {
	describer := &planTableDescriber{
		mapping:  mapping,
		physical: NewPhysicalNameSet(physicalNames),
		byQuery:  make(map[string][]TableReferenceInfo),
	}

	if len(mapping) == 0 {
		return describer
	}

	unique := make(map[string]TableReferenceInfo)

	for _, ref := range mapping {
		key := planReferenceKey(ref)
		if _, exists := unique[key]; exists {
			continue
		}

		unique[key] = ref
	}

	for _, ref := range unique {
		if q := CanonicalIdentifier(ref.QueryName); q != "" {
			describer.byQuery[q] = append(describer.byQuery[q], ref)
		}
	}

	return describer
}

func planReferenceKey(ref TableReferenceInfo) string {
	parts := []string{
		CanonicalIdentifier(ref.Name),
		CanonicalIdentifier(ref.Alias),
		CanonicalIdentifier(ref.TableName),
		CanonicalIdentifier(ref.QueryName),
		strings.ToLower(strings.TrimSpace(ref.Context)),
	}

	return strings.Join(parts, "|")
}

func DescribePlanTables(aliases []string, mapping map[string]TableReferenceInfo, physicalNames []string) []string {
	describer := newPlanTableDescriber(mapping, physicalNames)
	return describer.describeAll(aliases)
}

func (d *planTableDescriber) describeAll(aliases []string) []string {
	if len(aliases) == 0 {
		return nil
	}

	results := make([]string, 0, len(aliases))
	seen := make(map[string]struct{})

	for _, alias := range aliases {
		for _, desc := range d.describe(alias) {
			if desc == "" {
				continue
			}

			if _, exists := seen[desc]; exists {
				continue
			}

			results = append(results, desc)
			seen[desc] = struct{}{}
		}
	}

	return results
}

func (d *planTableDescriber) describe(alias string) []string {
	trimmed := strings.TrimSpace(alias)
	if trimmed == "" {
		return []string{"table '<unknown>' (physical table unresolved)"}
	}

	canonical := CanonicalIdentifier(trimmed)
	if canonical != "" {
		if ref, ok := d.mapping[canonical]; ok {
			return d.describeFromReference(trimmed, ref)
		}
	}

	if resolved, ok := d.resolvePhysical(trimmed); ok {
		return []string{fmt.Sprintf("table '%s'", resolved)}
	}

	return []string{fmt.Sprintf("table '%s' (physical table unresolved)", trimmed)}
}

func (d *planTableDescriber) resolvePhysical(name string) (string, bool) {
	if d.physical == nil {
		return "", false
	}

	return d.physical.Resolve(name)
}

func (d *planTableDescriber) describeFromReference(alias string, ref TableReferenceInfo) []string {
	targets := d.collectPhysicalTargets(alias, ref)
	if len(targets) == 0 {
		return []string{d.unresolvedDescription(alias, ref)}
	}

	context := classifyContext(ref.Context)
	aliasTrimmed := strings.TrimSpace(alias)
	aliasCanonical := CanonicalIdentifier(aliasTrimmed)

	descriptions := make([]string, 0, len(targets))
	for _, target := range targets {
		physical := strings.TrimSpace(target)
		if physical == "" {
			continue
		}

		targetCanonical := CanonicalIdentifier(physical)
		if context == "" {
			descriptions = append(descriptions, fmt.Sprintf("table '%s'", physical))
			continue
		}

		if aliasTrimmed != "" && targetCanonical == aliasCanonical {
			switch context {
			case "CTE", "subquery":
				continue
			case "join":
				descriptions = append(descriptions, fmt.Sprintf("table '%s' (%s)", physical, context))
			default:
				descriptions = append(descriptions, fmt.Sprintf("table '%s'", physical))
			}

			continue
		}

		switch context {
		case "CTE", "subquery":
			if aliasTrimmed == "" || targetCanonical == aliasCanonical {
				descriptions = append(descriptions, fmt.Sprintf("table '%s' (%s)", physical, context))
			} else {
				descriptions = append(descriptions, fmt.Sprintf("table '%s' in '%s'(%s)", physical, aliasTrimmed, context))
			}
		case "join":
			if aliasTrimmed != "" && !strings.EqualFold(aliasTrimmed, physical) {
				descriptions = append(descriptions, fmt.Sprintf("table '%s' in '%s'(%s)", physical, aliasTrimmed, context))
			} else {
				descriptions = append(descriptions, fmt.Sprintf("table '%s' (%s)", physical, context))
			}
		default:
			descriptions = append(descriptions, fmt.Sprintf("table '%s'", physical))
		}
	}

	if len(descriptions) == 0 {
		return []string{d.unresolvedDescription(alias, ref)}
	}

	return descriptions
}

func (d *planTableDescriber) collectPhysicalTargets(alias string, ref TableReferenceInfo) []string {
	targets := make(map[string]string)

	add := func(name string, force bool) {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return
		}

		canonical := CanonicalIdentifier(trimmed)
		if canonical == "" {
			return
		}

		if resolved, ok := d.resolvePhysical(trimmed); ok {
			trimmed = resolved
			canonical = CanonicalIdentifier(trimmed)
		} else if !force {
			return
		}

		if canonical == "" {
			return
		}

		if _, exists := targets[canonical]; !exists {
			targets[canonical] = trimmed
		}
	}

	aliasTrimmed := strings.TrimSpace(alias)

	add(ref.TableName, true)

	nameTrimmed := strings.TrimSpace(ref.Name)

	forceName := nameTrimmed != "" && !strings.EqualFold(nameTrimmed, aliasTrimmed)
	if strings.EqualFold(ref.Context, "cte") || strings.EqualFold(ref.Context, "subquery") {
		forceName = true
	}

	add(ref.Name, forceName)
	add(alias, false)

	visited := make(map[string]struct{})
	d.collectFromQuery(ref.Name, visited, add)
	d.collectFromQuery(alias, visited, add)
	d.collectFromQuery(ref.QueryName, visited, add)

	if len(targets) == 0 {
		add(ref.TableName, true)
		add(ref.Name, false)
	}

	if len(targets) == 0 {
		return nil
	}

	result := make([]string, 0, len(targets))
	for _, value := range targets {
		result = append(result, value)
	}

	sort.Strings(result)

	return result
}

func (d *planTableDescriber) collectFromQuery(identifier string, visited map[string]struct{}, add func(string, bool)) {
	canonical := CanonicalIdentifier(identifier)
	if canonical == "" {
		return
	}

	if _, exists := visited[canonical]; exists {
		return
	}

	visited[canonical] = struct{}{}

	children := d.byQuery[canonical]
	if len(children) == 0 {
		return
	}

	for _, child := range children {
		add(child.TableName, true)
		add(child.Name, false)
		add(child.Alias, false)

		if child.TableName == "" {
			d.collectFromQuery(child.Name, visited, add)
			d.collectFromQuery(child.Alias, visited, add)
		}
	}
}

func (d *planTableDescriber) unresolvedDescription(alias string, ref TableReferenceInfo) string {
	candidate := strings.TrimSpace(alias)
	if candidate == "" {
		candidate = strings.TrimSpace(ref.Name)
	}

	if candidate == "" {
		candidate = "<unknown>"
	}

	context := classifyContext(ref.Context)
	if context != "" {
		if aliasTrim := strings.TrimSpace(alias); aliasTrim != "" {
			return fmt.Sprintf("table '%s' in '%s'(%s) (physical table unresolved)", candidate, aliasTrim, context)
		}

		return fmt.Sprintf("table '%s' (%s, physical table unresolved)", candidate, context)
	}

	return fmt.Sprintf("table '%s' (physical table unresolved)", candidate)
}
