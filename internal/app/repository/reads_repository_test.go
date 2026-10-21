package repository

import (
	"context"
	"testing"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTaskRepositoryGetByStatuses_NoSiteFilter(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &TaskRepository{opt: o}

	_, _, err := r.GetByStatuses(context.Background(), 0, 20, "CreatedAt ASC", []string{"DRAFT", "COMPLETED"}, nil)

	require.NoError(t, err)
	assert.Equal(t, []string{
		`SELECT count(*) FROM "Tasks" WHERE TaskStatus IN ('DRAFT','COMPLETED')`,
		`SELECT * FROM "Tasks" WHERE TaskStatus IN ('DRAFT','COMPLETED') ORDER BY CreatedAt ASC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`,
	}, rec.sqls)
}

func TestTaskRepositoryGetByIDs_EmptyRunsNoQuery(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &TaskRepository{opt: o}

	tasks, err := r.GetByIDs(context.Background(), nil)

	require.NoError(t, err)
	assert.Empty(t, tasks)
	assert.Empty(t, rec.sqls)
}

func TestTaskRepositoryCountByStatus(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &TaskRepository{opt: o}

	_, err := r.CountByStatus(context.Background(), 1001)
	require.NoError(t, err)
	_, err = r.CountByStatus(context.Background(), 0)
	require.NoError(t, err)

	assert.Equal(t, []string{
		`SELECT TaskStatus, COUNT(1) as Count FROM "Tasks" WHERE CreatedBy = 1001 GROUP BY "TaskStatus"`,
		`SELECT TaskStatus, COUNT(1) as Count FROM "Tasks" GROUP BY "TaskStatus"`,
	}, rec.sqls)
}

func TestTaskHistoryRepositoryGet(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &TaskHistoryRepository{opt: o}

	_, _, err := r.Get(context.Background(), 40, 20, "CreatedAt DESC")

	require.NoError(t, err)
	assert.Equal(t, []string{
		`SELECT count(*) FROM "TaskHistories"`,
		`SELECT * FROM "TaskHistories" ORDER BY CreatedAt DESC OFFSET 40 ROWS FETCH NEXT 20 ROWS ONLY`,
	}, rec.sqls)
}

func TestApprovalRepositoryGetHeaderByTaskID_NotFound(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &ApprovalRepository{opt: o}

	header, err := r.GetHeaderByTaskID(context.Background(), "TS1")

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.Nil(t, header)
	assert.Equal(t, []string{
		`SELECT * FROM "ApprovalHDR" WHERE TaskID = 'TS1' ORDER BY ID DESC OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	}, rec.sqls)
}

func TestApprovalRepositoryGetPendingByApprover_NoSearch(t *testing.T) {
	o, rec := fakeDBOptions(t, sqlfake.Result{Match: "count(*)", Columns: []string{"count"}, Rows: nil})
	r := &ApprovalRepository{opt: o}

	_, _, err := r.GetPendingByApprover(context.Background(), 3001, "", "ApprovalDET.CreatedAt ASC", 20, 0)

	require.NoError(t, err)
	joins := `FROM "ApprovalDET" JOIN ApprovalHDR ON ApprovalHDR.ID = ApprovalDET.ApprovalHeaderID JOIN Tasks ON Tasks.TaskID = ApprovalHDR.TaskID JOIN Sites ON Sites.SiteID = Tasks.SiteID WHERE ApprovalDET.ApproverID = 3001 AND ApprovalDET.Status = 'PENDING'`
	assert.Equal(t, []string{
		`SELECT count(*) ` + joins,
		`SELECT ApprovalDET.* ` + joins + ` ORDER BY ApprovalDET.CreatedAt ASC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`,
	}, rec.sqls)
}

func TestSiteContextRepositoryReads(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &SiteContextRepository{opt: o}

	_, err := r.GetBySiteID(context.Background(), "ST1")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = r.GetCompetitorsBySiteID(context.Background(), "ST1")
	require.NoError(t, err)
	_, err = r.GetCustomNotesBySiteID(context.Background(), "ST1")
	require.NoError(t, err)

	assert.Equal(t, []string{
		`SELECT * FROM "SiteContexts" WHERE SiteID = 'ST1' ORDER BY "SiteContexts"."SiteID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "SiteCompetitors" WHERE SiteID = 'ST1' ORDER BY ID ASC`,
		`SELECT * FROM "SiteCustomNotes" WHERE SiteID = 'ST1' ORDER BY ID ASC`,
	}, rec.sqls)
}

func TestSiteRepositoryGet_NoFilters(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &SiteRepository{opt: o}

	_, _, err := r.Get(context.Background(), 0, 20, "CreatedAt ASC", "", "")

	require.NoError(t, err)
	assert.Equal(t, []string{
		`SELECT count(*) FROM "Sites"`,
		`SELECT * FROM "Sites" ORDER BY CreatedAt ASC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`,
	}, rec.sqls)
}

func TestSiteRepositoryGetActiveStores_CountFailure(t *testing.T) {
	dbErr := assert.AnError
	o, rec := fakeDBOptions(t, sqlfake.Result{Match: "SELECT COUNT(1)", Err: dbErr})
	r := &SiteRepository{opt: o}

	_, _, err := r.GetActiveStores(context.Background(), "x", 20, 0)

	assert.ErrorIs(t, err, dbErr)
	assert.Len(t, rec.sqls, 1)
}

func TestSiteContextRepositoryReplace_EmptyListOnlyDeletes(t *testing.T) {
	o, rec := fakeDBOptions(t)
	r := &SiteContextRepository{opt: o}

	require.NoError(t, r.ReplaceCompetitors(context.Background(), "ST1", nil))
	require.NoError(t, r.ReplaceCustomNotes(context.Background(), "ST1", nil))

	assert.Equal(t, []string{
		`DELETE FROM "SiteCompetitors" WHERE SiteID = 'ST1'`,
		`DELETE FROM "SiteCustomNotes" WHERE SiteID = 'ST1'`,
	}, rec.sqls)
}
