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

package graph

import (
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/query"
)

func queryWhereConditionFromModel(condition *model.WhereCondition) *query.WhereCondition {
	if condition == nil {
		return nil
	}

	return &query.WhereCondition{
		Type:   query.WhereConditionType(condition.Type),
		Atomic: queryAtomicWhereConditionFromModel(condition.Atomic),
		And:    queryOperationWhereConditionFromModel(condition.And),
		Or:     queryOperationWhereConditionFromModel(condition.Or),
	}
}

func queryAtomicWhereConditionFromModel(condition *model.AtomicWhereCondition) *query.AtomicWhereCondition {
	if condition == nil {
		return nil
	}

	return &query.AtomicWhereCondition{
		ColumnType: condition.ColumnType,
		Key:        condition.Key,
		Operator:   condition.Operator,
		Value:      condition.Value,
	}
}

func queryOperationWhereConditionFromModel(condition *model.OperationWhereCondition) *query.OperationWhereCondition {
	if condition == nil {
		return nil
	}

	children := make([]*query.WhereCondition, 0, len(condition.Children))
	for _, child := range condition.Children {
		children = append(children, queryWhereConditionFromModel(child))
	}

	return &query.OperationWhereCondition{
		Children: children,
	}
}

func querySortConditionsFromModel(sort []*model.SortCondition) []*query.SortCondition {
	conditions := make([]*query.SortCondition, 0, len(sort))
	for _, item := range sort {
		if item == nil {
			conditions = append(conditions, nil)
			continue
		}

		conditions = append(conditions, &query.SortCondition{
			Column:    item.Column,
			Direction: query.SortDirection(item.Direction),
		})
	}
	return conditions
}
