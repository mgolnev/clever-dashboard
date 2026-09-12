// Package ingestion принимает файл выгрузки Битрикса (HTML-«xls» или CSV) и
// приводит его к нейтральным доменным заказам. Это атомарный модуль приёма
// данных: он не знает о хранилище и метриках.
package ingestion

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Record — одна строка выгрузки: заголовок -> значение ячейки.
type Record map[string]string

func (r Record) get(keys ...string) string {
	for _, k := range keys {
		if v, ok := r[k]; ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	// Поиск по префиксу для длинных/изменчивых заголовков.
	for _, k := range keys {
		for h, v := range r {
			if strings.HasPrefix(h, k) && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

// Stream определяет формат по содержимому и отдаёт строки выгрузки по одной,
// не удерживая весь файл и все записи в памяти. Возвращает число строк данных.
func Stream(r io.Reader, yield func(Record) error) (int, error) {
	buffered := bufio.NewReaderSize(r, 64*1024)
	head, err := buffered.Peek(64 * 1024)
	if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
		return 0, fmt.Errorf("прочитать начало файла: %w", err)
	}
	trimmed := bytes.TrimSpace(head)
	trimmed = bytes.TrimSpace(bytes.TrimPrefix(trimmed, []byte{0xef, 0xbb, 0xbf}))
	if bytes.HasPrefix(trimmed, []byte("PK\x03\x04")) {
		return 0, fmt.Errorf("настоящий XLSX пока не поддерживается; экспортируйте заказы из Битрикса в XLS или CSV")
	}
	if bytes.HasPrefix(trimmed, []byte("<")) {
		return streamHTML(buffered, yield)
	}
	return streamCSV(buffered, head, yield)
}

// ParseFile оставлен как совместимый адаптер для небольших файлов и тестов.
func ParseFile(data []byte) ([]Record, error) {
	records := make([]Record, 0)
	_, err := Stream(bytes.NewReader(data), func(record Record) error {
		records = append(records, record)
		return nil
	})
	return records, err
}

func streamHTML(r io.Reader, yield func(Record) error) (int, error) {
	tokenizer := html.NewTokenizer(r)
	var headers []string
	var row []string
	var cell strings.Builder
	inTable := false
	inRow := false
	inCell := false
	tableDepth := 0
	rows := 0

	finishCell := func() {
		if !inCell {
			return
		}
		row = append(row, strings.TrimSpace(cell.String()))
		cell.Reset()
		inCell = false
	}
	finishRow := func() error {
		finishCell()
		inRow = false
		if len(row) == 0 {
			return nil
		}
		if headers == nil {
			headers = append([]string(nil), row...)
			return nil
		}
		rows++
		return yield(recordFromRow(headers, row))
	}

	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			if err := tokenizer.Err(); err != io.EOF {
				return rows, fmt.Errorf("parse html: %w", err)
			}
			if inRow {
				if err := finishRow(); err != nil {
					return rows, err
				}
			}
			if headers == nil {
				return rows, fmt.Errorf("в файле не найдена таблица с заголовками")
			}
			return rows, nil
		case html.StartTagToken:
			name, _ := tokenizer.TagName()
			tag := string(name)
			if tag == "table" {
				if !inTable {
					inTable = true
					tableDepth = 1
				} else {
					tableDepth++
				}
				continue
			}
			if !inTable || tableDepth != 1 {
				continue
			}
			switch tag {
			case "tr":
				if !inRow {
					inRow = true
					row = nil
				}
			case "td", "th":
				if inRow && !inCell {
					inCell = true
					cell.Reset()
				}
			}
		case html.TextToken:
			if inCell && inTable && tableDepth == 1 {
				cell.Write(tokenizer.Text())
			}
		case html.EndTagToken:
			name, _ := tokenizer.TagName()
			tag := string(name)
			if tag == "table" && inTable {
				tableDepth--
				if tableDepth == 0 {
					if inRow {
						if err := finishRow(); err != nil {
							return rows, err
						}
					}
					if headers == nil {
						return rows, fmt.Errorf("в файле не найдена таблица с заголовками")
					}
					return rows, nil
				}
				continue
			}
			if !inTable || tableDepth != 1 {
				continue
			}
			switch tag {
			case "td", "th":
				finishCell()
			case "tr":
				if inRow {
					if err := finishRow(); err != nil {
						return rows, err
					}
				}
			}
		}
	}
}

func streamCSV(r io.Reader, head []byte, yield func(Record) error) (int, error) {
	// Русские заголовки попадают в просмотренный префикс, поэтому кодировку
	// можно определить до потокового чтения остальных строк.
	if !utf8.Valid(head) {
		r = transform.NewReader(r, charmap.Windows1251.NewDecoder())
	}
	csvReader := csv.NewReader(r)
	csvReader.Comma = detectSeparator(head)
	csvReader.FieldsPerRecord = -1
	csvReader.LazyQuotes = true

	var headers []string
	rows := 0
	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return rows, fmt.Errorf("parse csv: %w", err)
		}
		if headers == nil {
			headers = trimAll(row)
			continue
		}
		rows++
		if err := yield(recordFromRow(headers, row)); err != nil {
			return rows, err
		}
	}
	if headers == nil {
		return rows, fmt.Errorf("пустой CSV без заголовков")
	}
	return rows, nil
}

func detectSeparator(data []byte) rune {
	head := data
	if len(head) > 4096 {
		head = head[:4096]
	}
	if bytes.Count(head, []byte(";")) > bytes.Count(head, []byte(",")) {
		return ';'
	}
	return ','
}

func trimAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.TrimSpace(s)
	}
	return out
}

func recordFromRow(headers, row []string) Record {
	record := make(Record, len(headers))
	for index, header := range headers {
		if index < len(row) {
			record[header] = strings.TrimSpace(row[index])
		}
	}
	return record
}
