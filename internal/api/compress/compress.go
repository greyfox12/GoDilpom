package compress

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/exp/slices"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
	//	content string
}

func (w gzipWriter) Write(b []byte) (int, error) {
	// w.Writer будет отвечать за gzip-сжатие, поэтому пишем в него
	//	w.content = w.Header().Get("Content-Type")
	return w.Writer.Write(b)
}

// ////////////////
type flateWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w flateWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Compress сжимает слайс байт.
func Compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	// создаём переменную w — в неё будут записываться входящие данные,
	// которые будут сжиматься и сохраняться в bytes.Buffer
	w, err := flate.NewWriter(&b, flate.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed init compress writer: %w", err)
	}
	// запись данных
	_, err = w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to compress temporary buffer: %w", err)
	}
	// обязательно нужно вызвать метод Close() — в противном случае часть данных
	// может не записаться в буфер b; если нужно выгрузить все упакованные данные
	// в какой-то момент сжатия, используйте метод Flush()
	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %w", err)
	}
	// переменная b содержит сжатые данные
	return b.Bytes(), nil
}

// Decompress распаковывает слайс байт.
func Decompress(data []byte, typCompress string) ([]byte, error) {
	var r io.Reader
	var err error
	switch typCompress {
	case "flate", "gzip":
		// переменная r будет читать входящие данные и распаковывать их
		r = flate.NewReader(bytes.NewReader(data))
		//		defer r.Close()
		//	case "gzip":
		//		r, err = gzip.NewReader(bytes.NewReader(data))
		//		defer r.Close()
		//		if err != nil {
		//			return nil, fmt.Errorf("failed gzip.NewReader: %v", err)
		//		}
	default:
		return nil, fmt.Errorf("failed decompress type: %v", typCompress)
	}

	var b bytes.Buffer
	// в переменную b записываются распакованные данные
	_, err = b.ReadFrom(r)
	if err != nil {
		return nil, fmt.Errorf("failed decompress data: %w", err)
	}

	return b.Bytes(), nil
}

func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// проверяем, что клиент поддерживает gzip-сжатие

		fmt.Printf("GZIP ENTER\n")

		headerCode := strings.Split(r.Header.Get("Accept-Encoding"), ",")
		for i := range headerCode {
			headerCode[i] = strings.TrimSpace(headerCode[i])
		}

		//		fmt.Printf("headerCode=%v\n", headerCode)
		//		fmt.Printf("headerCode[gzip]=%v\n", slices.IndexFunc(headerCode, func(c string) bool { return c == "gzip" }))
		//		fmt.Printf("r.Header.Get(Content-Type)=%v\n", w.Header())

		// Для gzip
		if slices.IndexFunc(headerCode, func(c string) bool { return c == "gzip" }) >= 0 {
			// создаём gzip.Writer поверх текущего w
			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				io.WriteString(w, err.Error())
				return
			}
			defer gz.Close()

			w.Header().Set("Content-Encoding", "gzip")
			// передаём обработчику страницы переменную типа gzipWriter для вывода данных
			next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
			return

		}

		// Для deflate
		if slices.IndexFunc(headerCode, func(c string) bool { return c == "deflate" }) >= 0 {
			// создаём gzip.Writer поверх текущего w
			fl, err := flate.NewWriter(w, flate.BestCompression)
			if err != nil {
				io.WriteString(w, err.Error())
				return
			}
			defer fl.Close()

			w.Header().Set("Content-Encoding", "deflate")
			// передаём обработчику страницы переменную типа gzipWriter для вывода данных
			next.ServeHTTP(flateWriter{ResponseWriter: w, Writer: fl}, r)
			return

		}
		// если gzip не поддерживается, передаём управление
		// дальше без изменений
		next.ServeHTTP(w, r)
	})
}

func GzipRead(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// проверяем, что клиент поддерживает gzip-сжатие

		fmt.Printf("GZIP Read ENTER\n")

		if r.Header.Get("Content-Encoding") == "gzip" || r.Header.Get("Content-Encoding") == "flate" {
			fmt.Printf("GzipRead Header gzip \n")
			r.Body = flate.NewReader(r.Body)
			next.ServeHTTP(w, r)

			return
		}

		// если gzip не поддерживается, передаём управление
		// дальше без изменений
		next.ServeHTTP(w, r)
	})
}
