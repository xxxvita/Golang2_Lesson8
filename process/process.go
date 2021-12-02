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
	FileSize int64
	workerId uint16
}

type ChanFindDuplicate chan FindDuplicate

// Настройки поведения для процесса анализа дубликатов
type Options struct {
	// true если требуется подтверждение перед удалением файла
	mustConfirmationDelete bool
	// true если требуется удалять файлы-дубликаты
	needRemoveDuplicate bool
	// Максимальное число потоков, анализируюбщие директории
	// -1 - бесконечное число потоков
	maxCountThread int16
	// Текущее число работабщих потоков, анализирующие директории
	currentThreadCount int16
	// Защита доступа к данным структуры из горутин
	mux sync.RWMutex
}

// Получение экземпляра указателя на структуру с настройками по умолчанию
func OptionsNew() *Options {
	return &Options{
		mustConfirmationDelete: true,
		needRemoveDuplicate:    false,
		maxCountThread:         -1,
		mux:                    sync.RWMutex{},
	}
}

// Сеттер для MustConfirmationDelete
func (o *Options) MustConfirmationDeleteSet(val bool) {
	o.mux.Lock()
	defer o.mux.Unlock()

	o.mustConfirmationDelete = val
}

// Геттер для MustConfirmationDelete
func (o *Options) MustConfirmationDeleteGet() bool {
	o.mux.RLock()
	defer o.mux.RUnlock()

	return o.mustConfirmationDelete
}

// Сеттер для needRemoveDuplicate
func (o *Options) NeedRemoveDuplicateSet(val bool) {
	o.mux.Lock()
	defer o.mux.Unlock()

	o.needRemoveDuplicate = val
}

// Геттер для needRemoveDuplicate
func (o *Options) NeedRemoveDuplicateGet() bool {
	o.mux.RLock()
	defer o.mux.RUnlock()

	return o.needRemoveDuplicate
}

// Геттер для maxCountThread
func (o *Options) MaxCountThreadGet() int16 {
	o.mux.RLock()
	defer o.mux.RUnlock()

	return o.maxCountThread
}

// Сеттер для maxCountThread
func (o *Options) MaxCountThreadSet(val int16) {
	o.mux.Lock()
	defer o.mux.Unlock()

	o.maxCountThread = val
}

// Геттер для currentThreadCount
func (o *Options) CurrentThreadCountGet() int16 {
	o.mux.RLock()
	defer o.mux.RUnlock()

	return o.currentThreadCount
}

// Сеттер для currentThreadCount
func (o *Options) CurrentThreadCountSet(val int16) {
	o.mux.Lock()
	defer o.mux.Unlock()

	o.currentThreadCount = val
}

// Если возможно добавить поток-воркер (currentThreadCount < MaxCountThread),
// то добавляется новая горутина и функция возвращает true, иначе возвращает false
func (o *Options) AddWorker() bool {
	o.mux.Lock()
	defer o.mux.Unlock()

	if o.maxCountThread == -1 || o.currentThreadCount < o.maxCountThread {
		o.currentThreadCount++
		return true
	}

	return false
}

// Уменьшение числа текущих потоков
func (o *Options) RemoveWorker() {
	o.mux.Lock()
	defer o.mux.Unlock()

	o.currentThreadCount--
}

