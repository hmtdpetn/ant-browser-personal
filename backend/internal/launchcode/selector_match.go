package launchcode

import (
	"ant-chrome/backend/internal/browser"
	"net/http"
	"sort"
	"strings"
)

func (s *LaunchServer) findProfilesBySelector(selector LaunchSelector) ([]browser.Profile, int, string) {
	if selector.IsEmpty() {
		return nil, http.StatusBadRequest, "selector is required"
	}
	if err := selector.Validate(); err != nil {
		return nil, http.StatusBadRequest, err.Error()
	}
	if s.browserMgr == nil {
		return nil, http.StatusInternalServerError, "advanced profile selector is not available"
	}

	snapshots := s.profileSnapshots()
	if len(snapshots) == 0 {
		return nil, http.StatusNotFound, "profile selector matched no instance"
	}

	if selector.Code != "" {
		profileID, err := s.service.Resolve(selector.Code)
		if err != nil {
			return nil, http.StatusNotFound, "launch code not found"
		}
		filtered := make([]browser.Profile, 0, 1)
		for _, item := range snapshots {
			if item.ProfileId == profileID {
				filtered = append(filtered, item)
				break
			}
		}
		snapshots = filtered
	}

	if selector.ProfileID != "" {
		snapshots = filterProfiles(snapshots, func(item browser.Profile) bool {
			return item.ProfileId == selector.ProfileID
		})
	}

	if selector.ProfileName != "" {
		snapshots = filterProfiles(snapshots, func(item browser.Profile) bool {
			return strings.EqualFold(strings.TrimSpace(item.ProfileName), selector.ProfileName)
		})
	}

	if selector.GroupID != "" {
		snapshots = filterProfiles(snapshots, func(item browser.Profile) bool {
			return strings.TrimSpace(item.GroupId) == selector.GroupID
		})
	}

	if len(selector.Tags) > 0 {
		snapshots = filterProfiles(snapshots, func(item browser.Profile) bool {
			return profileHasAllTags(item, selector.Tags)
		})
	}

	fuzzyQueries := selector.Keywords
	if selector.Key != "" {
		exactMatches := filterProfiles(snapshots, func(item browser.Profile) bool {
			return profileHasExactKeyword(item, selector.Key)
		})
		if len(exactMatches) > 0 {
			snapshots = exactMatches
		} else {
			fuzzyQueries = normalizeSelectorTerms(append([]string{selector.Key}, fuzzyQueries...))
		}
	}

	if len(fuzzyQueries) > 0 {
		snapshots = filterProfiles(snapshots, func(item browser.Profile) bool {
			return profileMatchesAllKeywordQueries(item, fuzzyQueries)
		})
	}

	if len(snapshots) == 0 {
		if selector.OnlyCode() {
			return nil, http.StatusNotFound, "launch code not found"
		}
		return nil, http.StatusNotFound, "profile selector matched no instance"
	}

	sortProfilesForSelector(snapshots)
	return snapshots, http.StatusOK, ""
}

func (s *LaunchServer) findProfileBySelector(selector LaunchSelector) (browser.Profile, int, string) {
	snapshots, status, errMsg := s.findProfilesBySelector(selector)
	if errMsg != "" {
		return browser.Profile{}, status, errMsg
	}
	if len(snapshots) > 1 && selector.MatchMode != launchMatchModeFirst {
		return browser.Profile{}, http.StatusConflict, buildAmbiguousSelectorError(snapshots)
	}
	return snapshots[0], http.StatusOK, ""
}

func (s *LaunchServer) profileSnapshots() []browser.Profile {
	if s.browserMgr == nil {
		return nil
	}

	s.browserMgr.Mutex.Lock()
	items := make([]browser.Profile, 0, len(s.browserMgr.Profiles))
	for _, profile := range s.browserMgr.Profiles {
		if profile == nil {
			continue
		}
		items = append(items, *profile)
	}
	s.browserMgr.Mutex.Unlock()

	if s.service != nil {
		for i := range items {
			if code, err := s.service.EnsureCode(items[i].ProfileId); err == nil {
				items[i].LaunchCode = code
			}
		}
	}
	return items
}

func filterProfiles(items []browser.Profile, keep func(browser.Profile) bool) []browser.Profile {
	filtered := make([]browser.Profile, 0, len(items))
	for _, item := range items {
		if keep(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func sortProfilesForSelector(items []browser.Profile) {
	sort.Slice(items, func(i, j int) bool {
		leftName := strings.ToLower(strings.TrimSpace(items[i].ProfileName))
		rightName := strings.ToLower(strings.TrimSpace(items[j].ProfileName))
		if leftName != rightName {
			return leftName < rightName
		}
		return items[i].ProfileId < items[j].ProfileId
	})
}
