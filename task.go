package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

type Task struct {
	ID         string
	Status     string
	Links      []string
	ArchiveURL string
	Errors     string
}

// проверяет, является ли картинка/документ разрещенным
// link - ссылка на картинку/документ пользователя
func validateExtension(link string) bool {
	ext := strings.ToLower(filepath.Ext(link))
	for _, e := range extensions {
		if ext == e {
			return true
		}
	}
	return false
}

// создает zip-файл со ссылками пользователя
// t - содержит информацию о текущем пользователе
func processTask(t *Task) {

	// очищаем место для след задачи
	defer func() { <-active }()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	var errs []string

	for i, link := range t.Links {
		resp, err := http.Get(link)
		if err != nil || resp.StatusCode != http.StatusOK {
			errs = append(errs, fmt.Sprintf("Ссылка %d недоступна: %s", i+1, link))
			continue
		}
		defer resp.Body.Close()

		fname := fmt.Sprintf("file_%d%s", i+1, filepath.Ext(link))
		fw, err := zw.Create(fname)
		if err != nil {
			errs = append(errs, fmt.Sprintf("Ошибка упаковки %s", link))
			continue
		}

		_, err = io.Copy(fw, resp.Body)
		if err != nil {
			errs = append(errs, fmt.Sprintf("Ошибка скачивания %s", link))
		}
	}

	zw.Close()

	taskMu.Lock()
	t.Status = "done"
	t.ArchiveURL = fmt.Sprintf("/archive/%s.zip", t.ID) // Симуляция URL (в реальности - endpoint для скачивания)
	if len(errs) > 0 {
		t.Errors = strings.Join(errs, "; ")
	}
	taskMu.Unlock()
}
