package TaskManager

import (
	"github.com/google/uuid"
	"sync"
)

type TaskList struct {
	Tasks []PaktumTask `json:"tasks"`
	sync.Mutex
}

type PaktumTask struct {
	TaskUID string `json:"task_uid"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Returned  interface{} `json:"output"`
	Done    bool   `json:"done"`
}

var TL TaskList = TaskList{
	Tasks: nil,
}

func init() {
	TL.Tasks = make([]PaktumTask, 0)
}

func NewTask(taskType string) string {
	TL.Lock()
	defer TL.Unlock()

	var newTask = PaktumTask{
		TaskUID: generateUUID(),
		Type:    taskType,
		Returned: "",
		Done:    false,
	}

	TL.Tasks = append(TL.Tasks, newTask)

	return newTask.TaskUID
}

func GetTask(taskUID string) PaktumTask {
	TL.Lock()
	defer TL.Unlock()

	for _, task := range TL.Tasks {
		if task.TaskUID == taskUID {
			return task
		}
	}

	return PaktumTask{}
}

func SetTaskStatus(taskUID string, status string) {
	TL.Lock()
	defer TL.Unlock()

	for i, task := range TL.Tasks {
		if task.TaskUID == taskUID {
			TL.Tasks[i].Status = status
			return
		}
	}
}

func SetTaskDone(taskUID string) {
	TL.Lock()
	defer TL.Unlock()

	for i, task := range TL.Tasks {
		if task.TaskUID == taskUID {
			TL.Tasks[i].Done = true
			return
		}
	}
}

func SetTaskOutput(taskUID string, output interface{}) {
	TL.Lock()
	defer TL.Unlock()

	for i, task := range TL.Tasks {
		if task.TaskUID == taskUID {
			TL.Tasks[i].Returned = output
			return
		}
	}
}

func GetTasks() []PaktumTask {
	TL.Lock()
	defer TL.Unlock()

	return TL.Tasks
}

func generateUUID() string {
	return uuid.New().String()
}