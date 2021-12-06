package process_test

import (
	"FindDuplicate/process"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

const TestDir = "TestDir"

// Пример запуска старта анализатора дубликатов файлов в директории
// Для перечисления всех дубликатов
func ExampleStartWatch() {
	// Запуск обхода указанной директории
	wg := sync.WaitGroup{}
	options := process.OptionsNew()

	fDir, err := os.Open("WorDir")
	if err != nil {
		log.Fatal("Не верно указана директория")
	}

	options.MustConfirmationDeleteSet(false)
	options.NeedRemoveDuplicateSet(false)

	err = process.StartWatch(options, fDir, &wg)
	if err != nil {
		panic(err)
	}
}

// Тест с текущей директорией, файлы не удаляются, разрешения пользователя не требуется
func TestStartWatch(t *testing.T) {
	// Запуск обхода указанной директории
	wg := sync.WaitGroup{}
	options := process.OptionsNew()

	fDir, err := os.Open(".")
	if err != nil {
		log.Fatal("Не верно указана директория")
	}

	options.MustConfirmationDeleteSet(false)
	options.NeedRemoveDuplicateSet(false)

	err = process.StartWatch(options, fDir, &wg)
	if err != nil {
		t.Error(err)
	}
}

// Тест создаёт директории для работы потоков поиска
// Сравнивает время работы процессов в одной горутине и в многопоточной модели горутин
func TestOneAndMultyGoroutine(t *testing.T) {
	tOne := time.Now()

	// Прогон с одним потоком
	wg := sync.WaitGroup{}
	options := process.OptionsNew()

	fDir, err := os.Open(".")
	if err != nil {
		log.Fatal("Не верно указана директория")
	}

	options.MustConfirmationDeleteSet(false)
	options.NeedRemoveDuplicateSet(false)
	options.MaxCountThreadSet(1)

	err = process.StartWatch(options, fDir, &wg)
	if err != nil {
		t.Error(err)
	}

	wg.Wait()

	deltaOne := time.Since(tOne)

	// Прогон с неограниченным числом потоков

	tMulty := time.Now()
	wg1 := sync.WaitGroup{}
	options1 := process.OptionsNew()

	options1.MustConfirmationDeleteSet(false)
	options1.NeedRemoveDuplicateSet(false)
	//options1.MaxCountThreadSet(1)

	err = process.StartWatch(options1, fDir, &wg1)
	if err != nil {
		t.Error(err)
	}

	wg1.Wait()
	deltaMulty := time.Since(tMulty)

	if deltaMulty > deltaOne {
		t.Failed()
	}

}
