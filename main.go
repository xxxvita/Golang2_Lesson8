package main

/*
go run main.go -a WorkDir
go run main.go -a -r WorkDir
*/

import (
	"FindDuplicate/process"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	flagR *bool
	flagA *bool
)

func main() {
	flagR = flag.Bool("r", false, "Удаляет найденные дубликаты в подкаталогах")
	flagA = flag.Bool("a", false, "Перед удалением не спрашивать подтверждения")
	flag.Parse()

	if len(os.Args) == 1 {
		panic("Число параметров должно быть больше одного")
	}

	fmt.Println("Start")

	fDir, err := os.Open(flag.Args()[0])
	if err != nil {
		log.Fatal("Не верно указана директория")
	}

	// Запуск обхода указанной директории
	wg := sync.WaitGroup{}
	options := process.OptionsNew()

	options.MustConfirmationDeleteSet(!*flagA)
	options.NeedRemoveDuplicateSet(*flagR)
	options.MaxCountThreadSet(1)

	err = process.StartWatch(options, fDir, &wg)
	if err != nil {
		panic(err)
	}

	fmt.Println("Finish")
}
