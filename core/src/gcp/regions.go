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

package gcp

// Region represents a GCP region.
type Region struct {
	ID          string
	Description string
}

// GetRegions returns the list of GCP regions.
// Based on https://cloud.google.com/about/locations
func GetRegions() []Region {
	return []Region{
		// North America
		{ID: "us-central1", Description: "Iowa"},
		{ID: "us-east1", Description: "South Carolina"},
		{ID: "us-east4", Description: "Northern Virginia"},
		{ID: "us-east5", Description: "Columbus"},
		{ID: "us-south1", Description: "Dallas"},
		{ID: "us-west1", Description: "Oregon"},
		{ID: "us-west2", Description: "Los Angeles"},
		{ID: "us-west3", Description: "Salt Lake City"},
		{ID: "us-west4", Description: "Las Vegas"},
		{ID: "northamerica-northeast1", Description: "Montréal"},
		{ID: "northamerica-northeast2", Description: "Toronto"},

		// South America
		{ID: "southamerica-east1", Description: "São Paulo"},
		{ID: "southamerica-west1", Description: "Santiago"},

		// Europe
		{ID: "europe-central2", Description: "Warsaw"},
		{ID: "europe-north1", Description: "Finland"},
		{ID: "europe-southwest1", Description: "Madrid"},
		{ID: "europe-west1", Description: "Belgium"},
		{ID: "europe-west2", Description: "London"},
		{ID: "europe-west3", Description: "Frankfurt"},
		{ID: "europe-west4", Description: "Netherlands"},
		{ID: "europe-west6", Description: "Zurich"},
		{ID: "europe-west8", Description: "Milan"},
		{ID: "europe-west9", Description: "Paris"},
		{ID: "europe-west10", Description: "Berlin"},
		{ID: "europe-west12", Description: "Turin"},

		// Asia Pacific
		{ID: "asia-east1", Description: "Taiwan"},
		{ID: "asia-east2", Description: "Hong Kong"},
		{ID: "asia-northeast1", Description: "Tokyo"},
		{ID: "asia-northeast2", Description: "Osaka"},
		{ID: "asia-northeast3", Description: "Seoul"},
		{ID: "asia-south1", Description: "Mumbai"},
		{ID: "asia-south2", Description: "Delhi"},
		{ID: "asia-southeast1", Description: "Singapore"},
		{ID: "asia-southeast2", Description: "Jakarta"},

		// Australia
		{ID: "australia-southeast1", Description: "Sydney"},
		{ID: "australia-southeast2", Description: "Melbourne"},

		// Middle East
		{ID: "me-central1", Description: "Doha"},
		{ID: "me-central2", Description: "Dammam"},
		{ID: "me-west1", Description: "Tel Aviv"},

		// Africa
		{ID: "africa-south1", Description: "Johannesburg"},
	}
}
