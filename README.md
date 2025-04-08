# PTAR - Parallel File Archiver

PTAR is a high-performance command-line utility written in Go that allows you to quickly archive large directories with files using parallel processing. The utility creates standard TAR archives, optimizing the process of reading files from disk by using multiple goroutines.

## Features

- **Parallel file reading** for maximum performance
- Creation of standard **TAR archives** compatible with all popular archivers
- **Customizable number of worker goroutines** for optimization for specific hardware
- **Detailed logging** of the archiving process to a separate file
- **Real-time progress display** of archiving
- **Automatic skipping** of output file and executable file to prevent recursive archiving
- **Error handling** with detailed output of information about problematic files

## Installation

### From source code

```bash
CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o ptar .
```


----
----


# PTAR - Параллельный архиватор файлов

PTAR - это высокопроизводительная утилита командной строки, написанная на Go, которая позволяет быстро архивировать большие директории с файлами, используя параллельную обработку. Утилита создает стандартные TAR-архивы, оптимизируя процесс чтения файлов с диска за счет использования нескольких горутин.

## Особенности

- **Параллельное чтение файлов** для максимальной производительности
- Создание стандартных **TAR-архивов**, совместимых со всеми популярными архиваторами
- **Настраиваемое количество рабочих горутин** для оптимизации под конкретное оборудование
- **Подробное логирование** процесса архивации в отдельный файл
- **Отображение прогресса** архивации в реальном времени
- **Автоматический пропуск** выходного файла и исполняемого файла для предотвращения рекурсивной архивации
- **Обработка ошибок** с детальным выводом информации о проблемных файлах

## Установка

### Из исходного кода

```bash
CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o ptar .
```
