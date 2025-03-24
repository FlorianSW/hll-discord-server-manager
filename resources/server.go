package resources

type Server struct {
	ServerId string `json:"server_id"`
	Name     string `json:"name"`

	CRConCredentials   *CRConCredentials   `json:"crcon_credentials"`
	TCAdminCredentials *TCAdminCredentials `json:"tcadmin_credentials"`
}

func (s Server) Id() string {
	return s.ServerId
}

type CRConCredentials struct {
	BaseUrl string `json:"base_url"`
	ApiKey  string `json:"api_key"`
}

type TCAdminCredentials struct {
	BaseUrl   string `json:"base_url"`
	ServiceId string `json:"service_id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}
