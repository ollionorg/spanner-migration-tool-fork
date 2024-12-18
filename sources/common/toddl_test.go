// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/spanner-migration-tool/common/constants"
	"github.com/GoogleCloudPlatform/spanner-migration-tool/internal"
	"github.com/GoogleCloudPlatform/spanner-migration-tool/mocks"
	"github.com/GoogleCloudPlatform/spanner-migration-tool/schema"
	"github.com/GoogleCloudPlatform/spanner-migration-tool/spanner/ddl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_quoteIfNeeded(t *testing.T) {

	type arg struct {
		s string
	}

	tests := []struct {
		name              string
		args              arg
		expectedTableName string
	}{
		{
			name: "quoteIfNeeded",
			args: arg{
				s: "table Name",
			},
			expectedTableName: "\"table Name\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIfNeeded(tt.args.s)
			if !reflect.DeepEqual(result, tt.expectedTableName) {
				t.Errorf("quoteIfNeeded() returned incorrect output, got %v and want %v", result, tt.expectedTableName)
			}
		})
	}
}

func Test_cvtPrimaryKeys(t *testing.T) {
	srcKeys := []schema.Key{
		{
			ColId: "c1",
			Desc:  true,
			Order: 1,
		},
	}
	resultIndexKey := []ddl.IndexKey{{
		ColId: "c1",
		Desc:  true,
		Order: 1,
	},
	}

	type arg struct {
		srcKeys []schema.Key
	}

	tests := []struct {
		name             string
		args             arg
		expectedIndexKey []ddl.IndexKey
	}{
		{
			name: "Creating PrimaryKeys for the tables",
			args: arg{
				srcKeys: srcKeys,
			},
			expectedIndexKey: resultIndexKey,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cvtPrimaryKeys(tt.args.srcKeys)
			if !reflect.DeepEqual(result, tt.expectedIndexKey) {
				t.Errorf("cvtPrimaryKeys() output doesn't match, got %v and want %v", result, tt.expectedIndexKey)
			}
		})
	}
}

func Test_cvtForeignKeys(t *testing.T) {
	conv := internal.Conv{
		SrcSchema: map[string]schema.Table{
			"t1": {
				Name:   "table1",
				Id:     "t1",
				ColIds: []string{"c1", "c2"},
				ColDefs: map[string]schema.Column{
					"c1": {Name: "a", Id: "c1", Type: schema.Type{Name: ddl.String, Mods: []int64{255}}},
					"c2": {Name: "b", Id: "c2", Type: schema.Type{Name: ddl.Numeric, Mods: []int64{6, 4}}},
				},
				ForeignKeys: []schema.ForeignKey{{Name: "fk1", Id: "f1", ColumnNames: []string{"a"}, ColIds: []string{"c1"}, ReferTableId: "t2", ReferColumnIds: []string{"c3"}, ReferTableName: "table2", ReferColumnNames: []string{"c"}, OnDelete: constants.FK_RESTRICT, OnUpdate: constants.FK_CASCADE}},
			},
			"t2": {
				Name:   "table2",
				Id:     "t2",
				ColIds: []string{"c3"},
				ColDefs: map[string]schema.Column{
					"c3": {Name: "c", Id: "c3", Type: schema.Type{Name: ddl.String, Mods: []int64{255}}},
				},
			},
		},
		UsedNames:    map[string]bool{},
		SchemaIssues: map[string]internal.TableIssues{},
	}
	spTableName := "table1"
	srcTableId := "t1"
	srcKeys := []schema.ForeignKey{
		{
			Name:             "fk1",
			ColIds:           []string{"c1"},
			ColumnNames:      []string{"a"},
			ReferTableId:     "t2",
			ReferTableName:   "table2",
			ReferColumnIds:   []string{"c3"},
			ReferColumnNames: []string{"c"},
			Id:               "f1",
			OnDelete:         constants.FK_RESTRICT,
			OnUpdate:         constants.FK_CASCADE,
		},
	}

	resultForeignKey := []ddl.Foreignkey{{
		Name:           "fk1",
		ColIds:         []string{"c1"},
		ReferTableId:   "t2",
		ReferColumnIds: []string{"c3"},
		Id:             "f1",
		OnDelete:       constants.FK_NO_ACTION,
		OnUpdate:       constants.FK_NO_ACTION,
	},
	}

	type arg struct {
		conv        *internal.Conv
		spTableName string
		srcTableTd  string
		srcKey      []schema.ForeignKey
		isRestore   bool
	}

	tc := []struct {
		name               string
		args               arg
		expectedForeignKey []ddl.Foreignkey
		Error              error
	}{
		{
			name: "creating foreign key in spanner schema",
			args: arg{
				conv:        &conv,
				spTableName: spTableName,
				srcTableTd:  srcTableId,
				srcKey:      srcKeys,
				isRestore:   false,
			},
			expectedForeignKey: resultForeignKey,
			Error:              nil,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			result := cvtForeignKeys(tt.args.conv, tt.args.spTableName, tt.args.srcTableTd, tt.args.srcKey, tt.args.isRestore)
			if !reflect.DeepEqual(result, tt.expectedForeignKey) {
				t.Errorf("cvtForeignKeys() output mismatch, got %v and want %v,%v", result, tt.expectedForeignKey, tt.Error)
			}
		})
	}
}

