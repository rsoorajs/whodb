/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package azure

import (
	"testing"
)

func TestGetRegions_NonEmpty(t *testing.T) {
	regions := GetRegions()
	if len(regions) == 0 {
		t.Error("expected non-empty region list")
	}
}

func TestGetRegions_HasGeographies(t *testing.T) {
	regions := GetRegions()

	geographies := make(map[string]int)
	for _, r := range regions {
		if r.Geography == "" {
			t.Errorf("region %s has empty geography", r.ID)
		}
		geographies[r.Geography]++
	}

	if geographies[GeographyAmericas] == 0 {
		t.Error("expected Americas regions")
	}
	if geographies[GeographyEurope] == 0 {
		t.Error("expected Europe regions")
	}
	if geographies[GeographyAsiaPac] == 0 {
		t.Error("expected Asia Pacific regions")
	}
	if geographies[GeographyMiddleEast] == 0 {
		t.Error("expected Middle East regions")
	}
	if geographies[GeographyAfrica] == 0 {
		t.Error("expected Africa regions")
	}
}

func TestGetRegions_KeyRegionsPresent(t *testing.T) {
	regions := GetRegions()

	regionIDs := make(map[string]bool)
	for _, r := range regions {
		regionIDs[r.ID] = true
	}

	keyRegions := []string{
		"eastus",
		"westus2",
		"westeurope",
		"northeurope",
		"southeastasia",
		"australiaeast",
		"japaneast",
		"brazilsouth",
		"uksouth",
	}

	for _, id := range keyRegions {
		if !regionIDs[id] {
			t.Errorf("expected key region %s to be present", id)
		}
	}
}

func TestGetRegions_NoDuplicateIDs(t *testing.T) {
	regions := GetRegions()
	seen := make(map[string]bool)
	for _, r := range regions {
		if seen[r.ID] {
			t.Errorf("duplicate region ID: %s", r.ID)
		}
		seen[r.ID] = true
	}
}

func TestGetRegions_AllFieldsPopulated(t *testing.T) {
	regions := GetRegions()
	for _, r := range regions {
		if r.ID == "" {
			t.Error("found region with empty ID")
		}
		if r.DisplayName == "" {
			t.Errorf("region %s has empty display name", r.ID)
		}
		if r.Geography == "" {
			t.Errorf("region %s has empty geography", r.ID)
		}
	}
}

func TestGetRegions_ValidGeographies(t *testing.T) {
	validGeographies := map[string]bool{
		GeographyAmericas:   true,
		GeographyEurope:     true,
		GeographyAsiaPac:    true,
		GeographyMiddleEast: true,
		GeographyAfrica:     true,
	}

	regions := GetRegions()
	for _, r := range regions {
		if !validGeographies[r.Geography] {
			t.Errorf("region %s has invalid geography: %s", r.ID, r.Geography)
		}
	}
}
