package provider

type Project struct {
	Id           int64  `json:"id"`
	Name         string `json:"name"`
	Organization string `json:"organization"`
	ApiToken     string `json:"api_token"`
}

type ProjectCreateInput struct {
	Name string `json:"name"`
}
