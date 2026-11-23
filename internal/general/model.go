package general

import "time"

// Response соответствует GeneralResponse в Swagger.
type Response struct {
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	Role            string     `json:"role"`
	CountOfAllDeals int64      `json:"count_of_all_deals"`
	CountClients    int64      `json:"count_clients"`
	CountTasks      int64      `json:"count_tasks"`
	Activities      []Activity `json:"activities"`
	Tasks           []Task     `json:"tasks"`
}

type Activity struct {
	ActorName string    `json:"actor_name"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

type Task struct {
	Title string     `json:"title"`
	DueAt *time.Time `json:"due_at"`
}
