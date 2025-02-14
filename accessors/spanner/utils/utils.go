package utils

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/spanner-migration-tool/spanner/ddl"
	"golang.org/x/exp/rand"
)

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand = rand.New(rand.NewSource(uint64(time.Now().UnixNano())))

	randomString := make([]byte, length)
	for i := range randomString {
		randomString[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(randomString)
}
func GenerateColumnDefsForTable(count int) map[string]ddl.ColumnDef {
	colums := make(map[string]ddl.ColumnDef)
	for i := 1; i <= count; i++ {
		colName := fmt.Sprintf("col%d", i)
		colId := fmt.Sprintf("c%d", i)
		colums[colId] = ddl.ColumnDef{Name: colName, Id: colId, T: ddl.Type{Name: ddl.Int64}}
	}
	return colums
}
func GenerateColIds(count int) []string {
	var colIds []string
	for i := 1; i <= count; i++ {
		colId := fmt.Sprintf("c%d", i)
		colIds = append(colIds, colId)
	}
	return colIds
}

func GenerateTables(count int) ddl.Schema {
	tables := make(ddl.Schema)

	for i := 1; i <= count; i++ {
		tableName := fmt.Sprintf("table%d", i)
		tableId := fmt.Sprintf("t%d", i)
		tables[tableId] = ddl.CreateTable{Name: tableName, Id: tableId, PrimaryKeys: []ddl.IndexKey{{ColId: "c1"}}, ColIds: []string{"c1"},
			ColDefs: map[string]ddl.ColumnDef{
				"c1": {Name: "col1", Id: "c1", T: ddl.Type{Name: ddl.Int64}},
			}}
	}
	return tables
}
