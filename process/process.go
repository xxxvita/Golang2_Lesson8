// Пакет process реализазует работу с поддиректориями, заданными
// через указание основной директории. Пакет анализирует файлы внутри директорий
// в нескольких потоках, и ищет дубликаты в них через специальную функцию анализатор.
//
//  func StartDuplicateFind(options Options, ch <-chan FindDuplicate, wg *sync.WaitGroup)
//
// Найденные дубликаты файлов могут быть просто перечислены
// пакетом или удалены. Удалять дубликаты можно с подтверждением пользователя
// через комендную строку. Для уточнения способа работы с дубликатамив пакете
// реализован тип-структура с описанием флагов. Эта структура требуется во всех
// функциях пакета.
//   Настройки поведения для процесса анализа дубликатов
//   type Options struct {
//      // true если требуется подтверждение перед удалением файла
//      MustConfirmationDelete bool
//      // true если требуется удалять файлы-дубликаты
//      NeedRemoveDuplicate bool
//    }
//
//
package process

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sync"
)

type FindDuplicate struct {
	DirName  string
	FileName string
}

type ChanFindDuplicate chan FindDuplicate

// Настройки поведения для процесса анализа дубликатов
type Options struct {
	// true если требуется подтверждение перед удалением файла
	MustConfirmationDelete bool
	// true если требуется удалять файлы-дубликаты
	NeedRemoveDuplicate bool
}

// Анализ файлов из канала на предмет дубликата
// в параметре options передаются настройки поведения анализатора
// в канкле сh передаётся структура с указанием директории, которая сейчас обрабатывается,
// если в options.MustConfirmationDelete == true или указывается название файла,
// совместно с директорией для дальнейшего анализа на дубликаты.
func StartDuplicateFind(options Options, ch <-chan FindDuplicate, wg *sync.WaitGroup) {
	log.Println("Старт поиска дубликатов...")
	defer wg.Done()
	defer log.Println("Поиск дубликатов прекращён")

	mapFiles := map[string]int{}

	for fd := range ch {
		// Если требуется прерывания на согласие пользователя ,
		// то в канале log-строка с текущим каталогом перед его обработкой
		if options.MustConfirmationDelete && fd.FileName == "" {
			log.Printf("Обработка каталога: %s", fd.DirName)
			continue
		}

		log.Printf("Найден файл: %s\n", fd.DirName+"/"+fd.FileName)

		// Если файл fd.FileName уже есть в списке, то это дубликат
		_, ok := mapFiles[fd.FileName]
		// Почему-то запись if _, ok := mapFiles[fd.FileName] ругается на знак подчёркивания
		if ok {
			log.Printf("Найден дубликат. Файл: %s\n", fd.DirName+"/"+fd.FileName)

			// Ожидание ввода пользователя
			if options.MustConfirmationDelete && options.NeedRemoveDuplicate {
				fmt.Printf("Удалить файл %s? (y, n)", fd.DirName+"/"+fd.FileName)
				scanner := bufio.NewScanner(os.Stdin)
				fl := true
				for fl {
					for scanner.Scan() {
						txt := scanner.Text()
						switch txt {
						case "y":
							{
								fmt.Printf("\n Файл %s удалён!\n", fd.DirName+"/"+fd.FileName)
								fl = false
							}
						case "n":
							{
								fmt.Printf("Пропуск\n")
								fl = false
							}
						default:
							fmt.Printf("\nНеверный ввод. Повторите (y/n):")
						}

						break
					}
				}
			} else {
				fmt.Printf("Файл %s удалён!\n", fd.DirName+"/"+fd.FileName)
			}
		} else {
			mapFiles[fd.FileName] = 1
		}
	}
}

// Обход дерева директорий с созданием для каждой поддиректории,
// включая заданную потока для отслеживания файлов-дубликатов
func StartWatch(options Options, fDir *os.File, wg *sync.WaitGroup) error {
	// Запуск слежения за дубликатами в каталогах

	chanDupl := make(ChanFindDuplicate)
	wgDupl := sync.WaitGroup{}

	wgDupl.Add(1)
	go StartDuplicateFind(options, chanDupl, &wgDupl)

	wg.Add(1)
	go func() {
		StartContentChanges(options, fDir, wg, &chanDupl)

		defer wg.Done()
	}()

	// Ожидание закрытия всех воркеров по поиску содержимого директорий
	wg.Wait()

	// Закрыте канала для прекращения работы потока по поиску дубликатов
	close(chanDupl)
	// Ожидание корректного закрытия потока анализа списка файлов на дубликаты
	wgDupl.Wait()

	return nil
}

// Запуск потока для отслеживания изменений в директории
func StartContentChanges(options Options, sDir *os.File, wg *sync.WaitGroup, signalChan *ChanFindDuplicate) {
	// Отправка сообщения "Обработка каталога" в поток анализа файлов
	// если требуется подтверждение от пользователя
	if options.MustConfirmationDelete {
		*signalChan <- FindDuplicate{DirName: sDir.Name()}
	} else {
		log.Printf("Обработка каталога: %s", sDir.Name())
	}

	fileNames, err := sDir.Readdirnames(-1)
	if err != nil {
		log.Fatal(err)
	}

	// Анализируем содержимое директоии (файл и директоии)
	for _, s := range fileNames {
		st, err := os.Stat(sDir.Name() + "/" + s)
		if err != nil {
			log.Fatal(err)
		}

		// Для каждого нового каталога запускается свой поток обработки
		if st.IsDir() {
			sCatalogName := sDir.Name() + "/" + st.Name()

			f, err := os.Open(sCatalogName)
			if err != nil {
				log.Printf("Ошибка чтения каталога (%s): %s\n", sCatalogName, err)
			}

			wg.Add(1)
			go func() {
				StartContentChanges(options, f, wg, signalChan)

				defer wg.Done()
			}()
		} else {
			// Отправка найденного файла в канал для его дальнейшего анализа
			*signalChan <- FindDuplicate{DirName: sDir.Name(), FileName: st.Name()}
		}
	}
}
