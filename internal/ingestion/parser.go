// Package ingestion принимает файл выгрузки Битрикса (HTML-«xls» или CSV) и
// приводит его к нейтральным доменным заказам. Это атомарный модуль приёма
// данных: он не знает о хранилище и метриках.
package ingestion

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/charmap"
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

// ParseFile определяет формат по содержимому и возвращает строки выгрузки.
func ParseFile(data []byte) ([]Record, error) {
	trimmed := bytes.TrimSpace(data)
	if bytes.HasPrefix(trimmed, []byte("<")) {
		return parseHTML(data)
	}
	return parseCSV(data)
}

func parseHTML(data []byte) ([]Record, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	var headers []string
	var records []Record
	doc.Find("table").First().Find("tr").Each(func(i int, tr *goquery.Selection) {
		var cells []string
		tr.Find("th,td").Each(func(_ int, c *goquery.Selection) {
			cells = append(cells, strings.TrimSpace(c.Text()))
		})
		if len(cells) == 0 {
			return
		}
		if headers == nil {
			headers = cells
			return
		}
		rec := make(Record, len(headers))
		for j, h := range headers {
			if j < len(cells) {
				rec[h] = cells[j]
			}
		}
		records = append(records, rec)
	})
	if headers == nil {
		return nil, fmt.Errorf("в файле не найдена таблица с заголовками")
	}
	return records, nil
}

func parseCSV(data []byte) ([]Record, error) {
	// Битрикс часто отдаёт CSV в windows-1251 с разделителем ';'.
	if !isValidUTF8(data) {
		if dec, err := charmap.Windows1251.NewDecoder().Bytes(data); err == nil {
			data = dec
		}
	}
	sep := detectSeparator(data)
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = sep
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	var headers []string
	var records []Record
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse csv: %w", err)
		}
		if headers == nil {
			headers = trimAll(row)
			continue
		}
		rec := make(Record, len(headers))
		for j, h := range headers {
			if j < len(row) {
				rec[h] = strings.TrimSpace(row[j])
			}
		}
		records = append(records, rec)
	}
	if headers == nil {
		return nil, fmt.Errorf("пустой CSV без заголовков")
	}
	return records, nil
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

func isValidUTF8(b []byte) bool {
	return strings.ToValidUTF8(string(b), "\uFFFD") == string(b) && !bytes.Contains(b, []byte("\uFFFD"))
}

func trimAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.TrimSpace(s)
	}
	return out
}
