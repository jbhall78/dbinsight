package main

type UserMap struct {
	users  []*UserMapItem
	config *Config
}

type UserMapItem struct {
	proxy_user   string
	proxy_pass   string
	backend_user string
	backend_pass string
}

func NewUserMap(config *Config) *UserMap {
	return &UserMap{
		config: config,
		users:  make([]*UserMapItem, 0),
	}
}
func NewUserMapItem(proxy_user string, proxy_pass string, backend_user string, backend_pass string) *UserMapItem {
	return &UserMapItem{
		proxy_user:   proxy_user,
		proxy_pass:   proxy_pass,
		backend_user: backend_user,
		backend_pass: backend_pass,
	}
}

func (userMap *UserMap) Initialize() error {

	for _, item := range userMap.config.AuthenticationMap {
		userMapItem := NewUserMapItem(item.ProxyUser, item.ProxyPassword, item.BackendUser, item.BackendPassword)
		userMap.users = append(userMap.users, userMapItem)
	}

	return nil
}