func Test_cvtIndexes(t *testing.T) {
	tableId := "t1"
	spColIds := []string{"c1", "c2", "c3"}

	spIndexes := []ddl.CreateIndex{
		{
			Name:    "indexName",
			TableId: "t1",
			Unique:  true,
			Keys: []ddl.IndexKey{
				{"c1", true, 1},
				{"c2", true, 2},
				{"c3", true, 3},
			},
			Id:              "t1",
			StoredColumnIds: []string{"c1", "c2", "c3"},
		},
	}

	srcIndexes := []schema.Index{
		{
			Name:   "indexName",
			Unique: true,
			Keys: []schema.Key{
				{"c1", true, 1},
				{"c2", true, 2},
				{"c3", true, 3},
			},
			Id:              "t1",
			StoredColumnIds: []string{"c1", "c2", "c3"},
		},
	}

	conv := internal.Conv{
		SrcSchema: map[string]schema.Table{
			"t1": {
				Name:   "table1",
				Id:     "t1",
				ColIds: []string{"c1", "c2", "c3"},
				ColDefs: map[string]schema.Column{
					"c1": {Name: "a", Id: "c1", Type: schema.Type{Name: ddl.String, Mods: []int64{255}}},
					"c2": {Name: "b", Id: "c2", Type: schema.Type{Name: ddl.String, Mods: []int64{255}}},
					"c3": {Name: "c", Id: "cc3", Type: schema.Type{Name: ddl.String, Mods: []int64{255}}},
				},
				Indexes: srcIndexes,
			},
		},
		UsedNames: map[string]bool{},
		SpSchema: ddl.Schema{
			"t1": ddl.CreateTable{
				Name:   "table1",
				ColIds: []string{"c1", "c2", "c3"},
				ColDefs: map[string]ddl.ColumnDef{
					"c1": {Name: "a", Id: "c1", T: ddl.Type{Name: ddl.String, Len: 255}},
					"c2": {Name: "b", Id: "c2", T: ddl.Type{Name: ddl.String, Len: 255}},
					"c3": {Name: "c", Id: "c3", T: ddl.Type{Name: ddl.String, Len: 255}},
				},
				Id: "t1",
			},
		},
	}

	type arg struct {
		conv       *internal.Conv
		tableId    string
		srcIndexes []schema.Index
		spColIds   []string
	}

	tests := []struct {
		name              string
		args              arg
		ExpectedSpIndexes []ddl.CreateIndex
	}{
		{
			name: "Adding Index to the table",
			args: arg{
				conv:       &conv,
				tableId:    tableId,
				srcIndexes: srcIndexes,
				spColIds:   spColIds,
			},
			ExpectedSpIndexes: spIndexes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cvtIndexes(tt.args.conv, tt.args.tableId, tt.args.srcIndexes, tt.args.spColIds, tt.args.conv.SpSchema[tt.args.tableId].ColDefs)
			if !reflect.DeepEqual(got, tt.ExpectedSpIndexes) {
				t.Errorf("cvtIndexes() = %v and wants %v", got, tt.ExpectedSpIndexes)
			}
		})
	}
}

