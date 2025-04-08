package main

import (
	"archive/tar"
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// fileData содержит информацию о файле для передачи в канал архиватору.
type fileData struct {
	RelPath string      // Относительный путь файла внутри архива
	Info    fs.FileInfo // Информация о файле
	Content []byte      // Содержимое файла
}

func main() {
	// Инициализация лог-файла
	logFile, err := os.OpenFile("ptar.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Ошибка открытия лог-файла: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Определение флагов командной строки
	inputDir := flag.String("dir", ".", "Директория с файлами для архивации")
	outputFile := flag.String("out", "output.tar", "Имя выходного tar архива")
	maxWorkers := flag.Int("workers", 16, "Максимальное количество горутин для чтения файлов")
	flag.Parse()

	// Проверка входной директории
	inputInfo, err := os.Stat(*inputDir)
	if err != nil {
		log.Fatalf("Ошибка доступа к директории '%s': %v", *inputDir, err)
	}
	if !inputInfo.IsDir() {
		log.Fatalf("'%s' не является директорией", *inputDir)
	}

	// Получаем абсолютный путь к выходному файлу
	absOutputFile, err := filepath.Abs(*outputFile)
	if err != nil {
		log.Fatalf("Ошибка получения абсолютного пути: %v", err)
	}

	// Подсчет общего количества файлов
	var totalFiles int
	err = filepath.WalkDir(*inputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			totalFiles++
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Ошибка при подсчете файлов: %v", err)
	}

	log.Printf("Начинаем архивацию директории '%s' в файл '%s' с %d рабочими горутинами. Всего файлов: %d",
		*inputDir, *outputFile, *maxWorkers, totalFiles)

	// Создание выходного файла с буферизацией на уровне ОС
	var tarFile *os.File
	if *outputFile == "-" {
		tarFile = os.Stdout
	} else {
		var err error
		tarFile, err = os.OpenFile(*outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatalf("Ошибка создания tar файла '%s': %v", *outputFile, err)
		}
	}
	defer tarFile.Close()

	// Создание писателя tar архива
	tw := tar.NewWriter(tarFile)
	defer tw.Close()

	// Канал для передачи данных о файлах от читателей к архиватору
	fileChan := make(chan fileData, *maxWorkers*2)

	// Канал для обработки ошибок
	errChan := make(chan error, 1)

	// Канал для сигнализации о прерывании работы
	doneChan := make(chan struct{})

	// WaitGroup для ожидания завершения всех читающих горутин
	var wg sync.WaitGroup

	// WaitGroup для ожидания завершения архиватора
	var archiverWg sync.WaitGroup

	// Семафор для ограничения количества одновременных операций чтения
	readSemaphore := make(chan struct{}, *maxWorkers)

	// Переменная для отслеживания возникновения ошибки
	var processingError error
	var errorMutex sync.Mutex

	// Функция для проверки наличия ошибки
	hasError := func() bool {
		errorMutex.Lock()
		defer errorMutex.Unlock()
		return processingError != nil
	}

	// Запуск горутины-архиватора
	archiverWg.Add(1)
	go func() {
		defer archiverWg.Done()
		for data := range fileChan {
			// Прерываем обработку, если возникла ошибка
			if hasError() {
				continue
			}

			// Создание заголовка tar
			header, err := tar.FileInfoHeader(data.Info, "")
			if err != nil {
				errorMutex.Lock()
				processingError = err
				errorMutex.Unlock()
				close(doneChan) // Сигнализируем о прерывании работы
				errChan <- err
				return
			}
			header.Name = data.RelPath

			// Запись заголовка
			if err := tw.WriteHeader(header); err != nil {
				errorMutex.Lock()
				processingError = err
				errorMutex.Unlock()
				close(doneChan) // Сигнализируем о прерывании работы
				errChan <- err
				return
			}

			// Запись содержимого файла (если это не директория)
			if !data.Info.IsDir() {
				if _, err := tw.Write(data.Content); err != nil {
					errorMutex.Lock()
					processingError = err
					errorMutex.Unlock()
					close(doneChan) // Сигнализируем о прерывании работы
					errChan <- err
					return
				}
			}

			log.Printf("Добавлен %s: %s (%d байт)",
				map[bool]string{true: "директория", false: "файл"}[data.Info.IsDir()],
				data.RelPath, header.Size)
		}
	}()

	// Счетчик обработанных файлов
	var processedFiles int64
	var processedMutex sync.Mutex

	// Обход директории
	err = filepath.WalkDir(*inputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("Ошибка доступа к '%s': %v", path, err)
			return nil
		}

		// Прерываем обработку, если возникла ошибка
		if hasError() {
			return filepath.SkipAll
		}

		// Получаем абсолютный путь к текущему файлу
		absPath, err := filepath.Abs(path)
		if err != nil {
			log.Printf("Ошибка получения абсолютного пути для '%s': %v", path, err)
			return nil
		}

		// Пропускаем выходной файл и исполняемый файл
		if absPath == absOutputFile || absPath == os.Args[0] {
			log.Printf("ПРОПУЩЕН ФАЙЛ: %s (причина: выходной файл или исполняемый файл)", path)
			return nil
		}

		// Вычисляем относительный путь для tar архива
		relPath, err := filepath.Rel(*inputDir, path)
		if err != nil {
			log.Printf("ОШИБКА: Не удалось получить относительный путь для '%s': %v", path, err)
			return nil
		}

		// Получаем информацию о файле
		info, err := d.Info()
		if err != nil {
			log.Printf("ОШИБКА: Не удалось получить информацию о файле '%s': %v", path, err)
			return nil
		}

		// Для директорий и файлов используем один и тот же механизм
		wg.Add(1)
		go func(filePath, relPath string, fileInfo fs.FileInfo) {
			defer wg.Done()

			// Получаем слот в семафоре только для файлов
			if !fileInfo.IsDir() {
				readSemaphore <- struct{}{}
				defer func() { <-readSemaphore }()
			}

			var content []byte
			var readErr error

			// Читаем содержимое только для файлов
			if !fileInfo.IsDir() {
				content, readErr = os.ReadFile(filePath)
				if readErr != nil {
					log.Printf("ОШИБКА: Не удалось прочитать файл '%s': %v", filePath, readErr)
					return
				}
			}

			select {
			case fileChan <- fileData{
				RelPath: relPath,
				Info:    fileInfo,
				Content: content,
			}:
				processedMutex.Lock()
				processedFiles++
				if processedFiles%1000 == 0 {
					log.Printf("ПРОГРЕСС: Обработано файлов: %d из %d", processedFiles, totalFiles)
				}
				processedMutex.Unlock()
			case <-doneChan:
				// Прерываем обработку по сигналу
				return
			}
		}(path, relPath, info)

		return nil
	})

	if err != nil {
		log.Printf("КРИТИЧЕСКАЯ ОШИБКА: Ошибка обхода директории '%s': %v", *inputDir, err)
		close(doneChan) // Сигнализируем об ошибке, чтобы горутины могли завершиться
	}

	// Ожидание завершения всех читающих горутин
	wg.Wait()

	// Закрытие канала после того, как все читатели завершились
	close(fileChan)

	// Ожидание завершения архиватора
	archiverWg.Wait()

	// Проверка на наличие ошибок
	select {
	case err := <-errChan:
		log.Printf("КРИТИЧЕСКАЯ ОШИБКА: Ошибка при архивации: %v", err)
	default:
		if processingError != nil {
			log.Printf("КРИТИЧЕСКАЯ ОШИБКА: Ошибка при архивации: %v", processingError)
		} else {
			log.Printf("УСПЕХ: Архивация завершена успешно. Обработано файлов: %d из %d", processedFiles, totalFiles)
		}
	}
}
