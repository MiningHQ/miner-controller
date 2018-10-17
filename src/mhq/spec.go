package mhq

// Progress holds information about the current download progress
type Progress struct {
	BytesCompleted int64
	BytesTotal     int64
}

// RecommendedMinerResponse contains the recommended miners (if any)
// from the MiningHQ API
type RecommendedMinerResponse struct {
	Status  string             `json:"Status"`
	Message string             `json:"Message"`
	Miners  []RecommendedMiner `json:"Miners"`
}

// RecommendedMiner contains the information to download a recommended miner
type RecommendedMiner struct {
	Name           string `json:"Name"`
	Version        string `json:"Version"`
	Type           string `json:"Type"`
	DownloadLink   string `json:"DownloadLink"`
	DownloadSHA512 string `json:"DownloadSHA512"`
	SizeBytes      int64  `json:"SizeBytes"`
}