func Test_cvtForeignKeysForAReferenceTable(t *testing.T) {
	conv := internal.Conv{
		SrcSchema: map[string]schema.Table{
			"t1": {
				Name:   "table1",
				Id:     "t1",
				ColIds: []string{"c1", "c2"},
				ColDefs: map[string]schema.Column{
					"c1": {Name: "a", Id: "c1", Type: schema.Type{Name: ddl.String, Mods: []int64{255}}},
					"c2": {Name: "b", Id: "c2", Type: schema.Type{Name: ddl.Numeric, Mods: []int64{6, 4}}},
				},
				ForeignKeys: []schema.ForeignKey{{Name: "fk1", Id: "f1", ColumnNames: []string{"a"}, ColIds: []string{"c1"}, ReferTableId: "t2", ReferColumnIds: []string{"c3"}, ReferTableName: "table2", ReferColumnNames: []string{"c"}, OnDelete: constants.FK_RESTRICT, OnUpdate: constants.FK_CASCADE}},
			},
			"t2": {
				Name:   "table2",
				Id:     "t2",
				ColIds: []string{"c3"},
				ColDefs: map[string]schema.Column{
					"c3": {Name: "c", Id: "c3", Type: schema.Type{Name: ddl.String, Mods: []int64{255}}},
				},
				ForeignKeys: []schema.ForeignKey{},
			},
		},
		UsedNames: map[string]bool{},
		SpSchema: ddl.Schema{
			"t2": ddl.CreateTable{
				Name:   "table2",
				ColIds: []string{"c3"},
				ColDefs: map[string]ddl.ColumnDef{
					"c3": {Name: "c", Id: "c3", T: ddl.Type{Name: ddl.String, Len: 255}},
				},
				Id:          "t2",
				ForeignKeys: []ddl.Foreignkey{},
			},
		},
	}
	tableId := "t2"
	referTableId := "t2"
	srcKey := []schema.ForeignKey{
		{
			Name:             "fk1",
			ColIds:           []string{"c1"},
			ColumnNames:      []string{"a"},
			ReferTableId:     "t2",
			ReferTableName:   "table2",
			ReferColumnIds:   []string{"c3"},
			ReferColumnNames: []string{"c"},
			Id:               "f1",
			OnDelete:         constants.FK_RESTRICT,
			OnUpdate:         constants.FK_CASCADE,
		},
	}
	resultForeignKey := []ddl.Foreignkey{
		{
			Name:           "fk1",
			ColIds:         []string{"c1"},
			ReferTableId:   "t2",
			ReferColumnIds: []string{"c3"},
			Id:             "f1",
			OnDelete:       constants.FK_NO_ACTION,
			OnUpdate:       constants.FK_NO_ACTION,
		},
		{
			Name:           "fk1",
			ColIds:         []string{"c1"},
			ReferTableId:   "t2",
			ReferColumnIds: []string{"c3"},
			Id:             "f1",
			OnDelete:       constants.FK_NO_ACTION,
			OnUpdate:       constants.FK_NO_ACTION,
		},
	}
	spKey := []ddl.Foreignkey{{
		Name:           "fk1",
		ColIds:         []string{"c1"},
		ReferTableId:   "t2",
		ReferColumnIds: []string{"c3"},
		Id:             "f1",
		OnDelete:       constants.FK_NO_ACTION,
		OnUpdate:       constants.FK_NO_ACTION,
	},
	}

	tc := []struct {
		conv               *internal.Conv
		TableId            string
		ReferTableId       string
		srcKeys            []schema.ForeignKey
		spKeys             []ddl.Foreignkey
		expectedForeignKey []ddl.Foreignkey
	}{
		{
			conv:               &conv,
			srcKeys:            srcKey,
			TableId:            tableId,
			ReferTableId:       referTableId,
			spKeys:             spKey,
			expectedForeignKey: resultForeignKey,
		},
	}

	for _, tt := range tc {
		t.Run("testCase for referTable", func(t *testing.T) {
			result := cvtForeignKeysForAReferenceTable(tt.conv, tt.TableId, tt.ReferTableId, tt.srcKeys, tt.spKeys)
			if !reflect.DeepEqual(result, tt.expectedForeignKey) {
				t.Errorf("cvtForeignKeysForAReferenceTable() = %v and wants %v ", result, tt.expectedForeignKey)
			}
		})
	}
}

