package notifications

type getFileResponse struct {
	Result struct {
		FilePath string `json:"file_path"`
	}
}
