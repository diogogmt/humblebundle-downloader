package hbclient

type HBError struct {
	Message string `json:"message"`
	Status  string `json:"errors"`
}

type Order struct {
	UID         string     `json:"uid"`
	GameKey     string     `json:"gamekey"`
	Created     string     `json:"created"`
	AmountSpent float64    `json:"amount_spent"`
	Product     *Product   `json:"product"`
	Products    []*Product `json:"subproducts"`
}

type Product struct {
	MachineName string      `json:"machine_name"`
	HumanName   string      `json:"human_name"`
	URL         string      `json:"url"`
	Downloads   []*Download `json:"downloads"`
}

type Download struct {
	MachineName string          `json:"machine_name"`
	HumanName   string          `json:"human_name"`
	Platform    string          `json:"platform"`
	Types       []*DownloadType `json:"download_struct"`
}

type DownloadType struct {
	Name      string          `json:"name"`
	HumanName string          `json:"-"`
	HumanSize string          `json:"human_size"`
	MD5       string          `json:"md5"`
	SHA1      string          `json:"sha1"`
	URL       DownloadTypeURL `json:"url"`
	FileSize  int64           `json:"file_size"`
}

type DownloadTypeURL struct {
	Web        string `json:"web"`
	BitTorrent string `json:"bittorrent"`
}
