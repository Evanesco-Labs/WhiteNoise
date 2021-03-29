package store

import "fmt"

const CreateTasks string = ""

const ScriptCreateTables string = ""

func GetCreateTables() string {
	sqlStmt := fmt.Sprintf(ScriptCreateTables, CreateTasks)
	return sqlStmt
}
