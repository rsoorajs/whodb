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

package engine

import "github.com/clidey/whodb/core/src/source"

// StorageUnit aliases the source-owned storage unit type for compatibility.
type StorageUnit = source.StorageUnit

// Record aliases the source-owned record type for compatibility.
type Record = source.Record

// ExternalModel aliases the source-owned external AI model configuration for
// compatibility.
type ExternalModel = source.ExternalModel

// Column aliases the source-owned column type for compatibility.
type Column = source.Column

// GetRowsResult aliases the source-owned row result type for compatibility.
type GetRowsResult = source.RowsResult

// GraphUnitRelationshipType aliases the source-owned graph relationship type.
type GraphUnitRelationshipType = source.GraphRelationshipType

// GraphUnitRelationship aliases the source-owned graph relationship type.
type GraphUnitRelationship = source.GraphRelationship

// GraphUnit aliases the source-owned graph unit type for compatibility.
type GraphUnit = source.GraphUnit

// ChatMessage aliases the source-owned chat message type for compatibility.
type ChatMessage = source.ChatMessage

// ForeignKeyRelationship aliases the source-owned FK relationship type.
type ForeignKeyRelationship = source.ForeignKeyRelationship

// SSLStatus aliases the source-owned SSL status type for compatibility.
type SSLStatus = source.SSLStatus

// TypeCategory aliases the source-owned type category.
type TypeCategory = source.TypeCategory

// TypeDefinition aliases the source-owned type definition.
type TypeDefinition = source.TypeDefinition

const (
	// TypeCategoryNumeric groups numeric types.
	TypeCategoryNumeric TypeCategory = source.TypeCategoryNumeric
	// TypeCategoryText groups text types.
	TypeCategoryText TypeCategory = source.TypeCategoryText
	// TypeCategoryBinary groups binary types.
	TypeCategoryBinary TypeCategory = source.TypeCategoryBinary
	// TypeCategoryDatetime groups date/time types.
	TypeCategoryDatetime TypeCategory = source.TypeCategoryDatetime
	// TypeCategoryBoolean groups boolean types.
	TypeCategoryBoolean TypeCategory = source.TypeCategoryBoolean
	// TypeCategoryJSON groups JSON/document types.
	TypeCategoryJSON TypeCategory = source.TypeCategoryJSON
	// TypeCategoryOther groups uncategorised types.
	TypeCategoryOther TypeCategory = source.TypeCategoryOther
)

const (
	// GraphUnitRelationshipTypeOneToOne identifies a one-to-one relationship.
	GraphUnitRelationshipTypeOneToOne GraphUnitRelationshipType = source.GraphRelationshipTypeOneToOne
	// GraphUnitRelationshipTypeOneToMany identifies a one-to-many relationship.
	GraphUnitRelationshipTypeOneToMany GraphUnitRelationshipType = source.GraphRelationshipTypeOneToMany
	// GraphUnitRelationshipTypeManyToOne identifies a many-to-one relationship.
	GraphUnitRelationshipTypeManyToOne GraphUnitRelationshipType = source.GraphRelationshipTypeManyToOne
	// GraphUnitRelationshipTypeManyToMany identifies a many-to-many relationship.
	GraphUnitRelationshipTypeManyToMany GraphUnitRelationshipType = source.GraphRelationshipTypeManyToMany
	// GraphUnitRelationshipTypeUnknown identifies an unknown relationship.
	GraphUnitRelationshipTypeUnknown GraphUnitRelationshipType = source.GraphRelationshipTypeUnknown
)
