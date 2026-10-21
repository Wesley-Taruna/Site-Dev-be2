package dto

// UploadResponse is POST /v1/upload/add (site-dev-api UploadResp): the
// stored name and its "<bucket>/<object...>" path.
type UploadResponse struct {
	FileName string `json:"file_name"`
	Path     string `json:"path"`
}

// UploadRemoveRequest is the body of POST /v1/upload/remove.
type UploadRemoveRequest struct {
	Path string `json:"path" valid:"required"`
}
