package domain

type FilterEvents struct {
	TeamOwner   []string `query:"team_owner"`
	ServiceName []string `query:"service_name"`
	State       []string `query:"state"`
	Page        uint     `query:"page"`
	Limit       uint     `query:"limit"`
}
