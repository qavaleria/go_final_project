package handlers

import (
	"database/sql"
	"encoding/json"
	"github.com/qavaleria/go_final_project/models"
	"github.com/qavaleria/go_final_project/tasks"
	"log"
	"net/http"
	//"strconv"
	"time"
)

const LimitDB = "50"

// обработчик для API запроса /api/task
func HandleTask(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		switch r.Method {
		case http.MethodPost:
			var task models.Task
			err := json.NewDecoder(r.Body).Decode(&task)
			if err != nil {
				log.Printf("Ошибка десериализации JSON: %v", err)
				http.Error(w, "Ошибка десериализации JSON: "+err.Error(), http.StatusBadRequest)
				return
			}

			if task.Title == "" {
				log.Printf("Не указан заголовок задачи")
				http.Error(w, "Не указан заголовок задачи", http.StatusBadRequest)
				return
			}

			now := time.Now()
			if task.Date == "" {
				task.Date = now.Format(tasks.FormatDate)
			} else {
				date, err := time.Parse(tasks.FormatDate, task.Date)
				if err != nil {
					log.Printf("Дата представлена в неправильном формате")
					http.Error(w, "Дата представлена в неправильном формате", http.StatusBadRequest)
					return
				}

				if date.Before(now) {
					if task.Repeat == "" {
						task.Date = now.Format(tasks.FormatDate)
					} else {
						nextDate, err := tasks.NextDate(now, task.Date, task.Repeat)
						if err != nil {
							log.Printf("Ошибка вычисления следующей даты:  %v", err)
							http.Error(w, "Ошибка вычисления следующей даты: "+err.Error(), http.StatusBadRequest)
							return
						}
						task.Date = nextDate
					}
				}
			}

			if task.Repeat != "" {
				if err := tasks.ValidateRepeatRule(task.Repeat); err != nil {
					log.Printf("Правило повторения указано в неправильном формате")
					http.Error(w, "Правило повторения указано в неправильном формате", http.StatusBadRequest)
					return
				}

				nextDate, err := tasks.NextDate(now, task.Date, task.Repeat)
				if err != nil {
					log.Printf("Ошибка вычисления следующей даты: %v", err)
					http.Error(w, "Ошибка вычисления следующей даты: "+err.Error(), http.StatusBadRequest)
					return
				}
				task.Date = nextDate
				log.Printf("Cледующая дата: %s", nextDate)
			}

			id, err := tasks.AddTask(db, task)
			if err != nil {
				log.Printf("Ошибка добавления задачи: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			response := map[string]interface{}{
				"id": id,
			}
			json.NewEncoder(w).Encode(response)
		case http.MethodGet:
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, `{"error": "Не указан идентификатор"}`, http.StatusBadRequest)
				return
			}

			var task models.Task
			query := `SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?`
			err := db.QueryRow(query, id).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
			if err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
				} else {
					log.Printf("Ошибка выполнения запроса: %v", err)
					http.Error(w, `{"error": "Ошибка выполнения запроса"}`, http.StatusInternalServerError)
				}
				return
			}
			json.NewEncoder(w).Encode(task)
		case http.MethodPut:
			var task models.Task
			err := json.NewDecoder(r.Body).Decode(&task)
			if err != nil {
				log.Printf("Ошибка десериализации JSON: %v", err)
				http.Error(w, `{"error": "Ошибка десериализации JSON"}`, http.StatusBadRequest)
				return
			}

			if task.ID == "" {
				log.Printf("Не указан идентификатор задачи")
				http.Error(w, `{"error": "Не указан идентификатор задачи"}`, http.StatusBadRequest)
				return
			}

			if task.Title == "" {
				log.Printf("Не указан заголовок задачи")
				http.Error(w, `{"error": "Не указан заголовок задачи"}`, http.StatusBadRequest)
				return
			}

			now := time.Now()
			if task.Date == "" {
				task.Date = now.Format(tasks.FormatDate)
			} else {
				date, err := time.Parse(tasks.FormatDate, task.Date)
				if err != nil {
					log.Printf("Дата представлена в неправильном формате")
					http.Error(w, `{"error": "Дата представлена в неправильном формате"}`, http.StatusBadRequest)
					return
				}

				if date.Before(now) {
					if task.Repeat == "" {
						task.Date = now.Format(tasks.FormatDate)
					} else {
						nextDate, err := tasks.NextDate(now, task.Date, task.Repeat)
						if err != nil {
							log.Printf("Ошибка вычисления следующей даты: %v", err)
							http.Error(w, `{"error": "Ошибка вычисления следующей даты"}`, http.StatusBadRequest)
							return
						}
						task.Date = nextDate
					}
				}
			}

			if task.Repeat != "" {
				if err := tasks.ValidateRepeatRule(task.Repeat); err != nil {
					log.Printf("Правило повторения указано в неправильном формате")
					http.Error(w, `{"error": "Правило повторения указано в неправильном формате"}`, http.StatusBadRequest)
					return
				}
			}

			query := `UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?`
			res, err := db.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.ID)
			if err != nil {
				log.Printf("Ошибка выполнения запроса: %v", err)
				http.Error(w, `{"error": "Ошибка выполнения запроса"}`, http.StatusInternalServerError)
				return
			}

			rowsAffected, err := res.RowsAffected()
			if err != nil {
				log.Printf("Ошибка получения результата запроса: %v", err)
				http.Error(w, `{"error": "Ошибка получения результата запроса"}`, http.StatusInternalServerError)
				return
			}

			if rowsAffected == 0 {
				http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
				return
			}

			json.NewEncoder(w).Encode(map[string]interface{}{})
		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, `{"error": "Не указан идентификатор задачи"}`, http.StatusBadRequest)
				return
			}

			deleteQuery := `DELETE FROM scheduler WHERE id = ?`
			res, err := db.Exec(deleteQuery, id)
			if err != nil {
				log.Printf("Ошибка выполнения запроса: %v", err)
				http.Error(w, `{"error": "Ошибка выполнения запроса"}`, http.StatusInternalServerError)
				return
			}

			rowsAffected, err := res.RowsAffected()
			if err != nil {
				log.Printf("Ошибка получения результата запроса: %v", err)
				http.Error(w, `{"error": "Ошибка получения результата запроса"}`, http.StatusInternalServerError)
				return
			}

			if rowsAffected == 0 {
				http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
				return
			}

			json.NewEncoder(w).Encode(map[string]interface{}{})
		default:
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		}
	}
}

