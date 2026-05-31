package backend

// SnapshotInfo 快照元数据
type SnapshotInfo struct {
	SnapshotId string  `json:"snapshotId"`
	ProfileId  string  `json:"profileId"`
	Name       string  `json:"name"`
	SizeMB     float64 `json:"sizeMB"`
	CreatedAt  string  `json:"createdAt"`
	FilePath   string  `json:"filePath,omitempty"`
}
