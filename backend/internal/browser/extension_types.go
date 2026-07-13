package browser

type Extension struct {
	ExtensionID  string `json:"extensionId"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	Description  string `json:"description"`
	IconDataURL  string `json:"iconDataUrl"`
	ManifestJSON string `json:"manifestJson"`
	SourceURL    string `json:"sourceUrl"`
	InstallDir   string `json:"installDir"`
	Enabled      bool   `json:"enabled"`
	InstalledAt  string `json:"installedAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type ExtensionLookupResult struct {
	ExtensionID string `json:"extensionId"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	StoreURL    string `json:"storeUrl"`
	Installable bool   `json:"installable"`
	Message     string `json:"message"`
}

type ProfileExtensionSettings struct {
	ProfileID    string   `json:"profileId"`
	Configured   bool     `json:"configured"`
	ExtensionIDs []string `json:"extensionIds"`
	UpdatedAt    string   `json:"updatedAt"`
}