func Test_SchemaToSpannerSequenceHelper(t *testing.T) {
	expectedConv := internal.MakeConv()
	expectedConv.SpSequences["s1"] = ddl.Sequence{
		Name:             "Sequence1",
		Id:               "s1",
		SequenceKind:     "BIT REVERSED POSITIVE",
		SkipRangeMin:     "1",
		SkipRangeMax:     "2",
		StartWithCounter: "3",
	}
	tc := []struct {
		expectedConv *internal.Conv
		srcSequence  ddl.Sequence
	}{
		{
			expectedConv: expectedConv,
			srcSequence: ddl.Sequence{
				Name:             "Sequence1",
				Id:               "s1",
				SequenceKind:     constants.AUTO_INCREMENT,
				SkipRangeMin:     "1",
				SkipRangeMax:     "2",
				StartWithCounter: "3",
			},
		},
	}

	for _, tt := range tc {
		conv := internal.MakeConv()
		ss := SchemaToSpannerImpl{}
		ss.SchemaToSpannerSequenceHelper(conv, tt.srcSequence)
		assert.Equal(t, expectedConv, conv)
	}
}

func Test_cvtCheckContraint(t *testing.T) {

	conv := internal.MakeConv()
	srcSchema := []schema.CheckConstraint{
		{
			Id:   "cc1",
			Name: "check_1",
			Expr: "age > 0",
		},
		{
			Id:   "cc2",
			Name: "check_2",
			Expr: "age < 99",
		},
		{
			Id:   "cc3",
			Name: "@invalid_name", // incompatabile name
			Expr: "age != 0",
		},
	}
	spSchema := []ddl.CheckConstraint{
		{
			Id:   "cc1",
			Name: "check_1",
			Expr: "age > 0",
		},
		{
			Id:   "cc2",
			Name: "check_2",
			Expr: "age < 99",
		},
		{
			Id:   "cc3",
			Name: "Ainvalid_name",
			Expr: "age != 0",
		},
	}
	result := cvtCheckConstraint(conv, srcSchema)
	assert.Equal(t, spSchema, result)
}

