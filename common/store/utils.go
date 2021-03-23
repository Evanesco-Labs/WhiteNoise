package store

import "fmt"

const DSP_TASK_TABLE_NAME = "dsp_tasks"

const CreateTasks string = "CREATE TABLE IF NOT EXISTS " + DSP_TASK_TABLE_NAME +
	" (id VARCHAR[255] NOT NULL PRIMARY KEY, fileHash VARCHAR[255] NOT NULL, fileName VARCHAR[255] NOT NULL, walletAddr VARCHAR[255] NOT NULL, type INTEGER NOT NULL, createdAt DATE, updatedAt DATE);"

const ScriptCreateTables string = `PRAGMA foreign_keys=off;
BEGIN TRANSACTION;
%s
COMMIT;
PRAGMA foreign_keys=on;
`

func GetCreateTables() string {
	sqlStmt := fmt.Sprintf(ScriptCreateTables, CreateTasks)
	return sqlStmt
}