// Анализ файлов из канала на предмет дубликата
// в параметре options передаются настройки поведения анализатора
// в канкле сh передаётся структура с указанием директории, которая сейчас обрабатывается,
// если в options.MustConfirmationDelete == true или указывается название файла,
// совместно с директорией для дальнейшего анализа на дубликаты.
func StartDuplicateFind(options *Options, ch <-chan FindDuplicate, wg *sync.WaitGroup) {
	log.Println("Старт поиска дубликатов...")
	defer wg.Done()
	defer log.Println("Поиск дубликатов прекращён")

	mapFiles := map[string]struct{}{}

	for fd := range ch {
		// Если требуется прерывания на согласие пользователя ,
		// то в канале log-строка с текущим каталогом перед его обработкой
		if options.MustConfirmationDeleteGet() && fd.FileName == "" {
			log.Printf("Обработка каталога (workerId: %d): %s", fd.workerId, fd.DirName)
			continue
		}

		log.Printf("Найден файл: %s\n", fd.DirName+"/"+fd.FileName)

		// Если файл fd.FileName + fd.Size уже есть в списке, то это дубликат
		sMapKey := fmt.Sprintf("%s_%d", fd.FileName, fd.FileSize)
		_, ok := mapFiles[sMapKey]
		// Почему-то запись if _, ok := mapFiles[fd.FileName] ругается на знак подчёркивания
		if ok {
			log.Printf("Найден дубликат. Файл: %s (size: %d)\n", fd.DirName+"/"+fd.FileName, fd.FileSize)

			// Ожидание ввода пользователя
			if options.MustConfirmationDeleteGet() && options.NeedRemoveDuplicateGet() {
				fmt.Printf("Удалить файл %s (size: %d)? (y, n)", fd.DirName+"/"+fd.FileName, fd.FileSize)
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
							fmt.Printf("Неверный ввод. Повторите (y/n):")
						}

						break
					}
				}
			} else {
				fmt.Printf("Файл %s удалён!\n", fd.DirName+"/"+fd.FileName)
			}
		} else {
			mapFiles[sMapKey] = struct{}{}
		}
	}
}

// Обход дерева директорий с созданием для каждой поддиректории,
// включая заданную потока для отслеживания файлов-дубликатов
func StartWatch(options *Options, fDir *os.File, wg *sync.WaitGroup) error {
	// Запуск слежения за дубликатами в каталогах

	chanDupl := make(ChanFindDuplicate)
	wgDupl := sync.WaitGroup{}

	wgDupl.Add(1)
	go StartDuplicateFind(options, chanDupl, &wgDupl)

	wg.Add(1)
	go func() {
		// Первый поток
		options.CurrentThreadCountSet(1)
		err := StartContentChanges(options, fDir, wg, &chanDupl, 1)
		if err != nil {
			log.Fatal(err)
		}

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
func StartContentChanges(options *Options, sDir *os.File, wg *sync.WaitGroup, signalChan *ChanFindDuplicate, idWorker uint16) error {
	// Отправка сообщения "Обработка каталога" в поток анализа файлов
	// если требуется подтверждение от пользователя
	if options.MustConfirmationDeleteGet() {
		*signalChan <- FindDuplicate{DirName: sDir.Name()}
	} else {
		log.Printf("Обработка каталога (workerId:%d): %s", idWorker, sDir.Name())
	}

	fileNames, err := sDir.Readdirnames(-1)
	if err != nil {
		return fmt.Errorf("ошибка чтения каталога %s: %s", sDir.Name(), err)
	}

	// Анализируем содержимое директории (файл и директории)
	for _, s := range fileNames {
		st, err := os.Stat(sDir.Name() + "/" + s)
		if err != nil {
			return fmt.Errorf("ошибка получения информации о файле в каталоге %s: %s", sDir.Name(), err)
		}

		// Для каждого нового каталога запускается свой поток обработки
		if st.IsDir() {
			sCatalogName := sDir.Name() + "/" + st.Name()

			f, err := os.Open(sCatalogName)
			if err != nil {
				return fmt.Errorf("ошибка чтения каталога %s: %s", sCatalogName, err)
			}

			// Если можно запустить воркер для анализа директории
			if options.AddWorker() {
				wg.Add(1)
				go func() {
					StartContentChanges(options, f, wg, signalChan, idWorker+1)

					defer wg.Done()
					defer options.RemoveWorker()
				}()
			} else {
				StartContentChanges(options, f, wg, signalChan, idWorker)
			}
		} else {
			// Отправка найденного файла в канал для его дальнейшего анализа
			*signalChan <- FindDuplicate{DirName: sDir.Name(), FileName: st.Name(), FileSize: st.Size()}
		}
	}

	return nil
}
