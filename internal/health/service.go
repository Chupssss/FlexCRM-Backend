package health

type Service struct {
	repo *RepoConn
}

// Функция создания основного сервиса(взаимодействие между бд и хэндлерами)
func NewService(repo_conn *RepoConn) *Service {
	return &Service{repo: repo_conn}
}