func HandleGetTasks(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		rows, err := db.Query(`SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT ?`, LimitDB)
		if err != nil {
			log.Printf("Ошибка выполнения запроса: %v", err)
			http.Error(w, "Ошибка выполнения запроса: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		tasks := []models.Task{} // Инициализируем пустой слайс

		for rows.Next() {
			var task models.Task
			if err := rows.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat); err != nil {
				log.Printf("Ошибка чтения строки: %v", err)
				http.Error(w, "Ошибка чтения строки: "+err.Error(), http.StatusInternalServerError)
				return
			}
			tasks = append(tasks, task)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Ошибка обработки результата: %v", err)
			http.Error(w, "Ошибка обработки результата: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"tasks": tasks,
		}
		json.NewEncoder(w).Encode(response)
	}
}

func HandleMarkTaskDone(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error": "Не указан идентификатор задачи"}`, http.StatusBadRequest)
			return
		}

		var task models.Task
		query := `SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?`
		err := db.QueryRow(query, id).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, `{"error": "Задача не найдена"}`, http.StatusNotFound)
			} else {
				log.Printf("Ошибка выполнения запроса: %v", err)
				http.Error(w, `{"error": "Ошибка выполнения запроса"}`, http.StatusInternalServerError)
			}
			return
		}

		if task.Repeat == "" {
			// Удаляем одноразовую задачу
			deleteQuery := `DELETE FROM scheduler WHERE id = ?`
			_, err := db.Exec(deleteQuery, id)
			if err != nil {
				log.Printf("Ошибка удаления задачи: %v", err)
				http.Error(w, `{"error": "Ошибка удаления задачи"}`, http.StatusInternalServerError)
				return
			}
		} else {
			// Рассчитываем следующую дату для периодической задачи
			now := time.Now()
			nextDate, err := tasks.NextDate(now, task.Date, task.Repeat)
			if err != nil {
				log.Printf("Ошибка вычисления следующей даты: %v", err)
				http.Error(w, `{"error": "Ошибка вычисления следующей даты"}`, http.StatusInternalServerError)
				return
			}

			// Обновляем дату задачи
			updateQuery := `UPDATE scheduler SET date = ? WHERE id = ?`
			_, err = db.Exec(updateQuery, nextDate, id)
			if err != nil {
				log.Printf("Ошибка обновления задачи: %v", err)
				http.Error(w, `{"error": "Ошибка обновления задачи"}`, http.StatusInternalServerError)
				return
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{})
	}
}
