package health

import "github.com/jackc/pgx/v5/pgxpool"

type RepoConn struct {
	Repo *pgxpool.Pool
}

// Функция сохранения подключения к БД в структуре
func SaveRepoConn(conn *pgxpool.Pool) *RepoConn {
	return &RepoConn{Repo: conn}
}
