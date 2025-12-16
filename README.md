# jpgtools

Переносимый CLI для пакетной работы с JPEG: включает две команды —
`compress` (повторяет функциональность `compress_jpgs.py`) и `overlay`
(`apply_black_overlay.py`). Логика целиком переписана на Go, а mozjpeg
(`cjpeg`, `djpeg`, `jpegtran`) встраивается в бинарь и автоматически
распаковывается во внутренний кэш, поэтому пользователю достаточно
скачать один исполняемый файл.

> ⚠️ Сейчас вместе с репозиторием поставляется архив mozjpeg только для
> `darwin/arm64`. Чтобы поддержать другие платформы, добавьте соответствующие
> сборки в `internal/mozjpeg/assets/<platform>/` и обновите логику выбора.

## Сборка

```bash
# Сборка CLI в текущей директории
go build -o jpgtools ./cmd/jpgtools

# Если окружение тянет устаревший GOROOT (например, go1.25.0),
# подскажите актуальный путь:
env GOCACHE=$(pwd)/.gocache \
    GOROOT=/opt/homebrew/opt/go/libexec \
    go build -o jpgtools ./cmd/jpgtools
```

Полученный бинарь можно копировать на другие машины без дополнительной
установки Python, mozjpeg или ImageMagick.

## Использование

### Сжатие JPEG до заданного размера

```bash
./jpgtools compress \
  --input /path/to/source \
  --output /path/to/output \
  --target-kb 300 \
  --initial-quality 85 \
  --min-quality 55 \
  --quality-step 5 \
  --max-width 2380 \
  --max-height 1600 \
  --min-width 1290 \
  --min-height 800 \
  --recursive \
  --overwrite
```

- Скрипт подбирает масштаб, чтобы уложиться в габариты, а затем запускает
  mozjpeg несколько раз, уменьшая `quality` шагом `quality-step`, пока
  файл не станет ≤ `target-kb`.
- Без `--output` создаётся каталог `./output_YYMMDDhhmm`.
- `--dry-run` только печатает план.

### Чёрный overlay поверх каждого JPEG

```bash
./jpgtools overlay \
  --input /path/to/source \
  --output /path/to/output \
  --alpha 0.2 \
  --quality 95 \
  --recursive \
  --overwrite
```

- Для каждого файла читается оригинальный JPEG, все каналы умножаются на
  `(1 - alpha)` (по умолчанию 20 % затемнения) и результат перекодируется
  через встроенный `cjpeg`.
- Аргументы `--input/--output/--recursive/--overwrite/--dry-run` ведут
  себя так же, как у `compress`.

## Кэш mozjpeg

При первом запуске любой команды встроенный архив распаковывается в
`$JPGTOOLS_CACHE_DIR/<version>/<platform>` (по умолчанию
`~/Library/Caches/jpgtools/...`). Чтобы переиспользовать уже установленный
набор утилит или сбросить кэш, удалите соответствующую директорию или
переопределите переменную `JPGTOOLS_CACHE_DIR`.
