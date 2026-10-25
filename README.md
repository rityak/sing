# sing

Общая библиотека протоколов (буферы, копирование, адреса).

Форк [metacubex/sing](https://github.com/metacubex/sing) для [miray](https://github.com/rityak/miray). Наш код — в ветке `miray`.

## Что изменено

- `bufio.ChunkReader.ReadBuffer` сначала отдаёт остаток чанка, который оставил в кэше короткий `Read`, и только потом читает следующий чанк. Раньше при буфере не меньше чанка кэш пропускался, и остаток терялся: сервер VMess, перед которым первое чтение короткое (peek ранних данных), терял часть первого чанка клиента.
- Пул буферов держит класс 72 КиБ (`MaxPooledBufferSize`, как sagernet/sing `cbeb0ed`), `buf.NewSize` сначала берёт из пула: полный чанк AEAD (65535 + 16 байт) больше не уходит в кучу на каждое чтение.
- `bufio.Copy`: после `DefaultIncreaseBufferAfter` (512 КБ) чтение идёт буферами в кадр писателя (`ReadWaitOptions.IncreaseBuffer`: MTU писателя или 64 КиБ плюс headroom, не больше `MaxPooledBufferSize`), как sagernet/sing `CopyWithIncreateBuffer`; векторизованный путь upstream не перенесён.
- `udpnat`: адрес источника, на который переехала сессия (NAT rebinding), обновляется под блокировкой — чтение `LocalAddr` обработчиком больше не гонка данных. Поведение то же.
