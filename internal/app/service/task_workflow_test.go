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
	assert.Equal(t, "Disetujui oleh Dept Head citra", approvalHistoryMessage("APPROVE", "citra", "Dept Head", ""))
	assert.Equal(t, "Disetujui oleh citra dengan catatan (ok)", approvalHistoryMessage("APPROVE", "citra", "", "ok"))
	assert.Equal(t, "Ditolak Final oleh Dept Head citra dengan catatan (x)", approvalHistoryMessage("REJECT", "citra", "Dept Head", "x"))
	assert.Equal(t, "Revisi dari DIV_HEAD dengan catatan (y)", approvalHistoryMessage("REVISION", "", "DIV_HEAD", "y"))
	// Wes, 26 Oct: notes optional for REJECT / REVISION; staff CANCEL
	assert.Equal(t, "Ditolak Final oleh Dept Head citra", approvalHistoryMessage("REJECT", "citra", "Dept Head", ""))
	assert.Equal(t, "Revisi dari Div Head dodi", approvalHistoryMessage("REVISION", "dodi", "Div Head", ""))
	assert.Equal(t, "Diteruskan ke Div Head", forwardHistoryMessage("Div Head"))
	assert.Equal(t, "Pengajuan dibatalkan oleh andi (Staf)", approvalHistoryMessage("CANCEL", "andi", "Staf", ""))
	assert.Equal(t, "Pengajuan dibatalkan oleh andi (Staf) dengan catatan (z)", approvalHistoryMessage("CANCEL", "andi", "Staf", "z"))
}

func TestSubmitTask_NoActiveMatrix(t *testing.T) {
	w, m := newTestWorkflow(t)
	m.task.On("GetByID", ctxAny, "TS1").Return(&model.Task{TaskID: "TS1", TaskStatus: constants.TaskStatusRevision}, nil)
	m.matrix.On("GetActiveChain", ctxAny).Return(nil, nil)

	err := w.submitTask(context.Background(), "TS1", 1001, "USER", "n")

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

	s.maybeGenerateAIInsight(context.Background(), "ST1", constants.TaskStatusSubmited, false, 1001, &insight, nil)

	m.insight.AssertNotCalled(t, "Generate")
}

func TestMaybeGenerateAIInsight_UpdatedSubmitSkips(t *testing.T) {
	w, m := newTestWorkflow(t)
	s := &SiteService{opt: testOptions(), taskService: &TaskService{workflow: w}, autoGenerateAIOnSave: true}

	s.maybeGenerateAIInsight(context.Background(), "ST1", constants.TaskStatusSubmited, true, 1001, nil, nil)

	m.queue.AssertNotCalled(t, "Enabled")
}

// generate_ai (Wes, 26 Oct): false never starts AI, true always does,
// missing follows [ai_insight] auto_generate_on_save (old rules when on).
func TestMaybeGenerateAIInsight_GenerateAIFlag(t *testing.T) {
	yes, no := true, false
	for name, tc := range map[string]struct {
		auto     bool
		isUpdate bool
		status   string
		flag     *bool
		generate bool
	}{
		"new send, flag false, auto on":    {true, false, constants.TaskStatusSubmited, &no, false},
		"new draft, flag false, auto on":   {true, false, constants.TaskStatusDraft, &no, false},
		"new send, no flag, auto off":      {false, false, constants.TaskStatusSubmited, nil, false},
		"new send, no flag, auto on (old)": {true, false, constants.TaskStatusSubmited, nil, true},
		"updated send, flag true":          {false, true, constants.TaskStatusSubmited, &yes, true},
		"new send, flag true, auto off":    {false, false, constants.TaskStatusSubmited, &yes, true},
	} {
		t.Run(name, func(t *testing.T) {
			w, m := newTestWorkflow(t)
			s := &SiteService{opt: testOptions(), taskService: &TaskService{workflow: w}, autoGenerateAIOnSave: tc.auto}
			if tc.generate {
				m.queue.On("Enabled").Return(false).Once()
				m.insight.On("Generate", ctxAny, "ST1", int64(1001)).Return(dto.AIInsightResponseGet{}, nil).Once()
			}

			s.maybeGenerateAIInsight(context.Background(), "ST1", tc.status, tc.isUpdate, 1001, nil, tc.flag)

			if tc.generate {
				m.insight.AssertCalled(t, "Generate", ctxAny, "ST1", int64(1001))
			} else {
				m.queue.AssertNotCalled(t, "Enabled")
				m.insight.AssertNotCalled(t, "Generate", ctxAny, "ST1", int64(1001))
			}
		})
	}
}

func TestLoadAutoGenerateAIOnSave(t *testing.T) {
	assert.True(t, loadAutoGenerateAIOnSave(nil), "no config = as before")
}

func TestApprovalHistoryMessage_HoldRelease(t *testing.T) {
	assert.Equal(t, "Site ditahan (Hold Status) oleh citra (Dept Head) dengan alasan (zonasi)", approvalHistoryMessage("HOLD", "citra", "Dept Head", "zonasi"))
	assert.Equal(t, "Hold Status dilepas oleh citra (Dept Head)", approvalHistoryMessage("RELEASE", "citra", "Dept Head", ""))
	assert.Equal(t, "Hold Status dilepas oleh citra (Dept Head) dengan catatan (aman)", approvalHistoryMessage("RELEASE", "citra", "Dept Head", "aman"))
}
