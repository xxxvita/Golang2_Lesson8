package process_test

import (
	"FindDuplicate/process"
	"log"
	"os"
	"sync"
	"testing"
)

// Пример запуска старта анализатора дубликатов файлов в директории
// Для перечисления всех дубликатов
func ExampleStartWatch() {
	// Запуск обхода указанной директории
	wg := sync.WaitGroup{}
	options := process.Options{}

	fDir, err := os.Open("WorDir")
	if err != nil {
		log.Fatal("Не верно указана директория")
	}

	options.MustConfirmationDelete = false
	options.NeedRemoveDuplicate = false

	err = process.StartWatch(options, fDir, &wg)
	if err != nil {
		panic(err)
	}
}

func TestStartWatch(t *testing.T) {
	// Запуск обхода указанной директории
	wg := sync.WaitGroup{}
	options := process.Options{}

	fDir, err := os.Open("WorDir")
	if err != nil {
		log.Fatal("Не верно указана директория")
	}

	options.MustConfirmationDelete = false
	options.NeedRemoveDuplicate = false

	err = process.StartWatch(options, fDir, &wg)
	if err != nil {
		t.Error(err)
	}
}
