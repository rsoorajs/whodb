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

// Package query defines the neutral filter and sort AST used by source and
// backend runtime code. It intentionally does not depend on GraphQL-generated
// model types.
package query

// WhereConditionType identifies the shape of a where condition node.
type WhereConditionType string

const (
	// WhereConditionTypeAtomic represents one leaf condition.
	WhereConditionTypeAtomic WhereConditionType = "Atomic"
	// WhereConditionTypeAnd represents an AND group.
	WhereConditionTypeAnd WhereConditionType = "And"
	// WhereConditionTypeOr represents an OR group.
	WhereConditionTypeOr WhereConditionType = "Or"
)

// AtomicWhereCondition describes one leaf filter predicate.
type AtomicWhereCondition struct {
	ColumnType string
	Key        string
	Operator   string
	Value      string
}

// OperationWhereCondition contains child where condition nodes.
type OperationWhereCondition struct {
	Children []*WhereCondition
}

// WhereCondition represents the source-neutral filter AST.
type WhereCondition struct {
	Type   WhereConditionType
	Atomic *AtomicWhereCondition
	And    *OperationWhereCondition
	Or     *OperationWhereCondition
}

// SortDirection identifies the direction of a sort expression.
type SortDirection string

const (
	// SortDirectionAsc sorts in ascending order.
	SortDirectionAsc SortDirection = "ASC"
	// SortDirectionDesc sorts in descending order.
	SortDirectionDesc SortDirection = "DESC"
)

// SortCondition represents one sort expression.
type SortCondition struct {
	Column    string
	Direction SortDirection
}