func TestVerifyCheckConstraintExpressions(t *testing.T) {
	tests := []struct {
		name                    string
		expressions             []ddl.CheckConstraint
		expectedResults         []internal.ExpressionVerificationOutput
		expectedCheckConstraint []ddl.CheckConstraint
		expectedResponse        bool
	}{
		{
			name: "AllValidExpressions",
			expressions: []ddl.CheckConstraint{
				{Expr: "(col1 > 0)", ExprId: "expr1", Name: "check1"},
			},
			expectedResults: []internal.ExpressionVerificationOutput{
				{Result: true, Err: nil, ExpressionDetail: internal.ExpressionDetail{Expression: "(col1 > 0)", Type: "CHECK", Metadata: map[string]string{"tableId": "t1", "colId": "c1", "checkConstraintName": "check1"}, ExpressionId: "expr1"}},
			},
			expectedCheckConstraint: []ddl.CheckConstraint{
				{Expr: "(col1 > 0)", ExprId: "expr1", Name: "check1"},
			},
			expectedResponse: false,
		},
		{
			name: "InvalidSyntaxError",
			expressions: []ddl.CheckConstraint{
				{Expr: "(col1 > 0)", ExprId: "expr1", Name: "check1"},
				{Expr: "(col1 > 18", ExprId: "expr2", Name: "check2"},
			},
			expectedResults: []internal.ExpressionVerificationOutput{
				{Result: true, Err: nil, ExpressionDetail: internal.ExpressionDetail{Expression: "(col1 > 0)", Type: "CHECK", Metadata: map[string]string{"tableId": "t1", "colId": "c1", "checkConstraintName": "check1"}, ExpressionId: "expr1"}},
				{Result: false, Err: errors.New("Syntax error ..."), ExpressionDetail: internal.ExpressionDetail{Expression: "(col1 > 18", Type: "CHECK", Metadata: map[string]string{"tableId": "t1", "colId": "c1", "checkConstraintName": "check2"}, ExpressionId: "expr2"}},
			},
			expectedCheckConstraint: []ddl.CheckConstraint{
				{Expr: "(col1 > 0)", ExprId: "expr1", Name: "check1"},
			},
			expectedResponse: true,
		},
		{
			name: "NameError",
			expressions: []ddl.CheckConstraint{
				{Expr: "(col1 > 0)", ExprId: "expr1", Name: "check1"},
				{Expr: "(col1 > 18)", ExprId: "expr2", Name: "check2"},
			},
			expectedResults: []internal.ExpressionVerificationOutput{
				{Result: true, Err: nil, ExpressionDetail: internal.ExpressionDetail{Expression: "(col1 > 0)", Type: "CHECK", Metadata: map[string]string{"tableId": "t1", "colId": "c1", "checkConstraintName": "check1"}, ExpressionId: "expr1"}},
				{Result: false, Err: errors.New("Unrecognized name ..."), ExpressionDetail: internal.ExpressionDetail{Expression: "(col1 > 18)", Type: "CHECK", Metadata: map[string]string{"tableId": "t1", "colId": "c1", "checkConstraintName": "check2"}, ExpressionId: "expr2"}},
			},
			expectedCheckConstraint: []ddl.CheckConstraint{
				{Expr: "(col1 > 0)", ExprId: "expr1", Name: "check1"},
			},
			expectedResponse: true,
		},
		{
			name: "TypeError",
			expressions: []ddl.CheckConstraint{
				{Expr: "(col1 > 0)", ExprId: "expr1", Name: "check1"},
				{Expr: "(col1 > 18)", ExprId: "expr2", Name: "check2"},
			},
			expectedResults: []internal.ExpressionVerificationOutput{
				{Result: true, Err: nil, ExpressionDetail: internal.ExpressionDetail{Expression: "(col1 > 0)", Type: "CHECK", Metadata: map[string]string{"tableId": "t1", "colId": "c1", "checkConstraintName": "check1"}, ExpressionId: "expr1"}},
				{Result: false, Err: errors.New("No matching signature for operator"), ExpressionDetail: internal.ExpressionDetail{Expression: "(col1 > 18)", Type: "CHECK", Metadata: map[string]string{"tableId": "t1", "colId": "c1", "checkConstraintName": "check2"}, ExpressionId: "expr2"}},
			},
			expectedCheckConstraint: []ddl.CheckConstraint{
				{Expr: "(col1 > 0)", ExprId: "expr1", Name: "check1"},
			},
			expectedResponse: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockAccessor := new(mocks.MockExpressionVerificationAccessor)
			handler := &SchemaToSpannerImpl{ExpressionVerificationAccessor: mockAccessor}

			conv := internal.MakeConv()

			ctx := context.Background()

			conv.SpSchema = map[string]ddl.CreateTable{
				"t1": {
					Name:        "table1",
					Id:          "t1",
					PrimaryKeys: []ddl.IndexKey{{ColId: "c1"}},
					ColIds:      []string{"c1"},
					ColDefs: map[string]ddl.ColumnDef{
						"c1": {Name: "col1", Id: "c1", T: ddl.Type{Name: ddl.Int64}},
					},
					CheckConstraints: tc.expressions,
				},
			}

			mockAccessor.On("VerifyExpressions", ctx, mock.Anything).Return(internal.VerifyExpressionsOutput{
				ExpressionVerificationOutputList: tc.expectedResults,
			})
			handler.VerifyExpressions(conv)
			assert.Equal(t, conv.SpSchema["t1"].CheckConstraints, tc.expectedCheckConstraint)

		})
	}
}
