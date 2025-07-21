package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	tasks      = make(map[string]*Task)  // In-memory хранилище задач
	taskMu     sync.Mutex                // Мьютекс для задач
	active     = make(chan struct{}, 3)  // Семафор для лимита активных задач (3)
)

func main() {
	http.HandleFunc("/create", createTask)
	http.HandleFunc("/add", addLink)
	http.HandleFunc("/status", getStatus)

	log.Printf("Сервер запущен на порту %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

// handler для обработки эндпоинта /create
// создает новую задачу пользователя
func createTask(w http.ResponseWriter, r *http.Request) {
	taskMu.Lock()
	defer taskMu.Unlock()

	if len(active) >= 3 {
		http.Error(w, "Сервер занят: максимум 3 задачи одновременно", http.StatusServiceUnavailable)
		return
	}

	id := fmt.Sprintf("task_%d", time.Now().UnixNano())
	tasks[id] = &Task{ID: id, Status: "pending", Links: []string{}}

	// добавляем новую задачу
	select {
	case active <- struct{}{}:
	default:
		//Не должно произойти из-за проверки выше
	}

	json.NewEncoder(w).Encode(map[string]string{"id": id})
}


// handler для обработки эндпоинта /add
// добавляет новую ссылку с картинкой/документом 
func addLink(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	link := r.URL.Query().Get("link")

	taskMu.Lock()
	task, ok := tasks[id]
	taskMu.Unlock()

	if !ok {
		http.Error(w, "Задача не найдена", http.StatusNotFound)
		return
	}

	if len(task.Links) >= 3 {
		http.Error(w, "Лимит: не более 3 файлов", http.StatusBadRequest)
		return
	}

	if !validateExtension(link) {
		http.Error(w, "Недопустимый тип: только .pdf или .jpeg", http.StatusBadRequest)
		return
	}

	task.Links = append(task.Links, link)

	if len(task.Links) == 3 {
		go processTask(task) // Асинхронная обработка
	}

	w.WriteHeader(http.StatusOK)
}


// handler для обработки эндпоинта /status
// возвращает статус 
func getStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	taskMu.Lock()
	task, ok := tasks[id]
	taskMu.Unlock()

	if !ok {
		http.Error(w, "Задача не найдена", http.StatusNotFound)
		return
	}

	resp := map[string]string{"status": task.Status}
	if task.Status == "done" {
		resp["archive"] = task.ArchiveURL
		resp["errors"] = task.Errors
	}

	json.NewEncoder(w).Encode(resp)
}
