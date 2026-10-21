package router

// Side-by-side check, module 6a (files): POST /v1/upload/add, /remove and
// GET /v1/attachment/:id/file against site-dev-api 20eeab4. The file
// storage is a mock (the real Google Cloud Storage client is a TODO, plan
// Q4).

import (
	"bytes"
	"database/sql/driver"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	jwtx "gitlab.com/fami-app-v2/familib/familabs/jwt"
)

// copy of site-dev-api 20eeab4 dto.UploadResp (unchanged)
type oldUploadResp struct {
	FileName string `json:"file_name"`
	Path     string `json:"path"`
}

func withStorage(t *testing.T) *mocks.IStorageClient {
	storage := mocks.NewIStorageClient(t)
	withClients(t, func(c *client.Clients) { c.Storage = storage })
	testEnv.Set("gcs.upload.temp_bucket", "test-bucket")
	testEnv.Set("gcs.upload.temp_prefix", "site-dev/test/temp")
	t.Cleanup(func() {
		testEnv.Set("gcs.upload.temp_bucket", "")
		testEnv.Set("gcs.upload.temp_prefix", "")
	})
	return storage
}

func upload(t *testing.T, h http.Handler, field, name, content string) (int, []byte) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile(field, name)
	require.NoError(t, err)
	_, _ = io.WriteString(part, content)
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, "/v1/upload/add", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	token, err := jwtx.NewJWT([]byte(testJwtKey)).Create(&jwtx.UserClaim{UserID: 1001, ExpiresAt: time.Now().Add(time.Hour).Unix()})
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code, rr.Body.Bytes()
}

func TestSideBySide_UploadAdd(t *testing.T) {
	storage := withStorage(t)
	var written string
	storage.On("Write", mock.Anything, "test-bucket", mock.MatchedBy(func(o string) bool {
		return regexp.MustCompile(`^site-dev/test/temp/depan-\d{6}\.jpg$`).MatchString(o)
	}), mock.Anything).Run(func(args mock.Arguments) {
		b, _ := io.ReadAll(args.Get(3).(io.Reader))
		written = string(b)
	}).Return(nil)
	h, _, _ := stack(t)

	code, body := upload(t, h, "file", "../../depan.jpg", "jpeg bytes")

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Equal(t, "jpeg bytes", written)
	// name is random: rebuild the expected answer from the returned name
	name := regexp.MustCompile(`depan-\d{6}\.jpg`).FindString(string(body))
	require.NotEmpty(t, name)
	assertSameJSON(t, oldDataEnvelope(t, oldUploadResp{FileName: name, Path: "test-bucket/site-dev/test/temp/" + name}), body)
}

func TestSideBySide_UploadAdd_NoFile(t *testing.T) {
	withStorage(t)
	h, _, _ := stack(t)

	code, body := upload(t, h, "photo", "x.jpg", "x")

	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assertSameJSON(t, []byte(`{"code":422,"status":"failure","message":"Unprocessable Entity","errors":{"file":"The file is invalid value."}}`), body)
}

func TestUploadAdd_StorageNotSetUpYet(t *testing.T) {
	// TODO marker: without the real storage client the endpoint answers 500
	h, _, _ := stack(t)

	code, body := upload(t, h, "file", "x.jpg", "x")

	assert.Equal(t, http.StatusInternalServerError, code)
	assert.Contains(t, string(body), "storage: Google Cloud Storage client is not set up yet (TODO, plan Q4)")
}

func TestSideBySide_UploadRemove(t *testing.T) {
	storage := withStorage(t)
	storage.On("Delete", mock.Anything, "test-bucket", "site-dev/test/temp/depan-123456.jpg").Return(nil)
	h, _, _ := stack(t)

	code, body := call(t, h, http.MethodPost, "/v1/upload/remove", map[string]any{"path": "test-bucket/site-dev/test/temp/depan-123456.jpg"}, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success"}`), body)

	code, body = call(t, h, http.MethodPost, "/v1/upload/remove", map[string]any{"path": "no-slash"}, 1001)
	assert.Equal(t, http.StatusInternalServerError, code)
	assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"invalid object path: \"no-slash\""}}`), body)
}

func TestSideBySide_AttachmentFile(t *testing.T) {
	storage := withStorage(t)
	storage.On("Open", mock.Anything, "bucket", "sites/ST1/depan.jpg").Return(io.NopCloser(strings.NewReader("jpeg bytes")), nil)
	h, rec, _ := stack(t, sqlfake.Result{Match: `FROM "SiteAttachments" WHERE SiteAttachmentID = @p1 ORDER BY "SiteAttachments"."SiteAttachmentID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY |41`,
		Columns: []string{"SiteAttachmentID", "SiteID", "FileName", "Path", "CreatedAt"}, Rows: [][]driver.Value{{int64(41), "ST1", "depan.jpg", "bucket/sites/ST1/depan.jpg", created}}})

	req := httptest.NewRequest(http.MethodGet, "/v1/attachment/41/file", nil)
	token, _ := jwtx.NewJWT([]byte(testJwtKey)).Create(&jwtx.UserClaim{UserID: 1001, ExpiresAt: time.Now().Add(time.Hour).Unix()})
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	assert.Equal(t, "image/jpeg", rr.Header().Get("Content-Type"))
	assert.Equal(t, "jpeg bytes", rr.Body.String())
	assert.Equal(t, []string{`SELECT * FROM "SiteAttachments" WHERE SiteAttachmentID = 41 ORDER BY "SiteAttachments"."SiteAttachmentID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`}, rec.sqls)

	code, body := call(t, h, http.MethodGet, "/v1/attachment/99/file", nil, 1001)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assertSameJSON(t, []byte(`{"code":422,"status":"failure","message":"Unprocessable Entity","errors":{"attachment":"The attachment is not found"}}`), body)
}
