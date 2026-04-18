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

package auth

import (
	"context"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/source"
)

// LoginSource persists source credentials when needed and returns a successful
// login status response.
func LoginSource(_ context.Context, credentials *source.Credentials) (*model.StatusResponse, error) {
	values := credentials.CloneValues()
	log.Debugf("[LoginSource] sourceType=%s, values=%d", credentials.SourceType, len(values))

	if credentials.ID != nil && *credentials.ID != "" {
		storedCredentials := &source.Credentials{
			ID:          credentials.ID,
			SourceType:  credentials.SourceType,
			Values:      values,
			AccessToken: credentials.AccessToken,
			IsProfile:   false,
		}
		if err := SaveCredentials(*credentials.ID, storedCredentials); err != nil {
			warnKeyringUnavailableOnce(err)
		}
	}

	return &model.StatusResponse{Status: true}, nil
}
