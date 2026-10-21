package service

import (
	"context"
	"errors"
	"testing"

	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/assert"
)

type workflowMocks struct {
	task     *mocks.ITaskRepository
	history  *mocks.ITaskHistoryRepository
	approval *mocks.IApprovalRepository
	matrix   *mocks.IApprovalMatrixRepository
	code     *mocks.IApprovalCodeRepository
	user     *mocks.IUserRepository
	role     *mocks.IRoleRepository
	site     *mocks.ISiteRepository
	queue    *mocks.ITaskQueueClient
	mail     *mocks.IMailClient
	insight  *mocks.IAIInsightService
}

func newTestWorkflow(t *testing.T) (*taskWorkflow, workflowMocks) {
	m := workflowMocks{mocks.NewITaskRepository(t), mocks.NewITaskHistoryRepository(t), mocks.NewIApprovalRepository(t), mocks.NewIApprovalMatrixRepository(t),
		mocks.NewIApprovalCodeRepository(t), mocks.NewIUserRepository(t), mocks.NewIRoleRepository(t), mocks.NewISiteRepository(t),
		mocks.NewITaskQueueClient(t), mocks.NewIMailClient(t), mocks.NewIAIInsightService(t)}
	return &taskWorkflow{opt: testOptions(), RepositoryTask: m.task, RepositoryTaskHistory: m.history, RepositoryApproval: m.approval,
		RepositoryApprovalMatrix: m.matrix, RepositoryApprovalCode: m.code, RepositoryUser: m.user, RepositoryRole: m.role,
		RepositorySite: m.site, ClientTaskQueue: m.queue, ClientMail: m.mail, ServicesAIInsight: m.insight}, m
}

func TestApprovalHistoryMessage(t *testing.T) {
	assert.Equal(t, "Site di approve oleh citra (Dept Head)", approvalHistoryMessage("APPROVE", "citra", "Dept Head", ""))
	assert.Equal(t, "Site di approve oleh citra dengan catatan (ok)", approvalHistoryMessage("APPROVE", "citra", "", "ok"))
	assert.Equal(t, "Site di reject oleh citra (Dept Head) dengan catatan (x)", approvalHistoryMessage("REJECT", "citra", "Dept Head", "x"))
	assert.Equal(t, "Site diminta revisi oleh  (DIV_HEAD) dengan catatan (y)", approvalHistoryMessage("REVISION", "", "DIV_HEAD", "y"))
}

func TestSubmitTask_NoActiveMatrix(t *testing.T) {
	w, m := newTestWorkflow(t)
	m.task.On("GetByID", ctxAny, "TS1").Return(&model.Task{TaskID: "TS1", TaskStatus: constants.TaskStatusRevision}, nil)
	m.matrix.On("GetActiveChain", ctxAny).Return(nil, nil)

	err := w.submitTask(context.Background(), "TS1", 1001, "n")

	assert.EqualError(t, err, "task_service: no active approval matrix configured")
	m.approval.AssertNotCalled(t, "CreateHeader")
}

func TestNotifyApprovalPending_NoEmailIsSkipped(t *testing.T) {
	w, m := newTestWorkflow(t)
	m.user.On("GetByID", ctxAny, int64(3001)).Return(&model.User{UserID: 3001}, nil)

	w.notifyApprovalPending(context.Background(), "TS1", "ST1", 1001, &model.ApprovalDetail{ID: 31, ApproverID: 3001}, "n")

	m.code.AssertNotCalled(t, "Set")
	m.mail.AssertNotCalled(t, "SendApprovalPending")
}

func TestNotifyApprovalPending_QueueOnEnqueues(t *testing.T) {
	w, m := newTestWorkflow(t)
	m.user.On("GetByID", ctxAny, int64(3001)).Return(&model.User{UserID: 3001, Username: "citra", Email: "c@example.com"}, nil)
	m.user.On("GetByID", ctxAny, int64(1001)).Return(nil, errors.New("gone"))
	m.site.On("GetByID", ctxAny, "ST1").Return(nil, errors.New("gone"))
	m.code.On("Set", ctxAny, "CODE", uint(31)).Return(errors.New("redis down")) // logged only
	m.role.On("GetByID", ctxAny, "DEPT_HEAD").Return(nil, errors.New("gone"))
	m.queue.On("Enabled").Return(true)
	m.queue.On("Enqueue", ctxAny, "approval-email-TS1", "/v1/queue/approval-email",
		[]byte(`{"to":"c@example.com","to_name":"citra","task_id":"TS1","site_name":"ST1","submitted_by":"","approver_role":"DEPT_HEAD","notes":"n","approval_code":"CODE"}`)).Return(nil)

	w.notifyApprovalPending(context.Background(), "TS1", "ST1", 1001, &model.ApprovalDetail{ID: 31, ApproverID: 3001, ApproverRoleID: "DEPT_HEAD", SecureApprovalCode: "CODE"}, "n")

	m.mail.AssertNotCalled(t, "SendApprovalPending")
}

func TestEnqueueAIInsight(t *testing.T) {
	w, m := newTestWorkflow(t)
	m.queue.On("Enabled").Return(false).Once()
	m.insight.On("Generate", ctxAny, "ST1", int64(1001)).Return(dto.AIInsightResponseGet{}, client.ErrGeminiNotSetUp) // logged only
	w.enqueueAIInsight(context.Background(), "ST1", 1001)

	m.queue.On("Enabled").Return(true).Once()
	m.queue.On("Enqueue", ctxAny, "ai-insight-ST1", "/v1/queue/ai-insight", []byte(`{"site_id":"ST1","actor_id":1001}`)).Return(nil)
	w.enqueueAIInsight(context.Background(), "ST1", 1001)
}

func TestApprove_NotAwaiting(t *testing.T) {
	s, tm := newTestTaskService(t)
	w, _ := newTestWorkflow(t)
	s.workflow = w
	tm.task.On("GetByID", ctxAny, "TS1").Return(&model.Task{TaskID: "TS1", TaskStatus: "REJECTED"}, nil)
	tm.approval.On("GetHeaderByTaskID", ctxAny, "TS1").Return(&model.ApprovalHeader{ID: 7, OverallStatus: "REJECTED"}, nil)

	err := s.Approve(context.Background(), "TS1", 3001, dto.TaskActionRequest{Decision: "APPROVE"})

	assert.EqualError(t, err, "task_service: task TS1 is not awaiting approval (status=REJECTED)")
}

func TestMaybeGenerateAIInsight_PersistsClientInsight(t *testing.T) {
	w, m := newTestWorkflow(t)
	s := &SiteService{opt: testOptions(), taskService: &TaskService{workflow: w}}
	insight := dto.AIInsightResponseGet{OverallScore: 70}
	m.insight.On("Persist", ctxAny, "ST1", insight, int64(1001)).Return(nil)

	s.maybeGenerateAIInsight(context.Background(), "ST1", constants.TaskStatusSubmited, false, 1001, &insight)

	m.insight.AssertNotCalled(t, "Generate")
}

func TestMaybeGenerateAIInsight_UpdatedSubmitSkips(t *testing.T) {
	w, m := newTestWorkflow(t)
	s := &SiteService{opt: testOptions(), taskService: &TaskService{workflow: w}}

	s.maybeGenerateAIInsight(context.Background(), "ST1", constants.TaskStatusSubmited, true, 1001, nil)

	m.queue.AssertNotCalled(t, "Enabled")
}
